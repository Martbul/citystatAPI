package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/martbul/citystatAPI/middleware"
	"github.com/martbul/citystatAPI/prisma/db"
	"github.com/martbul/citystatAPI/services"
)

type WebhookHandler struct {
	client      *db.PrismaClient
	userService *services.UserService
}

func NewWebhookHandler(client *db.PrismaClient, userService *services.UserService) *WebhookHandler {
	return &WebhookHandler{
		client:      client,
		userService: userService,
	}
}

func (h *WebhookHandler) HandleClerkWebhook(w http.ResponseWriter, r *http.Request) {
	// This is a simplified webhook handler
	// In production, you should verify the webhook signature
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		middleware.ErrorResponse(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	eventType, ok := payload["type"].(string)
	if !ok {
		middleware.ErrorResponse(w, "Missing event type", http.StatusBadRequest)
		return
	}

	switch eventType {
	case "user.created", "user.updated":
		// Extract user data and sync to database
		userData, ok := payload["data"].(map[string]interface{})
		if !ok {
			middleware.ErrorResponse(w, "Invalid user data", http.StatusBadRequest)
			return
		}

		userID, ok := userData["id"].(string)
		if !ok {
			middleware.ErrorResponse(w, "Missing user ID", http.StatusBadRequest)
			return
		}

		// Sync user to database
		_, err := h.userService.SyncUserFromClerk(context.Background(), userID)
		if err != nil {
			log.Printf("Failed to sync user %s: %v", userID, err)
			middleware.ErrorResponse(w, "Failed to sync user", http.StatusInternalServerError)
			return
		}

		log.Printf("Successfully synced user %s", userID)

	case "user.deleted":
		// Handle user deletion
		userData, ok := payload["data"].(map[string]interface{})
		if !ok {
			middleware.ErrorResponse(w, "Invalid user data", http.StatusBadRequest)
			return
		}

		userID, ok := userData["id"].(string)
		if !ok {
			middleware.ErrorResponse(w, "Missing user ID", http.StatusBadRequest)
			return
		}

		// Delete user from database (posts will be cascade deleted)
		_, err := h.client.User.FindUnique(
			db.User.ID.Equals(userID),
		).Delete().Exec(context.Background())

		if err != nil && err != db.ErrNotFound {
			log.Printf("Failed to delete user %s: %v", userID, err)
			middleware.ErrorResponse(w, "Failed to delete user", http.StatusInternalServerError)
			return
		}

		log.Printf("Successfully deleted user %s", userID)
	}

	middleware.JSONResponse(w, map[string]string{"message": "Webhook processed"}, http.StatusOK)
}
