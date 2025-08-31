package types

import "citystatAPI/prisma/db"

// RankResponse represents the response for user rank information
type RankResponse struct {
	ID        string `json:"id"`
	UserID    string `json:"userId"`
	Points    int    `json:"points"`
	Level     string `json:"level"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// LevelProgressResponse represents the progress towards next level
type LevelProgressResponse struct {
	CurrentLevel          string  `json:"currentLevel"`
	CurrentPoints         int     `json:"currentPoints"`
	NextLevel            string  `json:"nextLevel"`
	PointsToNextLevel    int     `json:"pointsToNextLevel"`
	ProgressPercentage   float64 `json:"progressPercentage"`
	CurrentLevelThreshold int     `json:"currentLevelThreshold"`
	NextLevelThreshold   int     `json:"nextLevelThreshold"`
}

// RankInfo represents rank information in other responses
type RankInfo struct {
	CurrentLevel       string  `json:"currentLevel"`
	CurrentPoints      int     `json:"currentPoints"`
	NextLevel         string  `json:"nextLevel"`
	PointsToNextLevel int     `json:"pointsToNextLevel"`
	ProgressPercentage float64 `json:"progressPercentage"`
}



// LeaderboardResponse represents the leaderboard response
type LeaderboardResponse struct {
	Leaderboard []LeaderboardEntry `json:"leaderboard"`
	UserRank    *LeaderboardEntry  `json:"userRank,omitempty"`
	TotalUsers  int                `json:"totalUsers"`
}

// SaveVisitedStreetsResponse represents the enhanced response for saving visited streets
type SaveVisitedStreetsResponse struct {
	Status          string    `json:"status"`
	NewStreetsCount int       `json:"newStreetsCount"`
	PointsAwarded   int       `json:"pointsAwarded"`
	RankInfo        *RankInfo `json:"rankInfo,omitempty"`
}

type LevelProgressInfo struct {
	CurrentLevel          db.Level `json:"currentLevel"`
	CurrentPoints         int      `json:"currentPoints"`
	NextLevel            db.Level `json:"nextLevel"`
	PointsToNextLevel    int      `json:"pointsToNextLevel"`
	ProgressPercentage   float64  `json:"progressPercentage"`
	CurrentLevelThreshold int      `json:"currentLevelThreshold"`
	NextLevelThreshold   int      `json:"nextLevelThreshold"`
}

type LeaderboardEntry struct {
	UserID    string  `json:"userId"`
	UserName  *string `json:"userName"`
	FirstName *string `json:"firstName"`
	LastName  *string `json:"lastName"`
	ImageURL  *string `json:"imageUrl"`
	Points    int     `json:"points"`
	Level     string  `json:"level"`
	Rank      int     `json:"rank"`
}

type LeaderboardData struct {
	Leaderboard []LeaderboardEntry `json:"leaderboard"`
	CurrentUser *LeaderboardEntry  `json:"currentUser,omitempty"`
	TotalUsers  int                `json:"totalUsers"`
}