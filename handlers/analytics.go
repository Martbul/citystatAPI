package handlers

import (
	"citystatAPI/middleware"
	"citystatAPI/services"
	"net/http"
)


type AnaliticsHandler struct {
	analyticsService *services.AnalyticsService
}


func NewAnaliticsHandler(analyticsService *services.AnalyticsService) *AnaliticsHandler {
	return &AnaliticsHandler{analyticsService: analyticsService}
}



func (h *AnaliticsHandler) GetMain2Stats(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		middleware.ErrorResponse(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}

	mainStats, err := h.analyticsService.GetMain2Stats(r.Context(), userID)
	if err != nil {
		middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	middleware.JSONResponse(w, mainStats, http.StatusOK)
}



func (h *AnaliticsHandler) GetMainRadarChartData(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		middleware.ErrorResponse(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}

	mainStats, err := h.analyticsService.GetMainRadarChartData(r.Context(), userID)
	if err != nil {
		middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	middleware.JSONResponse(w, mainStats, http.StatusOK)
}




func (h *AnaliticsHandler) GetMainRadarChartDataDetailed(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		middleware.ErrorResponse(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}

	mainStats, err := h.analyticsService.GetMainRadarChartDataDetailed(r.Context(), userID)
	if err != nil {
		middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	middleware.JSONResponse(w, mainStats, http.StatusOK)
}