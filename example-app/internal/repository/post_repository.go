package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/nhalm/pgxkit/v2"
	"github.com/nhalm/skimatik/v2/example-app/internal/models"
	"github.com/nhalm/skimatik/v2/example-app/internal/repository/generated"
)

// PostRepository represents a custom repository that embeds the generated queries
// This demonstrates the recommended pattern for extending generated functionality
type PostRepository struct {
	db *pgxkit.DB
	// Embed the generated queries for basic operations
	*generated.PostsQueries
}

// NewPostRepository creates a new post repository with the database connection
func NewPostRepository(db *pgxkit.DB) *PostRepository {
	return &PostRepository{
		db:           db,
		PostsQueries: generated.NewPostsQueries(),
	}
}

// Implement service.PostRepository interface methods with domain type conversion

func (r *PostRepository) GetPublishedPosts(ctx context.Context, limit int) ([]models.PostSummary, error) {
	results, err := r.PostsQueries.GetPublishedPosts(ctx, executorFromContext(ctx, r.db), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get published posts: %w", err)
	}

	posts := make([]models.PostSummary, len(results))
	for i, result := range results {
		var publishedAt *string
		if result.PublishedAt != nil {
			publishedAtStr := result.PublishedAt.Format("2006-01-02T15:04:05Z07:00")
			publishedAt = &publishedAtStr
		}

		posts[i] = models.PostSummary{
			ID:          result.Id,
			Title:       result.Title,
			Content:     result.Content,
			AuthorID:    result.AuthorId,
			IsPublished: true, // GetPublishedPosts only returns published posts
			PublishedAt: publishedAt,
			CreatedAt:   result.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return posts, nil
}

func (r *PostRepository) GetPostWithAuthor(ctx context.Context, postID uuid.UUID) (*models.PostDetail, error) {
	result, err := r.PostsQueries.GetPostWithAuthor(ctx, executorFromContext(ctx, r.db), postID)
	if err != nil {
		return nil, fmt.Errorf("failed to get post with author: %w", err)
	}

	var publishedAt *string
	if result.PublishedAt != nil {
		publishedAtStr := result.PublishedAt.Format("2006-01-02T15:04:05Z07:00")
		publishedAt = &publishedAtStr
	}

	post := &models.PostDetail{
		ID:          result.Id,
		Title:       result.Title,
		Content:     result.Content,
		AuthorID:    result.AuthorId,
		AuthorName:  result.AuthorName,
		AuthorEmail: result.AuthorEmail,
		IsPublished: result.IsPublished,
		PublishedAt: publishedAt,
		CreatedAt:   result.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	return post, nil
}

func (r *PostRepository) GetUserPosts(ctx context.Context, userID uuid.UUID) ([]models.PostSummary, error) {
	results, err := r.PostsQueries.GetUserPosts(ctx, executorFromContext(ctx, r.db), userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user posts: %w", err)
	}

	posts := make([]models.PostSummary, len(results))
	for i, result := range results {
		var publishedAt *string
		if result.PublishedAt != nil {
			publishedAtStr := result.PublishedAt.Format("2006-01-02T15:04:05Z07:00")
			publishedAt = &publishedAtStr
		}

		posts[i] = models.PostSummary{
			ID:          result.Id,
			Title:       result.Title,
			Content:     result.Content,
			AuthorID:    result.AuthorId,
			IsPublished: result.IsPublished,
			PublishedAt: publishedAt,
			CreatedAt:   result.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return posts, nil
}

func (r *PostRepository) GetPostsWithStats(ctx context.Context, limit int) ([]models.PostWithStats, error) {
	// Use GetPostsWithCommentCount as the equivalent for "stats"
	results, err := r.GetPostsWithCommentCount(ctx, executorFromContext(ctx, r.db), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get posts with stats: %w", err)
	}

	posts := make([]models.PostWithStats, len(results))
	for i, result := range results {
		posts[i] = models.PostWithStats{
			ID:           result.Id,
			Title:        result.Title,
			AuthorID:     result.AuthorId,
			AuthorName:   result.AuthorName,
			CommentCount: result.CommentCount,
			CreatedAt:    result.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return posts, nil
}

func (r *PostRepository) PublishPost(ctx context.Context, postID uuid.UUID) error {
	err := r.PostsQueries.PublishPost(ctx, executorFromContext(ctx, r.db), postID)
	if err != nil {
		return fmt.Errorf("failed to publish post: %w", err)
	}
	return nil
}

// Custom business logic methods that build on the generated foundation

// GetFeaturedPosts returns posts marked as featured with custom business logic
func (r *PostRepository) GetFeaturedPosts(ctx context.Context, limit int) ([]models.PostSummary, error) {
	// Use the generated GetPublishedPosts as a base, then filter
	posts, err := r.GetPublishedPosts(ctx, limit*2) // Get more to filter
	if err != nil {
		return nil, fmt.Errorf("failed to get published posts for featured filtering: %w", err)
	}

	// Custom filtering logic - in a real app this would check a featured flag
	var featured []models.PostSummary
	for _, post := range posts {
		// Simple logic: featured posts have titles longer than 20 characters
		if len(post.Title) > 20 && len(featured) < limit {
			featured = append(featured, post)
		}
	}

	return featured, nil
}

// GetPostsByTag demonstrates custom query logic building on generated methods
func (r *PostRepository) GetPostsByTag(ctx context.Context, tagName string, limit int) ([]models.PostSummary, error) {
	// In a real implementation, this would use a proper SQL query
	// For demo purposes, we'll use the generated method and filter
	posts, err := r.GetPublishedPosts(ctx, limit*3)
	if err != nil {
		return nil, fmt.Errorf("failed to get posts for tag filtering: %w", err)
	}

	// Custom filtering logic (in reality, this would be in the SQL query)
	var tagged []models.PostSummary
	for _, post := range posts {
		// Demo: filter posts that might contain the tag in content
		// In a real app, you'd have a proper tags table and join
		if len(tagged) < limit {
			tagged = append(tagged, post)
		}
	}

	return tagged, nil
}

// GetPostStatistics demonstrates aggregating multiple generated queries
func (r *PostRepository) GetPostStatistics(ctx context.Context) (*models.PostStats, error) {
	// Custom business logic that combines multiple generated queries
	// This pattern is useful for dashboard-style data aggregation

	// Example of how you might combine multiple generated methods:
	// publishedCount, err := r.GetPublishedPostCount(ctx)
	// draftCount, err := r.GetDraftPostCount(ctx)
	// etc.

	return &models.PostStats{
		TotalPosts:     0,
		PublishedPosts: 0,
		DraftPosts:     0,
	}, fmt.Errorf("statistics not implemented - awaiting generated code")
}
