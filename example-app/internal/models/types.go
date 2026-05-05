// Package models defines the business types passed between the example-app's
// service, repository, and API layers.
package models

import (
	"github.com/google/uuid"
)

// User types

// UserSummary represents a user summary for business logic
type UserSummary struct {
	ID       uuid.UUID
	Name     string
	Email    string
	IsActive bool
}

// UserDetail represents detailed user information for business logic
type UserDetail struct {
	ID          uuid.UUID
	Name        string
	Email       string
	IsActive    bool
	PostCount   int
	CreatedAt   string
	LastLoginAt *string
}

// UserStats represents user statistics
type UserStats struct {
	UserID       uuid.UUID
	PostCount    int
	CommentCount int
	LastActivity *string
}

// UserAuditEntry represents a single SCD Type 2 audit row from users_audit.
type UserAuditEntry struct {
	ID        uuid.UUID `json:"id"`
	ParentID  uuid.UUID `json:"parent_id"`
	Version   int       `json:"version"`
	Snapshot  string    `json:"snapshot"`
	ValidFrom string    `json:"valid_from"`
	ValidTo   *string   `json:"valid_to"`
}

// Post types

// PostSummary represents a post summary for business logic
type PostSummary struct {
	ID          uuid.UUID
	Title       string
	Content     string
	AuthorID    uuid.UUID
	IsPublished bool
	PublishedAt *string
	CreatedAt   string
}

// PostDetail represents detailed post information for business logic
type PostDetail struct {
	ID          uuid.UUID
	Title       string
	Content     string
	AuthorID    uuid.UUID
	AuthorName  string
	AuthorEmail string
	IsPublished bool
	PublishedAt *string
	CreatedAt   string
}

// PostWithStats represents a post with engagement statistics
type PostWithStats struct {
	ID           uuid.UUID
	Title        string
	AuthorID     uuid.UUID
	AuthorName   string
	CommentCount int
	CreatedAt    string
}

// PostStats represents aggregated post statistics
type PostStats struct {
	TotalPosts     int
	PublishedPosts int
	DraftPosts     int
}
