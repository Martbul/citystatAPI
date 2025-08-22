// internal/services/visitor.go
package services

import (
	"context"

	"citystatAPI/internal/repository"
	"citystatAPI/types"
)

type VisitorService struct {
	visitorRepo  repository.VisitorRepository
	settingsRepo repository.SettingsRepository
}

func NewVisitorService(visitorRepo repository.VisitorRepository, settingsRepo repository.SettingsRepository) *VisitorService {
	return &VisitorService{
		visitorRepo:  visitorRepo,
		settingsRepo: settingsRepo,
	}
}

func (s *VisitorService) GetLocationPermission(ctx context.Context, userID string) (bool, error) {
	settings, err := s.settingsRepo.GetUserSettings(ctx, userID)
	if err != nil {
		return false, err
	}
	return settings.EnabledLocationTracking.Bool, nil
}

func (s *VisitorService) SaveLocationPermission(ctx context.Context, userID string, hasLocationPermission bool) (bool, error) {
	enabled, err := s.settingsRepo.UpdateLocationPermission(ctx, userID, hasLocationPermission)
	if err != nil {
		return false, err
	}
	return enabled, nil
}

func (s *VisitorService) SaveVisitedStreets(ctx context.Context, userID string, req types.SaveVisitedStreetsRequest) error {
	for _, street := range req.VisitedStreets {
		// Check if record already exists
		exists, err := s.visitorRepo.CheckVisitedStreetExists(ctx, userID, req.SessionID, street.StreetID, street.EntryTimestamp)
		if err != nil {
			return err
		}

		if !exists {
			// Convert types for repository
			var durationSeconds *int32
			if street.DurationSeconds != nil {
				val := int32(*street.DurationSeconds)
				durationSeconds = &val
			}

			_, err = s.visitorRepo.CreateVisitedStreet(ctx, repository.CreateVisitedStreetParams{
				UserID:          userID,
				SessionID:       req.SessionID,
				StreetID:        street.StreetID,
				StreetName:      street.StreetName,
				EntryTimestamp:  street.EntryTimestamp,
				ExitTimestamp:   street.ExitTimestamp,
				DurationSeconds: durationSeconds,
				EntryLatitude:   street.EntryLatitude,
				EntryLongitude:  street.EntryLongitude,
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *VisitorService) GetVisitedStreets(ctx context.Context, userID string) ([]types.VisitedStreetResponse, error) {
	streets, err := s.visitorRepo.GetVisitedStreets(ctx, userID)
	if err != nil {
		return nil, err
	}

	results := make([]types.VisitedStreetResponse, len(streets))
	for i, street := range streets {
		entryLat, err := street.EntryLatitude.Float64Value()
		if err != nil {
			return nil, err
		}
		entryLon, err := street.EntryLongitude.Float64Value()
		if err != nil {
			return nil, err
		}

		result := types.VisitedStreetResponse{
			SessionID:      street.SessionID,
			StreetID:       street.StreetID,
			StreetName:     street.StreetName,
			EntryTimestamp: street.EntryTimestamp,
			EntryLatitude:  entryLat.Float64,
			EntryLongitude: entryLon.Float64,
		}


		// Handle optional fields based on pgx/v5 types
		if street.ExitTimestamp.Valid {
			exit := street.ExitTimestamp.Int64
			result.ExitTimestamp = &exit
		}

		if street.DurationSeconds.Valid {
			duration := int64(street.DurationSeconds.Int32)
			result.DurationSeconds = &duration
		}

		results[i] = result
	}

	return results, nil
}
