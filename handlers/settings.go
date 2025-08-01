package handlers

import (
	"citystatAPI/middleware"
	"citystatAPI/services"
	"citystatAPI/utils"
	"encoding/json"
	"fmt"
	"net/http"
)



type SettingsHandler struct {
	settingsService *services.SettingsService
}


func NewSettingsHandler(settingsService *services.SettingsService) *SettingsHandler {
	return &SettingsHandler{settingsService: settingsService}
}



func (s *SettingsHandler) EditUsername(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		middleware.ErrorResponse(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}

	// Parse request body

	  updateReq, err := utils.ParseJSON[map[string]interface{}](r)
	if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
		middleware.ErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	fmt.Println("req body parsed")

	// Update user with the provided data
	user, err := s.settingsService.EditUsername(r.Context(), userID, updateReq)
	if err != nil {
		middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Println("user updated successfuly")

	middleware.JSONResponse(w, user, http.StatusOK)
}
