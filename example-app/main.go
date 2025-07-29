package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	// "github.com/nhalm/skimatik/example-app/api/handlers"
	// "github.com/nhalm/skimatik/example-app/repository/generated"
	// "github.com/nhalm/skimatik/example-app/service"
)

func main() {
	// Get database URL from environment
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:password@localhost:5432/blog?sslmode=disable"
	}

	// Connect to database using standard pgx
	db, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(context.Background()); err != nil {
		log.Fatal("Failed to ping database:", err)
	}
	log.Println("✅ Connected to database")

	// TODO: These will be uncommented after running 'make generate'
	// Initialize generated queries
	// queries := generated.New(db)

	// Initialize services
	// userService := service.NewUserService(queries)
	// postService := service.NewPostService(queries)

	// Initialize handlers
	// userHandler := handlers.NewUserHandler(userService)
	// postHandler := handlers.NewPostHandler(postService)

	// Setup router with chi
	r := chi.NewRouter()

	// Add middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Timeout(60 * time.Second))

	// Add CORS middleware
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token")

			if r.Method == "OPTIONS" {
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	// API routes
	r.Route("/api", func(r chi.Router) {
		// Health check
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"healthy","timestamp":"` + time.Now().Format(time.RFC3339) + `"}`))
		})

		// TODO: These routes will be enabled after running 'make generate'
		// User routes
		// r.Route("/users", func(r chi.Router) {
		// 	r.Get("/", userHandler.GetActiveUsers)
		// 	r.Get("/search", userHandler.SearchUsers)
		// 	r.Get("/{id}", userHandler.GetUser)
		// 	r.Get("/{id}/stats", userHandler.GetUserStats)
		// 	r.Get("/{id}/posts", postHandler.GetUserPosts)
		// 	r.Delete("/{id}", userHandler.DeactivateUser)
		// })

		// Post routes
		// r.Route("/posts", func(r chi.Router) {
		// 	r.Get("/", postHandler.GetPublishedPosts)
		// 	r.Get("/with-stats", postHandler.GetPostsWithStats)
		// 	r.Get("/{id}", postHandler.GetPost)
		// 	r.Put("/{id}/publish", postHandler.PublishPost)
		// })
	})

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 Server starting on port %s", port)
	log.Printf("📋 Basic health check available at:")
	log.Printf("   GET  /api/health")
	log.Printf("")
	log.Printf("💡 To enable full API functionality:")
	log.Printf("   1. Run: make setup    (database + code generation)")
	log.Printf("   2. Run: make run      (start with full API)")

	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
