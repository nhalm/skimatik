package main

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/nhalm/pgxkit"
)

// Example application demonstrating skimatik shared utility patterns and repository embedding
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

// Interface defined by the team based on domain needs
type UserManager interface {
	CreateUser(ctx context.Context, params CreateUsersParams) (*Users, error)
	GetUser(ctx context.Context, id uuid.UUID) (*Users, error)
	UpdateUser(ctx context.Context, id uuid.UUID, params UpdateUsersParams) (*Users, error)
	DeleteUser(ctx context.Context, id uuid.UUID) error
	ListUsers(ctx context.Context) ([]Users, error)
	GetActiveUsers(ctx context.Context) ([]Users, error)
}

// Generated repository with shared utilities
type UsersRepository struct {
	db *pgxkit.DB
}

func NewUsersRepository(db *pgxkit.DB) *UsersRepository {
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

// Repository implementation that embeds generated repository and implements interface
type UserRepository struct {
	*UsersRepository // Embed generated repository
}

func NewUserRepository(db *pgxkit.DB) UserManager {
	return &UserRepository{
		UsersRepository: NewUsersRepository(db),
	}
}

// Interface methods automatically satisfied by embedded repository
func (r *UserRepository) CreateUser(ctx context.Context, params CreateUsersParams) (*Users, error) {
	return r.UsersRepository.Create(ctx, params)
}

func (r *UserRepository) GetUser(ctx context.Context, id uuid.UUID) (*Users, error) {
	return r.UsersRepository.GetByID(ctx, id)
}

func (r *UserRepository) UpdateUser(ctx context.Context, id uuid.UUID, params UpdateUsersParams) (*Users, error) {
	return r.UsersRepository.Update(ctx, id, params)
}

func (r *UserRepository) DeleteUser(ctx context.Context, id uuid.UUID) error {
	return r.UsersRepository.Delete(ctx, id)
}

func (r *UserRepository) ListUsers(ctx context.Context) ([]Users, error) {
	return r.UsersRepository.List(ctx)
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

// Service layer that uses the interface, fulfilled by the user's repository
type UserService struct {
	userRepo UserManager // Property of interface type
}

func NewUserService(userRepo UserManager) *UserService {
	return &UserService{
		userRepo: userRepo,
	}
}

// Service methods delegate to repository through interface
func (s *UserService) RegisterUser(ctx context.Context, name, email string) (*Users, error) {
	params := CreateUsersParams{
		Name:  name,
		Email: email,
	}
	return s.userRepo.CreateUser(ctx, params)
}

func (s *UserService) GetUserDashboard(ctx context.Context) ([]Users, error) {
	// Business logic can use any interface methods
	return s.userRepo.GetActiveUsers(ctx)
}

func main() {
	// Database connection
	ctx := context.Background()
	dsn := "postgres://skimatik:skimatik_test_password@localhost:5432/skimatik_test?sslmode=disable"

	db, err := pgxkit.New(ctx, dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Shutdown(context.Background())

	// Test connection
	if err := db.HealthCheck(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("✅ Connected to database successfully")

	// Demonstrate repository embedding pattern
	log.Println("\n🔧 Demonstrating Repository Embedding Pattern:")

	// 1. Create repository that implements interface and embeds generated repository
	userRepo := NewUserRepository(db)
	log.Println("✅ Created UserRepository (embeds generated repository)")

	// 2. Service has property of interface type, fulfilled by repository
	userService := NewUserService(userRepo)
	log.Println("✅ Created UserService (uses interface property)")

	// Demonstrate shared utility patterns
	log.Println("\n🚀 Demonstrating Shared Utility Patterns:")

	// Create a user using retry operation utilities
	user, err := userService.RegisterUser(ctx, "John Doe", "john@example.com")
	if err != nil {
		log.Printf("❌ Failed to register user: %v", err)
	} else {
		log.Printf("✅ Registered user: %s (ID: %s)", user.Name, user.Id)
	}

	// List users using shared database utilities
	users, err := userService.userRepo.ListUsers(ctx)
	if err != nil {
		log.Printf("❌ Failed to list users: %v", err)
	} else {
		log.Printf("✅ Listed %d users using shared database utilities", len(users))
	}

	// Get active users using custom business logic with shared utilities
	activeUsers, err := userService.GetUserDashboard(ctx)
	if err != nil {
		log.Printf("❌ Failed to get active users: %v", err)
	} else {
		log.Printf("✅ Retrieved %d active users using custom business logic", len(activeUsers))
	}

	// Demonstrate direct repository usage with retry
	if user != nil {
		updatedUser, err := userRepo.(*UserRepository).UsersRepository.CreateWithRetry(ctx, CreateUsersParams{
			Name:  "Jane Doe",
			Email: "jane@example.com",
		})
		if err != nil {
			log.Printf("❌ Failed to create user with retry: %v", err)
		} else {
			log.Printf("✅ Created user with retry utilities: %s (ID: %s)", updatedUser.Name, updatedUser.Id)
		}
	}

	log.Println("\n🎉 Example completed - demonstrated:")
	log.Println("   • Repository embedding patterns")
	log.Println("   • Shared database operation utilities")
	log.Println("   • Retry operation utilities")
	log.Println("   • Interface-driven design")
	log.Println("   • Service layer with interface properties")
}
