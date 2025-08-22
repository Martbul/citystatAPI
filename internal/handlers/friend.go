
// internal/handlers/friend.go
package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"citystatAPI/internal/middleware"
	"citystatAPI/internal/services"
	"citystatAPI/types"

	"github.com/gorilla/mux"
)

type FriendHandler struct {
	friendService *services.FriendService
}

func NewFriendHandler(friendService *services.FriendService) *FriendHandler {
	return &FriendHandler{friendService: friendService}
}

func (h *FriendHandler) AddFriend(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		middleware.ErrorResponse(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}

	var req types.AddFriendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.ErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.FriendID == "" {
		middleware.ErrorResponse(w, "Friend ID is required", http.StatusBadRequest)
		return
	}

	if req.FriendID == userID {
		middleware.ErrorResponse(w, "You cannot add yourself as a friend", http.StatusBadRequest)
		return
	}

	friend, err := h.friendService.AddFriend(r.Context(), userID, req.FriendID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			middleware.ErrorResponse(w, "User not found", http.StatusNotFound)
			return
		}
		if strings.Contains(err.Error(), "already friends") {
			middleware.ErrorResponse(w, "Already friends with this user", http.StatusBadRequest)
			return
		}
		middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := types.AddFriendResponse{
		Message: "Friend added successfully",
		Friend:  *friend,
	}
	middleware.JSONResponse(w, response, http.StatusOK)
}

func (h *FriendHandler) GetFriends(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		middleware.ErrorResponse(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}

	friends, err := h.friendService.GetUserFriends(r.Context(), userID)
	if err != nil {
		middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := types.FriendsListResponse{Friends: friends}
	middleware.JSONResponse(w, response, http.StatusOK)
}

func (h *FriendHandler) RemoveFriend(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		middleware.ErrorResponse(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	friendID := vars["friendId"]
	if friendID == "" {
		middleware.ErrorResponse(w, "Friend ID is required", http.StatusBadRequest)
		return
	}

	err := h.friendService.RemoveFriend(r.Context(), userID, friendID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			middleware.ErrorResponse(w, "Friendship not found", http.StatusNotFound)
			return
		}
		middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	middleware.JSONResponse(w, map[string]string{"message": "Friend removed successfully"}, http.StatusOK)
}