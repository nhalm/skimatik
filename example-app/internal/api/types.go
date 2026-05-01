package api

import "github.com/google/uuid"

// API Request/Response types for HTTP context
// These are the types that get serialized to/from JSON

// User API types

// UserSummaryResponse represents a user summary for API responses
type UserSummaryResponse struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Email    string    `json:"email"`
	IsActive bool      `json:"is_active"`
}

// UserDetailResponse represents detailed user information for API responses
type UserDetailResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	IsActive    bool      `json:"is_active"`
	PostCount   int       `json:"post_count"`
	CreatedAt   string    `json:"created_at"`
	LastLoginAt *string   `json:"last_login_at,omitempty"`
}

// UserStatsResponse represents user statistics for API responses
type UserStatsResponse struct {
	UserID       uuid.UUID `json:"user_id"`
	PostCount    int       `json:"post_count"`
	CommentCount int       `json:"comment_count"`
	LastActivity *string   `json:"last_activity,omitempty"`
}
