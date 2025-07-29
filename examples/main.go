package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Example application demonstrating skimatik generated repositories
// This shows real usage of generated repositories with shared utilities

// Note: In a real application, you would import your generated repositories:
// import "your-project/repositories"

// For this example, we'll simulate the generated repository structure
type Users struct {
	Id        uuid.UUID `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Email     string    `json:"email" db:"email"`
	IsActive  bool      `json:"is_active" db:"is_active"`
	CreatedAt string    `json:"created_at" db:"created_at"`
}

func (u Users) GetID() uuid.UUID { return u.Id }

type CreateUsersParams struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type UpdateUsersParams struct {
	Name  *string `json:"name,omitempty"`
	Email *string `json:"email,omitempty"`
}

type PaginationParams struct {
	Cursor string `json:"cursor,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

type PaginationResult[T any] struct {
	Items      []T     `json:"items"`
	HasMore    bool    `json:"has_more"`
	NextCursor *string `json:"next_cursor"`
}

// Example of a generated repository structure
type UsersRepository struct {
	db *pgxpool.Pool
}

func NewUsersRepository(db *pgxpool.Pool) *UsersRepository {
	return &UsersRepository{db: db}
}

// Example CRUD operations demonstrating shared utility usage
func (r *UsersRepository) Create(ctx context.Context, params CreateUsersParams) (*Users, error) {
	query := `INSERT INTO users (name, email, is_active) VALUES ($1, $2, true) RETURNING id, name, email, is_active, created_at`

	// Using shared database utilities (simulated)
	row := r.db.QueryRow(ctx, query, params.Name, params.Email)
	var user Users
	err := row.Scan(&user.Id, &user.Name, &user.Email, &user.IsActive, &user.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create user failed: %w", err)
	}
	return &user, nil
}

func (r *UsersRepository) GetByID(ctx context.Context, id uuid.UUID) (*Users, error) {
	query := `SELECT id, name, email, is_active, created_at FROM users WHERE id = $1`

	// Using shared database utilities (simulated)
	row := r.db.QueryRow(ctx, query, id)
	var user Users
	err := row.Scan(&user.Id, &user.Name, &user.Email, &user.IsActive, &user.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("get user failed: %w", err)
	}
	return &user, nil
}

func (r *UsersRepository) Update(ctx context.Context, id uuid.UUID, params UpdateUsersParams) (*Users, error) {
	// Build dynamic query based on provided fields
	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	if params.Name != nil {
		setParts = append(setParts, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, *params.Name)
		argIndex++
	}
	if params.Email != nil {
		setParts = append(setParts, fmt.Sprintf("email = $%d", argIndex))
		args = append(args, *params.Email)
		argIndex++
	}

	if len(setParts) == 0 {
		return r.GetByID(ctx, id) // No updates, just return existing
	}

	query := fmt.Sprintf(`UPDATE users SET %s WHERE id = $%d RETURNING id, name, email, is_active, created_at`,
		fmt.Sprintf("%v", setParts), argIndex)
	args = append(args, id)

	// Using shared database utilities (simulated)
	row := r.db.QueryRow(ctx, query, args...)
	var user Users
	err := row.Scan(&user.Id, &user.Name, &user.Email, &user.IsActive, &user.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("update user failed: %w", err)
	}
	return &user, nil
}

func (r *UsersRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`

	// Using shared database utilities (simulated)
	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete user failed: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

func (r *UsersRepository) List(ctx context.Context) ([]Users, error) {
	query := `SELECT id, name, email, is_active, created_at FROM users ORDER BY created_at DESC`

	// Using shared database utilities (simulated)
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list users failed: %w", err)
	}
	defer rows.Close()

	var results []Users
	for rows.Next() {
		var user Users
		err := rows.Scan(&user.Id, &user.Name, &user.Email, &user.IsActive, &user.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan user failed: %w", err)
		}
		results = append(results, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return results, nil
}

// Simulated retry operation utilities
func (r *UsersRepository) CreateWithRetry(ctx context.Context, params CreateUsersParams) (*Users, error) {
	// Simulate retry logic (in real app, this would use shared retry utilities)
	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		user, err := r.Create(ctx, params)
		if err == nil {
			return user, nil
		}
		if attempt == maxRetries-1 {
			return nil, fmt.Errorf("create user failed after %d attempts: %w", maxRetries, err)
		}
		log.Printf("Create attempt %d failed, retrying: %v", attempt+1, err)
	}
	return nil, fmt.Errorf("create user failed after %d attempts", maxRetries)
}

// Example repository implementation showing embedding pattern
type UserRepository struct {
	*UsersRepository // Embed generated repository
}

func NewUserRepository(conn *pgxpool.Pool) *UserRepository {
	return &UserRepository{
		UsersRepository: NewUsersRepository(conn),
	}
}

// Custom business logic using shared utilities pattern
func (r *UserRepository) GetActiveUsers(ctx context.Context) ([]Users, error) {
	query := `SELECT id, name, email, is_active, created_at FROM users WHERE is_active = true ORDER BY created_at DESC`

	// Using shared database utilities pattern (simulated)
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("get active users failed: %w", err)
	}
	defer rows.Close()

	var results []Users
	for rows.Next() {
		var user Users
		err := rows.Scan(&user.Id, &user.Name, &user.Email, &user.IsActive, &user.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan user failed: %w", err)
		}
		results = append(results, user)
	}

	return results, nil
}

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
	log.Println("Example endpoints demonstrating shared utility patterns:")
	log.Println("  GET  /users              - List users using shared database utilities")
	log.Println("  GET  /users/{id}         - Get user by ID with shared error handling")
	log.Println("  POST /users              - Create user with retry operation utilities")
	log.Println("  PUT  /users/{id}         - Update user using shared database patterns")
	log.Println("  DELETE /users/{id}       - Delete user with shared error handling")
	log.Println("  GET  /users/active       - Custom query using shared utility patterns")

	if err := http.ListenAndServe(":8080", server.Router()); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// APIServer demonstrates repository embedding and shared utility usage
type APIServer struct {
	userRepo *UserRepository
}

func NewAPIServer(conn *pgxpool.Pool) *APIServer {
	// Initialize repository with embedded generated repository
	userRepo := NewUserRepository(conn)

	return &APIServer{
		userRepo: userRepo,
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

	// User routes demonstrating real repository usage
	r.Route("/users", func(r chi.Router) {
		r.Get("/", s.handleListUsers)
		r.Post("/", s.handleCreateUser)
		r.Get("/active", s.handleGetActiveUsers) // Custom business logic
		r.Get("/{id}", s.handleGetUser)
		r.Put("/{id}", s.handleUpdateUser)
		r.Delete("/{id}", s.handleDeleteUser)
	})

	return r
}

func (s *APIServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Test database connection using repository
	_, err := s.userRepo.List(ctx)
	if err != nil {
		http.Error(w, "Database unhealthy", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "healthy",
		"database": "connected",
		"features": "shared utilities, retry operations, embedding patterns",
	})
}

// List users using generated repository with shared database utilities
func (s *APIServer) handleListUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Using generated repository with shared database utilities
	users, err := s.userRepo.List(ctx)
	if err != nil {
		log.Printf("Failed to list users: %v", err)
		http.Error(w, "Failed to list users", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"items":    users,
		"has_more": false, // Simplified for example
		"count":    len(users),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	log.Printf("Listed %d users using shared database utilities", len(users))
}

// Get user by ID demonstrating shared error handling
func (s *APIServer) handleGetUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idParam := chi.URLParam(r, "id")

	userID, err := uuid.Parse(idParam)
	if err != nil {
		http.Error(w, "Invalid user ID format", http.StatusBadRequest)
		return
	}

	// Using generated repository with shared error handling
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if err.Error() == "user not found" {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		log.Printf("Failed to get user %s: %v", userID, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)

	log.Printf("Retrieved user %s using shared error handling", userID)
}

// Create user with retry operation utilities
func (s *APIServer) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var params CreateUsersParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate input
	if params.Name == "" || params.Email == "" {
		http.Error(w, "Name and email are required", http.StatusBadRequest)
		return
	}

	// Using retry operation utilities for resilient creation
	user, err := s.userRepo.CreateWithRetry(ctx, params)
	if err != nil {
		log.Printf("Failed to create user with retry: %v", err)
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)

	log.Printf("Created user %s using retry operation utilities", user.Id)
}

// Update user using shared database patterns
func (s *APIServer) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idParam := chi.URLParam(r, "id")

	userID, err := uuid.Parse(idParam)
	if err != nil {
		http.Error(w, "Invalid user ID format", http.StatusBadRequest)
		return
	}

	var params UpdateUsersParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Using generated repository with shared database patterns
	user, err := s.userRepo.Update(ctx, userID, params)
	if err != nil {
		if err.Error() == "user not found" {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		log.Printf("Failed to update user %s: %v", userID, err)
		http.Error(w, "Failed to update user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)

	log.Printf("Updated user %s using shared database patterns", userID)
}

// Delete user with shared error handling
func (s *APIServer) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idParam := chi.URLParam(r, "id")

	userID, err := uuid.Parse(idParam)
	if err != nil {
		http.Error(w, "Invalid user ID format", http.StatusBadRequest)
		return
	}

	// Using generated repository with shared error handling
	err = s.userRepo.Delete(ctx, userID)
	if err != nil {
		if err.Error() == "user not found" {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		log.Printf("Failed to delete user %s: %v", userID, err)
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
	log.Printf("Deleted user %s using shared error handling", userID)
}

// Get active users demonstrating custom business logic with shared utilities
func (s *APIServer) handleGetActiveUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Using custom business logic with shared utility patterns
	users, err := s.userRepo.GetActiveUsers(ctx)
	if err != nil {
		log.Printf("Failed to get active users: %v", err)
		http.Error(w, "Failed to get active users", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"items":       users,
		"active_only": true,
		"count":       len(users),
		"note":        "Custom business logic using shared utility patterns",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	log.Printf("Retrieved %d active users using custom business logic with shared utilities", len(users))
}
