// Package main runs the example-app HTTP server, wiring repositories, services,
// and handlers together to demonstrate skimatik-generated code in a real app.
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/nhalm/pgxkit/v2"
	"github.com/nhalm/skimatik/v2/example-app/internal/api"
	"github.com/nhalm/skimatik/v2/example-app/internal/config"
	"github.com/nhalm/skimatik/v2/example-app/internal/repository"
	"github.com/nhalm/skimatik/v2/example-app/internal/service"
)

func main() {
	if err := run(); err != nil {
		log.Printf("fatal: %v", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	db := pgxkit.NewDB()
	if err := db.Connect(context.Background(), cfg.DatabaseURL); err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := db.Shutdown(shutdownCtx); err != nil {
			log.Printf("warning: database shutdown encountered error: %v", err)
		}
	}()

	if err := db.HealthCheck(context.Background()); err != nil {
		return fmt.Errorf("failed database health check: %w", err)
	}
	log.Println("✅ Connected to database")

	userRepo := repository.NewUserRepository(db)
	postRepo := repository.NewPostRepository(db)

	userService := service.NewUserService(userRepo)
	postService := service.NewPostService(postRepo)

	userHandler := api.NewUserHandler(userService)
	postHandler := api.NewPostHandler(postService)

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Timeout(60 * time.Second))

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

	r.Route("/api", func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`{"status":"healthy","timestamp":"` + time.Now().Format(time.RFC3339) + `"}`)); err != nil {
				slog.Error("write health response", "error", err)
			}
		})

		r.Route("/users", func(r chi.Router) {
			r.Get("/", userHandler.GetActiveUsers)
			r.Post("/", userHandler.CreateUser)
			r.Get("/search", userHandler.SearchUsers)
			r.Get("/{id}", userHandler.GetUser)
			r.Patch("/{id}/name", userHandler.UpdateUserName)
			r.Get("/{id}/audit", userHandler.GetUserAuditHistory)
			r.Get("/{id}/stats", userHandler.GetUserStats)
			r.Get("/{id}/posts", postHandler.GetUserPosts)
			r.Delete("/{id}", userHandler.DeactivateUser)
		})

		r.Route("/posts", func(r chi.Router) {
			r.Get("/", postHandler.GetPublishedPosts)
			r.Get("/with-stats", postHandler.GetPostsWithStats)
			r.Get("/{id}", postHandler.GetPost)
			r.Put("/{id}/publish", postHandler.PublishPost)

			r.Get("/featured", postHandler.GetFeaturedPosts)
			r.Get("/statistics", postHandler.GetPostStatistics)
			r.Get("/tag/{tag}", postHandler.GetPostsByTag)
		})
	})

	addr := ":" + strconv.Itoa(cfg.HTTPPort)
	log.Printf("🚀 Server starting on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		return fmt.Errorf("server failed to start: %w", err)
	}
	return nil
}

func loadConfig() (*config.Config, error) {
	cfg := &config.Config{}
	if err := config.LoadLogging(cfg); err != nil {
		return nil, err
	}
	if err := config.LoadDatabase(cfg); err != nil {
		return nil, err
	}
	if err := config.LoadHTTP(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
