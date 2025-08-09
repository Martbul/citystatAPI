package services

import (
	"citystatAPI/prisma/db"
	"citystatAPI/types"
	"context"
	"errors"
	"fmt"
    prismaTypes "github.com/steebchen/prisma-client-go/runtime/types"

	"github.com/shopspring/decimal"
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


// Service function
func (s *VisitorService) SaveVisitedStreets(ctx context.Context, clerkUserID string, req types.SaveVisitedStreetsRequest) error {
	for _, street := range req.VisitedStreets {
		// Convert types for Prisma compatibility
		entryTimestamp := prismaTypes.BigInt(street.EntryTimestamp)
		entryLatitude := decimal.NewFromFloat(street.EntryLatitude)
		entryLongitude := decimal.NewFromFloat(street.EntryLongitude)
		
		// Check if the record already exists to avoid duplicates
		existing, err := s.client.VisitedStreet.FindFirst(
			db.VisitedStreet.UserID.Equals(clerkUserID),
			db.VisitedStreet.SessionID.Equals(req.SessionID),
			db.VisitedStreet.StreetID.Equals(street.StreetID),
			db.VisitedStreet.EntryTimestamp.Equals(entryTimestamp),
		).Exec(ctx)
		
		// If error is not "not found", return the error
		if err != nil && !errors.Is(err, db.ErrNotFound) {
			return err
		}
	
		// If record doesn't exist, create it
		if existing == nil {
			// Prepare optional parameters
			var optionalParams []db.VisitedStreetSetParam
			
			if street.ExitTimestamp != nil {
				exitTimestamp := prismaTypes.BigInt(*street.ExitTimestamp)
				optionalParams = append(optionalParams, db.VisitedStreet.ExitTimestamp.Set(exitTimestamp))
			}
			
			if street.DurationSeconds != nil {
				optionalParams = append(optionalParams, db.VisitedStreet.DurationSeconds.Set(*street.DurationSeconds))
			}
			
			// Create the record - use User.Link for the relation
			_, err = s.client.VisitedStreet.CreateOne(
				db.VisitedStreet.SessionID.Set(req.SessionID),
				db.VisitedStreet.StreetID.Set(street.StreetID),
				db.VisitedStreet.StreetName.Set(street.StreetName),
				db.VisitedStreet.EntryTimestamp.Set(entryTimestamp),
				db.VisitedStreet.EntryLatitude.Set(entryLatitude),
				db.VisitedStreet.EntryLongitude.Set(entryLongitude),
				db.VisitedStreet.User.Link(db.User.ID.Equals(clerkUserID)),
				optionalParams...,
			).Exec(ctx)
			
			if err != nil {
				return err
			}
		}
	}
	
	return nil
}