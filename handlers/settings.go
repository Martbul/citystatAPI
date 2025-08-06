package handlers

import (
	"citystatAPI/middleware"
	"citystatAPI/services"
	"citystatAPI/utils"
	"fmt"
	"net/http"
)

type SettingsHandler struct {
	settingsService *services.SettingsService
}

func NewSettingsHandler(settingsService *services.SettingsService) *SettingsHandler {
	return &SettingsHandler{settingsService: settingsService}
}

func (s *SettingsHandler) GetUserSettings(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		middleware.ErrorResponse(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}

    settings, err := s.settingsService.GetUserSettings(r.Context(), userID)
    if err != nil {
		middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Println("user get user settings")
	middleware.JSONResponse(w, settings, http.StatusOK)
}

func (s *SettingsHandler) EditUsername(w http.ResponseWriter, r *http.Request) {
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

	user, err := s.settingsService.EditUsername(r.Context(), userID, updateReq)
	if err != nil {
		middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Println("user updated successfully")
	middleware.JSONResponse(w, user, http.StatusOK)
}

func (s *SettingsHandler) EditPhoneNumber(w http.ResponseWriter, r *http.Request) {
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

	fmt.Println("req body parsed for getting phone num")

	user, err := s.settingsService.EditPhoneNumber(r.Context(), userID, updateReq)
	if err != nil {
		middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Println("user updated successfully")
	middleware.JSONResponse(w, user, http.StatusOK)
}
