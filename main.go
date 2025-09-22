package main

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"

	appHandlers "citystatAPI/handlers"
	appMiddleware "citystatAPI/middleware"
	"citystatAPI/prisma/db"
	"citystatAPI/services"
	"citystatAPI/telemetry"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-redis/redis/v8"
	"golang.org/x/time/rate"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/hashicorp/go-hclog"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
)

var (
	client *db.PrismaClient
	// redisClient      *redis.Client
	// rateLimiter      *appMiddleware.RateLimiter
	rateLimiter       *appMiddleware.RateLimiter
	redisRateLimiter  *appMiddleware.RedisRateLimiter
	userService       *services.UserService
	settingsService   *services.SettingsService
	friendService     *services.FriendService
	visitorService    *services.VisitorService
	rankService       *services.RankService
	analyticsService  *services.AnalyticsService
	telemetryShutdown telemetry.TelemetryShutdown
)

func init() {
	telemetryConfig := telemetry.GetTelemetryConfigFromEnv()
	var err error
	telemetryShutdown, err = telemetry.InitTelemetry(telemetryConfig)
	if err != nil {
		log.Fatalf("Failed to initialize telemetry: %v", err)
	}

	// Initialize ALL metrics at once
	if err := telemetry.InitAllMetrics(); err != nil {
		log.Fatalf("Failed to initialize metrics: %v", err)
	}
	log.Println("✅ All metrics initialized")

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}
	log.Printf("DATABASE_URL loaded: %s", dbURL[:50]+"...")

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "localhost:6379"
	}

	u, err := url.Parse(redisURL)
	if err != nil {
		log.Fatalf("❌ Invalid REDIS_URL: %v", err)
	}

	addr := u.Host
	password := ""
	if u.User != nil {
		password, _ = u.User.Password()
	}

	useTLS := strings.HasPrefix(redisURL, "rediss://")

	opts := &redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           0,
		DialTimeout:  10 * time.Second,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		PoolSize:     10,
		PoolTimeout:  30 * time.Second,
	}

	if useTLS {
		opts.TLSConfig = &tls.Config{InsecureSkipVerify: false}
	}

	redisClient := redis.NewClient(opts)

	// redisClient := redis.NewClient(&redis.Options{
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("Redis connection failed: %v. Using in-memory rate limiting", err)
		rateLimiterConfig := appMiddleware.RateLimitConfig{
			DefaultRPS:   rate.Limit(100.0 / 60.0),
			DefaultBurst: 20,
			PremiumRPS:   rate.Limit(1000.0 / 60.0),
			PremiumBurst: 50,
		}
		rateLimiter = appMiddleware.NewRateLimiter(rateLimiterConfig)
	} else {
		log.Println("Redis connected successfully")
		// Use Redis rate limiter
		redisRateLimiter = appMiddleware.NewRedisRateLimiter(
			redisClient,
			100,         // default limit per minute
			1000,        // premium limit per minute
			time.Minute, // window duration
		)
	}

	clerk.SetKey(os.Getenv("CLERK_SECRET_KEY"))

	client = db.NewClient()
	if err := client.Prisma.Connect(); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Services
	userService = services.NewUserService(client)
	settingsService = services.NewSettingsService(client)
	friendService = services.NewFriendService(client)
	rankService = services.NewRankService(client)
	visitorService = services.NewVisitorService(client, rankService)
	analyticsService = services.NewAnalyticsService(client)
}

func main() {
	defer func() {
		// Shutdown telemetry
		if telemetryShutdown != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := telemetryShutdown(ctx); err != nil {
				log.Printf("Failed to shutdown telemetry: %v", err)
			}
		}

		if err := client.Prisma.Disconnect(); err != nil {
			log.Printf("Failed to disconnect from database: %v", err)
		}
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

	// Global middleware (applied to all routes)
	r.Use(otelmux.Middleware("citystat-api"))
	r.Use(appMiddleware.TelemetryMiddleware)
	if redisRateLimiter != nil {
		r.Use(appMiddleware.RedisRateLimitMiddleware(redisRateLimiter))
	} else {
		r.Use(appMiddleware.RateLimitMiddleware(rateLimiter))

	}

	// Setup metrics endpoint
	telemetry.SetupMetricsEndpoint(r)

	// Public invite routes (no auth required but still rate limited)
	r.HandleFunc("/invite", inviteHandler.ProcessInvite).Methods("GET")

	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "healthy", "service": "citystat-api"}`))
	}).Methods("GET")

	// API subrouter
	api := r.PathPrefix("/api").Subrouter()

	// Protected routes (auth + rate limiting)
	protected := api.PathPrefix("").Subrouter()
	protected.Use(appMiddleware.ClerkMiddleware)

	// User routes
	protected.HandleFunc("/user", userHandler.GetProfile).Methods("GET")
	protected.HandleFunc("/user/profile", userHandler.UpdateProfile).Methods("PUT")
	protected.HandleFunc("/user/details", userHandler.UpdateUserDetails).Methods("PUT")
	// protected.HandleFunc("/settings", userHandler.UpdateUserProfile).Methods("PUT")
	protected.HandleFunc("/user/note", userHandler.UpdateNote).Methods("PUT")
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
	protected.HandleFunc("/settings/username", settingsHandler.UpdateUsername).Methods("PUT")
	protected.HandleFunc("/settings/phone", settingsHandler.UpdatePhoneNumber).Methods("PUT")

	// Visitor routes
	protected.HandleFunc("/visitor/locationPermission", visitorHandler.GetLocationPermission).Methods("GET")
	protected.HandleFunc("/visitor/locationPermission", visitorHandler.SaveLocationPermission).Methods("POST")
	protected.HandleFunc("/visitor/streets", visitorHandler.GetVisitedStreets).Methods("GET")
	protected.HandleFunc("/visitor/streets", visitorHandler.SaveVisitedStreets).Methods("POST")
	// protected.HandleFunc("/visitor/streets/visitStats", visitorHandler.GetStreetVisitStats).Methods("GET")
	// protected.HandleFunc("/visitor/streets/visitStats", visitorHandler.SaveStreetVisitStats).Methods("POST")

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
		tempLogger.Info("Starting server with telemetry on port", "port", port)
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			tempLogger.Error("Error starting server", "error", err)
			os.Exit(1)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	signal.Notify(sigChan, os.Kill)

	sig := <-sigChan
	log.Println("Got signal:", sig)

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Shutdown HTTP server
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server shutdown complete")
}
