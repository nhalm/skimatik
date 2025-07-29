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

// Post API types

// PostSummaryResponse represents a post summary for API responses
type PostSummaryResponse struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	AuthorID    uuid.UUID `json:"author_id"`
	IsPublished bool      `json:"is_published"`
	PublishedAt *string   `json:"published_at,omitempty"`
	CreatedAt   string    `json:"created_at"`
}

// PostDetailResponse represents detailed post information for API responses
type PostDetailResponse struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	AuthorID    uuid.UUID `json:"author_id"`
	AuthorName  string    `json:"author_name"`
	AuthorEmail string    `json:"author_email"`
	IsPublished bool      `json:"is_published"`
	PublishedAt *string   `json:"published_at,omitempty"`
	CreatedAt   string    `json:"created_at"`
}

// PostWithStatsResponse represents a post with engagement statistics for API responses
type PostWithStatsResponse struct {
	ID           uuid.UUID `json:"id"`
	Title        string    `json:"title"`
	AuthorID     uuid.UUID `json:"author_id"`
	AuthorName   string    `json:"author_name"`
	CommentCount int       `json:"comment_count"`
	CreatedAt    string    `json:"created_at"`
}

// PostStatsResponse represents aggregated post statistics for API responses
type PostStatsResponse struct {
	TotalPosts     int `json:"total_posts"`
	PublishedPosts int `json:"published_posts"`
	DraftPosts     int `json:"draft_posts"`
}
