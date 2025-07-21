package handlers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/martbul/citystatAPI/middleware"
	"github.com/martbul/citystatAPI/prisma/db"
	"github.com/martbul/citystatAPI/services"
)

type WebhookHandler struct {
	client      *db.PrismaClient
	userService *services.UserService
}

type ClerkWebhookData struct {
	ID                string                 `json:"id"`
	EmailAddresses    []EmailAddress         `json:"email_addresses"`
	PrimaryEmailAddressID string             `json:"primary_email_address_id"`
	FirstName         *string                `json:"first_name"`
	LastName          *string                `json:"last_name"`
	ImageURL          string                 `json:"image_url"`
	CreatedAt         int64                  `json:"created_at"`
	UpdatedAt         int64                  `json:"updated_at"`
}

type EmailAddress struct {
	ID           string `json:"id"`
	EmailAddress string `json:"email_address"`
}

func NewWebhookHandler(client *db.PrismaClient, userService *services.UserService) *WebhookHandler {
	return &WebhookHandler{
		client:      client,
		userService: userService,
	}
}

// verifyWebhookSignature validates the Clerk webhook signature
func (h *WebhookHandler) verifyWebhookSignature(payload []byte, headers http.Header) bool {
	webhookSecret := os.Getenv("CLERK_WEBHOOK_SECRET")
	if webhookSecret == "" {
		log.Println("CLERK_WEBHOOK_SECRET not set")
		return false
	}

	svixID := headers.Get("svix-id")
	svixTimestamp := headers.Get("svix-timestamp")
	svixSignature := headers.Get("svix-signature")

	if svixID == "" || svixTimestamp == "" || svixSignature == "" {
		return false
	}

	// Create the signed payload
	signedPayload := fmt.Sprintf("%s.%s.%s", svixID, svixTimestamp, string(payload))

	// Verify timestamp (should be within 5 minutes)
	timestamp, err := strconv.ParseInt(svixTimestamp, 10, 64)
	if err != nil {
		return false
	}
	if time.Now().Unix()-timestamp > 300 { // 5 minutes
		return false
	}

	// Compute expected signature
	mac := hmac.New(sha256.New, []byte(webhookSecret))
	mac.Write([]byte(signedPayload))
	expectedSignature := "v1," + hex.EncodeToString(mac.Sum(nil))

	// Compare signatures
	return hmac.Equal([]byte(svixSignature), []byte(expectedSignature))
}

func (h *WebhookHandler) HandleClerkWebhook(w http.ResponseWriter, r *http.Request) {
	// Read the payload
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		middleware.ErrorResponse(w, "Failed to read payload", http.StatusBadRequest)
		return
	}

	// Verify webhook signature (uncomment in production)
	// if !h.verifyWebhookSignature(payload, r.Header) {
	// 	middleware.ErrorResponse(w, "Invalid signature", http.StatusUnauthorized)
	// 	return
	// }

	var webhookPayload map[string]interface{}
	if err := json.Unmarshal(payload, &webhookPayload); err != nil {
		middleware.ErrorResponse(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	eventType, ok := webhookPayload["type"].(string)
	if !ok {
		middleware.ErrorResponse(w, "Missing event type", http.StatusBadRequest)
		return
	}

	data, ok := webhookPayload["data"].(map[string]interface{})
	if !ok {
		middleware.ErrorResponse(w, "Missing data field", http.StatusBadRequest)
		return
	}

	switch eventType {
	case "user.created":
		h.handleUserCreated(w, data)
	case "user.updated":
		h.handleUserUpdated(w, data)
	case "user.deleted":
		h.handleUserDeleted(w, data)
	default:
		log.Printf("Unhandled event type: %s", eventType)
		middleware.JSONResponse(w, map[string]string{"message": "Event type not handled"}, http.StatusOK)
	}
}

func (h *WebhookHandler) handleUserCreated(w http.ResponseWriter, data map[string]interface{}) {
	userData := h.parseUserData(data)
	if userData == nil {
		middleware.ErrorResponse(w, "Invalid user data", http.StatusBadRequest)
		return
	}

	// Create user in database
	_, err := h.client.User.CreateOne(
		db.User.ID.Set(userData.ID),
		db.User.Email.Set(userData.GetPrimaryEmail()),
		db.User.FirstName.SetIfPresent(userData.FirstName),
		db.User.LastName.SetIfPresent(userData.LastName),
		db.User.ImageURL.SetIfPresent(&userData.ImageURL),
		db.User.CreatedAt.Set(time.Unix(userData.CreatedAt/1000, 0)),
		db.User.UpdatedAt.Set(time.Unix(userData.UpdatedAt/1000, 0)),
	).Exec(context.Background())

	if err != nil {
		log.Printf("Failed to create user %s: %v", userData.ID, err)
		middleware.ErrorResponse(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully created user %s", userData.ID)
	middleware.JSONResponse(w, map[string]string{"message": "User created"}, http.StatusOK)
}

func (h *WebhookHandler) handleUserUpdated(w http.ResponseWriter, data map[string]interface{}) {
	userData := h.parseUserData(data)
	if userData == nil {
		middleware.ErrorResponse(w, "Invalid user data", http.StatusBadRequest)
		return
	}

	// Update user in database
	_, err := h.client.User.FindUnique(
		db.User.ID.Equals(userData.ID),
	).Update(
		db.User.Email.Set(userData.GetPrimaryEmail()),
		db.User.FirstName.SetIfPresent(userData.FirstName),
		db.User.LastName.SetIfPresent(userData.LastName),
		db.User.ImageURL.SetIfPresent(&userData.ImageURL),
		db.User.UpdatedAt.Set(time.Unix(userData.UpdatedAt/1000, 0)),
	).Exec(context.Background())

	if err != nil {
		log.Printf("Failed to update user %s: %v", userData.ID, err)
		middleware.ErrorResponse(w, "Failed to update user", http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully updated user %s", userData.ID)
	middleware.JSONResponse(w, map[string]string{"message": "User updated"}, http.StatusOK)
}

func (h *WebhookHandler) handleUserDeleted(w http.ResponseWriter, data map[string]interface{}) {
	userID, ok := data["id"].(string)
	if !ok {
		middleware.ErrorResponse(w, "Missing user ID", http.StatusBadRequest)
		return
	}

	// Delete user from database
	_, err := h.client.User.FindUnique(
		db.User.ID.Equals(userID),
	).Delete().Exec(context.Background())

	if err != nil && err != db.ErrNotFound {
		log.Printf("Failed to delete user %s: %v", userID, err)
		middleware.ErrorResponse(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully deleted user %s", userID)
	middleware.JSONResponse(w, map[string]string{"message": "User deleted"}, http.StatusOK)
}

func (h *WebhookHandler) parseUserData(data map[string]interface{}) *ClerkWebhookData {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil
	}

	var userData ClerkWebhookData
	if err := json.Unmarshal(jsonData, &userData); err != nil {
		return nil
	}

	return &userData
}

func (userData *ClerkWebhookData) GetPrimaryEmail() string {
	for _, email := range userData.EmailAddresses {
		if email.ID == userData.PrimaryEmailAddressID {
			return email.EmailAddress
		}
	}
	if len(userData.EmailAddresses) > 0 {
		return userData.EmailAddresses[0].EmailAddress
	}
	return ""
}