package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/nhalm/skimatik/example-app/repository/generated"
)

// UserService handles business logic for user operations
type UserService interface {
	GetUserByEmail(ctx context.Context, email string) (*UserProfile, error)
	GetActiveUsers(ctx context.Context, limit int) ([]UserProfile, error)
	GetUserStats(ctx context.Context, userID uuid.UUID) (*UserStats, error)
	SearchUsers(ctx context.Context, query string, limit int) ([]UserProfile, error)
	DeactivateUser(ctx context.Context, userID uuid.UUID) error
}

// UserProfile represents a user profile for the API
type UserProfile struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Bio       *string   `json:"bio,omitempty"`
	IsActive  bool      `json:"is_active"`
	CreatedAt string    `json:"created_at"`
}

// UserStats represents user statistics
type UserStats struct {
	PostCount    int `json:"post_count"`
	CommentCount int `json:"comment_count"`
}

type userService struct {
	userQueries *generated.UsersQueries
}

// NewUserService creates a new user service
func NewUserService(userQueries *generated.UsersQueries) UserService {
	return &userService{
		userQueries: userQueries,
	}
}

// GetUserByEmail retrieves a user by email address
func (s *userService) GetUserByEmail(ctx context.Context, email string) (*UserProfile, error) {
	// Validate email format
	if !isValidEmail(email) {
		return nil, fmt.Errorf("invalid email format: %s", email)
	}

	user, err := s.userQueries.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return s.toUserProfile(&user), nil
}

// GetActiveUsers retrieves a list of active users
func (s *userService) GetActiveUsers(ctx context.Context, limit int) ([]UserProfile, error) {
	// Validate limit
	if limit <= 0 || limit > 100 {
		return nil, fmt.Errorf("limit must be between 1 and 100, got %d", limit)
	}

	users, err := s.userQueries.GetActiveUsers(ctx, int32(limit))
	if err != nil {
		return nil, fmt.Errorf("failed to get active users: %w", err)
	}

	profiles := make([]UserProfile, len(users))
	for i, user := range users {
		profiles[i] = *s.toUserProfile(&user)
	}

	return profiles, nil
}

// GetUserStats retrieves statistics for a user
func (s *userService) GetUserStats(ctx context.Context, userID uuid.UUID) (*UserStats, error) {
	stats, err := s.userQueries.GetUserStats(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user stats: %w", err)
	}

	return &UserStats{
		PostCount:    int(stats.PostCount),
		CommentCount: int(stats.CommentCount),
	}, nil
}

// SearchUsers searches for users by name or email
func (s *userService) SearchUsers(ctx context.Context, query string, limit int) ([]UserProfile, error) {
	// Validate inputs
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}
	if limit <= 0 || limit > 50 {
		return nil, fmt.Errorf("limit must be between 1 and 50, got %d", limit)
	}

	// Add wildcards for ILIKE search
	searchPattern := "%" + strings.TrimSpace(query) + "%"

	users, err := s.userQueries.SearchUsers(ctx, searchPattern, int32(limit))
	if err != nil {
		return nil, fmt.Errorf("failed to search users: %w", err)
	}

	profiles := make([]UserProfile, len(users))
	for i, user := range users {
		profiles[i] = *s.toUserProfile(&user)
	}

	return profiles, nil
}

// DeactivateUser deactivates a user account
func (s *userService) DeactivateUser(ctx context.Context, userID uuid.UUID) error {
	err := s.userQueries.DeactivateUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to deactivate user: %w", err)
	}

	return nil
}

// toUserProfile converts generated user row to user profile
func (s *userService) toUserProfile(user *generated.GetActiveUsersRow) *UserProfile {
	return &UserProfile{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Bio:       user.Bio,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// isValidEmail performs basic email validation
func isValidEmail(email string) bool {
	return strings.Contains(email, "@") && strings.Contains(email, ".") && len(email) > 5
}
