package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/nhalm/pgxkit"
	"github.com/nhalm/skimatik/example-app/api/handlers"
	"github.com/nhalm/skimatik/example-app/repository/generated"
	"github.com/nhalm/skimatik/example-app/service"
)

func main() {
	// Get database URL from environment or use default
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:password@localhost:5432/blog?sslmode=disable"
	}

	// Initialize database connection
	db, err := pgxkit.NewDB(databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}()

	// Test database connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.Ping(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("Successfully connected to database")

	// Initialize generated query repositories
	userQueries := generated.NewUsersQueries(db)
	postQueries := generated.NewPostsQueries(db)

	// Initialize services
	userService := service.NewUserService(userQueries)
	postService := service.NewPostService(postQueries)

	// Initialize handlers
	userHandler := handlers.NewUserHandler(userService)
	postHandler := handlers.NewPostHandler(postService)

	// Setup router and routes
	router := setupRoutes(userHandler, postHandler)

	// Setup HTTP server
	server := &http.Server{
		Addr:         ":8080",
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Println("Starting server on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Server shutting down...")

	// Graceful shutdown with timeout
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

// setupRoutes configures all HTTP routes
func setupRoutes(userHandler *handlers.UserHandler, postHandler *handlers.PostHandler) *mux.Router {
	router := mux.NewRouter()

	// API prefix
	api := router.PathPrefix("/api").Subrouter()

	// Add CORS middleware
	api.Use(corsMiddleware)

	// Add logging middleware
	api.Use(loggingMiddleware)

	// User routes
	api.HandleFunc("/users", userHandler.GetActiveUsers).Methods("GET")
	api.HandleFunc("/users/search", userHandler.SearchUsers).Methods("GET")
	api.HandleFunc("/users/email/{email}", userHandler.GetUserByEmail).Methods("GET")
	api.HandleFunc("/users/{id}", userHandler.DeactivateUser).Methods("DELETE")
	api.HandleFunc("/users/{id}/stats", userHandler.GetUserStats).Methods("GET")
	api.HandleFunc("/users/{id}/posts", postHandler.GetUserPosts).Methods("GET")

	// Post routes
	api.HandleFunc("/posts", postHandler.GetPublishedPosts).Methods("GET")
	api.HandleFunc("/posts/stats", postHandler.GetPostsWithStats).Methods("GET")
	api.HandleFunc("/posts/{id}", postHandler.GetPost).Methods("GET")
	api.HandleFunc("/posts/{id}/publish", postHandler.PublishPost).Methods("PUT")

	// Health check endpoint
	api.HandleFunc("/health", healthCheckHandler).Methods("GET")

	return router
}

// corsMiddleware adds CORS headers
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware logs HTTP requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		next.ServeHTTP(w, r)

		log.Printf(
			"%s %s %s %v",
			r.Method,
			r.RequestURI,
			r.RemoteAddr,
			time.Since(start),
		)
	})
}

// healthCheckHandler provides a simple health check endpoint
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy","service":"blog-api","timestamp":"` + time.Now().Format(time.RFC3339) + `"}`))
}
