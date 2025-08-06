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

	"citystatAPI/middleware"
	"citystatAPI/prisma/db"
	"citystatAPI/services"
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

// func (h *WebhookHandler) HandleClerkWebhook(w http.ResponseWriter, r *http.Request) {
// 		log.Println("🔥 Webhook received!")
// 	// Read the payload
// 	payload, err := io.ReadAll(r.Body)
// 	if err != nil {
// 				log.Printf("❌ Failed to read payload: %v", err)
// 		middleware.ErrorResponse(w, "Failed to read payload", http.StatusBadRequest)
// 		return
// 	}
// 	log.Printf("📦 Payload received: %s", string(payload))
// 	// Verify webhook signature (uncomment in production)
// 	// if !h.verifyWebhookSignature(payload, r.Header) {
// 	// 	middleware.ErrorResponse(w, "Invalid signature", http.StatusUnauthorized)
// 	// 	return
// 	// }

// 	var webhookPayload map[string]interface{}
// 	if err := json.Unmarshal(payload, &webhookPayload); err != nil {
// 		middleware.ErrorResponse(w, "Invalid JSON payload", http.StatusBadRequest)
// 		return
// 	}

// 	eventType, ok := webhookPayload["type"].(string)
// 	if !ok {
// 		middleware.ErrorResponse(w, "Missing event type", http.StatusBadRequest)
// 		return
// 	}

// 	data, ok := webhookPayload["data"].(map[string]interface{})
// 	if !ok {
// 		middleware.ErrorResponse(w, "Missing data field", http.StatusBadRequest)
// 		return
// 	}

// 	switch eventType {
// 	case "user.created":
// 		h.handleUserCreated(w, data)
// 	case "user.updated":
// 		h.handleUserUpdated(w, data)
// 	case "user.deleted":
// 		h.handleUserDeleted(w, data)
// 	default:
// 		log.Printf("Unhandled event type: %s", eventType)
// 		middleware.JSONResponse(w, map[string]string{"message": "Event type not handled"}, http.StatusOK)
// 	}


// }


// func (h *WebhookHandler) handleUserCreated(w http.ResponseWriter, data map[string]interface{}) {
// 	userData := h.parseUserData(data)
// 	if userData == nil {
// 		middleware.ErrorResponse(w, "Invalid user data", http.StatusBadRequest)
// 		return
// 	}

// 	log.Printf("🆕 Creating user: %+v", userData)

// 	// Create user in database - Fix the field names
// 	user, err := h.client.User.CreateOne(
// 		db.User.ID.Set(userData.ID),
// 		db.User.Email.Set(userData.GetPrimaryEmail()),
// 		db.User.FirstName.SetIfPresent(userData.FirstName),
// 		db.User.LastName.SetIfPresent(userData.LastName),
// 		db.User.ImageURL.SetIfPresent(&userData.ImageURL), // Make sure this matches your schema field name
// 		db.User.CreatedAt.Set(time.Unix(userData.CreatedAt/1000, 0)),
// 		db.User.UpdatedAt.Set(time.Unix(userData.UpdatedAt/1000, 0)),
// 	).Exec(context.Background())

// 	if err != nil {
// 		log.Printf("❌ Failed to create user %s: %v", userData.ID, err)
// 		middleware.ErrorResponse(w, "Failed to create user", http.StatusInternalServerError)
// 		return
// 	}

	
// 	err = h.ensureUserHasSettings(ctx, clerkUserID)
// 	if err != nil {
// 		fmt.Printf("[SyncUserFromClerk] Failed to ensure new user has settings: %v\n", err)
// 	}
// 	fmt.Printf("[SyncUserFromClerk] New user settings ensured\n")



// 	log.Printf("✅ Successfully created user: %+v", user)
// 	middleware.JSONResponse(w, map[string]string{"message": "User created"}, http.StatusOK)
// }


// func (h *WebhookHandler) ensureUserHasSettings(ctx context.Context, userID string) error {
// 	fmt.Printf("[ensureUserHasSettings] Ensuring settings for user ID: %s\n", userID)
// 	settings, err := h.client.Settings.FindUnique(
// 		db.Settings.UserID.Equals(userID),
// 	).Exec(ctx)

// 	if err == db.ErrNotFound {
// 		fmt.Printf("[ensureUserHasSettings] Settings not found for user ID: %s, creating default settings...\n", userID)
// 		settingsCreate, err := h.client.Settings.CreateOne(
// 			db.Settings.User.Link(db.User.ID.Equals(userID)),
// 			db.Settings.Theme.Set(db.ThemeAuto),
// 			db.Settings.Language.Set(db.LanguageEn),
// 			db.Settings.TextSize.Set(db.TextSizeMedium),
// 			db.Settings.FontStyle.Set("default"),
// 			db.Settings.ZoomLevel.Set("100"),
// 			db.Settings.ShowRoleColors.Set(db.RoleColorsNexttoname),
// 			db.Settings.MessagesAllowance.Set(db.MessagesAllowanceAllmsg),
// 			db.Settings.Motion.Set(db.MotionDontplaygifwhenpossibleshow),
// 			db.Settings.StickersAnimation.Set(db.StickersAnimationAlways),
// 			db.Settings.EnabledLocationTracking.Set(false),
// 			db.Settings.AllowCityStatDataUsage.Set(true),
// 			db.Settings.AllowDataPersonalizationUsage.Set(true),
// 			db.Settings.AllowInAppRewards.Set(true),
// 			db.Settings.AllowDataAnaliticsAndPerformance.Set(true),
// 			db.Settings.EnableInAppNotifications.Set(true),
// 			db.Settings.EnableSoundEffects.Set(true),
// 			db.Settings.EnableVibration.Set(true),
// 		).Exec(ctx)
// 		if err != nil {
// 			fmt.Printf("[ensureUserHasSettings] Failed to create settings for user ID: %s, error: %v\n", userID, err)
// 			return fmt.Errorf("failed to create settings: %w", err)
// 		}
// 		fmt.Printf("[ensureUserHasSettings] Default settings created for user ID: %s: %+v\n", userID, settingsCreate)
// 	} else if err != nil {
// 		fmt.Printf("[ensureUserHasSettings] Error checking settings for user ID: %s, error: %v\n", userID, err)
// 		return fmt.Errorf("error checking settings: %w", err)
// 	} else {
// 		fmt.Printf("[ensureUserHasSettings] Settings already exist for user ID: %s: %+v\n", userID, settings)
// 	}

// 	return nil
// }

func (h *WebhookHandler) HandleClerkWebhook(w http.ResponseWriter, r *http.Request) {
	log.Println("🔥 Webhook received!")
	// Read the payload
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("❌ Failed to read payload: %v", err)
		middleware.ErrorResponse(w, "Failed to read payload", http.StatusBadRequest)
		return
	}
	log.Printf("📦 Payload received: %s", string(payload))
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

	log.Printf("🆕 Creating user: %+v", userData)

	ctx := context.Background()

	// Create user in database - Fixed field names to match schema
	user, err := h.client.User.CreateOne(
		db.User.ID.Set(userData.ID),
		db.User.Email.Set(userData.GetPrimaryEmail()),
		db.User.FirstName.SetIfPresent(userData.FirstName),
		db.User.LastName.SetIfPresent(userData.LastName),
		db.User.ImageURL.SetIfPresent(&userData.ImageURL), // Fixed: ImageURL -> ImageUrl to match schema
		db.User.CreatedAt.Set(time.Unix(userData.CreatedAt/1000, 0)),
		db.User.UpdatedAt.Set(time.Unix(userData.UpdatedAt/1000, 0)),
	).Exec(ctx)

	if err != nil {
		log.Printf("❌ Failed to create user %s: %v", userData.ID, err)
		middleware.ErrorResponse(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	// Fixed: use userData.ID instead of undefined clerkUserID
	err = h.ensureUserHasSettings(ctx, userData.ID)
	if err != nil {
		log.Printf("[handleUserCreated] Failed to ensure new user has settings: %v\n", err)
		// Don't return here - user was created successfully, just log the error
	} else {
		log.Printf("[handleUserCreated] New user settings ensured\n")
	}

	log.Printf("✅ Successfully created user: %+v", user)
	middleware.JSONResponse(w, map[string]string{"message": "User created"}, http.StatusOK)
}

func (h *WebhookHandler) ensureUserHasSettings(ctx context.Context, userID string) error {
	log.Printf("[ensureUserHasSettings] Ensuring settings for user ID: %s\n", userID)
	settings, err := h.client.Settings.FindUnique(
		db.Settings.UserID.Equals(userID),
	).Exec(ctx)

	if err == db.ErrNotFound {
		log.Printf("[ensureUserHasSettings] Settings not found for user ID: %s, creating default settings...\n", userID)
		settingsCreate, err := h.client.Settings.CreateOne(
			db.Settings.User.Link(db.User.ID.Equals(userID)),
			// Fixed: Use correct enum values that match your schema
			db.Settings.Theme.Set(db.ThemeLight),                              // Fixed: ThemeAuto -> ThemeLight
			db.Settings.Language.Set(db.LanguageEn),
			db.Settings.TextSize.Set(db.TextSizeMedium),                       // Fixed: TextSizeMedium -> TextSizeMEDIUM
			db.Settings.FontStyle.Set("default"),
			db.Settings.ZoomLevel.Set("100"),
			db.Settings.ShowRoleColors.Set(db.RoleColorsInname),          // Fixed: RoleColorsNexttoname -> RoleColorsNEXTTONAME
			db.Settings.MessagesAllowance.Set(db.MessagesAllowanceAllmsg),     // Fixed: MessagesAllowanceAllmsg -> MessagesAllowanceALLMSG
			db.Settings.Motion.Set(db.MotionDontplaygifwhenpossibleshow),     // Fixed: MotionDontplaygifwhenpossibleshow -> MotionDONTPLAYGIFWHENPOSSIBLESHOW
			db.Settings.StickersAnimation.Set(db.StickersAnimationAlways),      // Fixed: StickersAnimationAlways -> StickersAnimationALWAYS
			db.Settings.EnabledLocationTracking.Set(false),
			db.Settings.AllowCityStatDataUsage.Set(true),
			db.Settings.AllowDataPersonalizationUsage.Set(true),
			db.Settings.AllowInAppRewards.Set(true),
			db.Settings.AllowDataAnaliticsAndPerformance.Set(true),
			db.Settings.EnableInAppNotifications.Set(true),
			db.Settings.EnableSoundEffects.Set(true),
			db.Settings.EnableVibration.Set(true),
		).Exec(ctx)
		if err != nil {
			log.Printf("[ensureUserHasSettings] Failed to create settings for user ID: %s, error: %v\n", userID, err)
			return fmt.Errorf("failed to create settings: %w", err)
		}
		log.Printf("[ensureUserHasSettings] Default settings created for user ID: %s: %+v\n", userID, settingsCreate)
	} else if err != nil {
		log.Printf("[ensureUserHasSettings] Error checking settings for user ID: %s, error: %v\n", userID, err)
		return fmt.Errorf("error checking settings: %w", err)
	} else {
		log.Printf("[ensureUserHasSettings] Settings already exist for user ID: %s: %+v\n", userID, settings)
	}

	return nil
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
	log.Printf("🔍 Parsing user data: %+v", data)
	
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("❌ Failed to marshal data: %v", err)
		return nil
	}

	var userData ClerkWebhookData
	if err := json.Unmarshal(jsonData, &userData); err != nil {
		log.Printf("❌ Failed to unmarshal user data: %v", err)
		return nil
	}

	log.Printf("✅ Parsed user data: %+v", userData)
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