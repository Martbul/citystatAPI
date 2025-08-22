
// internal/handlers/invite.go
package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"citystatAPI/internal/middleware"
	"citystatAPI/internal/services"
	"citystatAPI/types"
)

type InviteHandler struct {
	userService   *services.UserService
	friendService *services.FriendService
}

func NewInviteHandler(userService *services.UserService, friendService *services.FriendService) *InviteHandler {
	return &InviteHandler{
		userService:   userService,
		friendService: friendService,
	}
}

func (h *InviteHandler) ProcessInvite(w http.ResponseWriter, r *http.Request) {
	invitedBy := r.URL.Query().Get("invitedBy")
	if invitedBy == "" {
		middleware.ErrorResponse(w, "Missing invitedBy parameter", http.StatusBadRequest)
		return
	}

	invitingUser, err := h.userService.GetOrCreateUser(r.Context(), invitedBy)
	if err != nil {
		middleware.ErrorResponse(w, "Invalid invite link", http.StatusNotFound)
		return
	}

	// Convert db.User to response format
	response := types.InviteInfoResponse{
		InvitedBy: types.InviteUserInfo{
			ID: invitingUser.ID,
			// TODO: Convert pgx types to string pointers
			// UserName:  convertPgxText(invitingUser.UserName),
			// FirstName: convertPgxText(invitingUser.FirstName),
			// LastName:  convertPgxText(invitingUser.LastName),
			// ImageURL:  convertPgxText(invitingUser.ImageUrl),
		},
		Message: "You've been invited to join CityStat!",
	}

	middleware.JSONResponse(w, response, http.StatusOK)
}

func (h *InviteHandler) AcceptInvite(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		middleware.ErrorResponse(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}

	var req types.AcceptInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.ErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.InvitedBy == "" {
		middleware.ErrorResponse(w, "InvitedBy is required", http.StatusBadRequest)
		return
	}

	if req.InvitedBy == userID {
		middleware.ErrorResponse(w, "You cannot add yourself as a friend", http.StatusBadRequest)
		return
	}

	friend, err := h.friendService.AddFriend(r.Context(), userID, req.InvitedBy)
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

	response := types.AcceptInviteResponse{
		Message: "Invite accepted successfully",
		Friend:  *friend,
	}
	middleware.JSONResponse(w, response, http.StatusOK)
}

func (h *InviteHandler) GetInviteLink(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		middleware.ErrorResponse(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}

	user, err := h.userService.GetOrCreateUser(r.Context(), userID)
	if err != nil {
		middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	baseURL := "https://yourapp.com/invite" // TODO: Get from config
	inviteLink := baseURL + "?invitedBy=" + user.ID

	// TODO: Add username to link if available
	// if user.UserName.Valid {
	//     inviteLink += "&userName=" + user.UserName.String
	// }

	response := types.InviteLinkResponse{
		InviteLink: inviteLink,
		Message:    "Invite link generated successfully",
	}

	middleware.JSONResponse(w, response, http.StatusOK)
}
	