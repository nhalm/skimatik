package api

import (
	"context"

	"github.com/google/uuid"
	"github.com/nhalm/skimatik/example-app/domain"
)

// UserService defines what the API layer needs from a user service
// This interface is owned by the API package (the consumer)
type UserService interface {
	GetActiveUsers(ctx context.Context, limit int) ([]domain.UserSummary, error)
	GetUserByEmail(ctx context.Context, email string) (*domain.UserDetail, error)
	SearchUsers(ctx context.Context, query string) ([]domain.UserSummary, error)
	GetUserStats(ctx context.Context, userID uuid.UUID) (*domain.UserStats, error)
	DeactivateUser(ctx context.Context, userID uuid.UUID) error
	GetUser(ctx context.Context, userID uuid.UUID) (*domain.UserDetail, error)
}

// PostService defines what the API layer needs from a post service
// This interface is owned by the API package (the consumer)
type PostService interface {
	GetPublishedPosts(ctx context.Context, limit int) ([]domain.PostSummary, error)
	GetPostWithAuthor(ctx context.Context, postID uuid.UUID) (*domain.PostDetail, error)
	GetUserPosts(ctx context.Context, userID uuid.UUID) ([]domain.PostSummary, error)
	GetPostsWithStats(ctx context.Context, limit int) ([]domain.PostWithStats, error)
	PublishPost(ctx context.Context, postID uuid.UUID) error

	// Custom business methods
	GetFeaturedPosts(ctx context.Context, limit int) ([]domain.PostSummary, error)
	GetPostsByTag(ctx context.Context, tagName string, limit int) ([]domain.PostSummary, error)
	GetPostStatistics(ctx context.Context) (*domain.PostStats, error)
}
