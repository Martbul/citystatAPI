package handlers

import (
	"net/http"
	"strconv"

	"citystatAPI/middleware"
	"citystatAPI/services"
	"citystatAPI/types"
)

type RankHandler struct {
	rankService *services.RankService
}

func NewRankHandler(rankService *services.RankService) *RankHandler {
	return &RankHandler{rankService: rankService}
}

// GetUserRank handles GET /api/rank
func (h *RankHandler) GetUserRank(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		middleware.ErrorResponse(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}

	rank, err := h.rankService.GetUserRank(r.Context(), userID)
	if err != nil {
		middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := types.RankResponse{
		ID:           rank.ID,
		UserID:       rank.UserID,
		Points:       rank.Points,
		Level:        string(rank.Level),
		CreatedAt:    rank.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:    rank.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	middleware.JSONResponse(w, response, http.StatusOK)
}

// GetLevelProgress handles GET /api/rank/progress
func (h *RankHandler) GetLevelProgress(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		middleware.ErrorResponse(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}

	progress, err := h.rankService.GetLevelProgress(r.Context(), userID)
	if err != nil {
		middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := types.LevelProgressResponse{
		CurrentLevel:          string(progress.CurrentLevel),
		CurrentPoints:         progress.CurrentPoints,
		NextLevel:            string(progress.NextLevel),
		PointsToNextLevel:    progress.PointsToNextLevel,
		ProgressPercentage:   progress.ProgressPercentage,
		CurrentLevelThreshold: progress.CurrentLevelThreshold,
		NextLevelThreshold:   progress.NextLevelThreshold,
	}

	middleware.JSONResponse(w, response, http.StatusOK)
}

// GetLeaderboard handles GET /api/rank/leaderboard
func (h *RankHandler) GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		middleware.ErrorResponse(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}

	// Get limit from query params (default 10)
	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	leaderboard, err := h.rankService.GetLeaderboard(r.Context(), limit, userID)
	if err != nil {
		middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	middleware.JSONResponse(w, leaderboard, http.StatusOK)
}