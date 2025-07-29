package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/nhalm/skimatik/example-app/domain"
)

// PostRepository defines what the service layer needs from a post repository
// This interface is owned by the consumer (service), not the implementer (repository package)
// The repository should return domain types, not database-specific types
type PostRepository interface {
	// Basic generated query methods - all return domain types
	GetPublishedPosts(ctx context.Context, limit int32) ([]domain.PostSummary, error)
	GetPostWithAuthor(ctx context.Context, postID uuid.UUID) (*domain.PostDetail, error)
	GetUserPosts(ctx context.Context, userID uuid.UUID) ([]domain.PostSummary, error)
	GetPostsWithStats(ctx context.Context, limit int32) ([]domain.PostWithStats, error)
	PublishPost(ctx context.Context, postID uuid.UUID) error

	// Custom repository methods that extend generated functionality
	GetFeaturedPosts(ctx context.Context, limit int) ([]domain.PostSummary, error)
	GetPostsByTag(ctx context.Context, tagName string, limit int) ([]domain.PostSummary, error)
	GetPostStatistics(ctx context.Context) (*domain.PostStats, error)
}
