
// internal/repository/settings.go
package repository

import (
	"context"
	"database/sql"
	"fmt"

	"citystatAPI/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
)

type SettingsRepository interface {
	GetUserSettings(ctx context.Context, userID string) (*db.Setting, error)
	CreateUserSettings(ctx context.Context, params CreateSettingsParams) (*db.Setting, error)
	UpdateUserSettings(ctx context.Context, params UpdateSettingsParams) (*db.Setting, error)
	UpdateLocationPermission(ctx context.Context, userID string, enabled bool) (bool, error)
}

type CreateSettingsParams struct {
	UserID                             string
	Theme                              *db.Theme
	Language                           *db.Language
	TextSize                           *db.TextSize
	ZoomLevel                          *string
	FontStyle                          *string
	MessagesAllowance                  *db.MessagesAllowance
	ShowRoleColors                     *db.RoleColors
	Motion                             *db.Motion
	StickersAnimation                  *db.StickersAnimation
	EnabledLocationTracking            *bool
	AllowCityStatDataUsage             *bool
	AllowDataPersonalizationUsage      *bool
	AllowInAppRewards                  *bool
	AllowDataAnalyticsAndPerformance   *bool
	EnableInAppNotifications           *bool
	EnableSoundEffects                 *bool
	EnableVibration                    *bool
}

type UpdateSettingsParams struct {
	UserID                             string
	Theme                              *db.Theme
	Language                           *db.Language
	TextSize                           *db.TextSize
	ZoomLevel                          *string
	FontStyle                          *string
	MessagesAllowance                  *db.MessagesAllowance
	ShowRoleColors                     *db.RoleColors
	Motion                             *db.Motion
	StickersAnimation                  *db.StickersAnimation
	EnabledLocationTracking            *bool
	AllowCityStatDataUsage             *bool
	AllowDataPersonalizationUsage      *bool
	AllowInAppRewards                  *bool
	AllowDataAnalyticsAndPerformance   *bool
	EnableInAppNotifications           *bool
	EnableSoundEffects                 *bool
	EnableVibration                    *bool
}

type settingsRepository struct {
	queries *db.Queries
}

func NewSettingsRepository(queries *db.Queries) SettingsRepository {
	return &settingsRepository{queries: queries}
}

func (r *settingsRepository) GetUserSettings(ctx context.Context, userID string) (*db.Setting, error) {
	settings, err := r.queries.GetUserSettings(ctx, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("settings not found")
		}
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}
	return &settings, nil
}

func (r *settingsRepository) CreateUserSettings(ctx context.Context, params CreateSettingsParams) (*db.Setting, error) {
	sqlcParams := db.CreateUserSettingsParams{
		UserID: params.UserID,
		Theme: db.NullTheme{
			Theme: getTheme(params.Theme),
			Valid: params.Theme != nil,
		},
		Language: db.NullLanguage{
			Language: getLanguage(params.Language),
			Valid:    params.Language != nil,
		},
		TextSize: db.NullTextSize{
			TextSize: getTextSize(params.TextSize),
			Valid:    params.TextSize != nil,
		},
		ZoomLevel: pgtype.Text{
			String: getStringOrDefault(params.ZoomLevel, "100"),
			Valid:  params.ZoomLevel != nil,
		},
		FontStyle: pgtype.Text{
			String: getStringOrDefault(params.FontStyle, "default"),
			Valid:  params.FontStyle != nil,
		},
		MessagesAllowance: db.NullMessagesAllowance{
			MessagesAllowance: getMessagesAllowance(params.MessagesAllowance),
			Valid:             params.MessagesAllowance != nil,
		},
		ShowRoleColors: db.NullRoleColors{
			RoleColors: getRoleColors(params.ShowRoleColors),
			Valid:      params.ShowRoleColors != nil,
		},
		Motion: db.NullMotion{
			Motion: getMotion(params.Motion),
			Valid:  params.Motion != nil,
		},
		StickersAnimation: db.NullStickersAnimation{
			StickersAnimation: getStickersAnimation(params.StickersAnimation),
			Valid:             params.StickersAnimation != nil,
		},
		EnabledLocationTracking: pgtype.Bool{
			Bool:  getBoolOrDefault(params.EnabledLocationTracking, false),
			Valid: params.EnabledLocationTracking != nil,
		},
		AllowCityStatDataUsage: pgtype.Bool{
			Bool:  getBoolOrDefault(params.AllowCityStatDataUsage, true),
			Valid: params.AllowCityStatDataUsage != nil,
		},
		AllowDataPersonalizationUsage: pgtype.Bool{
			Bool:  getBoolOrDefault(params.AllowDataPersonalizationUsage, true),
			Valid: params.AllowDataPersonalizationUsage != nil,
		},
		AllowInAppRewards: pgtype.Bool{
			Bool:  getBoolOrDefault(params.AllowInAppRewards, true),
			Valid: params.AllowInAppRewards != nil,
		},
		AllowDataAnalyticsAndPerformance: pgtype.Bool{
			Bool:  getBoolOrDefault(params.AllowDataAnalyticsAndPerformance, true),
			Valid: params.AllowDataAnalyticsAndPerformance != nil,
		},
		EnableInAppNotifications: pgtype.Bool{
			Bool:  getBoolOrDefault(params.EnableInAppNotifications, true),
			Valid: params.EnableInAppNotifications != nil,
		},
		EnableSoundEffects: pgtype.Bool{
			Bool:  getBoolOrDefault(params.EnableSoundEffects, true),
			Valid: params.EnableSoundEffects != nil,
		},
		EnableVibration: pgtype.Bool{
			Bool:  getBoolOrDefault(params.EnableVibration, true),
			Valid: params.EnableVibration != nil,
		},
	}

	settings, err := r.queries.CreateUserSettings(ctx, sqlcParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create settings: %w", err)
	}
	return &settings, nil
}

func (r *settingsRepository) UpdateUserSettings(ctx context.Context, params UpdateSettingsParams) (*db.Setting, error) {
	sqlcParams := db.UpdateUserSettingsParams{
		UserID: params.UserID,
		Theme: db.NullTheme{
			Theme: getTheme(params.Theme),
			Valid: params.Theme != nil,
		},
		Language: db.NullLanguage{
			Language: getLanguage(params.Language),
			Valid:    params.Language != nil,
		},
		TextSize: db.NullTextSize{
			TextSize: getTextSize(params.TextSize),
			Valid:    params.TextSize != nil,
		},
		ZoomLevel: pgtype.Text{
			String: stringValue(params.ZoomLevel),
			Valid:  params.ZoomLevel != nil,
		},
		FontStyle: pgtype.Text{
			String: stringValue(params.FontStyle),
			Valid:  params.FontStyle != nil,
		},
		MessagesAllowance: db.NullMessagesAllowance{
			MessagesAllowance: getMessagesAllowance(params.MessagesAllowance),
			Valid:             params.MessagesAllowance != nil,
		},
		ShowRoleColors: db.NullRoleColors{
			RoleColors: getRoleColors(params.ShowRoleColors),
			Valid:      params.ShowRoleColors != nil,
		},
		Motion: db.NullMotion{
			Motion: getMotion(params.Motion),
			Valid:  params.Motion != nil,
		},
		StickersAnimation: db.NullStickersAnimation{
			StickersAnimation: getStickersAnimation(params.StickersAnimation),
			Valid:             params.StickersAnimation != nil,
		},
		EnabledLocationTracking: pgtype.Bool{
			Bool:  boolValue(params.EnabledLocationTracking),
			Valid: params.EnabledLocationTracking != nil,
		},
		AllowCityStatDataUsage: pgtype.Bool{
			Bool:  boolValue(params.AllowCityStatDataUsage),
			Valid: params.AllowCityStatDataUsage != nil,
		},
		AllowDataPersonalizationUsage: pgtype.Bool{
			Bool:  boolValue(params.AllowDataPersonalizationUsage),
			Valid: params.AllowDataPersonalizationUsage != nil,
		},
		AllowInAppRewards: pgtype.Bool{
			Bool:  boolValue(params.AllowInAppRewards),
			Valid: params.AllowInAppRewards != nil,
		},
		AllowDataAnalyticsAndPerformance: pgtype.Bool{
			Bool:  boolValue(params.AllowDataAnalyticsAndPerformance),
			Valid: params.AllowDataAnalyticsAndPerformance != nil,
		},
		EnableInAppNotifications: pgtype.Bool{
			Bool:  boolValue(params.EnableInAppNotifications),
			Valid: params.EnableInAppNotifications != nil,
		},
		EnableSoundEffects: pgtype.Bool{
			Bool:  boolValue(params.EnableSoundEffects),
			Valid: params.EnableSoundEffects != nil,
		},
		EnableVibration: pgtype.Bool{
			Bool:  boolValue(params.EnableVibration),
			Valid: params.EnableVibration != nil,
		},
	}

	settings, err := r.queries.UpdateUserSettings(ctx, sqlcParams)
	if err != nil {
		return nil, fmt.Errorf("failed to update settings: %w", err)
	}
	return &settings, nil
}

func (r *settingsRepository) UpdateLocationPermission(ctx context.Context, userID string, enabled bool) (bool, error) {
	result, err := r.queries.UpdateLocationPermission(ctx, db.UpdateLocationPermissionParams{
		UserID:                  userID,
		EnabledLocationTracking: pgtype.Bool{Bool: enabled, Valid: true},
	})
	if err != nil {
		return false, fmt.Errorf("failed to update location permission: %w", err)
	}
	return result.Bool, nil
}

// Helper functions for settings defaults
func getTheme(t *db.Theme) db.Theme {
	if t != nil {
		return *t
	}
	return db.ThemeLIGHT
}

func getLanguage(l *db.Language) db.Language {
	if l != nil {
		return *l
	}
	return db.LanguageEn
}

func getTextSize(ts *db.TextSize) db.TextSize {
	if ts != nil {
		return *ts
	}
	return db.TextSizeMEDIUM
}

func getMessagesAllowance(ma *db.MessagesAllowance) db.MessagesAllowance {
	if ma != nil {
		return *ma
	}
	return db.MessagesAllowanceALLMSG
}

func getRoleColors(rc *db.RoleColors) db.RoleColors {
	if rc != nil {
		return *rc
	}
	return db.RoleColorsNEXTTONAME
}

func getMotion(m *db.Motion) db.Motion {
	if m != nil {
		return *m
	}
	return db.MotionDONTPLAYGIFWHENPOSSIBLESHOW
}

func getStickersAnimation(sa *db.StickersAnimation) db.StickersAnimation {
	if sa != nil {
		return *sa
	}
	return db.StickersAnimationALWAYS
}

func getStringOrDefault(s *string, def string) string {
	if s != nil {
		return *s
	}
	return def
}

func getBoolOrDefault(b *bool, def bool) bool {
	if b != nil {
		return *b
	}
	return def
}