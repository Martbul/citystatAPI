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


func (s *VisitorService) GetVisitedStreets(ctx context.Context, clerkUserID string) ([]types.VisitedStreetRequest, error) {
	visitedStreets, err := s.client.VisitedStreet.FindMany(
		db.VisitedStreet.UserID.Equals(clerkUserID),
	).Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	var result []types.VisitedStreetRequest
	for _, street := range visitedStreets {
		var exitTimestamp *int64
		if street.ExitTimestamp != nil {
			v,_ := street.ExitTimestamp()
			val := int64(v)
			exitTimestamp = &val
		}

		var durationSeconds *int
		if street.DurationSeconds != nil {
						v,_ := street.DurationSeconds()

			val := v
			durationSeconds = &val
		}

		result = append(result, types.VisitedStreetRequest{
			StreetID:        street.StreetID,
			StreetName:      street.StreetName,
			EntryTimestamp:  int64(street.EntryTimestamp),
			ExitTimestamp:   exitTimestamp,
			DurationSeconds: durationSeconds,
			EntryLatitude:   street.EntryLatitude.InexactFloat64(),
			EntryLongitude:  street.EntryLongitude.InexactFloat64(),
		})
	}

	return result, nil
}

// func (s *VisitorService) SaveVisitedStreets(ctx context.Context, clerkUserID string, req types.SaveVisitedStreetsRequest) error {
// 	for _, street := range req.VisitedStreets {
// 		entryTimestamp := prismaTypes.BigInt(street.EntryTimestamp)
// 		entryLatitude := decimal.NewFromFloat(street.EntryLatitude)
// 		entryLongitude := decimal.NewFromFloat(street.EntryLongitude)
		
// 		// Check if the record already exists to avoid duplicates
// 		existing, err := s.client.VisitedStreet.FindFirst(
// 			db.VisitedStreet.UserID.Equals(clerkUserID),
// 			db.VisitedStreet.SessionID.Equals(req.SessionID),
// 			db.VisitedStreet.StreetID.Equals(street.StreetID),
// 			db.VisitedStreet.EntryTimestamp.Equals(entryTimestamp),
// 		).Exec(ctx)
		
// 		// If error is not "not found", return the error
// 		if err != nil && !errors.Is(err, db.ErrNotFound) {
// 			return err
// 		}
	
// 		// If record doesn't exist, create it
// 		if existing == nil {
// 			// Prepare optional parameters
// 			var optionalParams []db.VisitedStreetSetParam
			
// 			if street.ExitTimestamp != nil {
// 				exitTimestamp := prismaTypes.BigInt(*street.ExitTimestamp)
// 				optionalParams = append(optionalParams, db.VisitedStreet.ExitTimestamp.Set(exitTimestamp))
// 			}
			
// 			if street.DurationSeconds != nil {
// 				optionalParams = append(optionalParams, db.VisitedStreet.DurationSeconds.Set(*street.DurationSeconds))
// 			}
			
// 			// Create the record - use User.Link for the relation
// 			_, err = s.client.VisitedStreet.CreateOne(
// 				db.VisitedStreet.SessionID.Set(req.SessionID),
// 				db.VisitedStreet.StreetID.Set(street.StreetID),
// 				db.VisitedStreet.StreetName.Set(street.StreetName),
// 				db.VisitedStreet.EntryTimestamp.Set(entryTimestamp),
// 				db.VisitedStreet.EntryLatitude.Set(entryLatitude),
// 				db.VisitedStreet.EntryLongitude.Set(entryLongitude),
// 				db.VisitedStreet.User.Link(db.User.ID.Equals(clerkUserID)),
// 				optionalParams...,
// 			).Exec(ctx)
			
// 			if err != nil {
// 				return err
// 			}
// 		}
// 	}
	
// 	return nil
// }

func (s *VisitorService) SaveVisitedStreets(ctx context.Context, clerkUserID string, req types.SaveVisitedStreetsRequest) error {
	seen := make(map[string]struct{})
	for _, street := range req.VisitedStreets {
		// Create a unique key based on StreetID and StreetName to avoid duplicates in this batch
		key := street.StreetID + "|" + street.StreetName
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}

		entryTimestamp := prismaTypes.BigInt(street.EntryTimestamp)
		entryLatitude := decimal.NewFromFloat(street.EntryLatitude)
		entryLongitude := decimal.NewFromFloat(street.EntryLongitude)

		// Check if the record already exists to avoid duplicates in the database
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
			var optionalParams []db.VisitedStreetSetParam

			if street.ExitTimestamp != nil {
				exitTimestamp := prismaTypes.BigInt(*street.ExitTimestamp)
				optionalParams = append(optionalParams, db.VisitedStreet.ExitTimestamp.Set(exitTimestamp))
			}

			if street.DurationSeconds != nil {
				optionalParams = append(optionalParams, db.VisitedStreet.DurationSeconds.Set(*street.DurationSeconds))
			}

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
