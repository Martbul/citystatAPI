package handlers

import (
	"citystatAPI/middleware"
	"citystatAPI/services"
	"citystatAPI/types"
	"encoding/json"
	"net/http"
)

type VisitorHandler struct {
	visitorService *services.VisitorService
}


func NewVisitorHandler(visitorService *services.VisitorService) *VisitorHandler {
	return &VisitorHandler{visitorService: visitorService}
}


func (h *VisitorHandler) GetLocationPermission(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.GetUserID(r)
	if !ok {
		middleware.ErrorResponse(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}

	hasLocationPermission, err := h.visitorService.GetLocationPermission(r.Context(), userID)
	if err != nil {
		middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	middleware.JSONResponse(w, hasLocationPermission, http.StatusOK)
}



func (h *VisitorHandler) SaveLocationPermission(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.GetUserID(r)
	if !ok {
		middleware.ErrorResponse(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}

var req types.SaveLocationPermitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.ErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}


	hasLocationPermission, err := h.visitorService.SaveLocationPermission(r.Context(), userID, req.HasLocationPermission)
	if err != nil {
	
		middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := types.SaveLocationPermitionResponse{
		Success: hasLocationPermission,
	}
	middleware.JSONResponse(w, response, http.StatusOK)}


// func (h *VisitorHandler) AddVisitedStreets(w http.ResponseWriter, r *http.Request) {
// 	userID, ok := middleware.GetUserID(r)
// 	if !ok {
// 		middleware.ErrorResponse(w, "User ID not found in context", http.StatusUnauthorized)
// 		return
// 	}

// 	var req types.AddVisitedStreetsReq
// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 		middleware.ErrorResponse(w, "Invalid request body", http.StatusBadRequest)
// 		return
// 	}

// 	if req.FriendID == "" {
// 		middleware.ErrorResponse(w, "Friend ID is required", http.StatusBadRequest)
// 		return
// 	}

// 	// Check if trying to add themselves
// 	if req.FriendID == userID {
// 		middleware.ErrorResponse(w, "You cannot add yourself as a friend", http.StatusBadRequest)
// 		return
// 	}

// 	friend, err := h.friendService.AddFriend(r.Context(), userID, req.FriendID)
// 	if err != nil {
// 		// Handle specific error cases
// 		if strings.Contains(err.Error(), "not found") {
// 			middleware.ErrorResponse(w, "User not found", http.StatusNotFound)
// 			return
// 		}
// 		if strings.Contains(err.Error(), "already friends") {
// 			middleware.ErrorResponse(w, "Already friends with this user", http.StatusBadRequest)
// 			return
// 		}
// 		middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}

// 	response := types.AddFriendResponse{
// 		Message: "Friend added successfully",
// 		Friend:  *friend,
// 	}
// 	middleware.JSONResponse(w, response, http.StatusOK)
// }
