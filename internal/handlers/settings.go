
// internal/handlers/settings.go
package handlers

import (
	"encoding/json"
	"net/http"

	"citystatAPI/internal/middleware"
	"citystatAPI/internal/services"
)

type SettingsHandler struct {
	settingsService *services.SettingsService
}

func NewSettingsHandler(settingsService *services.SettingsService) *SettingsHandler {
	return &SettingsHandler{settingsService: settingsService}
}

func (h *SettingsHandler) GetUserSettings(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		middleware.ErrorResponse(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}

	settings, err := h.settingsService.GetUserSettings(r.Context(), userID)
	if err != nil {
		middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	middleware.JSONResponse(w, settings, http.StatusOK)
}

func (h *SettingsHandler) UpdateUserSettings(w http.ResponseWriter, r *http.Request) {
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

	// Handle nested settings object
	if settingsData, ok := updates["settings"].(map[string]interface{}); ok {
		updates = settingsData
	}

	settings, err := h.settingsService.UpdateUserSettings(r.Context(), userID, updates)
	if err != nil {
		middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	middleware.JSONResponse(w, settings, http.StatusOK)
}

func (h *SettingsHandler) EditUsername(w http.ResponseWriter, r *http.Request) {
	// This should be handled by the user service, not settings service
	middleware.ErrorResponse(w, "Username editing not implemented in this refactor", http.StatusNotImplemented)
}

func (h *SettingsHandler) EditPhoneNumber(w http.ResponseWriter, r *http.Request) {
	// This should be handled by the user service, not settings service
	middleware.ErrorResponse(w, "Phone number editing not implemented in this refactor", http.StatusNotImplemented)
}