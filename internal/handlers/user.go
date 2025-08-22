
// internal/handlers/user.go
package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"citystatAPI/internal/middleware"
	"citystatAPI/internal/services"
	"citystatAPI/types"
	"citystatAPI/utils"
)

type UserHandler struct {
	userService *services.UserService
}

func NewUserHandler(userService *services.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
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

	middleware.JSONResponse(w, user, http.StatusOK)
}

func (h *UserHandler) UpdateUserDetails(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		middleware.ErrorResponse(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}

	var updateReq types.UserUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
		middleware.ErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	user, err := h.userService.UpdateUserDetails(r.Context(), userID, updateReq)
	if err != nil {
		middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	middleware.JSONResponse(w, user, http.StatusOK)
}

func (h *UserHandler) UpdateUserProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		middleware.ErrorResponse(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		middleware.ErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Convert to UserUpdateRequest
	updateReq := types.UserUpdateRequest{}
	if firstName, ok := updates["firstName"].(string); ok {
		updateReq.FirstName = &firstName
	}
	if lastName, ok := updates["lastName"].(string); ok {
		updateReq.LastName = &lastName
	}
	if userName, ok := updates["userName"].(string); ok {
		updateReq.UserName = &userName
	}
	if imageURL, ok := updates["imageURL"].(string); ok {
		updateReq.ImageURL = &imageURL
	}
	if completedTutorial, ok := updates["completedTutorial"].(bool); ok {
		updateReq.CompletedTutorial = &completedTutorial
	}

	user, err := h.userService.UpdateUserDetails(r.Context(), userID, updateReq)
	if err != nil {
		middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	middleware.JSONResponse(w, user, http.StatusOK)
}

func (h *UserHandler) EditNote(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		middleware.ErrorResponse(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}

	updateReq, err := utils.ParseJSON[map[string]interface{}](r)
	if err != nil {
		middleware.ErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	note, ok := updateReq["newNote"].(string)
	if !ok {
		middleware.ErrorResponse(w, "newNote field is required and must be a string", http.StatusBadRequest)
		return
	}

	user, err := h.userService.UpdateNote(r.Context(), userID, note)
	if err != nil {
		middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	middleware.JSONResponse(w, user, http.StatusOK)
}

func (h *UserHandler) SyncProfileFromClerk(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		middleware.ErrorResponse(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}

	user, err := h.userService.SyncUserFromClerk(r.Context(), userID)
	if err != nil {
		middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	middleware.JSONResponse(w, user, http.StatusOK)
}

func (h *UserHandler) SearchUsers(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		middleware.ErrorResponse(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}

	username := r.URL.Query().Get("username")
	if username == "" {
		middleware.ErrorResponse(w, "Username parameter is required", http.StatusBadRequest)
		return
	}

	username = strings.TrimSpace(username)
	if len(username) < 1 {
		middleware.ErrorResponse(w, "Username must be at least 1 character", http.StatusBadRequest)
		return
	}

	users, err := h.userService.SearchUsers(r.Context(), userID, username)
	if err != nil {
		middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := types.SearchUsersResponse{Users: users}
	middleware.JSONResponse(w, response, http.StatusOK)
}