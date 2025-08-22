// internal/handlers/visitor.go
package handlers

import (
	"encoding/json"
	"net/http"

	"citystatAPI/internal/middleware"
	"citystatAPI/internal/services"
	"citystatAPI/types"
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

	middleware.JSONResponse(w, map[string]bool{"hasLocationPermission": hasLocationPermission}, http.StatusOK)
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

	success, err := h.visitorService.SaveLocationPermission(r.Context(), userID, req.HasLocationPermission)
	if err != nil {
		middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := types.SaveLocationPermitionResponse{
		Success: success,
	}
	middleware.JSONResponse(w, response, http.StatusOK)
}

func (h *VisitorHandler) SaveVisitedStreets(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		middleware.ErrorResponse(w, "User ID not found", http.StatusUnauthorized)
		return
	}

	var req types.SaveVisitedStreetsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.ErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := h.visitorService.SaveVisitedStreets(r.Context(), userID, req)
	if err != nil {
		middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	middleware.JSONResponse(w, map[string]string{"status": "success"}, http.StatusOK)
}

func (h *VisitorHandler) GetVisitedStreets(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		middleware.ErrorResponse(w, "User ID not found", http.StatusUnauthorized)
		return
	}

	visitedStreets, err := h.visitorService.GetVisitedStreets(r.Context(), userID)
	if err != nil {
		middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	middleware.JSONResponse(w, map[string]interface{}{
		"status":          "success",
		"visited_streets": visitedStreets,
	}, http.StatusOK)
}
