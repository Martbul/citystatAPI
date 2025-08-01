package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	appHandlers "citystatAPI/handlers"
	appMiddleware "citystatAPI/middleware"
	"citystatAPI/prisma/db"
	"citystatAPI/services"
	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/go-openapi/runtime/middleware"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/hashicorp/go-hclog"
	"github.com/joho/godotenv"
)

var (
	client          *db.PrismaClient
	userService     *services.UserService
	settingsService *services.SettingsService
	friendService   *services.FriendService
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}
	log.Printf("DATABASE_URL loaded: %s", dbURL[:50]+"...")

	clerk.SetKey(os.Getenv("CLERK_SECRET_KEY"))

	client = db.NewClient()
	if err := client.Prisma.Connect(); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	userService = services.NewUserService(client)
	settingsService = services.NewSettingsService(client)
	friendService = services.NewFriendService(client)
}

func main() {
	defer func() {
		if err := client.Prisma.Disconnect(); err != nil {
			log.Printf("Failed to disconnect: %v", err)
		}
	}()

	tempLogger := hclog.Default()

	userHandler := appHandlers.NewUserHandler(userService)
	settingsHandler := appHandlers.NewSettingsHandler(settingsService)
	friendHandler := appHandlers.NewFriendHandler(friendService)
	inviteHandler := appHandlers.NewInviteHandler(userService, friendService)

	webhookHandler := appHandlers.NewWebhookHandler(client, userService)

	r := mux.NewRouter()

	// Public invite routes (no auth required for initial processing)
	r.HandleFunc("/invite", inviteHandler.ProcessInvite).Methods("GET")

	// API subrouter
	api := r.PathPrefix("/api").Subrouter()

	// Protected routes
	protected := api.PathPrefix("").Subrouter()
	protected.Use(appMiddleware.ClerkMiddleware)

	// User routes
	protected.HandleFunc("/user", userHandler.GetProfile).Methods("GET")
	protected.HandleFunc("/user", userHandler.UpdateProfile).Methods("PUT")
	protected.HandleFunc("/user/profile", userHandler.EditProfile).Methods("PUT")
	protected.HandleFunc("/user/note", userHandler.EditNote).Methods("PUT")
	// Friend routes
	protected.HandleFunc("/users/search", friendHandler.SearchUsers).Methods("GET")
	protected.HandleFunc("/friends/profile", friendHandler.GetFriendProfile).Methods("POST")
	protected.HandleFunc("/friends/add", friendHandler.AddFriend).Methods("POST")
	protected.HandleFunc("/friends/list", friendHandler.GetFriends).Methods("GET")
	protected.HandleFunc("/friends/{friendId}", friendHandler.RemoveFriend).Methods("DELETE")

	// Invite routes
	protected.HandleFunc("/invite/accept", inviteHandler.AcceptInvite).Methods("POST")
	protected.HandleFunc("/invite/link", inviteHandler.GetInviteLink).Methods("GET")

	// Settings routes
	protected.HandleFunc("/settings/account", friendHandler.SearchUsers).Methods("GET")
	protected.HandleFunc("/settings/username", settingsHandler.EditUsername).Methods("PUT")

	//Clerk routes
	protected.HandleFunc("/user/sync", userHandler.SyncProfileFromClerk).Methods("POST")
	r.HandleFunc("/webhooks", webhookHandler.HandleClerkWebhook).Methods("POST")

	opts := middleware.RedocOpts{SpecURL: "/swagger.yaml"}
	swaggerHandler := middleware.Redoc(opts, nil)

	r.Handle("/docs", swaggerHandler)
	//serving a the swagger.yaml file
	r.Handle("/swagger.yaml", http.FileServer(http.Dir("./")))
	corsHandler := handlers.CORS(
		handlers.AllowedOrigins([]string{"*"}), // Configure this for production
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"Content-Type", "Authorization"}),
	)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3333"
	}
	port = ":" + port

	server := http.Server{
		Addr:         port,
		Handler:      corsHandler(r),                                            // set the default handler
		ErrorLog:     tempLogger.StandardLogger(&hclog.StandardLoggerOptions{}), // set the logger for the server
		ReadTimeout:  5 * time.Second,                                           // max time to read request from the client
		WriteTimeout: 10 * time.Second,                                          // max time to write response to the client
		IdleTimeout:  120 * time.Second,                                         // max time for connections using TCP Keep-Alive
	}

	go func() {
		tempLogger.Info("Starting server on port ")
		tempLogger.Info(port)

		err := server.ListenAndServe()
		if err != nil {
			tempLogger.Error("Error starting server", "error", err)
			os.Exit(1)
		}
	}()

	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, os.Interrupt)
	signal.Notify(sigChan, os.Kill)

	sig := <-sigChan
	log.Println("Got signal:", sig)

	timeoutContext, _ := context.WithTimeout(context.Background(), 30*time.Second)

	server.Shutdown(timeoutContext)
}
