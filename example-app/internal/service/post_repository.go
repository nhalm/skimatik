// Package service contains the example-app's business-logic layer and the
// consumer-owned repository interfaces it depends on.
package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/nhalm/skimatik/v2/example-app/internal/models"
)

// PostRepository defines what the service layer needs from a post repository
// This interface is owned by the consumer (service), not the implementer (repository package)
// The repository should return domain types, not database-specific types
type PostRepository interface {
	// Basic generated query methods - all return domain types
	GetPublishedPosts(ctx context.Context, limit int) ([]models.PostSummary, error)
	GetPostWithAuthor(ctx context.Context, postID uuid.UUID) (*models.PostDetail, error)
	GetUserPosts(ctx context.Context, userID uuid.UUID) ([]models.PostSummary, error)
	GetPostsWithStats(ctx context.Context, limit int) ([]models.PostWithStats, error)
	PublishPost(ctx context.Context, postID uuid.UUID) error

	// Custom repository methods that extend generated functionality
	GetFeaturedPosts(ctx context.Context, limit int) ([]models.PostSummary, error)
	GetPostsByTag(ctx context.Context, tagName string, limit int) ([]models.PostSummary, error)
	GetPostStatistics(ctx context.Context) (*models.PostStats, error)
}
