
// internal/services/settings.go
package services

import (
	"context"
	"fmt"
	"strings"

	"citystatAPI/internal/db"
	"citystatAPI/internal/repository"
)

type SettingsService struct {
	settingsRepo repository.SettingsRepository
}

func NewSettingsService(settingsRepo repository.SettingsRepository) *SettingsService {
	return &SettingsService{
		settingsRepo: settingsRepo,
	}
}

func (s *SettingsService) GetUserSettings(ctx context.Context, userID string) (*db.Setting, error) {
	settings, err := s.settingsRepo.GetUserSettings(ctx, userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			// Create default settings if they don't exist
			return s.settingsRepo.CreateUserSettings(ctx, repository.CreateSettingsParams{
				UserID: userID,
			})
		}
		return nil, err
	}
	return settings, nil
}

func (s *SettingsService) UpdateUserSettings(ctx context.Context, userID string, updates map[string]interface{}) (*db.Setting, error) {
	params := repository.UpdateSettingsParams{
		UserID: userID,
	}

	// Parse updates and set appropriate fields
	if theme, ok := updates["theme"].(string); ok {
		t := db.Theme(theme)
		params.Theme = &t
	}
	if language, ok := updates["language"].(string); ok {
		l := db.Language(language)
		params.Language = &l
	}
	if textSize, ok := updates["textSize"].(string); ok {
		ts := db.TextSize(textSize)
		params.TextSize = &ts
	}
	if zoomLevel, ok := updates["zoomLevel"].(string); ok {
		params.ZoomLevel = &zoomLevel
	}
	if fontStyle, ok := updates["fontStyle"].(string); ok {
		params.FontStyle = &fontStyle
	}
	if messagesAllowance, ok := updates["messagesAllowance"].(string); ok {
		ma := db.MessagesAllowance(messagesAllowance)
		params.MessagesAllowance = &ma
	}
	if showRoleColors, ok := updates["showRoleColors"].(string); ok {
		rc := db.RoleColors(showRoleColors)
		params.ShowRoleColors = &rc
	}
	if motion, ok := updates["motion"].(string); ok {
		m := db.Motion(motion)
		params.Motion = &m
	}
	if stickersAnimation, ok := updates["stickersAnimation"].(string); ok {
		sa := db.StickersAnimation(stickersAnimation)
		params.StickersAnimation = &sa
	}

	// Boolean settings
	if val, ok := updates["enabledLocationTracking"].(bool); ok {
		params.EnabledLocationTracking = &val
	}
	if val, ok := updates["allowCityStatDataUsage"].(bool); ok {
		params.AllowCityStatDataUsage = &val
	}
	if val, ok := updates["allowDataPersonalizationUsage"].(bool); ok {
		params.AllowDataPersonalizationUsage = &val
	}
	if val, ok := updates["allowInAppRewards"].(bool); ok {
		params.AllowInAppRewards = &val
	}
	if val, ok := updates["allowDataAnalyticsAndPerformance"].(bool); ok {
		params.AllowDataAnalyticsAndPerformance = &val
	}
	if val, ok := updates["enableInAppNotifications"].(bool); ok {
		params.EnableInAppNotifications = &val
	}
	if val, ok := updates["enableSoundEffects"].(bool); ok {
		params.EnableSoundEffects = &val
	}
	if val, ok := updates["enableVibration"].(bool); ok {
		params.EnableVibration = &val
	}

	return s.settingsRepo.UpdateUserSettings(ctx, params)
}

func (s *SettingsService) EditUsername(ctx context.Context, userID string, username string) (*db.Setting, error) {
	// This should actually update the user table, not settings
	// But keeping it here for compatibility with your current structure
	return nil, fmt.Errorf("username editing should be handled by user service")
}

func (s *SettingsService) EditPhoneNumber(ctx context.Context, userID string, phone string) (*db.Setting, error) {
	// This should actually update the user table, not settings
	// But keeping it here for compatibility with your current structure
	return nil, fmt.Errorf("phone number editing should be handled by user service")
}