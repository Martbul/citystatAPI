package services

import (
	"context"
	"fmt"
	"math"

	"citystatAPI/prisma/db"
	"citystatAPI/types"
)

type RankService struct {
	client *db.PrismaClient
}

func NewRankService(client *db.PrismaClient) *RankService {
	return &RankService{client: client}
}
// GetUserRank retrieves or creates a user's rank
func (s *RankService) GetUserRank(ctx context.Context, userID string) (*db.RankModel, error) {
	// Try to get existing rank
	rank, err := s.client.Rank.FindUnique(
		db.Rank.UserID.Equals(userID),
	).Exec(ctx)
	
	if err == db.ErrNotFound {
		// Create new rank with default values
		rank, err = s.client.Rank.CreateOne(
			db.Rank.User.Link(db.User.ID.Equals(userID)),
			db.Rank.Points.Set(0),
			db.Rank.Level.Set(db.LevelIron),
		).Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create rank: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to get rank: %w", err)
	}
	
	return rank, nil
}


// AddPointsForVisitedStreet adds 10 points for each visited street and updates rank
func (s *RankService) AddPointsForVisitedStreet(ctx context.Context, userID string, streetsCount int) (*db.RankModel, error) {
	// Get or create user rank
	rank, err := s.GetUserRank(ctx, userID)
	if err != nil {
		return nil, err
	}
	
	// Calculate points to add (10 points per street)
	pointsToAdd := streetsCount * 10
	newPoints := rank.Points + pointsToAdd
	
	// Calculate new level based on exponential progression
	newLevel := s.calculateLevel(newPoints)
	
	// Update rank in database
	updatedRank, err := s.client.Rank.FindUnique(
		db.Rank.UserID.Equals(userID),
	).Update(
		db.Rank.Points.Set(newPoints),
		db.Rank.Level.Set(newLevel),
	).Exec(ctx)
	
	if err != nil {
		return nil, fmt.Errorf("failed to update rank: %w", err)
	}
	
	fmt.Printf("Updated rank for user %s: %d points, level %s\n", userID, newPoints, string(newLevel))
	return updatedRank, nil
}

// calculateLevel determines the level based on points using exponential progression
// Starting at 100 points for first level up, then exponentially increasing
func (s *RankService) calculateLevel(points int) db.Level {
	// Define level thresholds with exponential growth
	// Base: 100 points for Bronze, then multiply by ~2.5 each time
	levelThresholds := map[db.Level]int{
		db.LevelIron:     0,     // 0 points
		db.LevelBronze:   100,   // 100 points
		db.LevelSilver:   250,   // 250 points (2.5x)
		db.LevelGold:     625,   // 625 points (2.5x)
		db.LevelDimond:   1563,  // 1563 points (2.5x)
		db.LevelPlatinum: 3907,  // 3907 points (2.5x)
		db.LevelMaster:   9768,  // 9768 points (2.5x)
	}
	
	// Determine level based on points
	if points >= levelThresholds[db.LevelMaster] {
		return db.LevelMaster
	} else if points >= levelThresholds[db.LevelPlatinum] {
		return db.LevelPlatinum
	} else if points >= levelThresholds[db.LevelDimond] {
		return db.LevelDimond
	} else if points >= levelThresholds[db.LevelGold] {
		return db.LevelGold
	} else if points >= levelThresholds[db.LevelSilver] {
		return db.LevelSilver
	} else if points >= levelThresholds[db.LevelBronze] {
		return db.LevelBronze
	}
	
	return db.LevelIron
}

// GetLevelProgress returns current level, points, and progress to next level
func (s *RankService) GetLevelProgress(ctx context.Context, userID string) (*types.LevelProgressInfo, error) {
	rank, err := s.GetUserRank(ctx, userID)
	if err != nil {
		return nil, err
	}
	
	currentLevel := rank.Level
	currentPoints := rank.Points
	
	// Get next level info
	nextLevel, nextLevelThreshold := s.getNextLevelInfo(currentLevel)
	currentLevelThreshold := s.getCurrentLevelThreshold(currentLevel)
	
	// Calculate progress percentage
	var progressPercentage float64
	if nextLevel != currentLevel {
		pointsNeededForNext := nextLevelThreshold - currentLevelThreshold
		pointsEarnedInCurrentLevel := currentPoints - currentLevelThreshold
		if pointsNeededForNext > 0 {
			progressPercentage = float64(pointsEarnedInCurrentLevel) / float64(pointsNeededForNext) * 100
		}
	} else {
		progressPercentage = 100 // Already at max level
	}
	
	return &types.LevelProgressInfo{
		CurrentLevel:         currentLevel,
		CurrentPoints:        currentPoints,
		NextLevel:           nextLevel,
		PointsToNextLevel:   nextLevelThreshold - currentPoints,
		ProgressPercentage:  progressPercentage,
		CurrentLevelThreshold: currentLevelThreshold,
		NextLevelThreshold:   nextLevelThreshold,
	}, nil
}

func (s *RankService) getCurrentLevelThreshold(level db.Level) int {
	thresholds := map[db.Level]int{
		db.LevelIron:     0,
		db.LevelBronze:   100,
		db.LevelSilver:   250,
		db.LevelGold:     625,
		db.LevelDimond:   1563,
		db.LevelPlatinum: 3907,
		db.LevelMaster:   9768,
	}
	return thresholds[level]
}

func (s *RankService) getNextLevelInfo(currentLevel db.Level) (db.Level, int) {
	levelOrder := []db.Level{
		db.LevelIron,
		db.LevelBronze,
		db.LevelSilver,
		db.LevelGold,
		db.LevelDimond,
		db.LevelPlatinum,
		db.LevelMaster,
	}
	
	thresholds := map[db.Level]int{
		db.LevelIron:     0,
		db.LevelBronze:   100,
		db.LevelSilver:   250,
		db.LevelGold:     625,
		db.LevelDimond:   1563,
		db.LevelPlatinum: 3907,
		db.LevelMaster:   9768,
	}
	
	for i, level := range levelOrder {
		if level == currentLevel && i < len(levelOrder)-1 {
			nextLevel := levelOrder[i+1]
			return nextLevel, thresholds[nextLevel]
		}
	}
	
	// Already at max level
	return currentLevel, thresholds[currentLevel]
}

// Alternative exponential calculation using mathematical formula
func (s *RankService) calculateLevelMathematical(points int) db.Level {
	if points < 100 {
		return db.LevelIron
	}
	
	// Using exponential formula: threshold = 100 * (2.5^(level-1))
	// Solve for level: level = log(threshold/100) / log(2.5) + 1
	level := math.Log(float64(points)/100.0) / math.Log(2.5) + 1
	
	switch {
	case level >= 6:
		return db.LevelMaster
	case level >= 5:
		return db.LevelPlatinum
	case level >= 4:
		return db.LevelDimond
	case level >= 3:
		return db.LevelGold
	case level >= 2:
		return db.LevelSilver
	case level >= 1:
		return db.LevelBronze
	default:
		return db.LevelIron
	}
}

// GetLeaderboard returns top users by points with their ranking
func (s *RankService) GetLeaderboard(ctx context.Context, limit int, currentUserID string) (*types.LeaderboardData, error) {
	if limit <= 0 {
		limit = 10 // Default to top 10
	}

	// Get top users by points
	topRanks, err := s.client.Rank.FindMany().With(
		db.Rank.User.Fetch(),
	).OrderBy(
		db.Rank.Points.Order(db.DESC),
	).Take(limit).Exec(ctx)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get leaderboard: %w", err)
	}

	// Convert to leaderboard entries
	leaderboard := make([]types.LeaderboardEntry, len(topRanks))
	var currentUserRank *types.LeaderboardEntry
	
	for i, rank := range topRanks {
		user := rank.User()
		
		
		firstName, _ := user.FirstName()
		lastName, _ := user.LastName()
		userName, _ := user.UserName()
		imageURL := user.ImageURL
		
		entry := types.LeaderboardEntry{
			UserID:    user.ID,
			UserName:  &userName,
			FirstName: &firstName,
			LastName:  &lastName,
			ImageURL:  &imageURL,
			Points:    rank.Points,
			Level:     string(rank.Level),
			Rank:      i + 1,
		}
		
		leaderboard[i] = entry
		
		// Check if this is the current user
		if user.ID == currentUserID {
			currentUserRank = &entry
		}
	}
	
	// If current user is not in top results, find their rank
	if currentUserRank == nil {
		currentUserRank, err = s.getCurrentUserRank(ctx, currentUserID)
		if err != nil {
			fmt.Printf("Warning: failed to get current user rank: %v\n", err)
		}
	}
	
	// Get total number of ranked users
	totalUsers, err := s.client.Rank.FindMany().Exec(ctx)
	if err != nil {
		fmt.Printf("Warning: failed to get total user count: %v\n", err)
	}
	
	return &types.LeaderboardData{
		Leaderboard: leaderboard,
		CurrentUser: currentUserRank,
		TotalUsers:  len(totalUsers),
	}, nil
}

func (s *RankService) getCurrentUserRank(ctx context.Context, userID string) (*types.LeaderboardEntry, error) {
	// Get current user's rank
	userRank, err := s.client.Rank.FindUnique(
		db.Rank.UserID.Equals(userID),
	).With(
		db.Rank.User.Fetch(),
	).Exec(ctx)
	
	if err != nil {
		return nil, err
	}
	
	user := userRank.User()
	
	// Count users with more points to determine rank position
	higherRanks, err := s.client.Rank.FindMany(
		db.Rank.Points.Gt(userRank.Points),
	).Exec(ctx)
	
	if err != nil {
		return nil, err
	}
	
	firstName, _ := user.FirstName()
	lastName, _ := user.LastName()
	userName, _ := user.UserName()
	imageURL := user.ImageURL
	
	return &types.LeaderboardEntry{
		UserID:    user.ID,
		UserName:  &userName,
		FirstName: &firstName,
		LastName:  &lastName,
		ImageURL:  &imageURL,
		Points:    userRank.Points,
		Level:     string(userRank.Level),
		Rank:      len(higherRanks) + 1,
	}, nil
}

