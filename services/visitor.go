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

		// Calculate duration if exit timestamp is provided
		var durationSeconds *int
		var exitTimestamp *prismaTypes.BigInt
		if street.ExitTimestamp != nil {
			exitTs := prismaTypes.BigInt(*street.ExitTimestamp)
			exitTimestamp = &exitTs
			duration := int(*street.ExitTimestamp - street.EntryTimestamp)
			durationSeconds = &duration
		}
		if street.DurationSeconds != nil {
			durationSeconds = street.DurationSeconds
		}

		// Check if a record with the same userId and streetId already exists (unique constraint)
		existingRecord, err := s.client.VisitedStreet.FindUnique(
			db.VisitedStreet.UserIDStreetID(
				db.VisitedStreet.UserID.Equals(clerkUserID),
				db.VisitedStreet.StreetID.Equals(street.StreetID),
			),
		).Exec(ctx)
		if err != nil && !errors.Is(err, db.ErrNotFound) {
			return nil, err
		}
		if existingRecord == nil {
			// New street for this user
			totalTimeSpent := 0
			if durationSeconds != nil {
				totalTimeSpent = *durationSeconds
			}
			averageTimeSpent := totalTimeSpent // first visit

			// Required fields (exact types expected)
			reqStreetID := db.VisitedStreet.StreetID.Set(street.StreetID)
			reqStreetName := db.VisitedStreet.StreetName.Set(street.StreetName)
			reqEntryTimestamp := db.VisitedStreet.EntryTimestamp.Set(entryTimestamp)
			reqFirstVisit := db.VisitedStreet.FirstVisit.Set(entryTimestamp)
			reqLastVisit := db.VisitedStreet.LastVisit.Set(entryTimestamp)
			reqUser := db.VisitedStreet.User.Link(db.User.ID.Equals(clerkUserID))

			// Optional fields
			optionalParams := []db.VisitedStreetSetParam{
				db.VisitedStreet.SessionID.Set(req.SessionID),
				db.VisitedStreet.EntryLatitude.Set(entryLatitude),
				db.VisitedStreet.EntryLongitude.Set(entryLongitude),
				db.VisitedStreet.VisitCount.Set(1),
				db.VisitedStreet.TotalTimeSpent.Set(totalTimeSpent),
				db.VisitedStreet.AverageTimeSpent.Set(averageTimeSpent),
			}
			if exitTimestamp != nil {
				optionalParams = append(optionalParams, db.VisitedStreet.ExitTimestamp.Set(*exitTimestamp))
			}
			if durationSeconds != nil {
				optionalParams = append(optionalParams, db.VisitedStreet.DurationSeconds.Set(*durationSeconds))
			}

			// Correct call
			_, err = s.client.VisitedStreet.CreateOne(
				reqStreetID,
				reqStreetName,
				reqEntryTimestamp,
				reqFirstVisit,
				reqLastVisit,
				reqUser,
				optionalParams..., // variadic optional fields
			).Exec(ctx)
			if err != nil {
				return nil, err
			}

			newStreetsCount++
			fmt.Printf("New street added for user %s: %s (ID: %s)\n", clerkUserID, street.StreetName, street.StreetID)
		} else {
			// This street already exists for this user - update the statistics
			newVisitCount := existingRecord.VisitCount + 1
			newTotalTimeSpent := existingRecord.TotalTimeSpent
			if durationSeconds != nil {
				newTotalTimeSpent += *durationSeconds
			}

			newAverageTimeSpent := 0
			if newVisitCount > 0 {
				newAverageTimeSpent = newTotalTimeSpent / newVisitCount
			}

			// Determine first visit (earliest timestamp)
			firstVisit := existingRecord.FirstVisit
			if street.EntryTimestamp < int64(firstVisit) {
				firstVisit = entryTimestamp
			}

			// Update last visit to current entry
			lastVisit := entryTimestamp

			// Prepare update parameters
			updateParams := []db.VisitedStreetSetParam{
				db.VisitedStreet.VisitCount.Set(newVisitCount),
				db.VisitedStreet.FirstVisit.Set(firstVisit),
				db.VisitedStreet.LastVisit.Set(lastVisit),
				db.VisitedStreet.TotalTimeSpent.Set(newTotalTimeSpent),
				db.VisitedStreet.AverageTimeSpent.Set(newAverageTimeSpent),
				// Update the latest visit details
				db.VisitedStreet.EntryTimestamp.Set(entryTimestamp),
				db.VisitedStreet.EntryLatitude.Set(entryLatitude),
				db.VisitedStreet.EntryLongitude.Set(entryLongitude),
			}

			// Add optional parameters for latest visit
			if req.SessionID != "" {
				updateParams = append(updateParams, db.VisitedStreet.SessionID.Set(req.SessionID))
			}
			if exitTimestamp != nil {
				updateParams = append(updateParams, db.VisitedStreet.ExitTimestamp.Set(*exitTimestamp))
			}
			if durationSeconds != nil {
				updateParams = append(updateParams, db.VisitedStreet.DurationSeconds.Set(*durationSeconds))
			}

			_, err = s.client.VisitedStreet.FindUnique(
				db.VisitedStreet.UserIDStreetID(
					db.VisitedStreet.UserID.Equals(clerkUserID),
					db.VisitedStreet.StreetID.Equals(street.StreetID),
				),
			).Update(updateParams...).Exec(ctx)
			if err != nil {
				return nil, err
			}

			fmt.Printf("Updated street stats for user %s: %s (ID: %s) - Visit #%d\n",
				clerkUserID, street.StreetName, street.StreetID, newVisitCount)
		}
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

func (s *VisitorService) GetVisitedStreets(ctx context.Context, clerkUserID string) (*types.VisitedStreetsResponse, error) {
	visitedStreets, err := s.client.VisitedStreet.FindMany(
		db.VisitedStreet.UserID.Equals(clerkUserID),
	).OrderBy(
		db.VisitedStreet.LastVisit.Order(db.DESC),
	).Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get visited streets: %w", err)
	}

	var streets []types.VisitedStreetDetails
	for _, street := range visitedStreets {

		sessionId, _ := street.SessionID()
		entLat, _ := street.EntryLatitude()
		entLng, _ := street.EntryLongitude()

		streetDetail := types.VisitedStreetDetails{
			StreetID:         street.StreetID,
			StreetName:       street.StreetName,
			SessionID:        &sessionId,
			EntryTimestamp:   int64(street.EntryTimestamp),
			EntryLatitude:    &entLat,
			EntryLongitude:   &entLng,
			VisitCount:       int64(street.VisitCount),
			FirstVisit:       int64(street.FirstVisit),
			LastVisit:        int64(street.LastVisit),
			TotalTimeSpent:   int64(street.TotalTimeSpent),
			AverageTimeSpent: int64(street.AverageTimeSpent),
		}

		if street.ExitTimestamp != nil {
			extTms, _ := street.ExitTimestamp()
			exitTime := int64(extTms)
			streetDetail.ExitTimestamp = &exitTime
		}

		if street.DurationSeconds != nil {
			durSec, _ := street.DurationSeconds()
			streetDetail.DurationSeconds = &durSec
		}

		streets = append(streets, streetDetail)
	}

	return &types.VisitedStreetsResponse{
		Status: "success",
		Data:   streets,
		Count:  len(streets),
	}, nil
}
