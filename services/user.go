package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"citystatAPI/prisma/db"
	"citystatAPI/types"

	"github.com/clerk/clerk-sdk-go/v2/user"
)


type UserUpdateRequest2 struct {
    FirstName                 string    `json:"firstName,omitempty"`
    LastName                  string    `json:"lastName,omitempty"`
    UserName                  string    `json:"userName,omitempty"`
    ImageURL                  string    `json:"imageUrl,omitempty"`
    CompletedTutorial         bool      `json:"completedTutorial,omitempty"`
    IsLocationTrackingEnabled bool      `json:"isLocationTrackingEnabled,omitempty"`
    SelectedCity              *CityData2 `json:"selectedCity,omitempty"`
}

type CityData2 struct {
    Name        string  `json:"name"`
    Country     string  `json:"country"`
    State       string  `json:"state"`
    Lat         float64 `json:"lat"`
    Lng         float64 `json:"lng"`
    DisplayName string  `json:"display_name"`
}

type CityDataProcessingRequest struct {
	UserID string
	User   *db.UserModel
}

type UserService struct {
	client         *db.PrismaClient
	cityDataQueue  chan CityDataProcessingRequest
	shutdownCtx    context.Context
	shutdownCancel context.CancelFunc
}

func (s *UserService) cityDataProcessor() {
	for {
		select {
		case req := <-s.cityDataQueue:
			// Create context with timeout for this specific job
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			
			fmt.Printf("Processing city data for user %s in background\n", req.UserID)
			if err := s.updateUserCityData(ctx, req.User); err != nil {
				fmt.Printf("Failed to process city data for user %s: %v\n", req.UserID, err)
			} else {
				fmt.Printf("Successfully processed city data for user %s\n", req.UserID)
			}
			
			cancel()
			
		case <-s.shutdownCtx.Done():
			fmt.Println("City data processor shutting down")
			return
		}
	}
}

func NewUserService(client *db.PrismaClient) *UserService {
	ctx, cancel := context.WithCancel(context.Background())
	s := &UserService{
		client:         client,
		cityDataQueue:  make(chan CityDataProcessingRequest, 50),
		shutdownCtx:    ctx,
		shutdownCancel: cancel,
	}
	
	// Start background workers
	go s.cityDataProcessor()
	
	return s
}


func (s *UserService) UpdateUserDetails(ctx context.Context, clerkUserID string, updates UserUpdateRequest2 ) (*db.UserModel, error) {
	fmt.Println("updating user")
	fmt.Println(updates)

	// Ensure user exists first
	existingUser, err := s.client.User.FindUnique(
		db.User.ID.Equals(clerkUserID),
	).With(
		db.User.Settings.Fetch(),
	).Exec(ctx)
	if err != nil {
		if err == db.ErrNotFound {
			// User doesn't exist, sync from Clerk first
			user, syncErr := s.SyncUserFromClerk(ctx, clerkUserID)
			if syncErr != nil {
				return nil, fmt.Errorf("failed to sync user from Clerk: %w", syncErr)
			}
			existingUser = user
		} else {
			return nil, fmt.Errorf("error checking existing user: %w", err)
		}
	}

	// Build update operations based on provided fields
	updateOps := []db.UserSetParam{}

	// Handle basic user fields
	if updates.FirstName != "" {
		updateOps = append(updateOps, db.User.FirstName.Set(updates.FirstName))
	}
	if updates.LastName != "" {
		updateOps = append(updateOps, db.User.LastName.Set(updates.LastName))
	}
	if updates.UserName != "" {
		updateOps = append(updateOps, db.User.UserName.Set(updates.UserName))
	}
	if updates.ImageURL != "" {
		updateOps = append(updateOps, db.User.ImageURL.Set(updates.ImageURL))
	}
	
	// Handle CompletedTutorial - check if it's different from existing value
	if existingUser.CompletedTutorial != updates.CompletedTutorial {
		updateOps = append(updateOps, db.User.CompletedTutorial.Set(updates.CompletedTutorial))
	}

	// Handle city data - simplified to only use SelectedCity
	var cityNameForStats *string

	if updates.SelectedCity != nil {
		city := updates.SelectedCity
		
		// Update all city-related fields
		updateOps = append(updateOps, db.User.CityName.Set(city.Name))
		updateOps = append(updateOps, db.User.CityCountry.Set(city.Country))
		updateOps = append(updateOps, db.User.CityLat.Set(city.Lat))
		updateOps = append(updateOps, db.User.CityLng.Set(city.Lng))
		updateOps = append(updateOps, db.User.CityDisplayName.Set(city.DisplayName))

		// Update the legacy city field for backward compatibility
		updateOps = append(updateOps, db.User.City.Set(city.Name))

		// Handle state
		if city.State != "" {
			updateOps = append(updateOps, db.User.CityState.Set(city.State))
		}

		cityNameForStats = &city.Name
		
		fmt.Printf("City data will be updated: %s, %s, %s (%.6f, %.6f)\n", 
			city.Name, city.Country, city.State, city.Lat, city.Lng)
	}

	// Handle location tracking permission update in Settings
	var settingsUpdateNeeded bool
	var settingsUpdateOps []db.SettingsSetParam

	// Check if location tracking permission needs to be updated
	if settings, ok := existingUser.Settings(); ok {
		// Settings exist, check if the value is different
		if settings.EnabledLocationTracking != updates.IsLocationTrackingEnabled {
			settingsUpdateNeeded = true
			settingsUpdateOps = append(settingsUpdateOps,
				db.Settings.EnabledLocationTracking.Set(updates.IsLocationTrackingEnabled),
			)
			fmt.Printf("Location tracking permission will be updated to: %v\n", updates.IsLocationTrackingEnabled)
		}
	} else {
		// No settings exist yet, this shouldn't happen if user was properly created
		// but handle it gracefully
		fmt.Printf("Warning: User %s has no settings, this should not happen\n", clerkUserID)
	}

	// Perform user update if there are changes
	var updatedUser *db.UserModel
	if len(updateOps) > 0 {
		updatedUser, err = s.client.User.FindUnique(
			db.User.ID.Equals(clerkUserID),
		).Update(updateOps...).Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to update user: %w", err)
		}
		fmt.Printf("Successfully updated user %s with %d fields\n", clerkUserID, len(updateOps))
	} else {
		updatedUser = existingUser
		fmt.Printf("No user fields to update for user %s\n", clerkUserID)
	}

	// Update settings if location permission changed
	if settingsUpdateNeeded {
		_, err = s.client.Settings.FindUnique(
			db.Settings.UserID.Equals(clerkUserID),
		).Update(settingsUpdateOps...).Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to update location permission: %w", err)
		}
		fmt.Printf("Successfully updated location tracking permission for user %s to: %v\n", clerkUserID, updates.IsLocationTrackingEnabled)
	}

	// Queue city data processing instead of doing it synchronously
	if cityNameForStats != nil {
		select {
		case s.cityDataQueue <- CityDataProcessingRequest{
			UserID: clerkUserID,
			User:   updatedUser,
		}:
			fmt.Printf("Queued city data processing for user %s\n", clerkUserID)
		default:
			fmt.Printf("City data queue full, will retry city data processing for user %s\n", clerkUserID)
		}

		// Initialize CityStat synchronously (this should be fast)
		err = s.initializeCityStatsIfNeeded(ctx, clerkUserID, cityNameForStats)
		if err != nil {
			// Log error but don't fail the user update
			fmt.Printf("Warning: failed to initialize city stats for user %s: %v\n", clerkUserID, err)
		}
	}

	return updatedUser, nil
}



func (s *UserService) updateUserCityData(ctx context.Context, existingUser *db.UserModel) error {
	cityName, ok := existingUser.City()
	if !ok {
		fmt.Printf("Error: failed to get total user city: %v\n", ok)
		return fmt.Errorf("failed to get city name from user")
	}

	bbox, err := getCityBoundingBox(ctx, cityName)
	if err != nil {
		return fmt.Errorf("failed to get city boundaries: %w", err)
	}

	// Create Overpass query for the city
	overpassQuery := fmt.Sprintf(`
[out:json][timeout:30];
(
  way["highway"~"^(primary|secondary|tertiary|residential|trunk|motorway|unclassified|living_street|service|footway|path|cycleway|track)$"]
    (%f,%f,%f,%f);
);
out geom;
`, bbox.South, bbox.West, bbox.North, bbox.East)

	// Make request to Overpass API
	overpassData, err := queryOverpassAPI(ctx, overpassQuery)
	if err != nil {
		return fmt.Errorf("failed to query Overpass API: %w", err)
	}

	stats := calculateStreetStats(cityName, overpassData)



	_, err = s.client.User.FindUnique(
		db.User.ID.Equals(existingUser.ID),
	).Update(
		db.User.CityAllStreetsCount.Set(stats.TotalStreetsCity),
		db.User.CityAllKilometers.Set(stats.TotalKilometersCity),
		db.User.CityBboxNorth.Set(bbox.North),
		db.User.CityBboxSouth.Set(bbox.South),
		db.User.CityBboxEast.Set(bbox.East),
		db.User.CityBboxWest.Set(bbox.West),
	).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}
//! add the city bounding box to priosma so that you do0nt have to do a call to overpass again

func (s *UserService) Shutdown() {
	fmt.Println("Shutting down UserService...")
	s.shutdownCancel()
	
	// Give some time for current jobs to finish
	time.Sleep(5 * time.Second)
	
	// Drain remaining jobs from queue
	close(s.cityDataQueue)
	for req := range s.cityDataQueue {
		fmt.Printf("Draining queued city data job for user %s\n", req.UserID)
	}
}
// calculateStreetStats processes the Overpass data and calculates statistics
func calculateStreetStats(cityName string, data *types.OverpassResponse) *types.City2MainStats {
	stats := &types.City2MainStats{
		City:        cityName,
		StreetTypes: make(map[string]int),
	}

	var totalDistance float64

	for _, element := range data.Elements {
		if element.Type == "way" && len(element.Geometry) > 1 {
			stats.TotalStreetsCity++

			// Count street types
			highway := element.Tags.Highway
			if highway != "" {
				stats.StreetTypes[highway]++
			}

			// Calculate distance for this way
			wayDistance := 0.0
			for i := 0; i < len(element.Geometry)-1; i++ {
				dist := haversineDistance(
					element.Geometry[i].Lat, element.Geometry[i].Lon,
					element.Geometry[i+1].Lat, element.Geometry[i+1].Lon,
				)
				wayDistance += dist
			}
			totalDistance += wayDistance
		}
	}

	stats.TotalKilometersCity = totalDistance
	return stats
}
func getCityBoundingBox(ctx context.Context, cityName string) (*BoundingBox, error) {
	nominatimURL := fmt.Sprintf(
		"https://nominatim.openstreetmap.org/search?q=%s&format=json&limit=1",
		url.QueryEscape(cityName),
	)

	req, err := http.NewRequestWithContext(ctx, "GET", nominatimURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "CityStreetAnalyzer/1.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var nominatimResults []struct {
		BoundingBox []string `json:"boundingbox"`
	}

	if err := json.Unmarshal(body, &nominatimResults); err != nil {
		return nil, err
	}

	if len(nominatimResults) == 0 {
		return nil, fmt.Errorf("city not found: %s", cityName)
	}

	bbox := nominatimResults[0].BoundingBox
	if len(bbox) != 4 {
		return nil, fmt.Errorf("invalid bounding box data")
	}

	// Parse bounding box coordinates
	south := parseFloat(bbox[0])
	north := parseFloat(bbox[1])
	west := parseFloat(bbox[2])
	east := parseFloat(bbox[3])

	return &BoundingBox{
		North: north,
		South: south,
		East:  east,
		West:  west,
	}, nil
}
// Helper method to initialize CityStat if it doesn't exist
func (s *UserService) initializeCityStatsIfNeeded(ctx context.Context, userID string, cityName *string) error {
	if cityName == nil {
		return nil
	}

	// Check if CityStat already exists
	_, err := s.client.CityStat.FindUnique(
		db.CityStat.UserID.Equals(userID),
	).Exec(ctx)

	if err == db.ErrNotFound {
		// Create new CityStat with proper relation setup
		_, err = s.client.CityStat.CreateOne(
			db.CityStat.City.Set(*cityName),
			db.CityStat.User.Link(
				db.User.ID.Equals(userID),
			),
		).Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to create city stats: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to check existing city stats: %w", err)
	}

	return nil
}

func (s *UserService) SyncUserFromClerk(ctx context.Context, clerkUserID string) (*db.UserModel, error) {
	fmt.Printf("[SyncUserFromClerk] Starting sync for Clerk user ID: %s\n", clerkUserID)

	clerkUser, err := user.Get(ctx, clerkUserID)
	if err != nil {
		fmt.Printf("[SyncUserFromClerk] Error fetching user from Clerk: %v\n", err)
		return nil, fmt.Errorf("failed to fetch user from Clerk: %w", err)
	}
	fmt.Printf("[SyncUserFromClerk] Clerk user fetched: %+v\n", clerkUser)

	var email string
	if len(clerkUser.EmailAddresses) > 0 {
		email = clerkUser.EmailAddresses[0].EmailAddress
		fmt.Printf("[SyncUserFromClerk] Using email: %s\n", email)
	} else {
		fmt.Printf("[SyncUserFromClerk] No email addresses found for Clerk user\n")
	}

	var imageUrl *string
	if clerkUser.ImageURL != nil && *clerkUser.ImageURL != "" {
		imageUrl = clerkUser.ImageURL
		fmt.Printf("[SyncUserFromClerk] Using image URL: %s\n", *imageUrl)
	} else {
		fmt.Printf("[SyncUserFromClerk] No image URL found for Clerk user\n")
	}

	existingUser, err := s.client.User.FindUnique(
		db.User.ID.Equals(clerkUserID),
	).Exec(ctx)
	if err != nil && err != db.ErrNotFound {
		fmt.Printf("[SyncUserFromClerk] Error checking existing user in DB: %v\n", err)
		return nil, fmt.Errorf("error checking existing user: %w", err)
	}
	if existingUser != nil {
		fmt.Printf("[SyncUserFromClerk] User exists in DB, updating...\n")
		updatedUser, err := s.client.User.FindUnique(
			db.User.ID.Equals(clerkUserID),
		).Update(
			db.User.Email.Set(email),
			db.User.FirstName.SetIfPresent(clerkUser.FirstName),
			db.User.LastName.SetIfPresent(clerkUser.LastName),
			db.User.ImageURL.SetIfPresent(imageUrl),
		).Exec(ctx)
		if err != nil {
			fmt.Printf("[SyncUserFromClerk] Failed to update user in DB: %v\n", err)
			return nil, fmt.Errorf("failed to update user: %w", err)
		}
		fmt.Printf("[SyncUserFromClerk] User updated in DB: %+v\n", updatedUser)

		err = s.ensureUserHasSettings(ctx, clerkUserID)
		if err != nil {
			fmt.Printf("[SyncUserFromClerk] Failed to ensure user has settings: %v\n", err)
			return nil, fmt.Errorf("failed to ensure user has settings: %w", err)
		}
		fmt.Printf("[SyncUserFromClerk] User settings ensured\n")
		return updatedUser, nil
	}

	fmt.Printf("[SyncUserFromClerk] User does not exist in DB, creating new user...\n")
	newUser, err := s.client.User.CreateOne(
		db.User.ID.Set(clerkUserID),
		db.User.Email.Set(email),
		db.User.FirstName.SetIfPresent(clerkUser.FirstName),
		db.User.LastName.SetIfPresent(clerkUser.LastName),
		db.User.UserName.SetIfPresent(clerkUser.Username),
		db.User.ImageURL.SetIfPresent(imageUrl),
	).Exec(ctx)
	if err != nil {
		fmt.Printf("[SyncUserFromClerk] Failed to create user in DB: %v\n", err)
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	fmt.Printf("[SyncUserFromClerk] New user created in DB: %+v\n", newUser)

	_, err = s.client.User.FindUnique(db.User.ID.Equals(clerkUserID)).Exec(ctx)
	if err != nil {
		fmt.Printf("[SyncUserFromClerk] User not found after creation when creating settings: %v\n", err)
		val := fmt.Errorf("user not found when creating settings: %w", err)
		return nil, val
	}

	err = s.ensureUserHasSettings(ctx, clerkUserID)
	if err != nil {
		fmt.Printf("[SyncUserFromClerk] Failed to ensure new user has settings: %v\n", err)
		return nil, fmt.Errorf("failed to ensure new user has settings: %w", err)
	}
	fmt.Printf("[SyncUserFromClerk] New user settings ensured\n")

	return newUser, nil
}

func (s *UserService) ensureUserHasSettings(ctx context.Context, userID string) error {
	fmt.Printf("[ensureUserHasSettings] Ensuring settings for user ID: %s\n", userID)
	settings, err := s.client.Settings.FindUnique(
		db.Settings.UserID.Equals(userID),
	).Exec(ctx)

	if err == db.ErrNotFound {
		fmt.Printf("[ensureUserHasSettings] Settings not found for user ID: %s, creating default settings...\n", userID)
		settingsCreate, err := s.client.Settings.CreateOne(
			db.Settings.User.Link(db.User.ID.Equals(userID)),
			db.Settings.Theme.Set(db.ThemeSystem),
			db.Settings.Language.Set(db.LanguageEn),
			db.Settings.TextSize.Set(db.TextSizeMedium),
			db.Settings.FontStyle.Set("default"),
			db.Settings.ZoomLevel.Set("100"),
			db.Settings.ShowRoleColors.Set(db.RoleColorsNexttoname),
			db.Settings.MessagesAllowance.Set(db.MessagesAllowanceAllmsg),
			db.Settings.Motion.Set(db.MotionDontplaygifwhenpossibleshow),
			db.Settings.StickersAnimation.Set(db.StickersAnimationAlways),
			db.Settings.EnabledLocationTracking.Set(false),
			db.Settings.AllowCityStatDataUsage.Set(true),
			db.Settings.AllowDataPersonalizationUsage.Set(true),
			db.Settings.AllowInAppRewards.Set(true),
			db.Settings.AllowDataAnaliticsAndPerformance.Set(true),
			db.Settings.EnableInAppNotifications.Set(true),
			db.Settings.EnableSoundEffects.Set(true),
			db.Settings.EnableVibration.Set(true),
		).Exec(ctx)
		if err != nil {
			fmt.Printf("[ensureUserHasSettings] Failed to create settings for user ID: %s, error: %v\n", userID, err)
			return fmt.Errorf("failed to create settings: %w", err)
		}
		fmt.Printf("[ensureUserHasSettings] Default settings created for user ID: %s: %+v\n", userID, settingsCreate)
	} else if err != nil {
		fmt.Printf("[ensureUserHasSettings] Error checking settings for user ID: %s, error: %v\n", userID, err)
		return fmt.Errorf("error checking settings: %w", err)
	} else {
		fmt.Printf("[ensureUserHasSettings] Settings already exist for user ID: %s: %+v\n", userID, settings)
	}

	return nil
}

func (s *UserService) GetOrCreateUser(ctx context.Context, clerkUserID string) (*db.UserModel, error) {
	user, err := s.client.User.FindUnique(
		db.User.ID.Equals(clerkUserID),
	).With(
		db.User.Settings.Fetch(),
	).Exec(ctx)

	if err == nil {
		return user, nil
	}

	if err == db.ErrNotFound {
		return s.SyncUserFromClerk(ctx, clerkUserID)
	}

	return nil, fmt.Errorf("database error: %w", err)
}

func (s *UserService) UpdateNote(ctx context.Context, clerkUserID string, updates map[string]interface{}) (*db.UserModel, error) {
	note, ok := updates["newNote"].(string)
	if !ok {
		return nil, fmt.Errorf("username field is required and must be a string")
	}

	updatedUser, err := s.client.User.FindUnique(
		db.User.ID.Equals(clerkUserID),
	).Update(
		db.User.Note.Set(note),
	).Exec(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to update note: %w", err)
	}

	return updatedUser, nil
}


func (s *UserService) UpdateActiveHours(ctx context.Context, clerkUserID string, updates map[string]interface{}) (*db.UserModel, error) {
    fmt.Printf("UpdateActiveHours service called with userID: %s, updates: %+v\n", clerkUserID, updates)
    
    activeHoursRaw, ok := updates["activeHours"]
    if !ok {
        return nil, fmt.Errorf("activeHours field is required")
    }

    // Handle different numeric types that might come from JSON
    var activeHours float64
    switch v := activeHoursRaw.(type) {
    case float64:
        activeHours = v
    case float32:
        activeHours = float64(v)
    case int:
        activeHours = float64(v)
    case int64:
        activeHours = float64(v)
    default:
        return nil, fmt.Errorf("activeHours must be a number, got type %T with value %v", v, v)
    }

    fmt.Printf("Parsed activeHours: %f\n", activeHours)

    updatedUser, err := s.client.User.FindUnique(
        db.User.ID.Equals(clerkUserID),
    ).Update(
        db.User.ActiveHours.Set(activeHours),
    ).Exec(ctx)

    if err != nil {
        fmt.Printf("Database error: %v\n", err)
        return nil, fmt.Errorf("failed to update activeHours: %w", err)
    }

    fmt.Printf("Database update successful: %+v\n", updatedUser)
    return updatedUser, nil
}

func (s *UserService) UpdateUserImage(ctx context.Context, clerkUserID string, imageURL string) (*db.UserModel, error) {
	updatedUser, err := s.client.User.FindUnique(
		db.User.ID.Equals(clerkUserID),
	).Update(
		db.User.ImageURL.Set(imageURL),
	).Exec(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to update user image: %w", err)
	}

	return updatedUser, nil
}

// func (s *UserService) UpdateUserSettings(ctx context.Context, clerkUserID string, settingsUpdate map[string]interface{}) (*db.UserModel, error) {
// 	fmt.Println("Updating user settings for:", clerkUserID)
// 	fmt.Println("Settings data:", settingsUpdate)

// 	// Ensure user exists first
// 	existingUser, err := s.client.User.FindUnique(
// 		db.User.ID.Equals(clerkUserID),
// 	).With(
// 		db.User.Settings.Fetch(),
// 	).Exec(ctx)

// 	if err != nil {
// 		if err == db.ErrNotFound {
// 			return nil, fmt.Errorf("user not found")
// 		}
// 		return nil, fmt.Errorf("error checking existing user: %w", err)
// 	}

// 	// Build settings update operations
// 	settingsOps := []db.SettingsSetParam{}

// 	if theme, ok := settingsUpdate["theme"].(db.Theme); ok {
// 		settingsOps = append(settingsOps, db.Settings.Theme.Set(theme))
// 	}
// 	if language, ok := settingsUpdate["language"].(db.Language); ok {
// 		settingsOps = append(settingsOps, db.Settings.Language.Set(language))
// 	}
// 	if textSize, ok := settingsUpdate["textSize"].(db.TextSize); ok {
// 		settingsOps = append(settingsOps, db.Settings.TextSize.Set(textSize))
// 	}
// 	if fontStyle, ok := settingsUpdate["fontStyle"].(string); ok {
// 		settingsOps = append(settingsOps, db.Settings.FontStyle.Set(fontStyle))
// 	}
// 	if zoomLevel, ok := settingsUpdate["zoomLevel"].(string); ok {
// 		settingsOps = append(settingsOps, db.Settings.ZoomLevel.Set(zoomLevel))
// 	}
// 	if showRoleColors, ok := settingsUpdate["showRoleColors"].(db.RoleColors); ok {
// 		settingsOps = append(settingsOps, db.Settings.ShowRoleColors.Set(showRoleColors))
// 	}
// 	if messagesAllowance, ok := settingsUpdate["messagesAllowance"].(db.MessagesAllowance); ok {
// 		settingsOps = append(settingsOps, db.Settings.MessagesAllowance.Set(messagesAllowance))
// 	}
// 	if motion, ok := settingsUpdate["motion"].(db.Motion); ok {
// 		settingsOps = append(settingsOps, db.Settings.Motion.Set(motion))
// 	}
// 	if stickersAnimation, ok := settingsUpdate["stickersAnimation"].(db.StickersAnimation); ok {
// 		settingsOps = append(settingsOps, db.Settings.StickersAnimation.Set(stickersAnimation))
// 	}

// 	// Boolean settings
// 	if enabledLocationTracking, ok := settingsUpdate["enabledLocationTracking"].(bool); ok {
// 		settingsOps = append(settingsOps, db.Settings.EnabledLocationTracking.Set(enabledLocationTracking))
// 	}
// 	if allowCityStatDataUsage, ok := settingsUpdate["allowCityStatDataUsage"].(bool); ok {
// 		settingsOps = append(settingsOps, db.Settings.AllowCityStatDataUsage.Set(allowCityStatDataUsage))
// 	}
// 	if allowDataPersonalizationUsage, ok := settingsUpdate["allowDataPersonalizationUsage"].(bool); ok {
// 		settingsOps = append(settingsOps, db.Settings.AllowDataPersonalizationUsage.Set(allowDataPersonalizationUsage))
// 	}
// 	if allowInAppRewards, ok := settingsUpdate["allowInAppRewards"].(bool); ok {
// 		settingsOps = append(settingsOps, db.Settings.AllowInAppRewards.Set(allowInAppRewards))
// 	}
// 	if allowDataAnaliticsAndPerformance, ok := settingsUpdate["allowDataAnaliticsAndPerformance"].(bool); ok {
// 		settingsOps = append(settingsOps, db.Settings.AllowDataAnaliticsAndPerformance.Set(allowDataAnaliticsAndPerformance))
// 	}
// 	if enableInAppNotifications, ok := settingsUpdate["enableInAppNotifications"].(bool); ok {
// 		settingsOps = append(settingsOps, db.Settings.EnableInAppNotifications.Set(enableInAppNotifications))
// 	}
// 	if enableSoundEffects, ok := settingsUpdate["enableSoundEffects"].(bool); ok {
// 		settingsOps = append(settingsOps, db.Settings.EnableSoundEffects.Set(enableSoundEffects))
// 	}
// 	if enableVibration, ok := settingsUpdate["enableVibration"].(bool); ok {
// 		settingsOps = append(settingsOps, db.Settings.EnableVibration.Set(enableVibration))
// 	}

// 	if len(settingsOps) == 0 {
// 		return existingUser, nil
// 	}

// fmt.Println("settings opts:")
// 	fmt.Println(settingsOps)

// 	// Check if user has settings record
// 	settings, hasSettings := existingUser.Settings()
// 	if !hasSettings || settings == nil {
// 		// Create new settings record
// 		_, err = s.client.Settings.CreateOne(
// 			db.Settings.User.Link(db.User.ID.Equals(clerkUserID)),
// 			settingsOps...,
// 		).Exec(ctx)
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to create settings: %w", err)
// 		}
// 	} else {
// 		// Update existing settings
// 		_, err = s.client.Settings.FindUnique(
// 			db.Settings.UserID.Equals(clerkUserID),
// 		).Update(settingsOps...).Exec(ctx)
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to update settings: %w", err)
// 		}
// 	}

// 	// Return updated user with settings
// 	updatedUser, err := s.client.User.FindUnique(
// 		db.User.ID.Equals(clerkUserID),
// 	).With(
// 		db.User.Settings.Fetch(),
// 		db.User.Friends.Fetch(),
// 		db.User.CityStats.Fetch().With(
// 			db.CityStat.StreetWalks.Fetch(),
// 		),
// 	).Exec(ctx)

// 	if err != nil {
// 		return nil, fmt.Errorf("failed to fetch updated user: %w", err)
// 	}

// 	return updatedUser, nil
// }

func (s *UserService) UpdateUserSettings(ctx context.Context, clerkUserID string, settingsUpdate map[string]interface{}) (*db.UserModel, error) {
	fmt.Println("Updating user settings for:", clerkUserID)
	fmt.Println("Settings data (raw):", settingsUpdate)

	// Extract nested settings map if present
	rawSettings := settingsUpdate
	if nested, ok := settingsUpdate["settings"].(map[string]interface{}); ok {
		rawSettings = nested
	}

	fmt.Println("Parsed settings:", rawSettings)

	// Ensure user exists first
	existingUser, err := s.client.User.FindUnique(
		db.User.ID.Equals(clerkUserID),
	).With(
		db.User.Settings.Fetch(),
	).Exec(ctx)

	if err != nil {
		if err == db.ErrNotFound {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("error checking existing user: %w", err)
	}

	// Build settings update operations
	settingsOps := []db.SettingsSetParam{}

	if themeStr, ok := rawSettings["theme"].(string); ok {
		settingsOps = append(settingsOps, db.Settings.Theme.Set(db.Theme(themeStr)))
	}
	if languageStr, ok := rawSettings["language"].(string); ok {
		settingsOps = append(settingsOps, db.Settings.Language.Set(db.Language(languageStr)))
	}
	if textSizeStr, ok := rawSettings["textSize"].(string); ok {
		settingsOps = append(settingsOps, db.Settings.TextSize.Set(db.TextSize(textSizeStr)))
	}
	if fontStyle, ok := rawSettings["fontStyle"].(string); ok {
		settingsOps = append(settingsOps, db.Settings.FontStyle.Set(fontStyle))
	}
	if zoomLevel, ok := rawSettings["zoomLevel"].(string); ok {
		settingsOps = append(settingsOps, db.Settings.ZoomLevel.Set(zoomLevel))
	}
	if showRoleColorsStr, ok := rawSettings["showRoleColors"].(string); ok {
		settingsOps = append(settingsOps, db.Settings.ShowRoleColors.Set(db.RoleColors(showRoleColorsStr)))
	}
	if messagesAllowanceStr, ok := rawSettings["messagesAllowance"].(string); ok {
		settingsOps = append(settingsOps, db.Settings.MessagesAllowance.Set(db.MessagesAllowance(messagesAllowanceStr)))
	}
	if motionStr, ok := rawSettings["motion"].(string); ok {
		settingsOps = append(settingsOps, db.Settings.Motion.Set(db.Motion(motionStr)))
	}
	if stickersAnimationStr, ok := rawSettings["stickersAnimation"].(string); ok {
		settingsOps = append(settingsOps, db.Settings.StickersAnimation.Set(db.StickersAnimation(stickersAnimationStr)))
	}

	// Boolean settings
	if enabledLocationTracking, ok := rawSettings["enabledLocationTracking"].(bool); ok {
		settingsOps = append(settingsOps, db.Settings.EnabledLocationTracking.Set(enabledLocationTracking))
	}
	if allowCityStatDataUsage, ok := rawSettings["allowCityStatDataUsage"].(bool); ok {
		settingsOps = append(settingsOps, db.Settings.AllowCityStatDataUsage.Set(allowCityStatDataUsage))
	}
	if allowDataPersonalizationUsage, ok := rawSettings["allowDataPersonalizationUsage"].(bool); ok {
		settingsOps = append(settingsOps, db.Settings.AllowDataPersonalizationUsage.Set(allowDataPersonalizationUsage))
	}
	if allowInAppRewards, ok := rawSettings["allowInAppRewards"].(bool); ok {
		settingsOps = append(settingsOps, db.Settings.AllowInAppRewards.Set(allowInAppRewards))
	}
	if allowDataAnaliticsAndPerformance, ok := rawSettings["allowDataAnaliticsAndPerformance"].(bool); ok {
		settingsOps = append(settingsOps, db.Settings.AllowDataAnaliticsAndPerformance.Set(allowDataAnaliticsAndPerformance))
	}
	if enableInAppNotifications, ok := rawSettings["enableInAppNotifications"].(bool); ok {
		settingsOps = append(settingsOps, db.Settings.EnableInAppNotifications.Set(enableInAppNotifications))
	}
	if enableSoundEffects, ok := rawSettings["enableSoundEffects"].(bool); ok {
		settingsOps = append(settingsOps, db.Settings.EnableSoundEffects.Set(enableSoundEffects))
	}
	if enableVibration, ok := rawSettings["enableVibration"].(bool); ok {
		settingsOps = append(settingsOps, db.Settings.EnableVibration.Set(enableVibration))
	}

	if len(settingsOps) == 0 {
		fmt.Println("No valid settings provided to update.")
		return existingUser, nil
	}

	fmt.Println("Settings operations to apply:", settingsOps)

	// Check if user has settings record
	settings, hasSettings := existingUser.Settings()
	if !hasSettings || settings == nil {
		fmt.Println("Creating new settings record")
		_, err = s.client.Settings.CreateOne(
			db.Settings.User.Link(db.User.ID.Equals(clerkUserID)),
			settingsOps...,
		).Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create settings: %w", err)
		}
	} else {
		fmt.Println("Updating existing settings record")
		_, err = s.client.Settings.FindUnique(
			db.Settings.UserID.Equals(clerkUserID),
		).Update(settingsOps...).Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to update settings: %w", err)
		}
	}

	// Return updated user with settings
	updatedUser, err := s.client.User.FindUnique(
		db.User.ID.Equals(clerkUserID),
	).With(
		db.User.Settings.Fetch(),
		db.User.Friends.Fetch(),
	).Exec(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch updated user: %w", err)
	}

	return updatedUser, nil
}

// UpdateUserProfile handles mixed user and settings updates
// func (s *UserService) UpdateUserProfile(ctx context.Context, clerkUserID string, updates map[string]interface{}) (*db.UserModel, error) {
// 	fmt.Println("Updating user profile for:", clerkUserID)
// 	fmt.Println("Profile data:", updates)

// 	// Check if this is a settings-only update
// 	if settingsData, hasSettings := updates["settings"]; hasSettings {
// 		if settingsMap, ok := settingsData.(map[string]interface{}); ok {
// 			return s.UpdateUserSettings(ctx, clerkUserID, settingsMap)
// 		}
// 	}

// 	// Handle regular user field updates
// 	return s.UpdateUserDetails(ctx, clerkUserID, types.UserUpdateRequest{
// 		FirstName:         getStringPointer(updates, "firstName"),
// 		LastName:          getStringPointer(updates, "lastName"),
// 		UserName:          getStringPointer(updates, "userName"),
// 		ImageURL:          getStringPointer(updates, "imageURL"),
// 		CompletedTutorial: getBoolPointer(updates, "completedTutorial"),
// 	})
// }

// Helper functions
func getStringPointer(data map[string]interface{}, key string) *string {
	if val, ok := data[key].(string); ok {
		return &val
	}
	return nil
}

func getBoolPointer(data map[string]interface{}, key string) *bool {
	if val, ok := data[key].(bool); ok {
		return &val
	}
	return nil
}



// UserSearchResultWithScore represents a search result with ranking score
type UserSearchResultWithScore struct {
	types.UserSearchResult
	Score int // Higher score = better match
}

func (s *UserService) SearchUsers(ctx context.Context, currentUserID, username string) ([]types.UserSearchResult, error) {
	// Get current user's friends to check friend status
	currentUserFriends, err := s.client.Friend.FindMany(
		db.Friend.UserID.Equals(currentUserID),
	).Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user friends: %w", err)
	}

	// Create a map for quick friend lookup
	friendMap := make(map[string]bool)
	for _, friend := range currentUserFriends {
		friendMap[friend.FriendID] = true
	}

	// Normalize search term
	searchTerm := strings.ToLower(strings.TrimSpace(username))
	if searchTerm == "" {
		return []types.UserSearchResult{}, nil
	}
	
	// Search across username, first name, and last name fields
	// Using OR conditions to cast a wider net for fuzzy matching
	users, err := s.client.User.FindMany(
		db.User.And(
			db.User.Or(
				// Search in username field
				db.User.UserName.Contains(username),
				// Search in first name field
				db.User.FirstName.Contains(username),
				// Search in last name field  
				db.User.LastName.Contains(username),
				// Search for cases where search term might match "FirstName LastName"
				db.User.Or(
					db.User.And(
						db.User.FirstName.StartsWith(strings.Split(username, " ")[0]),
						db.User.LastName.StartsWith(func() string {
							parts := strings.Split(username, " ")
							if len(parts) > 1 {
								return parts[1]
							}
							return ""
						}()),
					),
				),
			),
			db.User.ID.Not(currentUserID), // Exclude current user
		),
	).Take(100).Exec(ctx) // Get more results for better ranking

	if err != nil {
		return nil, fmt.Errorf("failed to search users: %w", err)
	}

	// Score and rank results
	var scoredResults []UserSearchResultWithScore
	for _, user := range users {
		firstName, _ := user.FirstName()
		lastName, _ := user.LastName()
		userName, _ := user.UserName()
		imageURL := user.ImageURL

		// Calculate match score across all searchable fields
		score := calculateMultiFieldMatchScore(searchTerm, userName, firstName, lastName)
		
		// Only include results with a reasonable score
		if score > 0 {
			result := UserSearchResultWithScore{
				UserSearchResult: types.UserSearchResult{
					ID:        user.ID,
					UserName:  &userName,
					FirstName: &firstName,
					LastName:  &lastName,
					ImageURL:  &imageURL,
					IsFriend:  friendMap[user.ID],
				},
				Score: score,
			}
			scoredResults = append(scoredResults, result)
		}
	}

	// Sort by score (highest first), then by username for consistency
	sort.Slice(scoredResults, func(i, j int) bool {
		if scoredResults[i].Score != scoredResults[j].Score {
			return scoredResults[i].Score > scoredResults[j].Score
		}
		// Secondary sort by username for consistent ordering
		return strings.ToLower(*scoredResults[i].UserName) < strings.ToLower(*scoredResults[j].UserName)
	})

	// Convert back to regular results and limit to 10
	results := make([]types.UserSearchResult, 0, 10)
	for i, scored := range scoredResults {
		if i >= 10 {
			break
		}
		results = append(results, scored.UserSearchResult)
	}

	return results, nil
}

// calculateMultiFieldMatchScore returns a score for how well the user matches the search term
// Searches across username, first name, last name, and full name combinations
func calculateMultiFieldMatchScore(searchTerm, userName, firstName, lastName string) int {
	if searchTerm == "" {
		return 0
	}

	score := 0
	searchLower := strings.ToLower(searchTerm)
	userNameLower := strings.ToLower(userName)
	firstNameLower := strings.ToLower(firstName)
	lastNameLower := strings.ToLower(lastName)
	fullNameLower := strings.ToLower(strings.TrimSpace(firstName + " " + lastName))

	// USERNAME MATCHING (highest priority)
	if userNameLower == searchLower {
		score += 1000 // Exact username match
	} else if strings.HasPrefix(userNameLower, searchLower) {
		score += 800 // Username starts with search
	} else if strings.Contains(userNameLower, searchLower) {
		score += 600 // Username contains search
	}

	// FIRST NAME MATCHING
	if firstNameLower == searchLower {
		score += 500 // Exact first name match
	} else if strings.HasPrefix(firstNameLower, searchLower) {
		score += 400 // First name starts with search
	} else if strings.Contains(firstNameLower, searchLower) {
		score += 300 // First name contains search
	}

	// LAST NAME MATCHING
	if lastNameLower == searchLower {
		score += 500 // Exact last name match
	} else if strings.HasPrefix(lastNameLower, searchLower) {
		score += 400 // Last name starts with search
	} else if strings.Contains(lastNameLower, searchLower) {
		score += 300 // Last name contains search
	}

	// FULL NAME MATCHING (for searches like "John Doe")
	if fullNameLower == searchLower {
		score += 900 // Exact full name match
	} else if strings.HasPrefix(fullNameLower, searchLower) {
		score += 700 // Full name starts with search
	} else if strings.Contains(fullNameLower, searchLower) {
		score += 500 // Full name contains search
	}

	// Handle partial full name searches (e.g., "John D" matching "John Doe")
	searchParts := strings.Fields(searchLower)
	if len(searchParts) >= 2 {
		firstPart := searchParts[0]
		secondPart := searchParts[1]
		
		if strings.HasPrefix(firstNameLower, firstPart) && strings.HasPrefix(lastNameLower, secondPart) {
			score += 850 // Both first and last name match parts
		}
	}

	// FUZZY MATCHING BONUSES
	// Bonus for similar length usernames
	usernameLengthDiff := abs(len(searchLower) - len(userNameLower))
	if usernameLengthDiff <= 2 && len(searchLower) > 2 {
		score += 100 - (usernameLengthDiff * 20)
	}

	// Levenshtein distance for username (for typo tolerance)
	if score < 600 && len(searchLower) > 2 { // Only for non-obvious matches
		distance := levenshteinDistance(searchLower, userNameLower)
		maxLen := max(len(searchLower), len(userNameLower))
		
		if maxLen > 0 {
			similarity := float64(maxLen-distance) / float64(maxLen)
			if similarity > 0.7 { // 70% similarity threshold
				score += int(similarity * 150)
			}
		}
	}

	// Bonus for friends (social relevance)
	// Note: This would need to be passed in if you want to prioritize friends
	// if isFriend {
	//     score += 50
	// }

	return score
}

// Helper functions
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// levenshteinDistance calculates the Levenshtein distance between two strings
func levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
	}

	for i := 0; i <= len(s1); i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= len(s2); j++ {
		matrix[0][j] = j
	}

	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}

			matrix[i][j] = min3(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

func min3(a, b, c int) int {
	if a <= b && a <= c {
		return a
	} else if b <= c {
		return b
	}
	return c
}

func (s *UserService) GetUsersSameCity(ctx context.Context, clerkUserID string) ([]types.UserSearchResult, error) {
	fmt.Printf("Getting users in same city for: %s\n", clerkUserID)

	// context timeout to prevent hanging requests
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	currentUser, err := s.client.User.FindUnique(
		db.User.ID.Equals(clerkUserID),
	).Exec(ctx)
	if err != nil {
		if err == db.ErrNotFound {
			fmt.Printf("Current user not found: %s\n", clerkUserID)
			return nil, fmt.Errorf("current user not found")
		}
		fmt.Printf("Database error fetching current user: %v\n", err)
		return nil, fmt.Errorf("failed to fetch current user: %w", err)
	}

	cityName, hasCity := currentUser.CityName()
	if !hasCity || cityName == "" {
		fmt.Printf("User %s has no city set\n", clerkUserID)
		return nil, fmt.Errorf("current user has no city set")
	}

	fmt.Printf("Searching for users in city: %s\n", cityName)

	//  Fetch current user's friends (both directions)
	friendIDs := make(map[string]struct{})

	// Add current user to excluded IDs to avoid returning themselves
	friendIDs[clerkUserID] = struct{}{}

	// Fetch outgoing friendships
	friendships, err := s.client.Friend.FindMany(
		db.Friend.UserID.Equals(clerkUserID),
	).Exec(ctx)
	if err != nil {
		fmt.Printf("Warning: failed to fetch outgoing friendships: %v\n", err)
	} else {
		for _, f := range friendships {
			friendIDs[f.FriendID] = struct{}{}
		}
	}

	// Fetch incoming friendships
	friendOf, err := s.client.Friend.FindMany(
		db.Friend.FriendID.Equals(clerkUserID),
	).Exec(ctx)
	if err != nil {
		fmt.Printf("Warning: failed to fetch incoming friendships: %v\n", err)
		// Don't return error here - continue without friend filtering
	} else {
		for _, f := range friendOf {
			friendIDs[f.UserID] = struct{}{}
		}
	}

	// Convert to slice for query
	excludedIDs := make([]string, 0, len(friendIDs))
	for id := range friendIDs {
		excludedIDs = append(excludedIDs, id)
	}

	fmt.Printf("Excluding %d user IDs from results\n", len(excludedIDs))

	// 3. Fetch users in same city - with better query structure
	var usersInCity []db.UserModel
	var queryErr error

	if len(excludedIDs) > 0 {
		// Query with exclusions
		usersInCity, queryErr = s.client.User.FindMany(
			db.User.And(
				db.User.CityName.Equals(cityName),
				db.User.ID.NotIn(excludedIDs),
			),
		).Take(20).Exec(ctx)
	} else {
		// Query without exclusions (fallback)
		usersInCity, queryErr = s.client.User.FindMany(
			db.User.And(
				db.User.CityName.Equals(cityName),
				db.User.ID.Not(clerkUserID), // Just exclude current user
			),
		).Take(20).Exec(ctx)
	}

	if queryErr != nil {
		fmt.Printf("Database error fetching users in same city: %v\n", queryErr)
		return nil, fmt.Errorf("failed to fetch users in same city: %w", queryErr)
	}

	fmt.Printf("Found %d users in city %s\n", len(usersInCity), cityName)

	//  Convert to response format with null safety
	results := make([]types.UserSearchResult, len(usersInCity))
	for i, user := range usersInCity {
		// Handle optional fields safely
		var firstName, lastName, userName string
		var imageURL string

		if fn, hasFN := user.FirstName(); hasFN {
			firstName = fn
		}
		if ln, hasLN := user.LastName(); hasLN {
			lastName = ln
		}
		if un, hasUN := user.UserName(); hasUN {
			userName = un
		}

		imageURL = user.ImageURL

		results[i] = types.UserSearchResult{
			ID:        user.ID,
			UserName:  &userName,
			FirstName: &firstName,
			LastName:  &lastName,
			ImageURL:  &imageURL,
			IsFriend:  false,
		}
	}

	return results, nil
}

// Add this health check method to your UserService
func (s *UserService) HealthCheck(ctx context.Context) error {
	// Simple query to check database connection
	_, err := s.client.User.FindFirst().Exec(ctx)
	if err != nil && err != db.ErrNotFound {
		return err
	}
	return nil
}
