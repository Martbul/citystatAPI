package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"citystatAPI/middleware"
	"citystatAPI/services"
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

func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		middleware.ErrorResponse(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}

	// Parse request body
	var updateReq types.UserUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
		middleware.ErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	fmt.Println("req body parsed")

	// Update user with the provided data
	user, err := h.userService.UpdateUser(r.Context(), userID, updateReq)
	if err != nil {
		middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Println("user updated successfuly")

	middleware.JSONResponse(w, user, http.StatusOK)
}


func (h *UserHandler) EditProfile(w http.ResponseWriter, r *http.Request) {
	// userID, ok := middleware.GetUserID(r)
	// if !ok {
	// 	middleware.ErrorResponse(w, "User ID not found in context", http.StatusUnauthorized)
	// 	return
	// }

	// var updateReq types.UserEditProfileRequest
	// if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
	// 	middleware.ErrorResponse(w, "Invalid request body", http.StatusBadRequest)
	// 	return
	// }
	// fmt.Println("req body parsed")

	// user, err := h.userService.UpdateUser(r.Context(), userID, updateReq)
	// if err != nil {
	// 	middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
	// 	return
	// }
	// fmt.Println("user updated successfuly")

	// middleware.JSONResponse(w, user, http.StatusOK)
}

// func (h *UserHandler) EditProfile(w http.ResponseWriter, r *http.Request) {
//     // Parse into generic map
//     var updateReq map[string]interface{}
//     if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
//         middleware.ErrorResponse(w, "Invalid request body", http.StatusBadRequest)
//         return
//     }
    
//     // Pass directly to service layer
//     user, err := h.userService.UpdateUser(r.Context(), userID, updateReq)
//     if err != nil {
//         middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
//         return
//     }
    
//     middleware.JSONResponse(w, user, http.StatusOK)
// }


// SyncProfileFromClerk - separate endpoint for syncing from Clerk
func (h *UserHandler) SyncProfileFromClerk(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		middleware.ErrorResponse(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}

	// Ensure user exists
	_, err := h.userService.GetOrCreateUser(r.Context(), userID)
	if err != nil {
		middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Sync latest data from Clerk
	user, err := h.userService.SyncUserFromClerk(r.Context(), userID)
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

    // Use ONLY your ParseJSON helper - remove the json.NewDecoder line
    updateReq, err := utils.ParseJSON[map[string]interface{}](r)
    if err != nil {
        middleware.ErrorResponse(w, "Invalid request body", http.StatusBadRequest)
        return
    }
    
    fmt.Println("req body parsed")
    
    user, err := h.userService.EditNote(r.Context(), userID, updateReq)
    if err != nil {
        middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    fmt.Println("user updated successfully")
    middleware.JSONResponse(w, user, http.StatusOK)
}

// Add these methods to your handlers/user.go file

// UpdateUserSettings handles PUT /user/settings  
func (h *UserHandler) UpdateUserSettings(w http.ResponseWriter, r *http.Request) {
    userID, ok := middleware.GetUserID(r)
    if !ok {
        middleware.ErrorResponse(w, "User ID not found in context", http.StatusUnauthorized)
        return
    }

    var settingsReq map[string]interface{}
    if err := json.NewDecoder(r.Body).Decode(&settingsReq); err != nil {
        middleware.ErrorResponse(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    fmt.Println("Settings update request:", settingsReq)

    user, err := h.userService.UpdateUserSettings(r.Context(), userID, settingsReq)
    if err != nil {
        middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
        return
    }

    fmt.Println("Settings updated successfully")
    middleware.JSONResponse(w, user, http.StatusOK)
}

func (h *UserHandler) UpdateUserProfile(w http.ResponseWriter, r *http.Request) {
    userID, ok := middleware.GetUserID(r)
    if !ok {
        middleware.ErrorResponse(w, "User ID not found in context", http.StatusUnauthorized)
        return
    }

    // Parse into generic map to handle both user fields and settings
    var updateReq map[string]interface{}
    if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
        middleware.ErrorResponse(w, "Invalid request body", http.StatusBadRequest)
        return
    }
    
    fmt.Println("Profile update request:", updateReq)
    
    user, err := h.userService.UpdateUserProfile(r.Context(), userID, updateReq)
    if err != nil {
        middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    fmt.Println("Profile updated successfully")
    middleware.JSONResponse(w, user, http.StatusOK)
}