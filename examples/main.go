package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Example application demonstrating dbutil-gen generated repositories
// This shows how to use table-based repositories with pagination in a web API

func main() {
	// Database connection
	ctx := context.Background()
	dsn := "postgres://dbutil:dbutil_test_password@localhost:5432/dbutil_test?sslmode=disable"

	conn, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer conn.Close()

	// Test connection
	if err := conn.Ping(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Connected to database successfully")

	// Create API server with generated repositories
	server := NewAPIServer(conn)

	// Start server
	log.Println("Starting server on :8080")
	log.Println("Example endpoints:")
	log.Println("  GET  /users              - List users with pagination")
	log.Println("  GET  /users/{id}         - Get user by ID")
	log.Println("  POST /users              - Create new user")
	log.Println("  PUT  /users/{id}         - Update user")
	log.Println("  DELETE /users/{id}       - Delete user")
	log.Println("  GET  /posts              - List posts with pagination")
	log.Println("  GET  /posts/{id}         - Get post by ID")
	log.Println("  POST /posts              - Create new post")

	if err := http.ListenAndServe(":8080", server.Router()); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// APIServer demonstrates how to structure an application using generated repositories
type APIServer struct {
	conn *pgxpool.Pool
	// In a real application, you would inject the generated repositories here
	// userRepo *repositories.UsersRepository
	// postRepo *repositories.PostsRepository
	// etc.
}

func NewAPIServer(conn *pgxpool.Pool) *APIServer {
	return &APIServer{
		conn: conn,
	}
}

func (s *APIServer) Router() http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	// Routes
	r.Get("/health", s.handleHealth)

	// User routes
	r.Route("/users", func(r chi.Router) {
		r.Get("/", s.handleListUsers)
		r.Post("/", s.handleCreateUser)
		r.Get("/{id}", s.handleGetUser)
		r.Put("/{id}", s.handleUpdateUser)
		r.Delete("/{id}", s.handleDeleteUser)
		r.Get("/{id}/posts", s.handleListUserPosts)
	})

	// Post routes
	r.Route("/posts", func(r chi.Router) {
		r.Get("/", s.handleListPosts)
		r.Post("/", s.handleCreatePost)
		r.Get("/{id}", s.handleGetPost)
	})

	return r
}

func (s *APIServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Test database connection
	if err := s.conn.Ping(ctx); err != nil {
		http.Error(w, "Database unhealthy", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "healthy",
		"database": "connected",
	})
}

// Example: List users with pagination
func (s *APIServer) handleListUsers(w http.ResponseWriter, r *http.Request) {
	// Parse pagination parameters
	cursor := r.URL.Query().Get("cursor")
	limit := 20 // default
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	// In a real application with generated repositories, this would be:
	// userRepo := repositories.NewUsersRepository(s.conn)
	// result, err := userRepo.ListPaginated(ctx, repositories.PaginationParams{
	//     Cursor: cursor,
	//     Limit:  limit,
	// })

	// For this example, we'll simulate the response structure
	mockResponse := map[string]interface{}{
		"items": []map[string]interface{}{
			{
				"id":         "01234567-89ab-cdef-0123-456789abcdef",
				"name":       "John Doe",
				"email":      "john@example.com",
				"created_at": "2025-01-15T10:30:00Z",
			},
			{
				"id":         "01234567-89ab-cdef-0123-456789abcde0",
				"name":       "Jane Smith",
				"email":      "jane@example.com",
				"created_at": "2025-01-15T10:31:00Z",
			},
		},
		"has_more":    true,
		"next_cursor": "eyJpZCI6IjAxMjM0NTY3LTg5YWItY2RlZi0wMTIzLTQ1Njc4OWFiY2RlMCJ9",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(mockResponse)

	log.Printf("Listed users: cursor=%s, limit=%d", cursor, limit)
}

// Example: Get user by ID
func (s *APIServer) handleGetUser(w http.ResponseWriter, r *http.Request) {
	vars := chi.URLParam(r, "id")

	userID, err := uuid.Parse(vars)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// In a real application:
	// userRepo := repositories.NewUsersRepository(s.conn)
	// user, err := userRepo.GetByID(ctx, userID)
	// if err != nil {
	//     if err == pgx.ErrNoRows {
	//         http.Error(w, "User not found", http.StatusNotFound)
	//         return
	//     }
	//     http.Error(w, "Internal server error", http.StatusInternalServerError)
	//     return
	// }

	// Mock response
	mockUser := map[string]interface{}{
		"id":         userID.String(),
		"name":       "John Doe",
		"email":      "john@example.com",
		"created_at": "2025-01-15T10:30:00Z",
		"is_active":  true,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(mockUser)

	log.Printf("Retrieved user: %s", userID)
}

// Example: Create new user
func (s *APIServer) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	// In a real application, you would:
	// userRepo := repositories.NewUsersRepository(s.conn)
	// user, err := userRepo.Create(ctx, repositories.CreateUsersParams{...})

	var req struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate input
	if req.Name == "" || req.Email == "" {
		http.Error(w, "Name and email are required", http.StatusBadRequest)
		return
	}

	// Example response (in real app, this would come from the repository)
	user := map[string]interface{}{
		"id":         uuid.New(),
		"name":       req.Name,
		"email":      req.Email,
		"created_at": "2024-01-01T12:00:00Z",
		"updated_at": "2024-01-01T12:00:00Z",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

// Example: Update user
func (s *APIServer) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	vars := chi.URLParam(r, "id")

	userID, err := uuid.Parse(vars)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Name  *string `json:"name,omitempty"`
		Email *string `json:"email,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// In a real application, you would:
	// userRepo := repositories.NewUsersRepository(s.conn)
	//
	// // Build update params based on provided fields
	// updateParams := repositories.UpdateUsersParams{}
	// if req.Name != nil {
	//     updateParams.Name = *req.Name
	// }
	// if req.Email != nil {
	//     updateParams.Email = *req.Email
	// }
	//
	// user, err := userRepo.Update(ctx, userID, updateParams)
	// if err != nil {
	//     if err == pgx.ErrNoRows {
	//         http.Error(w, "User not found", http.StatusNotFound)
	//         return
	//     }
	//     http.Error(w, "Failed to update user", http.StatusInternalServerError)
	//     return
	// }

	// Example response
	user := map[string]interface{}{
		"id":         userID,
		"name":       "Updated Name",
		"email":      "updated@example.com",
		"created_at": "2024-01-01T12:00:00Z",
		"updated_at": "2024-01-01T12:30:00Z",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// Example: Delete user
func (s *APIServer) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	vars := chi.URLParam(r, "id")

	_, err := uuid.Parse(vars) // Validate ID format
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// In a real application, you would:
	// userRepo := repositories.NewUsersRepository(s.conn)
	// err := userRepo.Delete(ctx, userID)
	// if err != nil {
	//     if err == pgx.ErrNoRows {
	//         http.Error(w, "User not found", http.StatusNotFound)
	//         return
	//     }
	//     http.Error(w, "Failed to delete user", http.StatusInternalServerError)
	//     return
	// }

	w.WriteHeader(http.StatusNoContent)
}

// Example: List posts with pagination
func (s *APIServer) handleListPosts(w http.ResponseWriter, r *http.Request) {
	// Parse pagination parameters (in real app, these would be used)
	// limit := 20
	// if l := r.URL.Query().Get("limit"); l != "" {
	// 	if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
	// 		limit = parsed
	// 	}
	// }
	// cursor := r.URL.Query().Get("cursor")

	// In a real application, you would:
	// postRepo := repositories.NewPostsRepository(s.conn)
	// result, err := postRepo.ListPaginated(ctx, repositories.PaginationParams{
	//     Cursor: cursor,
	//     Limit:  limit,
	// })

	// Example response
	posts := []map[string]interface{}{
		{
			"id":         uuid.New(),
			"title":      "Example Post 1",
			"content":    "This is the content of post 1",
			"user_id":    uuid.New(),
			"created_at": "2024-01-01T12:00:00Z",
			"updated_at": "2024-01-01T12:00:00Z",
		},
		{
			"id":         uuid.New(),
			"title":      "Example Post 2",
			"content":    "This is the content of post 2",
			"user_id":    uuid.New(),
			"created_at": "2024-01-01T11:30:00Z",
			"updated_at": "2024-01-01T11:30:00Z",
		},
	}

	response := map[string]interface{}{
		"items":       posts,
		"has_more":    false,
		"next_cursor": nil,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Example: Get post by ID
func (s *APIServer) handleGetPost(w http.ResponseWriter, r *http.Request) {
	vars := chi.URLParam(r, "id")

	postID, err := uuid.Parse(vars)
	if err != nil {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	// In a real application, you would:
	// postRepo := repositories.NewPostsRepository(s.conn)
	// post, err := postRepo.GetByID(ctx, postID)

	// Example response
	post := map[string]interface{}{
		"id":         postID,
		"title":      "Example Post",
		"content":    "This is the content of the post",
		"user_id":    uuid.New(),
		"created_at": "2024-01-01T12:00:00Z",
		"updated_at": "2024-01-01T12:00:00Z",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(post)
}

// Example: Create new post
func (s *APIServer) handleCreatePost(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title   string    `json:"title"`
		Content string    `json:"content"`
		UserID  uuid.UUID `json:"user_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate input
	if req.Title == "" || req.Content == "" {
		http.Error(w, "Title and content are required", http.StatusBadRequest)
		return
	}

	// In a real application, you would:
	// postRepo := repositories.NewPostsRepository(s.conn)
	// post, err := postRepo.Create(ctx, repositories.CreatePostsParams{
	//     Title:   req.Title,
	//     Content: req.Content,
	//     UserID:  req.UserID,
	// })

	// Example response
	post := map[string]interface{}{
		"id":         uuid.New(),
		"title":      req.Title,
		"content":    req.Content,
		"user_id":    req.UserID,
		"created_at": "2024-01-01T12:00:00Z",
		"updated_at": "2024-01-01T12:00:00Z",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(post)
}

// Example: List posts for a specific user
func (s *APIServer) handleListUserPosts(w http.ResponseWriter, r *http.Request) {
	vars := chi.URLParam(r, "id")

	_, err := uuid.Parse(vars) // Validate ID format but don't store
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Parse pagination parameters (in real app, these would be used)
	// limit := 20
	// if l := r.URL.Query().Get("limit"); l != "" {
	// 	if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
	// 		limit = parsed
	// 	}
	// }
	// cursor := r.URL.Query().Get("cursor")

	// In a real application, you would:
	// postRepo := repositories.NewPostsRepository(s.conn)
	// result, err := postRepo.ListByUserIDPaginated(ctx, repositories.ListByUserIDPaginatedParams{
	//     UserID: userID,
	//     Cursor: cursor,
	//     Limit:  limit,
	// })

	// Example response
	posts := []map[string]interface{}{
		{
			"id":         uuid.New(),
			"title":      "User's Post 1",
			"content":    "This is a post by the user",
			"user_id":    uuid.New(), // In real app, this would be the actual userID
			"created_at": "2024-01-01T12:00:00Z",
			"updated_at": "2024-01-01T12:00:00Z",
		},
	}

	response := map[string]interface{}{
		"items":       posts,
		"has_more":    false,
		"next_cursor": nil,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
