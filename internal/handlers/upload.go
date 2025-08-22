
// internal/handlers/upload.go
package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"

	"citystatAPI/internal/middleware"
)

type UploadHandler struct{}

func NewUploadHandler() *UploadHandler {
	return &UploadHandler{}
}

func (h *UploadHandler) UploadThingProxy(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, x-uploadthing-version, x-uploadthing-api-key")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	apiKey := os.Getenv("UPLOADTHING_SECRET")
	if apiKey == "" {
		middleware.ErrorResponse(w, "UploadThing API key not configured", http.StatusInternalServerError)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		middleware.ErrorResponse(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	uploadThingURL := "https://api.uploadthing.com" + strings.TrimPrefix(r.URL.Path, "/api/uploadthing")
	if r.URL.RawQuery != "" {
		uploadThingURL += "?" + r.URL.RawQuery
	}

	req, err := http.NewRequest(r.Method, uploadThingURL, bytes.NewReader(body))
	if err != nil {
		middleware.ErrorResponse(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	for key, values := range r.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	req.Header.Set("x-uploadthing-api-key", apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		middleware.ErrorResponse(w, "Failed to proxy request to UploadThing", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (h *UploadHandler) HandleImageUpload(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		middleware.ErrorResponse(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}

	var uploadData struct {
		URL  string `json:"url"`
		Key  string `json:"key"`
		Name string `json:"name"`
		Size int64  `json:"size"`
	}

	if err := json.NewDecoder(r.Body).Decode(&uploadData); err != nil {
		middleware.ErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Update user image URL using user service
	response := map[string]interface{}{
		"message":  "Image uploaded successfully",
		"imageUrl": uploadData.URL,
		"userId":   userID,
	}

	middleware.JSONResponse(w, response, http.StatusOK)
}