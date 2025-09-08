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
	client      *db.PrismaClient
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

	fmt.Println("enableLocationTracking")
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





// Updated function
func (s *VisitorService) GetVisitedStreets(ctx context.Context, clerkUserID string) (*types.GetVisitedStreetsResponse, error) {
    visitedStreets, err := s.client.VisitedStreet.FindMany(
        db.VisitedStreet.UserID.Equals(clerkUserID),
    ).Exec(ctx)
    
    if err != nil {
        return &types.GetVisitedStreetsResponse{
            Data:    []types.VisitedStreetRequest{},
            Message: fmt.Sprintf("Failed to retrieve visited streets: %v", err),
            Status:  "error",
        }, fmt.Errorf("database error: %w", err)
    }

    // Handle case where user exists but has no visited streets
    if len(visitedStreets) == 0 {
        return &types.GetVisitedStreetsResponse{
            Data:    []types.VisitedStreetRequest{},
            Message: "No visited streets found for this user",
            Status:  "success",
        }, nil
    }

    var result []types.VisitedStreetRequest
    
    for _, street := range visitedStreets {
        var exitTimestamp *int64
        if street.ExitTimestamp != nil {
            v, _ := street.ExitTimestamp()
            val := int64(v)
            exitTimestamp = &val
        }
        
        var durationSeconds *int
        if street.DurationSeconds != nil {
            v, _ := street.DurationSeconds()
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

    return &types.GetVisitedStreetsResponse{
        Data:    result,
        Message: fmt.Sprintf("Successfully retrieved %d visited streets", len(result)),
        Status:  "success",
    }, nil
}


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
		Status:          "success",
		NewStreetsCount: newStreetsCount,
		PointsAwarded:   newStreetsCount * 10,
	}

	if levelProgress != nil {
		response.RankInfo = &types.RankInfo{
			CurrentLevel:       string(levelProgress.CurrentLevel),
			CurrentPoints:      levelProgress.CurrentPoints,
			NextLevel:          string(levelProgress.NextLevel),
			PointsToNextLevel:  levelProgress.PointsToNextLevel,
			ProgressPercentage: levelProgress.ProgressPercentage,
		}
	}

	return response, nil
}

func (s *VisitorService) GetStreetVisitStats(ctx context.Context, clerkUserID string) (*types.StreetVisitApiResponse, error) {
	// Use FindMany instead of FindUnique since UserID is not a unique field
	visitedStreetsStats, err := s.client.StreetVisitStat.FindMany(
		db.StreetVisitStat.UserID.Equals(clerkUserID),
	).Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("Failed to get visited streets stats: %w", err)
	}

	fmt.Println("visited streets stats:", visitedStreetsStats)
	
	// Convert db.StreetVisitStatModel to types.StreetStat
	var streetStats []types.StreetStat
	for _, stat := range visitedStreetsStats {
		streetStats = append(streetStats, types.StreetStat{
			StreetID:         stat.StreetID,
			StreetName:       stat.StreetName,
			VisitCount:       int64(stat.VisitCount),
			FirstVisit:       int64(stat.FirstVisit),
			LastVisit:        int64(stat.LastVisit),
			TotalTimeSpent:   int64(stat.TotalTimeSpent),
			AverageTimeSpent: int64(stat.AverageTimeSpent),
		})
	}

	ret := &types.StreetVisitApiResponse{
		Status: "success",
		Data:   streetStats,
	}
	return ret, nil
}

func (s *VisitorService) SaveStreetVisitStats(ctx context.Context, clerkUserID string, req types.SaveStreetVisitStatsRequest) (*types.StreetVisitApiResponse, error) {
	var visitedStreetsStats []types.StreetStat

	for _, streetData := range req.StreetStats {
		// Check if a record with the same streetId exists
		existingStreetById, err := s.client.StreetVisitStat.FindFirst(
			db.StreetVisitStat.UserID.Equals(clerkUserID),
			db.StreetVisitStat.StreetID.Equals(streetData.StreetID),
		).Exec(ctx)
		if err != nil && !errors.Is(err, db.ErrNotFound) {
			return nil, fmt.Errorf("error checking existing street by ID: %w", err)
		}

		// Check if a record with the same streetName exists (fallback for streets without ID)
		existingByName, err := s.client.StreetVisitStat.FindFirst(
			db.StreetVisitStat.UserID.Equals(clerkUserID),
			db.StreetVisitStat.StreetName.Equals(streetData.StreetName),
		).Exec(ctx)
		if err != nil && !errors.Is(err, db.ErrNotFound) {
			return nil, fmt.Errorf("error checking existing street by name: %w", err)
		}

		var updatedStreetStat *db.StreetVisitStatModel

		// Priority: Update by StreetID if exists, then by Name, otherwise create new
		if existingStreetById != nil {
			// Update existing record found by StreetID
			newVisitCount := existingStreetById.VisitCount + int(streetData.VisitCount)
			newTotalTimeSpent := existingStreetById.TotalTimeSpent + int(streetData.TotalTimeSpent)
			newAverageTimeSpent := int64(0)
			if newVisitCount > 0 {
				newAverageTimeSpent = int64(newTotalTimeSpent / newVisitCount)
			}

			// Determine first visit (earliest timestamp)
			firstVisit := existingStreetById.FirstVisit
			if streetData.FirstVisit < int64(firstVisit) {
				firstVisit = prismaTypes.BigInt(streetData.FirstVisit)
			}

			// Determine last visit (latest timestamp)
			lastVisit := existingStreetById.LastVisit
			if streetData.LastVisit > int64(lastVisit) {
				lastVisit = prismaTypes.BigInt(streetData.LastVisit)
			}

			updatedStreetStat, err = s.client.StreetVisitStat.FindUnique(
				db.StreetVisitStat.ID.Equals(existingStreetById.ID),
			).Update(
				db.StreetVisitStat.VisitCount.Set(newVisitCount),
				db.StreetVisitStat.FirstVisit.Set(firstVisit),
				db.StreetVisitStat.LastVisit.Set(lastVisit),
				db.StreetVisitStat.TotalTimeSpent.Set(newTotalTimeSpent),
				db.StreetVisitStat.AverageTimeSpent.Set(int(newAverageTimeSpent)),
				db.StreetVisitStat.StreetName.Set(streetData.StreetName), // Update name in case it changed
			).Exec(ctx)
			if err != nil {
				return nil, fmt.Errorf("error updating street visit stat by ID: %w", err)
			}

		} else if existingByName != nil {
			// Update existing record found by Name (and update StreetID if provided)
			newVisitCount := existingByName.VisitCount + int(streetData.VisitCount)
			newTotalTimeSpent := existingByName.TotalTimeSpent + int(streetData.TotalTimeSpent)
			newAverageTimeSpent := int64(0)
			if newVisitCount > 0 {
				newAverageTimeSpent = int64(newTotalTimeSpent / newVisitCount)
			}

			// Determine first visit (earliest timestamp)
			firstVisit := existingByName.FirstVisit
			if streetData.FirstVisit < int64(firstVisit) {
				firstVisit = prismaTypes.BigInt(streetData.FirstVisit)
			}

			// Determine last visit (latest timestamp)
			lastVisit := existingByName.LastVisit
			if streetData.LastVisit > int64(lastVisit) {
				lastVisit = prismaTypes.BigInt(streetData.LastVisit)
			}

			updateParams := []db.StreetVisitStatSetParam{
				db.StreetVisitStat.VisitCount.Set(newVisitCount),
				db.StreetVisitStat.FirstVisit.Set(firstVisit),
				db.StreetVisitStat.LastVisit.Set(lastVisit),
				db.StreetVisitStat.TotalTimeSpent.Set(newTotalTimeSpent),
				db.StreetVisitStat.AverageTimeSpent.Set(int(newAverageTimeSpent)),
			}

			// Update StreetID if it's provided and different
			if streetData.StreetID != "" && streetData.StreetID != existingByName.StreetID {
				updateParams = append(updateParams, db.StreetVisitStat.StreetID.Set(streetData.StreetID))
			}

			updatedStreetStat, err = s.client.StreetVisitStat.FindUnique(
				db.StreetVisitStat.ID.Equals(existingByName.ID),
			).Update(updateParams...).Exec(ctx)
			if err != nil {
				return nil, fmt.Errorf("error updating street visit stat by name: %w", err)
			}

		} else {
			averageTimeSpent := 0
			if streetData.VisitCount > 0 {
				averageTimeSpent = int(streetData.TotalTimeSpent / streetData.VisitCount)
			}

			// Alternative syntax - pass the parameters as a slice
			query := `INSERT INTO street_visit_stats (user_id, street_id, street_name, visit_count, first_visit, last_visit, total_time_spent, average_time_spent) 
          VALUES ($1, $2, $3, $4, $5, $6, $7, $8) 
          RETURNING *`

			params := []interface{}{
				clerkUserID,
				streetData.StreetID,
				streetData.StreetName,
				int(streetData.VisitCount),
				streetData.FirstVisit,
				streetData.LastVisit,
				int(streetData.TotalTimeSpent),
				averageTimeSpent,
			}

			var results []db.StreetVisitStatModel
			err := s.client.Prisma.QueryRaw(query, params...).Exec(ctx, &results)

			if err != nil {
				return nil, fmt.Errorf("error creating new street visit stat: %w", err)
			}

			if len(results) > 0 {
				updatedStreetStat = &results[0]
			} else {
				return nil, fmt.Errorf("no result returned from insert")
			}
			if err != nil {
				return nil, fmt.Errorf("error creating new street visit stat: %w", err)
			}
		}

		// Convert the updated/created record back to the response format
		responseStreetStat := types.StreetStat{
			StreetID:         updatedStreetStat.StreetID,
			StreetName:       updatedStreetStat.StreetName,
			VisitCount:       int64(updatedStreetStat.VisitCount),
			FirstVisit:       int64(updatedStreetStat.FirstVisit),
			LastVisit:        int64(updatedStreetStat.LastVisit),
			TotalTimeSpent:   int64(updatedStreetStat.TotalTimeSpent),
			AverageTimeSpent: int64(updatedStreetStat.AverageTimeSpent),
		}

		visitedStreetsStats = append(visitedStreetsStats, responseStreetStat)
	}

	ret := &types.StreetVisitApiResponse{
		Status: "success",
		Data:   visitedStreetsStats,
	}
	return ret, nil
}
