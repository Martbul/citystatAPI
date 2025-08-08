package services

import (
	"citystatAPI/prisma/db"
	"context"
	"errors"
	"fmt"
)

type VisitorService struct {
	client *db.PrismaClient
}

func NewVisitorService(client *db.PrismaClient) *VisitorService {
	return &VisitorService{client: client}
}

func (s *VisitorService) GetLocationPermission(ctx context.Context, clerkUserID string) (bool, error) {
    settings, err := s.client.Settings.FindUnique(
        db.Settings.UserID.Equals(clerkUserID),
    ).Select(
        db.Settings.EnabledLocationTracking.Field(),
    ).Exec(context.Background())

    if err != nil {
        if errors.Is(err, db.ErrNotFound) {
            return false, nil 
        }
        return false, fmt.Errorf("database error: %w", err)
    }

	 fmt.Println("enableLocationTracking" )
	 fmt.Println(settings.EnabledLocationTracking)

    return settings.EnabledLocationTracking, nil
}




func (s *VisitorService) SaveLocationPermission(ctx context.Context, clerkUserID string, hasLocationPermission bool) (bool, error) {
    updatedSettings, err := s.client.Settings.FindUnique(
        db.Settings.UserID.Equals(clerkUserID),
    ).Update(
        db.Settings.EnabledLocationTracking.Set(hasLocationPermission),
    ).Exec(ctx)

    if err != nil {
        return false, fmt.Errorf("database error: %w", err)
    }

    fmt.Println("Updated enabledLocationTracking:", updatedSettings.EnabledLocationTracking)
    return updatedSettings.EnabledLocationTracking, nil
}