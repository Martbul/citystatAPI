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
    	rankService *RankService
}

func NewVisitorService(client *db.PrismaClient, rankService *RankService) *VisitorService {
	return &VisitorService{
		client:      client,
		rankService: rankService,
	}
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
//     seen := make(map[string]struct{})
//     for _, street := range req.VisitedStreets {
//         // Create a unique key based on StreetID and StreetName to avoid duplicates in this batch
//         key := street.StreetID + "|" + street.StreetName
//         if _, exists := seen[key]; exists {
//             continue
//         }
//         seen[key] = struct{}{}

//         entryTimestamp := prismaTypes.BigInt(street.EntryTimestamp)
//         entryLatitude := decimal.NewFromFloat(street.EntryLatitude)
//         entryLongitude := decimal.NewFromFloat(street.EntryLongitude)

//         // Check if a record with the same StreetID OR StreetName already exists
//         existingByID, err := s.client.VisitedStreet.FindFirst(
//             db.VisitedStreet.UserID.Equals(clerkUserID),
//             db.VisitedStreet.StreetID.Equals(street.StreetID),
//         ).Exec(ctx)
//         if err != nil && !errors.Is(err, db.ErrNotFound) {
//             return err
//         }

//         existingByName, err := s.client.VisitedStreet.FindFirst(
//             db.VisitedStreet.UserID.Equals(clerkUserID),
//             db.VisitedStreet.StreetName.Equals(street.StreetName),
//         ).Exec(ctx)
//         if err != nil && !errors.Is(err, db.ErrNotFound) {
//             return err
//         }

//         // If neither StreetID nor StreetName exists, create the record
//         if existingByID == nil && existingByName == nil {
//             var optionalParams []db.VisitedStreetSetParam
//             if street.ExitTimestamp != nil {
//                 exitTimestamp := prismaTypes.BigInt(*street.ExitTimestamp)
//                 optionalParams = append(optionalParams, db.VisitedStreet.ExitTimestamp.Set(exitTimestamp))
//             }
//             if street.DurationSeconds != nil {
//                 optionalParams = append(optionalParams, db.VisitedStreet.DurationSeconds.Set(*street.DurationSeconds))
//             }

//             _, err = s.client.VisitedStreet.CreateOne(
//                 db.VisitedStreet.SessionID.Set(req.SessionID),
//                 db.VisitedStreet.StreetID.Set(street.StreetID),
//                 db.VisitedStreet.StreetName.Set(street.StreetName),
//                 db.VisitedStreet.EntryTimestamp.Set(entryTimestamp),
//                 db.VisitedStreet.EntryLatitude.Set(entryLatitude),
//                 db.VisitedStreet.EntryLongitude.Set(entryLongitude),
//                 db.VisitedStreet.User.Link(db.User.ID.Equals(clerkUserID)),
//                 optionalParams...,
//             ).Exec(ctx)
//             if err != nil {
//                 return err
//             }
//         }
//         // If either StreetID or StreetName exists, skip this record (don't save)
//     }
//     return nil
// }



func (s *VisitorService) SaveVisitedStreets(ctx context.Context, clerkUserID string, req types.SaveVisitedStreetsRequest) (*types.SaveVisitedStreetsResponse, error) {
	seen := make(map[string]struct{})
	newStreetsCount := 0

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

		// Check if a record with the same StreetID OR StreetName already exists
		existingByID, err := s.client.VisitedStreet.FindFirst(
			db.VisitedStreet.UserID.Equals(clerkUserID),
			db.VisitedStreet.StreetID.Equals(street.StreetID),
		).Exec(ctx)
		if err != nil && !errors.Is(err, db.ErrNotFound) {
			return nil, err
		}

		existingByName, err := s.client.VisitedStreet.FindFirst(
			db.VisitedStreet.UserID.Equals(clerkUserID),
			db.VisitedStreet.StreetName.Equals(street.StreetName),
		).Exec(ctx)
		if err != nil && !errors.Is(err, db.ErrNotFound) {
			return nil, err
		}

		// If neither StreetID nor StreetName exists, create the record
		if existingByID == nil && existingByName == nil {
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
				return nil, err
			}
			
			newStreetsCount++
			fmt.Printf("New street added for user %s: %s (ID: %s)\n", clerkUserID, street.StreetName, street.StreetID)
		}
		// If either StreetID or StreetName exists, skip this record (don't save)
	}

	// Award points for new streets visited (10 points per street)
	var updatedRank *db.RankModel
	var err error
	if newStreetsCount > 0 {
		updatedRank, err = s.rankService.AddPointsForVisitedStreet(ctx, clerkUserID, newStreetsCount)
		if err != nil {
			// Log error but don't fail the street saving operation
			fmt.Printf("Warning: Failed to update rank for user %s: %v\n", clerkUserID, err)
		} else {
			fmt.Printf("Awarded %d points (%d streets) to user %s. New total: %d points, Level: %s\n", 
				newStreetsCount*10, newStreetsCount, clerkUserID, updatedRank.Points, string(updatedRank.Level))
		}
	}

	// Get level progress info
	var levelProgress *types.LevelProgressInfo
	if updatedRank != nil {
		levelProgress, err = s.rankService.GetLevelProgress(ctx, clerkUserID)
		if err != nil {
			fmt.Printf("Warning: Failed to get level progress for user %s: %v\n", clerkUserID, err)
		}
	}

	// Return response with ranking information
	response := &types.SaveVisitedStreetsResponse{
		Status:           "success",
		NewStreetsCount:  newStreetsCount,
		PointsAwarded:    newStreetsCount * 10,
	}

	if levelProgress != nil {
		response.RankInfo = &types.RankInfo{
			CurrentLevel:       string(levelProgress.CurrentLevel),
			CurrentPoints:      levelProgress.CurrentPoints,
			NextLevel:         string(levelProgress.NextLevel),
			PointsToNextLevel: levelProgress.PointsToNextLevel,
			ProgressPercentage: levelProgress.ProgressPercentage,
		}
	}

	return response, nil
}