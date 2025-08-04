// handlers/upload.go
package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"

	"citystatAPI/middleware"
)

type UploadHandler struct{}

func NewUploadHandler() *UploadHandler {
	return &UploadHandler{}
}

// UploadThingProxy handles all UploadThing API requests
func (h *UploadHandler) UploadThingProxy(w http.ResponseWriter, r *http.Request) {
	// Enable CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, x-uploadthing-version, x-uploadthing-api-key")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Get UploadThing API key from environment
	apiKey := os.Getenv("UPLOADTHING_SECRET")
	if apiKey == "" {
		middleware.ErrorResponse(w, "UploadThing API key not configured", http.StatusInternalServerError)
		return
	}

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		middleware.ErrorResponse(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Build UploadThing API URL
	uploadThingURL := "https://api.uploadthing.com" + strings.TrimPrefix(r.URL.Path, "/api/uploadthing")
	if r.URL.RawQuery != "" {
		uploadThingURL += "?" + r.URL.RawQuery
	}

	// Create request to UploadThing
	req, err := http.NewRequest(r.Method, uploadThingURL, bytes.NewReader(body))
	if err != nil {
		middleware.ErrorResponse(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	// Copy headers from original request
	for key, values := range r.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// Add/Override UploadThing authentication
	req.Header.Set("x-uploadthing-api-key", apiKey)

	// Make request to UploadThing
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		middleware.ErrorResponse(w, "Failed to proxy request to UploadThing", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Copy status code and body
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// HandleImageUpload processes the upload and updates user profile
func (h *UploadHandler) HandleImageUpload(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		middleware.ErrorResponse(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}

	var uploadData struct {
		URL      string `json:"url"`
		Key      string `json:"key"`
		Name     string `json:"name"`
		Size     int64  `json:"size"`
	}

	if err := json.NewDecoder(r.Body).Decode(&uploadData); err != nil {
		middleware.ErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Here you could update the user's profile with the new image URL
	// For example, if you add this to your UserService:
	// user, err := h.userService.UpdateUserImage(r.Context(), userID, uploadData.URL)
	// if err != nil {
	//     middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
	//     return
	// }

	response := map[string]interface{}{
		"message":  "Image uploaded successfully",
		"imageUrl": uploadData.URL,
		"userId":   userID,
	}

	middleware.JSONResponse(w, response, http.StatusOK)
}