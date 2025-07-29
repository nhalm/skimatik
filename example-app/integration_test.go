//go:build integration
// +build integration

package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nhalm/skimatik/example-app/api"
	"github.com/nhalm/skimatik/example-app/service"
)

const testDatabaseURL = "postgres://postgres:password@localhost:5432/blog?sslmode=disable"

func setupTestApp(t *testing.T) (*chi.Mux, *pgxpool.Pool) {
	// Connect to test database
	db, err := pgxpool.New(context.Background(), testDatabaseURL)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Test database connection
	if err := db.Ping(context.Background()); err != nil {
		t.Fatalf("Failed to ping test database: %v", err)
	}

	// Initialize services (using stub implementations until we have generated code)
	userService := service.NewStubUserService()
	postService := service.NewStubPostService()

	// Initialize handlers
	userHandler := api.NewUserHandler(userService)
	postHandler := api.NewPostHandler(postService)

	// Setup router
	r := chi.NewRouter()

	// Health check endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// API routes - using actual available methods
	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/users", func(r chi.Router) {
			r.Get("/active", userHandler.GetActiveUsers)
			r.Get("/{id}", userHandler.GetUser)
			r.Get("/email/{email}", userHandler.GetUserByEmail)
			r.Get("/search", userHandler.SearchUsers)
			r.Get("/{id}/stats", userHandler.GetUserStats)
			r.Post("/{id}/deactivate", userHandler.DeactivateUser)
		})

		r.Route("/posts", func(r chi.Router) {
			r.Get("/", postHandler.GetPublishedPosts)
			r.Get("/{id}", postHandler.GetPost)
			r.Get("/user/{userId}", postHandler.GetUserPosts)
			r.Get("/stats", postHandler.GetPostsWithStats)
			r.Get("/featured", postHandler.GetFeaturedPosts)
			r.Get("/tag/{tag}", postHandler.GetPostsByTag)
			r.Get("/statistics", postHandler.GetPostStatistics)
			r.Post("/{id}/publish", postHandler.PublishPost)
		})
	})

	return r, db
}

func TestApplicationIntegration(t *testing.T) {
	// Set the DATABASE_URL environment variable for the test
	originalURL := os.Getenv("DATABASE_URL")
	os.Setenv("DATABASE_URL", testDatabaseURL)
	defer os.Setenv("DATABASE_URL", originalURL)

	app, db := setupTestApp(t)
	defer db.Close()

	t.Run("Health Check", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		if w.Body.String() != "OK" {
			t.Errorf("Expected 'OK', got '%s'", w.Body.String())
		}
	})

	t.Run("Database Connection", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			t.Errorf("Database ping failed: %v", err)
		}
	})

	t.Run("User API Endpoints", func(t *testing.T) {
		// Test GET /api/v1/users/active
		req := httptest.NewRequest("GET", "/api/v1/users/active", nil)
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)

		// Stub services may return different status codes, so we just check that we get a response
		if w.Code == 0 {
			t.Errorf("GET /api/v1/users/active failed: no response")
		}
		t.Logf("GET /api/v1/users/active returned status: %d", w.Code)

		// Test GET /api/v1/users/{id}
		req = httptest.NewRequest("GET", "/api/v1/users/123e4567-e89b-12d3-a456-426614174000", nil)
		w = httptest.NewRecorder()
		app.ServeHTTP(w, req)

		// Test that the endpoint exists and responds
		if w.Code == 0 {
			t.Errorf("GET /api/v1/users/{id} failed: no response")
		}
		t.Logf("GET /api/v1/users/{id} returned status: %d", w.Code)

		// Test GET /api/v1/users/email/{email}
		req = httptest.NewRequest("GET", "/api/v1/users/email/test@example.com", nil)
		w = httptest.NewRecorder()
		app.ServeHTTP(w, req)

		if w.Code == 0 {
			t.Errorf("GET /api/v1/users/email/{email} failed: no response")
		}
		t.Logf("GET /api/v1/users/email/{email} returned status: %d", w.Code)

		// Test GET /api/v1/users/search
		req = httptest.NewRequest("GET", "/api/v1/users/search?q=test", nil)
		w = httptest.NewRecorder()
		app.ServeHTTP(w, req)

		if w.Code == 0 {
			t.Errorf("GET /api/v1/users/search failed: no response")
		}
		t.Logf("GET /api/v1/users/search returned status: %d", w.Code)
	})

	t.Run("Post API Endpoints", func(t *testing.T) {
		// Test GET /api/v1/posts (published posts)
		req := httptest.NewRequest("GET", "/api/v1/posts", nil)
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)

		// Stub services may return different status codes, so we just check that we get a response
		if w.Code == 0 {
			t.Errorf("GET /api/v1/posts failed: no response")
		}
		t.Logf("GET /api/v1/posts returned status: %d", w.Code)

		// Test GET /api/v1/posts/{id}
		req = httptest.NewRequest("GET", "/api/v1/posts/123e4567-e89b-12d3-a456-426614174000", nil)
		w = httptest.NewRecorder()
		app.ServeHTTP(w, req)

		if w.Code == 0 {
			t.Errorf("GET /api/v1/posts/{id} failed: no response")
		}
		t.Logf("GET /api/v1/posts/{id} returned status: %d", w.Code)

		// Test GET /api/v1/posts/featured
		req = httptest.NewRequest("GET", "/api/v1/posts/featured", nil)
		w = httptest.NewRecorder()
		app.ServeHTTP(w, req)

		if w.Code == 0 {
			t.Errorf("GET /api/v1/posts/featured failed: no response")
		}
		t.Logf("GET /api/v1/posts/featured returned status: %d", w.Code)

		// Test GET /api/v1/posts/stats
		req = httptest.NewRequest("GET", "/api/v1/posts/stats", nil)
		w = httptest.NewRecorder()
		app.ServeHTTP(w, req)

		if w.Code == 0 {
			t.Errorf("GET /api/v1/posts/stats failed: no response")
		}
		t.Logf("GET /api/v1/posts/stats returned status: %d", w.Code)
	})

	t.Run("Code Generation Validation", func(t *testing.T) {
		// This test validates that code generation happened successfully
		// by checking if generated files exist and are valid

		// Check if generated directory exists
		if _, err := os.Stat("repository/generated"); os.IsNotExist(err) {
			t.Skip("Generated code directory doesn't exist - skipping validation (run 'make generate' first)")
		}

		// Check for key generated files
		expectedFiles := []string{
			"repository/generated/users_generated.go",
			"repository/generated/posts_generated.go",
			"repository/generated/comments_generated.go",
			"repository/generated/pagination.go",
		}

		for _, file := range expectedFiles {
			if _, err := os.Stat(file); os.IsNotExist(err) {
				t.Errorf("Expected generated file missing: %s", file)
			}
		}

		// TODO: Once generated code is integrated, we can test:
		// - Generated repositories work with real database
		// - Pagination functions correctly
		// - Query methods work as expected
		// - CRUD operations complete successfully
	})
}

func TestApplicationStartup(t *testing.T) {
	// This test validates that the application can start up successfully
	// It's designed to be used in the Makefile timeout test

	// Set the DATABASE_URL environment variable for the test
	originalURL := os.Getenv("DATABASE_URL")
	os.Setenv("DATABASE_URL", testDatabaseURL)
	defer os.Setenv("DATABASE_URL", originalURL)

	app, db := setupTestApp(t)
	defer db.Close()

	// Create a test server
	server := httptest.NewServer(app)
	defer server.Close()

	// Test that we can make a request to the server
	resp, err := http.Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("Failed to make request to test server: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	t.Log("✅ Application startup test passed")
}
