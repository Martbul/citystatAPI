package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"citystatAPI/middleware"
	"citystatAPI/services"
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

// ProcessInvite handles GET /invite?invitedBy=userId&userName=username
func (h *InviteHandler) ProcessInvite(w http.ResponseWriter, r *http.Request) {
	// This endpoint can be accessed without authentication for initial invite processing
	// but will require auth for actually adding the friend relationship
	
	invitedBy := r.URL.Query().Get("invitedBy")
	// userName := r.URL.Query().Get("userName")
	
	if invitedBy == "" {
		middleware.ErrorResponse(w, "Missing invitedBy parameter", http.StatusBadRequest)
		return
	}

	// Get the inviting user's information
	invitingUser, err := h.userService.GetOrCreateUser(r.Context(), invitedBy)
	if err != nil {
		middleware.ErrorResponse(w, "Invalid invite link", http.StatusNotFound)
		return
	}
	invitingUserName,_ := invitingUser.UserName()
		invitingFirstName,_ := invitingUser.FirstName()

	invitingLastName,_ := invitingUser.LastName()
	invitingInageURL := invitingUser.ImageURL

	response := types.InviteInfoResponse{
		InvitedBy: types.InviteUserInfo{
			ID:        invitingUser.ID,
			UserName:  &invitingUserName,
			FirstName: &invitingFirstName,
			LastName: &invitingLastName,
			ImageURL:  &invitingInageURL,
		},
		Message: "You've been invited to join CityStat!",
	}

	middleware.JSONResponse(w, response, http.StatusOK)
}

// AcceptInvite handles POST /invite/accept
func (h *InviteHandler) AcceptInvite(w http.ResponseWriter, r *http.Request) {
	// Get current user (must be authenticated)
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

	// Check if trying to add themselves
	if req.InvitedBy == userID {
		middleware.ErrorResponse(w, "You cannot add yourself as a friend", http.StatusBadRequest)
		return
	}

	// Add the friend relationship (bidirectional)
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

// GetInviteLink handles GET /invite/link - generates invite link for current user
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

	// You can get this from environment or config
	baseURL := "https://yourapp.com/invite" // or from env variable
	inviteLink := baseURL + "?invitedBy=" + user.ID
	uName,_ := user.UserName()
	if user.UserName != nil {
		inviteLink += "&userName=" + uName
	}

	response := types.InviteLinkResponse{
		InviteLink: inviteLink,
		Message:    "Invite link generated successfully",
	}

	middleware.JSONResponse(w, response, http.StatusOK)
}