package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"citystatAPI/internal/config"
	"citystatAPI/internal/db"
	"citystatAPI/internal/handlers"
	"citystatAPI/internal/middleware"
	"citystatAPI/internal/repository"
	"citystatAPI/internal/services"

	"github.com/clerk/clerk-sdk-go/v2"
	gorillaHandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/hashicorp/go-hclog"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	// Setup logging
	logger := hclog.Default()

	// Initialize Clerk
	clerk.SetKey(cfg.ClerkSecretKey)

	// Connect to database
	ctx := context.Background()
	dbPool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer dbPool.Close()

	// Test database connection
	if err := dbPool.Ping(ctx); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	// Initialize database queries
	queries := db.New(dbPool)

	// Initialize repositories
	repos := &repository.Repositories{
		User:     repository.NewUserRepository(queries),
		Friend:   repository.NewFriendRepository(queries),
		Settings: repository.NewSettingsRepository(queries),
		Visitor:  repository.NewVisitorRepository(queries),
	}

	// Initialize services
	svc := &services.Services{
		User:     services.NewUserService(repos.User, repos.Settings),
		Friend:   services.NewFriendService(repos.Friend, repos.User),
		Settings: services.NewSettingsService(repos.Settings),
		Visitor:  services.NewVisitorService(repos.Visitor, repos.Settings),
	}

	// Initialize handlers
	h := &handlers.Handlers{
		User:     handlers.NewUserHandler(svc.User),
		Friend:   handlers.NewFriendHandler(svc.Friend),
		Settings: handlers.NewSettingsHandler(svc.Settings),
		Visitor:  handlers.NewVisitorHandler(svc.Visitor),
		Upload:   handlers.NewUploadHandler(),
		Webhook:  handlers.NewWebhookHandler(svc.User),
		Invite:   handlers.NewInviteHandler(svc.User, svc.Friend),
	}

	// Setup routes
	router := setupRoutes(h)

	// Setup CORS
	corsHandler := gorillaHandlers.CORS(
		gorillaHandlers.AllowedOrigins([]string{"*"}), // Configure for production
		gorillaHandlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		gorillaHandlers.AllowedHeaders([]string{"Content-Type", "Authorization"}),
	)

	// Create server
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      corsHandler(router),
		ErrorLog:     logger.StandardLogger(&hclog.StandardLoggerOptions{}),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Info("Starting server", "port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	logger.Info("Shutting down server...")

	// Shutdown server gracefully
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
	}

	logger.Info("Server exited")
}

func setupRoutes(h *handlers.Handlers) *mux.Router {
	r := mux.NewRouter()

	// Public routes
	r.HandleFunc("/invite", h.Invite.ProcessInvite).Methods("GET")
	r.HandleFunc("/webhooks", h.Webhook.HandleClerkWebhook).Methods("POST")

	// API routes
	api := r.PathPrefix("/api").Subrouter()

	// Protected routes
	protected := api.PathPrefix("").Subrouter()
	protected.Use(middleware.ClerkMiddleware)

	// User routes
	protected.HandleFunc("/user", h.User.GetProfile).Methods("GET")
	protected.HandleFunc("/user/details", h.User.UpdateUserDetails).Methods("PUT")
	protected.HandleFunc("/user/profile", h.User.UpdateUserProfile).Methods("PUT")
	protected.HandleFunc("/user/note", h.User.EditNote).Methods("PUT")
	protected.HandleFunc("/user/sync", h.User.SyncProfileFromClerk).Methods("POST")
	protected.HandleFunc("/users/search", h.User.SearchUsers).Methods("GET")

	// Friend routes
	protected.HandleFunc("/friends/add", h.Friend.AddFriend).Methods("POST")
	protected.HandleFunc("/friends/list", h.Friend.GetFriends).Methods("GET")
	protected.HandleFunc("/friends/{friendId}", h.Friend.RemoveFriend).Methods("DELETE")

	// Invite routes
	protected.HandleFunc("/invite/accept", h.Invite.AcceptInvite).Methods("POST")
	protected.HandleFunc("/invite/link", h.Invite.GetInviteLink).Methods("GET")

	// Settings routes
	protected.HandleFunc("/settings", h.Settings.GetUserSettings).Methods("GET")
	protected.HandleFunc("/settings", h.Settings.UpdateUserSettings).Methods("PUT")
	protected.HandleFunc("/settings/username", h.Settings.EditUsername).Methods("PUT")
	protected.HandleFunc("/settings/phone", h.Settings.EditPhoneNumber).Methods("PUT")

	// Visitor routes
	protected.HandleFunc("/visitor/locationPermission", h.Visitor.GetLocationPermission).Methods("GET")
	protected.HandleFunc("/visitor/locationPermission", h.Visitor.SaveLocationPermission).Methods("POST")
	protected.HandleFunc("/visitor/streets", h.Visitor.GetVisitedStreets).Methods("GET")
	protected.HandleFunc("/visitor/streets", h.Visitor.SaveVisitedStreets).Methods("POST")

	// Upload routes
	protected.PathPrefix("/uploadthing").HandlerFunc(h.Upload.UploadThingProxy)
	protected.HandleFunc("/upload/complete", h.Upload.HandleImageUpload).Methods("POST")

	return r
}