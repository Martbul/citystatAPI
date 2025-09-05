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
	// "github.com/go-redis/redis/v8"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/hashicorp/go-hclog"
	"github.com/joho/godotenv"
)

var (
	client *db.PrismaClient
	// redisClient      *redis.Client
	// rateLimiter      *appMiddleware.RateLimiter
	userService      *services.UserService
	settingsService  *services.SettingsService
	friendService    *services.FriendService
	visitorService   *services.VisitorService
	rankService      *services.RankService
	analyticsService *services.AnalyticsService
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

	// Initialize Redis client
	// redisURL := os.Getenv("REDIS_URL")
	// if redisURL == "" {
	// 	redisURL = "localhost:6379" // Default for development
	// }

	// redisClient = redis.NewClient(&redis.Options{
	// 	Addr:         redisURL,
	// 	Password:     os.Getenv("REDIS_PASSWORD"),
	// 	DB:           0,
	// 	DialTimeout:  10 * time.Second,
	// 	ReadTimeout:  30 * time.Second,
	// 	WriteTimeout: 30 * time.Second,
	// 	PoolSize:     10,
	// 	PoolTimeout:  30 * time.Second,
	// })

	// Test Redis connection
	// ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// defer cancel()

	// if err := redisClient.Ping(ctx).Err(); err != nil {
	// 	log.Printf("Warning: Redis connection failed: %v. Rate limiting will use fallback mode.", err)
	// } else {
	// 	log.Println("Redis connected successfully")
	// }

	clerk.SetKey(os.Getenv("CLERK_SECRET_KEY"))

	client = db.NewClient()
	if err := client.Prisma.Connect(); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Initialize services
	userService = services.NewUserService(client)
	settingsService = services.NewSettingsService(client)
	friendService = services.NewFriendService(client)
	rankService = services.NewRankService(client)
	visitorService = services.NewVisitorService(client, rankService)
	analyticsService = services.NewAnalyticsService(client)

	// Initialize rate limiter
	// rateLimiter = appMiddleware.NewRateLimiter(redisClient)

	// Set up user tiers (you'd typically load this from your database)
	// setupUserTiers()

}

// func setupUserTiers() {
// 	// Example: Set user tiers based on your business logic
// 	// In production, you'd fetch this from your database

// 	// You could have a service method to check user subscription status
// 	// rateLimiter.SetUserTier("premium_user_id", appMiddleware.TierPremium)
// 	// rateLimiter.SetUserTier("enterprise_user_id", appMiddleware.TierEnterprise)

// 	log.Println("User tiers initialized")
// }

func main() {
	defer func() {
		if err := client.Prisma.Disconnect(); err != nil {
			log.Printf("Failed to disconnect from database: %v", err)
		}
		// if err := redisClient.Close(); err != nil {
		// 	log.Printf("Failed to disconnect from Redis: %v", err)
		// }
	}()

	tempLogger := hclog.Default()

	userHandler := appHandlers.NewUserHandler(userService)
	settingsHandler := appHandlers.NewSettingsHandler(settingsService)
	visitorHandler := appHandlers.NewVisitorHandler(visitorService)
	friendHandler := appHandlers.NewFriendHandler(friendService)
	inviteHandler := appHandlers.NewInviteHandler(userService, friendService)
	uploadHandler := appHandlers.NewUploadHandler()
	webhookHandler := appHandlers.NewWebhookHandler(client, userService)
	rankHandler := appHandlers.NewRankHandler(rankService)
	analiticsHandler := appHandlers.NewAnaliticsHandler(analyticsService)

	r := mux.NewRouter()

	// Add rate limiting middleware BEFORE other middlewares
	// r.Use(rateLimiter.SmartRateLimiter())

	// Public invite routes (no auth required but still rate limited)
	r.HandleFunc("/invite", inviteHandler.ProcessInvite).Methods("GET")

	//! Add rate limit monitoring endpoint for admins
	// r.HandleFunc("/admin/rate-limit/stats", getRateLimitStats).Methods("GET")

	// API subrouter
	api := r.PathPrefix("/api").Subrouter()

	// Protected routes
	protected := api.PathPrefix("").Subrouter()
	protected.Use(appMiddleware.ClerkMiddleware)

	// User routes
	protected.HandleFunc("/user", userHandler.GetProfile).Methods("GET")
	protected.HandleFunc("/user/details", userHandler.UpdateUserDetails).Methods("PUT")
	protected.HandleFunc("/settings", userHandler.UpdateUserProfile).Methods("PUT")
	protected.HandleFunc("/user/profile", userHandler.EditProfile).Methods("PUT")
	protected.HandleFunc("/user/note", userHandler.EditNote).Methods("PUT")
	protected.HandleFunc("/users/search", userHandler.SearchUsers).Methods("GET")
	protected.HandleFunc("/users/sameCity", userHandler.GetUsersSameCity).Methods("GET")
	protected.HandleFunc("/users/activeHours", userHandler.UpdateActiveHours).Methods("PUT")

	// Friend routes
	protected.HandleFunc("/friends/profile", friendHandler.GetFriendProfile).Methods("POST")
	protected.HandleFunc("/friends/add", friendHandler.AddFriend).Methods("POST")
	protected.HandleFunc("/friends/list", friendHandler.GetFriends).Methods("GET")
	protected.HandleFunc("/friends/{friendId}", friendHandler.RemoveFriend).Methods("DELETE")

	// Invite routes
	protected.HandleFunc("/invite/accept", inviteHandler.AcceptInvite).Methods("POST")
	protected.HandleFunc("/invite/link", inviteHandler.GetInviteLink).Methods("GET")

	// Settings routes
	protected.HandleFunc("/settings", settingsHandler.GetUserSettings).Methods("GET")
	protected.HandleFunc("/user/settings", userHandler.UpdateUserSettings).Methods("PUT")
	protected.HandleFunc("/settings/account", userHandler.SearchUsers).Methods("GET")
	protected.HandleFunc("/settings/username", settingsHandler.EditUsername).Methods("PUT")
	protected.HandleFunc("/settings/phone", settingsHandler.EditPhoneNumber).Methods("PUT")

	// Visitor routes
	protected.HandleFunc("/visitor/locationPermission", visitorHandler.GetLocationPermission).Methods("GET")
	protected.HandleFunc("/visitor/locationPermission", visitorHandler.SaveLocationPermission).Methods("POST")
	protected.HandleFunc("/visitor/streets", visitorHandler.GetVisitedStreets).Methods("GET")
	protected.HandleFunc("/visitor/streets", visitorHandler.SaveVisitedStreets).Methods("POST")
	protected.HandleFunc("/visitor/streets/visitStats", visitorHandler.GetStreetVisitStats).Methods("GET")
	protected.HandleFunc("/visitor/streets/visitStats", visitorHandler.SaveStreetVisitStats).Methods("POST")

	// Rank routes
	protected.HandleFunc("/rank", rankHandler.GetUserRank).Methods("GET")
	protected.HandleFunc("/rank/progress", rankHandler.GetLevelProgress).Methods("GET")
	protected.HandleFunc("/rank/leaderboard", rankHandler.GetLeaderboard).Methods("GET")
	protected.HandleFunc("/rank/leaderboard/local", rankHandler.GetLocalLeaderboard).Methods("GET")

	// Analitics routes
	protected.HandleFunc("/analytics/main2stats", analiticsHandler.GetMain2Stats).Methods("GET")
	protected.HandleFunc("/analytics/mainRadarChartData", analiticsHandler.GetMainRadarChartData).Methods("GET")
	protected.HandleFunc("/analytics/mainRadarChartData/detailed", analiticsHandler.GetMainRadarChartDataDetailed).Methods("GET")

	
	// Documents
	//! privacy policy
	//! terms of service
	//! open source licens

	// Add UploadThing routes
	protected.PathPrefix("/uploadthing").HandlerFunc(uploadHandler.UploadThingProxy)
	protected.HandleFunc("/upload/complete", uploadHandler.HandleImageUpload).Methods("POST")

	//Clerk routes
	protected.HandleFunc("/user/sync", userHandler.SyncProfileFromClerk).Methods("POST")
	r.HandleFunc("/webhooks", webhookHandler.HandleClerkWebhook).Methods("POST")

	// Documentation routes
	opts := middleware.RedocOpts{SpecURL: "/swagger.yaml"}
	swaggerHandler := middleware.Redoc(opts, nil)
	r.Handle("/docs", swaggerHandler)
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
