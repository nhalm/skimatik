// Package api contains HTTP handlers, response helpers, and the consumer-owned
// service interfaces used by the example-app's HTTP layer.
package api

import (
	"context"

	"github.com/google/uuid"
	"github.com/nhalm/skimatik/v2/example-app/internal/models"
)

// UserService defines what the API layer needs from a user service
// This interface is owned by the API package (the consumer)
type UserService interface {
	GetActiveUsers(ctx context.Context, limit int) ([]models.UserSummary, error)
	SearchUsers(ctx context.Context, query string) ([]models.UserSummary, error)
	GetUserStats(ctx context.Context, userID uuid.UUID) (*models.UserStats, error)
	DeactivateUser(ctx context.Context, userID uuid.UUID) error
	GetUser(ctx context.Context, userID uuid.UUID) (*models.UserDetail, error)
}

// PostService defines what the API layer needs from a post service
// This interface is owned by the API package (the consumer)
type PostService interface {
	GetPublishedPosts(ctx context.Context, limit int) ([]models.PostSummary, error)
	GetPostWithAuthor(ctx context.Context, postID uuid.UUID) (*models.PostDetail, error)
	GetUserPosts(ctx context.Context, userID uuid.UUID) ([]models.PostSummary, error)
	GetPostsWithStats(ctx context.Context, limit int) ([]models.PostWithStats, error)
	PublishPost(ctx context.Context, postID uuid.UUID) error

	// Custom business methods
	GetFeaturedPosts(ctx context.Context, limit int) ([]models.PostSummary, error)
	GetPostsByTag(ctx context.Context, tagName string, limit int) ([]models.PostSummary, error)
	GetPostStatistics(ctx context.Context) (*models.PostStats, error)
}
