package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/go-openapi/runtime/middleware"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/hashicorp/go-hclog"
	"github.com/joho/godotenv"
	appHandlers "github.com/martbul/citystatAPI/handlers"
	appMiddleware "github.com/martbul/citystatAPI/middleware"
	"github.com/martbul/citystatAPI/prisma/db"
	"github.com/martbul/citystatAPI/services"
)

// var bindAddress = env.String("BIND_ADDRESS", false, ":9090", "Bind address for the server")
var (
	client      *db.PrismaClient
	userService *services.UserService
)

func init() {
    if err := godotenv.Load(); err != nil {
        log.Println("No .env file found")
    }
    
    dbURL := os.Getenv("DATABASE_URL")
    if dbURL == "" {
        log.Fatal("DATABASE_URL environment variable is not set")
    }
    log.Printf("DATABASE_URL loaded: %s", dbURL[:50]+"...") // Print first 50 chars for debugging
    
    clerk.SetKey(os.Getenv("CLERK_SECRET_KEY"))
    
    client = db.NewClient()
    if err := client.Prisma.Connect(); err != nil {
        log.Fatal("Failed to connect to database:", err)
    }
    
    userService = services.NewUserService(client)
}

func main() {
	defer func() {
		if err := client.Prisma.Disconnect(); err != nil {
			log.Printf("Failed to disconnect: %v", err)
		}
	}()

	tempLogger := hclog.Default()

	userHandler := appHandlers.NewUserHandler(userService)
//	postHandler := appHandlers.NewPostHandler(client, userService)
	webhookHandler := appHandlers.NewWebhookHandler(client, userService)

	r := mux.NewRouter()

	// API subrouter
	api := r.PathPrefix("/api").Subrouter()


	////api.HandleFunc("/posts", postHandler.GetPosts).Methods("GET")
	//api.HandleFunc("/posts/{id}", postHandler.GetPostByID).Methods("GET")

	// Protected routes
	protected := api.PathPrefix("").Subrouter()
	protected.Use(appMiddleware.ClerkMiddleware)


	protected.HandleFunc("/user", userHandler.GetProfile).Methods("GET")
	protected.HandleFunc("/user", userHandler.UpdateProfile).Methods("PUT")

	// Post routes
	//protected.HandleFunc("/posts", postHandler.CreatePost).Methods("POST")
	//protected.HandleFunc("/posts/{id}", postHandler.UpdatePost).Methods("PUT")
	//protected.HandleFunc("/posts/{id}", postHandler.DeletePost).Methods("DELETE")
//	protected.HandleFunc("/my-posts", postHandler.GetMyPosts).Methods("GET")

	// Webhook routes (separate from API)
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
	//CORS
	//ch := handlers.CORS(handlers.AllowedOrigins([]string{"http://localhost:3000"}))
	port := os.Getenv("PORT")
	if port == "" {
		port = ":3333"
	}
	server := http.Server{
		Addr:         port,
		Handler:      corsHandler(r),                                            // set the default handler
		ErrorLog:     tempLogger.StandardLogger(&hclog.StandardLoggerOptions{}), // set the logger for the server
		ReadTimeout:  5 * time.Second,                                           // max time to read request from the client
		WriteTimeout: 10 * time.Second,                                          // max time to write response to the client
		IdleTimeout:  120 * time.Second,                                         // max time for connections using TCP Keep-Alive
	}

	//! DON`T UNDERSTAND
	//wrappingt he service in a go func in order to not block
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
	// broadcasting a message on the sigChan whenever an opperating system kill's command or interupt(now when you do ctrl + c and kill the running server it will gracefuly shutdown)
	signal.Notify(sigChan, os.Interrupt)
	signal.Notify(sigChan, os.Kill)

	sig := <-sigChan
	log.Println("Got signal:", sig)

	timeoutContext, _ := context.WithTimeout(context.Background(), 30*time.Second) //allowing 30 sec for gracefuls shutdow, after them the server will forcefully shutdown

	// this is graceful shutdown,the server will no longer accept new requests and will wait until it has completed all the old requests, before shuting down
	server.Shutdown(timeoutContext)
}
