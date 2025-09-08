package services

import (
	"citystatAPI/prisma/db"
	"citystatAPI/types"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

type AnalyticsService struct {
	client *db.PrismaClient
}

func NewAnalyticsService(client *db.PrismaClient) *AnalyticsService {
	return &AnalyticsService{client: client}
}
func (s *AnalyticsService) GetMain2Stats(ctx context.Context, clerkUserID string) (*types.City2MainStats, error) {
	// Get user data including cached city stats and bounding box
	currUser, err := s.client.User.FindUnique(
		db.User.ID.Equals(clerkUserID),
	).Select(
		db.User.City.Field(),
		db.User.CityAllStreetsCount.Field(),
		db.User.CityAllKilometers.Field(),
		db.User.CityBboxNorth.Field(),
		db.User.CityBboxSouth.Field(),
		db.User.CityBboxEast.Field(),
		db.User.CityBboxWest.Field(),
	).Exec(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get user data: %w", err)
	}

	cityName, ok := currUser.City()
	if !ok {
		return nil, fmt.Errorf("user has no city set")
	}

	// Check if we have cached city data
	totalStreetsCity, hasCachedStreets := currUser.CityAllStreetsCount()
	totalKilometersCity, hasCachedKilometers := currUser.CityAllKilometers()
	
	var bbox *BoundingBox
	
	// Try to get bounding box from database first
	if north, hasNorth := currUser.CityBboxNorth(); hasNorth {
		if south, hasSouth := currUser.CityBboxSouth(); hasSouth {
			if east, hasEast := currUser.CityBboxEast(); hasEast {
				if west, hasWest := currUser.CityBboxWest(); hasWest {
					bbox = &BoundingBox{
						North: north,
						South: south,
						East:  east,
						West:  west,
					}
				}
			}
		}
	}

	// If we don't have bounding box cached, fetch it from API
	if bbox == nil {
		fetchedBbox, err := getCityBoundingBox(ctx, cityName)
		if err != nil {
			return nil, fmt.Errorf("failed to get city boundaries: %w", err)
		}
		bbox = fetchedBbox
		
		// Cache the bounding box for future use
		go func() {
			bgCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			
			_, err := s.client.User.FindUnique(
				db.User.ID.Equals(clerkUserID),
			).Update(
				db.User.CityBboxNorth.Set(bbox.North),
				db.User.CityBboxSouth.Set(bbox.South),
				db.User.CityBboxEast.Set(bbox.East),
				db.User.CityBboxWest.Set(bbox.West),
			).Exec(bgCtx)
			
			if err != nil {
				fmt.Printf("Failed to cache bounding box for user %s: %v\n", clerkUserID, err)
			}
		}()
	}

	// Get user's visited streets data
	userVisitedStreets, err := s.getUserVisitedStreets(ctx, clerkUserID, bbox)
	if err != nil {
		return nil, fmt.Errorf("failed to get user visited streets: %w", err)
	}

	// If we have cached city data, use it
	if hasCachedStreets && hasCachedKilometers {
		stats := &types.City2MainStats{
			City:                cityName,
			TotalStreetsCity:    totalStreetsCity,
			TotalKilometersCity: totalKilometersCity,
			StreetTypes:         make(map[string]int), // We could cache this too if needed
		}

		// Calculate user-specific stats
		s.calculateUserStats(stats, userVisitedStreets)
		
		return stats, nil
	}

	// Fallback: If no cached data, fetch from Overpass API
	fmt.Printf("No cached city data for user %s, fetching from Overpass API\n", clerkUserID)
	
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
		return nil, fmt.Errorf("failed to query Overpass API: %w", err)
	}

	// Calculate statistics including user coverage
	stats := calculateStreetStatsWithUserData(cityName, overpassData, userVisitedStreets)

	// Cache the city data for future use
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		_, err := s.client.User.FindUnique(
			db.User.ID.Equals(clerkUserID),
		).Update(
			db.User.CityAllStreetsCount.Set(stats.TotalStreetsCity),
			db.User.CityAllKilometers.Set(stats.TotalKilometersCity),
		).Exec(bgCtx)
		
		if err != nil {
			fmt.Printf("Failed to cache city stats for user %s: %v\n", clerkUserID, err)
		}
	}()

	return stats, nil
}

func calculateStreetStatsWithUserData(cityName string, cityData *types.OverpassResponse, userVisitedStreets []UserVisitedStreet) *types.City2MainStats {
	stats := &types.City2MainStats{
		City:        cityName,
		StreetTypes: make(map[string]int),
	}

	var totalCityDistance float64
	
	// Create a map to store all city streets for matching
	cityStreets := make(map[string]*types.Element)
	
	// Process city data
	for _, element := range cityData.Elements {
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
			totalCityDistance += wayDistance
			
			// Store street with its distance for user matching
			streetKey := fmt.Sprintf("%d", element.ID)
			cityStreets[streetKey] = &element
		}
	}

	// Process user visited streets
	visitedStreetIDs := make(map[string]bool)
	var totalUserDistance float64

	for _, userStreet := range userVisitedStreets {
		// Deduplicate by street ID
		if !visitedStreetIDs[userStreet.StreetID] {
			visitedStreetIDs[userStreet.StreetID] = true
			stats.TotalStreetsCovered++

			// Try to match with city streets to calculate distance
			if cityElement, exists := cityStreets[userStreet.StreetID]; exists {
				streetDistance := calculateStreetDistance(cityElement)
				totalUserDistance += streetDistance
			} else {
				// Fallback: estimate distance based on duration and average walking speed
				if userStreet.Duration != nil && *userStreet.Duration > 0 {
					// Assume average walking speed of 5 km/h
					estimatedDistance := (float64(*userStreet.Duration) / 3600.0) * 5.0
					totalUserDistance += estimatedDistance
				}
			}
		}
	}

	// Alternative approach: Calculate user coverage based on unique street names
	// This is more accurate if street IDs don't match between your data and OSM
	if stats.TotalStreetsCovered == 0 {
		visitedStreetNames := make(map[string]bool)
		for _, userStreet := range userVisitedStreets {
			if !visitedStreetNames[userStreet.StreetName] {
				visitedStreetNames[userStreet.StreetName] = true
				stats.TotalStreetsCovered++
				
				// Estimate distance based on street name matching with OSM data
				distance := estimateDistanceByStreetName(userStreet.StreetName, cityData)
				totalUserDistance += distance
			}
		}
	}

	// Set final values
	stats.TotalKilometersCity = totalCityDistance
	stats.TotalKilometersCovered = totalUserDistance
	
	// Calculate percentage coverage
	if stats.TotalStreetsCity > 0 {
		stats.PercentCityStreetCouverage = (float64(stats.TotalStreetsCovered) / float64(stats.TotalStreetsCity)) * 100
	}

	return stats
}

// calculateStreetDistance calculates the total distance of a street from its geometry
func calculateStreetDistance(element *types.Element) float64 {
	if len(element.Geometry) < 2 {
		return 0.0
	}
	
	var totalDistance float64
	for i := 0; i < len(element.Geometry)-1; i++ {
		dist := haversineDistance(
			element.Geometry[i].Lat, element.Geometry[i].Lon,
			element.Geometry[i+1].Lat, element.Geometry[i+1].Lon,
		)
		totalDistance += dist
	}
	return totalDistance
}

func estimateDistanceByStreetName(streetName string, cityData *types.OverpassResponse) float64 {
	for _, element := range cityData.Elements {
		if element.Type == "way" && element.Tags.Name == streetName {
			return calculateStreetDistance(&element)
		}
	}
	
	// Fallback: return average street length (adjust based on your city)
	return 0.1 // 100 meters as default
}

// getUserVisitedStreets retrieves the streets that the user has visited within the city bounds
func (s *AnalyticsService) getUserVisitedStreets(ctx context.Context, clerkUserID string, bbox *BoundingBox) ([]UserVisitedStreet, error) {
	// Convert Decimal lat/lng to float64 for comparison
	southDecimal := decimal.NewFromFloat(bbox.South)
	northDecimal := decimal.NewFromFloat(bbox.North)
	westDecimal := decimal.NewFromFloat(bbox.West)
	eastDecimal := decimal.NewFromFloat(bbox.East)

	visitedStreets, err := s.client.VisitedStreet.FindMany(
		db.VisitedStreet.UserID.Equals(clerkUserID),
		db.VisitedStreet.EntryLatitude.Gte(southDecimal),
		db.VisitedStreet.EntryLatitude.Lte(northDecimal),
		db.VisitedStreet.EntryLongitude.Gte(westDecimal),
		db.VisitedStreet.EntryLongitude.Lte(eastDecimal),
	).Exec(ctx)

	if err != nil {
		return nil, err
	}

	// Convert to our working type
	var userStreets []UserVisitedStreet
	for _, street := range visitedStreets {
		entryLat, _ := street.EntryLatitude.Float64()
		entryLng, _ := street.EntryLongitude.Float64()
		drtS,_ := street.DurationSeconds()
		
		userStreets = append(userStreets, UserVisitedStreet{
			StreetID:   street.StreetID,
			StreetName: street.StreetName,
			EntryLat:   entryLat,
			EntryLng:   entryLng,
			Duration:   &drtS,
		})
	}

	return userStreets, nil
}

type UserVisitedStreet struct {
	StreetID   string
	StreetName string
	EntryLat   float64
	EntryLng   float64
	Duration   *int // in seconds, can be nil
}

// calculateUserStats calculates user-specific statistics when we have cached city data
func (s *AnalyticsService) calculateUserStats(stats *types.City2MainStats, userVisitedStreets []UserVisitedStreet) {
	visitedStreetIDs := make(map[string]bool)
	var totalUserDistance float64

	for _, userStreet := range userVisitedStreets {
		// Deduplicate by street ID
		if !visitedStreetIDs[userStreet.StreetID] {
			visitedStreetIDs[userStreet.StreetID] = true
			stats.TotalStreetsCovered++

			// Estimate distance based on duration and average walking speed
			if userStreet.Duration != nil && *userStreet.Duration > 0 {
				// Assume average walking speed of 5 km/h
				estimatedDistance := (float64(*userStreet.Duration) / 3600.0) * 5.0
				totalUserDistance += estimatedDistance
			} else {
				// Fallback: use average street length
				totalUserDistance += 0.1 // 100 meters as default
			}
		}
	}

	// Alternative approach: Calculate user coverage based on unique street names if no street IDs
	if stats.TotalStreetsCovered == 0 {
		visitedStreetNames := make(map[string]bool)
		for _, userStreet := range userVisitedStreets {
			if !visitedStreetNames[userStreet.StreetName] {
				visitedStreetNames[userStreet.StreetName] = true
				stats.TotalStreetsCovered++
				// Use average street length estimation
				totalUserDistance += 0.15 // 150 meters as average street length
			}
		}
	}

	// Set final values
	stats.TotalKilometersCovered = totalUserDistance
	
	// Calculate percentage coverage
	if stats.TotalStreetsCity > 0 {
		stats.PercentCityStreetCouverage = (float64(stats.TotalStreetsCovered) / float64(stats.TotalStreetsCity)) * 100
	}
}




// queryOverpassAPI makes the POST request to Overpass API (recreating your JS fetch)
func queryOverpassAPI(ctx context.Context, overpassQuery string) (*types.OverpassResponse, error) {
	// Prepare form data (equivalent to your JS body)
	formData := url.Values{}
	formData.Set("data", overpassQuery)

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		"https://overpass-api.de/api/interpreter",
		strings.NewReader(formData.Encode()),
	)
	if err != nil {
		return nil, err
	}

	// Set headers (equivalent to your JS headers)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "CityStreetAnalyzer/1.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("overpass API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var overpassResp types.OverpassResponse
	if err := json.Unmarshal(body, &overpassResp); err != nil {
		return nil, fmt.Errorf("failed to parse Overpass response: %w", err)
	}

	return &overpassResp, nil
}


// haversineDistance calculates the distance between two points in kilometers
func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // Earth's radius in kilometers

	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180

	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

// Helper function to parse float from string
func parseFloat(s string) float64 {
	if val, err := fmt.Sscanf(s, "%f", new(float64)); err == nil && val == 1 {
		var result float64
		fmt.Sscanf(s, "%f", &result)
		return result
	}
	return 0.0
}

type BoundingBox struct {
	North, South, East, West float64
}

// MonthlyIntervalData represents the data structure for chart visualization
type MonthlyIntervalData struct {
	CurrentMonth  MonthIntervals `json:"currentMonth"`
	PreviousMonth MonthIntervals `json:"previousMonth"`
}

type MonthIntervals struct {
	Month     string         `json:"month"`     // "2024-09" format
	MonthName string         `json:"monthName"` // "September 2024"
	Intervals map[string]int `json:"intervals"` // interval -> count
	Total     int            `json:"total"`
}

type IntervalRange struct {
	Start int
	End   int
	Label string
}

func (s *AnalyticsService) 	GetMainRadarChartData(ctx context.Context, clerkUserID string) (*MonthlyIntervalData, error) {

	// Define the intervals
	intervals := []IntervalRange{
		{Start: 1, End: 6, Label: "1-6"},
		{Start: 7, End: 11, Label: "7-11"},
		{Start: 12, End: 16, Label: "12-16"},
		{Start: 17, End: 21, Label: "17-21"},
		{Start: 22, End: 26, Label: "22-26"},
		{Start: 27, End: 31, Label: "27-31"},
	}

	// Calculate date range (past 2 months)
	now := time.Now()
	startDate := now.AddDate(0, -2, 0).Truncate(24 * time.Hour)

	// Get current and previous month info
	currentMonth := now.AddDate(0, -1, 0)  // Last month
	previousMonth := now.AddDate(0, -2, 0) // 2 months ago
	fmt.Println("-------DEBUG------")
	fmt.Println(currentMonth)
	fmt.Println(previousMonth)

	// Query visited streets for the past 2 months
	visitedStreets, err := s.client.VisitedStreet.FindMany(
		db.VisitedStreet.UserID.Equals(clerkUserID),
		db.VisitedStreet.CreatedAt.Gte(startDate),
	).Exec(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get visited streets: %w", err)
	}

	if visitedStreets == nil {
		return nil, fmt.Errorf("user not found")
	}

	// Initialize result structure
	result := &MonthlyIntervalData{
		CurrentMonth: MonthIntervals{
			Month:     currentMonth.Format("2006-01"),
			MonthName: currentMonth.Format("January 2006"),
			Intervals: make(map[string]int),
			Total:     0,
		},
		PreviousMonth: MonthIntervals{
			Month:     previousMonth.Format("2006-01"),
			MonthName: previousMonth.Format("January 2006"),
			Total:     0,
			Intervals: make(map[string]int),
		},
	}

	// Initialize all intervals with 0
	for _, interval := range intervals {
		result.CurrentMonth.Intervals[interval.Label] = 0
		result.PreviousMonth.Intervals[interval.Label] = 0
	}

	// Process visited streets
	for _, visitedStreet := range visitedStreets {
		fmt.Println(visitedStreet)
		visitDate := visitedStreet.CreatedAt
				fmt.Println(visitDate)

		day := visitDate.Day()

		// Determine which interval this day falls into
		intervalLabel := getIntervalLabel(day, intervals)

		// Determine which month this belongs to
		if isSameMonth(visitDate, currentMonth) {
			result.CurrentMonth.Intervals[intervalLabel]++
			result.CurrentMonth.Total++
		} else if isSameMonth(visitDate, previousMonth) {
			result.PreviousMonth.Intervals[intervalLabel]++
			result.PreviousMonth.Total++
		}
	}

	return result, nil
}

// Helper function to determine which interval a day belongs to
func getIntervalLabel(day int, intervals []IntervalRange) string {
	for _, interval := range intervals {
		if day >= interval.Start && day <= interval.End {
			return interval.Label
		}
	}
	return "unknown" // shouldn't happen for valid days 1-31
}

// Helper function to check if two dates are in the same month
func isSameMonth(date1, date2 time.Time) bool {
	return date1.Year() == date2.Year() && date1.Month() == date2.Month()
}

// Alternative version if you want more detailed breakdown with specific days
type DetailedMonthlyData struct {
	CurrentMonth  DetailedMonth `json:"current_month"`
	PreviousMonth DetailedMonth `json:"previous_month"`
}

type DetailedMonth struct {
	Month     string                    `json:"month"`
	MonthName string                    `json:"month_name"`
	Intervals map[string]IntervalDetail `json:"intervals"`
	Total     int                       `json:"total"`
}

type IntervalDetail struct {
	Count int            `json:"count"`
	Days  map[string]int `json:"days"` // day -> count
	Range string         `json:"range"`
}

func (s *AnalyticsService) GetMainRadarChartDataDetailed(ctx context.Context, clerkUserID string) (*DetailedMonthlyData, error) {
	intervals := []IntervalRange{
		{Start: 1, End: 6, Label: "1-6"},
		{Start: 7, End: 11, Label: "7-11"},
		{Start: 12, End: 16, Label: "12-16"},
		{Start: 17, End: 21, Label: "17-21"},
		{Start: 22, End: 26, Label: "22-26"},
		{Start: 27, End: 31, Label: "27-31"},
	}

	now := time.Now()
	startDate := now.AddDate(0, -2, 0).Truncate(24 * time.Hour)
	// Get current and previous month info

	currentMonth := now.AddDate(0, -1, 0)  // Last month
	previousMonth := now.AddDate(0, -2, 0) // 2 months ago

	visitedStreets, err := s.client.VisitedStreet.FindMany(
		db.VisitedStreet.UserID.Equals(clerkUserID),
		db.VisitedStreet.CreatedAt.Gte(startDate),
	).Exec(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get visited streets: %w", err)
	}

	if visitedStreets == nil {
		return nil, fmt.Errorf("user not found")
	}

	// Initialize detailed result structure
	result := &DetailedMonthlyData{
		CurrentMonth: DetailedMonth{
			Month:     currentMonth.Format("2006-01"),
			MonthName: currentMonth.Format("January 2006"),
			Intervals: make(map[string]IntervalDetail),
			Total:     0,
		},
		PreviousMonth: DetailedMonth{
			Month:     previousMonth.Format("2006-01"),
			MonthName: previousMonth.Format("January 2006"),
			Intervals: make(map[string]IntervalDetail),
			Total:     0,
		},
	}

	// Initialize intervals
	for _, interval := range intervals {
		result.CurrentMonth.Intervals[interval.Label] = IntervalDetail{
			Count: 0,
			Days:  make(map[string]int),
			Range: interval.Label,
		}
		result.PreviousMonth.Intervals[interval.Label] = IntervalDetail{
			Count: 0,
			Days:  make(map[string]int),
			Range: interval.Label,
		}
	}

	// Process visited streets
	for _, visitedStreet := range visitedStreets {
		visitDate := visitedStreet.CreatedAt
		day := visitDate.Day()
		dayStr := fmt.Sprintf("%d", day)

		intervalLabel := getIntervalLabel(day, intervals)

		if isSameMonth(visitDate, currentMonth) {
			detail := result.CurrentMonth.Intervals[intervalLabel]
			detail.Count++
			detail.Days[dayStr]++
			result.CurrentMonth.Intervals[intervalLabel] = detail
			result.CurrentMonth.Total++
		} else if isSameMonth(visitDate, previousMonth) {
			detail := result.PreviousMonth.Intervals[intervalLabel]
			detail.Count++
			detail.Days[dayStr]++
			result.PreviousMonth.Intervals[intervalLabel] = detail
			result.PreviousMonth.Total++
		}
	}

	return result, nil
}
