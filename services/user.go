package services

import (
	"context"
	"fmt"

	"citystatAPI/prisma/db"
	"citystatAPI/types"

	"github.com/clerk/clerk-sdk-go/v2/user"
)


type UserService struct {
	client *db.PrismaClient
}


func NewUserService(client *db.PrismaClient) *UserService {
	return &UserService{client: client}
}

func (s *UserService) UpdateUser(ctx context.Context, clerkUserID string, updates types.UserUpdateRequest) (*db.UserModel, error) {
		fmt.Println("updating user")

	fmt.Println(updates)
	// Ensure user exists first
	existingUser, err := s.client.User.FindUnique(
		db.User.ID.Equals(clerkUserID),
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

	if updates.FirstName != nil {
		updateOps = append(updateOps, db.User.FirstName.Set(*updates.FirstName))
	}
	if updates.LastName != nil {
		updateOps = append(updateOps, db.User.LastName.Set(*updates.LastName))
	}
	if updates.UserName != nil {
		updateOps = append(updateOps, db.User.UserName.Set(*updates.UserName))
	}
	if updates.ImageURL != nil {
		updateOps = append(updateOps, db.User.ImageURL.Set(*updates.ImageURL))
	}
		if updates.CompletedTutorial != nil {
		updateOps = append(updateOps, db.User.CompletedTutorial.Set(*updates.CompletedTutorial))
	}


	// If no updates provided, return existing user
	if len(updateOps) == 0 {
		return existingUser, nil
	}

	// Perform the update
	updatedUser, err := s.client.User.FindUnique(
		db.User.ID.Equals(clerkUserID),
	).Update(updateOps...).Exec(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return updatedUser, nil
}



func (s *UserService) SyncUserFromClerk(ctx context.Context, clerkUserID string) (*db.UserModel, error) {
	// Get user from Clerk
	clerkUser, err := user.Get(ctx, clerkUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user from Clerk: %w", err)
	}

	// Prepare user data
	var email string
	if len(clerkUser.EmailAddresses) > 0 {
		email = clerkUser.EmailAddresses[0].EmailAddress
	}

	var imageUrl *string
	if clerkUser.ImageURL != nil && *clerkUser.ImageURL != "" {
		imageUrl = clerkUser.ImageURL
	}

	// Try to find existing user
	existingUser, err := s.client.User.FindUnique(
		db.User.ID.Equals(clerkUserID),
	).Exec(ctx)
	if err != nil && err != db.ErrNotFound {
		return nil, fmt.Errorf("error checking existing user: %w", err)
	}

	// User exists, update it
	if existingUser != nil {
		updatedUser, err := s.client.User.FindUnique(
			db.User.ID.Equals(clerkUserID),
		).Update(
			db.User.Email.Set(email),
			db.User.FirstName.SetIfPresent(clerkUser.FirstName),
			db.User.LastName.SetIfPresent(clerkUser.LastName),
			db.User.ImageURL.SetIfPresent(imageUrl),
		).Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to update user: %w", err)
		}

		// Ensure existing user has settings (in case they were created before settings were implemented)
		err = s.ensureUserHasSettings(ctx, clerkUserID)
		if err != nil {
			return nil, fmt.Errorf("failed to ensure user has settings: %w", err)
		}

		return updatedUser, nil
	}

	// User doesn't exist, create new one
	newUser, err := s.client.User.CreateOne(
		db.User.ID.Set(clerkUserID),
		db.User.Email.Set(email),
		db.User.FirstName.SetIfPresent(clerkUser.FirstName),
		db.User.LastName.SetIfPresent(clerkUser.LastName),
		db.User.UserName.SetIfPresent(clerkUser.Username),
		db.User.ImageURL.SetIfPresent(imageUrl),
	).Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Create default settings for the new user
	_, err = s.client.Settings.CreateOne(
		db.Settings.User.Link(db.User.ID.Equals(clerkUserID)),
		// Optionally set explicit defaults (or rely on schema defaults)
		// db.Settings.Theme.Set(db.ThemeLight),
		// db.Settings.Language.Set(db.LanguageEn),
		// db.Settings.EnabledLocationTracking.Set(false),
		// ... other explicit defaults if needed
	).Exec(ctx)
	if err != nil {
		// Log the error but don't fail user creation since user was already created
		// You might want to handle this differently based on your requirements
		return nil, fmt.Errorf("failed to create user settings: %w", err)
	}

	return newUser, nil
}

func (s *UserService) ensureUserHasSettings(ctx context.Context, userID string) error {
	// Check if settings already exist
	_, err := s.client.Settings.FindUnique(
		db.Settings.UserID.Equals(userID),
	).Exec(ctx)

	if err == db.ErrNotFound {
		// Settings don't exist, create them
		_, err = s.client.Settings.CreateOne(
			db.Settings.User.Link(db.User.ID.Equals(userID)),
		).Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to create settings: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("error checking settings: %w", err)
	}

	return nil
}


// // SyncUserFromClerk creates or updates user from Clerk data
// func (s *UserService) SyncUserFromClerk(ctx context.Context, clerkUserID string) (*db.UserModel, error) {
// 	// Get user from Clerk
// 	clerkUser, err := user.Get(ctx, clerkUserID)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to fetch user from Clerk: %w", err)
// 	}

// 	// Prepare user data
// 	var email string
// 	if len(clerkUser.EmailAddresses) > 0 {
// 		email = clerkUser.EmailAddresses[0].EmailAddress
// 	}

// 	var imageUrl *string
// 	if clerkUser.ImageURL != nil && *clerkUser.ImageURL != "" {
// 		imageUrl = clerkUser.ImageURL
// 	}

// 	// Try to find existing user
// 	existingUser, err := s.client.User.FindUnique(
// 		db.User.ID.Equals(clerkUserID),
// 	).Exec(ctx)

// 	if err != nil && err != db.ErrNotFound {
// 		return nil, fmt.Errorf("error checking existing user: %w", err)
// 	}

// 	// User exists, update it
// 	if existingUser != nil {
// 		updatedUser, err := s.client.User.FindUnique(
// 			db.User.ID.Equals(clerkUserID),
// 		).Update(
// 			db.User.Email.Set(email),
// 			db.User.FirstName.SetIfPresent(clerkUser.FirstName),
// 			db.User.LastName.SetIfPresent(clerkUser.LastName),
// 			db.User.ImageURL.SetIfPresent(imageUrl),
// 		).Exec(ctx)

// 		if err != nil {
// 			return nil, fmt.Errorf("failed to update user: %w", err)
// 		}

// 		return updatedUser, nil
// 	}

// 	// User doesn't exist, create new one
// 	newUser, err := s.client.User.CreateOne(
// 		db.User.ID.Set(clerkUserID),
// 		db.User.Email.Set(email),
// 		db.User.FirstName.SetIfPresent(clerkUser.FirstName),
// 		db.User.LastName.SetIfPresent(clerkUser.LastName),
// 		db.User.UserName.SetIfPresent(clerkUser.Username),
// 		db.User.ImageURL.SetIfPresent(imageUrl),
// 	).Exec(ctx)

// 	if err != nil {
// 		return nil, fmt.Errorf("failed to create user: %w", err)
// 	}

// 	return newUser, nil
// }

// GetOrCreateUser ensures user exists in database
func (s *UserService) GetOrCreateUser(ctx context.Context, clerkUserID string) (*db.UserModel, error) {
	// Try to get user from database first
	user, err := s.client.User.FindUnique(
		db.User.ID.Equals(clerkUserID),
	).Exec(ctx)

	if err == nil {
		return user, nil
	}

	if err == db.ErrNotFound {
		// User not in database, sync from Clerk
		return s.SyncUserFromClerk(ctx, clerkUserID)
	}

	return nil, fmt.Errorf("database error: %w", err)
}


func (s *UserService) EditNote(ctx context.Context, clerkUserID string, updates map[string]interface{}) (*db.UserModel, error) {
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


// Add this method to your services/user.go file

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


func (s *UserService) UpdateUserSettings(ctx context.Context, clerkUserID string, settingsUpdate map[string]interface{}) (*db.UserModel, error) {
    fmt.Println("Updating user settings for:", clerkUserID)
    fmt.Println("Settings data:", settingsUpdate)

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

    if theme, ok := settingsUpdate["theme"].(db.Theme); ok {
        settingsOps = append(settingsOps, db.Settings.Theme.Set(theme))
    }
    if language, ok := settingsUpdate["language"].(db.Language); ok {
        settingsOps = append(settingsOps, db.Settings.Language.Set(language))
    }
    if textSize, ok := settingsUpdate["textSize"].(db.TextSize); ok {
        settingsOps = append(settingsOps, db.Settings.TextSize.Set(textSize))
    }
    if fontStyle, ok := settingsUpdate["fontStyle"].(string); ok {
        settingsOps = append(settingsOps, db.Settings.FontStyle.Set(fontStyle))
    }
    if zoomLevel, ok := settingsUpdate["zoomLevel"].(string); ok {
        settingsOps = append(settingsOps, db.Settings.ZoomLevel.Set(zoomLevel))
    }
    if showRoleColors, ok := settingsUpdate["showRoleColors"].(db.RoleColors); ok {
        settingsOps = append(settingsOps, db.Settings.ShowRoleColors.Set(showRoleColors))
    }
    if messagesAllowance, ok := settingsUpdate["messagesAllowance"].(db.MessagesAllowance); ok {
        settingsOps = append(settingsOps, db.Settings.MessagesAllowance.Set(messagesAllowance))
    }
    if motion, ok := settingsUpdate["motion"].(db.Motion); ok {
        settingsOps = append(settingsOps, db.Settings.Motion.Set(motion))
    }
    if stickersAnimation, ok := settingsUpdate["stickersAnimation"].(db.StickersAnimation); ok {
        settingsOps = append(settingsOps, db.Settings.StickersAnimation.Set(stickersAnimation))
    }
    
    // Boolean settings
    if enabledLocationTracking, ok := settingsUpdate["enabledLocationTracking"].(bool); ok {
        settingsOps = append(settingsOps, db.Settings.EnabledLocationTracking.Set(enabledLocationTracking))
    }
    if allowCityStatDataUsage, ok := settingsUpdate["allowCityStatDataUsage"].(bool); ok {
        settingsOps = append(settingsOps, db.Settings.AllowCityStatDataUsage.Set(allowCityStatDataUsage))
    }
    if allowDataPersonalizationUsage, ok := settingsUpdate["allowDataPersonalizationUsage"].(bool); ok {
        settingsOps = append(settingsOps, db.Settings.AllowDataPersonalizationUsage.Set(allowDataPersonalizationUsage))
    }
    if allowInAppRewards, ok := settingsUpdate["allowInAppRewards"].(bool); ok {
        settingsOps = append(settingsOps, db.Settings.AllowInAppRewards.Set(allowInAppRewards))
    }
    if allowDataAnaliticsAndPerformance, ok := settingsUpdate["allowDataAnaliticsAndPerformance"].(bool); ok {
        settingsOps = append(settingsOps, db.Settings.AllowDataAnaliticsAndPerformance.Set(allowDataAnaliticsAndPerformance))
    }
    if enableInAppNotifications, ok := settingsUpdate["enableInAppNotifications"].(bool); ok {
        settingsOps = append(settingsOps, db.Settings.EnableInAppNotifications.Set(enableInAppNotifications))
    }
    if enableSoundEffects, ok := settingsUpdate["enableSoundEffects"].(bool); ok {
        settingsOps = append(settingsOps, db.Settings.EnableSoundEffects.Set(enableSoundEffects))
    }
    if enableVibration, ok := settingsUpdate["enableVibration"].(bool); ok {
        settingsOps = append(settingsOps, db.Settings.EnableVibration.Set(enableVibration))
    }

    if len(settingsOps) == 0 {
        return existingUser, nil
    }

    // Check if user has settings record
    settings, hasSettings := existingUser.Settings()
    if !hasSettings || settings == nil {
        // Create new settings record
        _, err = s.client.Settings.CreateOne(
            db.Settings.User.Link(db.User.ID.Equals(clerkUserID)),
            settingsOps...,
        ).Exec(ctx)
        if err != nil {
            return nil, fmt.Errorf("failed to create settings: %w", err)
        }
    } else {
        // Update existing settings
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
        db.User.CityStats.Fetch().With(
            db.CityStat.StreetWalks.Fetch(),
        ),
    ).Exec(ctx)

    if err != nil {
        return nil, fmt.Errorf("failed to fetch updated user: %w", err)
    }

    return updatedUser, nil
}

// UpdateUserProfile handles mixed user and settings updates
func (s *UserService) UpdateUserProfile(ctx context.Context, clerkUserID string, updates map[string]interface{}) (*db.UserModel, error) {
    fmt.Println("Updating user profile for:", clerkUserID)
    fmt.Println("Profile data:", updates)

    // Check if this is a settings-only update
    if settingsData, hasSettings := updates["settings"]; hasSettings {
        if settingsMap, ok := settingsData.(map[string]interface{}); ok {
            return s.UpdateUserSettings(ctx, clerkUserID, settingsMap)
        }
    }

    // Handle regular user field updates
    return s.UpdateUser(ctx, clerkUserID, types.UserUpdateRequest{
        FirstName:         getStringPointer(updates, "firstName"),
        LastName:          getStringPointer(updates, "lastName"),
        UserName:          getStringPointer(updates, "userName"),
        ImageURL:          getStringPointer(updates, "imageURL"),
        CompletedTutorial: getBoolPointer(updates, "completedTutorial"),
    })
}

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