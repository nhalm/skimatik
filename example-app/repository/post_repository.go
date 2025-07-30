package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/nhalm/skimatik/example-app/domain"
	"github.com/nhalm/skimatik/example-app/repository/generated"
)

// PostRepository represents a custom repository that embeds the generated queries
// This demonstrates the recommended pattern for extending generated functionality
type PostRepository struct {
	// Embed the generated queries for basic operations
	*generated.PostsQueries
}

// NewPostRepository creates a new post repository with the generated queries
func NewPostRepository(queries *generated.PostsQueries) *PostRepository {
	return &PostRepository{
		PostsQueries: queries,
	}
}

// Implement service.PostRepository interface methods with domain type conversion

func (r *PostRepository) GetPublishedPosts(ctx context.Context, limit int32) ([]domain.PostSummary, error) {
	results, err := r.PostsQueries.GetPublishedPosts(ctx, fmt.Sprintf("%d", limit))
	if err != nil {
		return nil, fmt.Errorf("failed to get published posts: %w", err)
	}

	posts := make([]domain.PostSummary, len(results))
	for i, result := range results {
		var publishedAt *string
		if result.PublishedAt.Valid {
			publishedAtStr := result.PublishedAt.Time.Format("2006-01-02T15:04:05Z07:00")
			publishedAt = &publishedAtStr
		}

		posts[i] = domain.PostSummary{
			ID:          uuid.UUID(result.Id.Bytes),
			Title:       result.Title.String,
			Content:     result.Content.String,
			AuthorID:    uuid.UUID(result.AuthorId.Bytes),
			IsPublished: true, // GetPublishedPosts only returns published posts
			PublishedAt: publishedAt,
			CreatedAt:   result.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return posts, nil
}

func (r *PostRepository) GetPostWithAuthor(ctx context.Context, postID uuid.UUID) (*domain.PostDetail, error) {
	result, err := r.PostsQueries.GetPostWithAuthor(ctx, postID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get post with author: %w", err)
	}

	var publishedAt *string
	if result.PublishedAt.Valid {
		publishedAtStr := result.PublishedAt.Time.Format("2006-01-02T15:04:05Z07:00")
		publishedAt = &publishedAtStr
	}

	post := &domain.PostDetail{
		ID:          uuid.UUID(result.Id.Bytes),
		Title:       result.Title.String,
		Content:     result.Content.String,
		AuthorID:    uuid.UUID(result.AuthorId.Bytes),
		AuthorName:  result.AuthorName.String,
		AuthorEmail: result.AuthorEmail.String,
		IsPublished: result.IsPublished.Bool,
		PublishedAt: publishedAt,
		CreatedAt:   result.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}

	return post, nil
}

func (r *PostRepository) GetUserPosts(ctx context.Context, userID uuid.UUID) ([]domain.PostSummary, error) {
	results, err := r.PostsQueries.GetUserPosts(ctx, userID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get user posts: %w", err)
	}

	posts := make([]domain.PostSummary, len(results))
	for i, result := range results {
		var publishedAt *string
		if result.PublishedAt.Valid {
			publishedAtStr := result.PublishedAt.Time.Format("2006-01-02T15:04:05Z07:00")
			publishedAt = &publishedAtStr
		}

		posts[i] = domain.PostSummary{
			ID:          uuid.UUID(result.Id.Bytes),
			Title:       result.Title.String,
			Content:     result.Content.String,
			AuthorID:    uuid.UUID(result.AuthorId.Bytes),
			IsPublished: result.IsPublished.Bool,
			PublishedAt: publishedAt,
			CreatedAt:   result.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return posts, nil
}

func (r *PostRepository) GetPostsWithStats(ctx context.Context, limit int32) ([]domain.PostWithStats, error) {
	// Use GetPostsWithCommentCount as the equivalent for "stats"
	results, err := r.PostsQueries.GetPostsWithCommentCount(ctx, fmt.Sprintf("%d", limit))
	if err != nil {
		return nil, fmt.Errorf("failed to get posts with stats: %w", err)
	}

	posts := make([]domain.PostWithStats, len(results))
	for i, result := range results {
		posts[i] = domain.PostWithStats{
			ID:           uuid.UUID(result.Id.Bytes),
			Title:        result.Title.String,
			AuthorID:     uuid.UUID(result.AuthorId.Bytes),
			AuthorName:   result.AuthorName.String,
			CommentCount: int(result.CommentCount.Int64),
			CreatedAt:    result.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return posts, nil
}

func (r *PostRepository) PublishPost(ctx context.Context, postID uuid.UUID) error {
	err := r.PostsQueries.PublishPost(ctx, postID)
	if err != nil {
		return fmt.Errorf("failed to publish post: %w", err)
	}
	return nil
}

// Custom business logic methods that build on the generated foundation

// GetFeaturedPosts returns posts marked as featured with custom business logic
func (r *PostRepository) GetFeaturedPosts(ctx context.Context, limit int) ([]domain.PostSummary, error) {
	// Use the generated GetPublishedPosts as a base, then filter
	posts, err := r.GetPublishedPosts(ctx, int32(limit*2)) // Get more to filter
	if err != nil {
		return nil, fmt.Errorf("failed to get published posts for featured filtering: %w", err)
	}

	// Custom filtering logic - in a real app this would check a featured flag
	var featured []domain.PostSummary
	for _, post := range posts {
		// Simple logic: featured posts have titles longer than 20 characters
		if len(post.Title) > 20 && len(featured) < limit {
			featured = append(featured, post)
		}
	}

	return featured, nil
}

// GetTrendingPosts returns posts with high engagement with custom business logic
func (r *PostRepository) GetTrendingPosts(ctx context.Context, limit int) ([]domain.PostSummary, error) {
	// Use the generated GetPublishedPosts as a base, then apply trending logic
	posts, err := r.GetPublishedPosts(ctx, int32(limit*3))
	if err != nil {
		return nil, fmt.Errorf("failed to get published posts: %w", err)
	}

	// Custom trending logic - in a real app this would check engagement metrics
	var trending []domain.PostSummary
	for _, post := range posts {
		// Simple logic: trending posts were published recently
		// In a real app, you'd check views, comments, likes, etc.
		if len(trending) < limit {
			trending = append(trending, post)
		}
	}

	return trending, nil
}

// GetPostSummary returns a summary of a post with custom formatting
func (r *PostRepository) GetPostSummary(ctx context.Context, id uuid.UUID) (string, error) {
	// This would use a generated GetPost function
	// For now, return a placeholder
	return fmt.Sprintf("Summary for post %s", id), nil
}

// Custom domain conversion methods

// Example of how you might extend generated functionality:
//
// func (r *PostRepository) CreatePostWithValidation(ctx context.Context, title, content string, authorId uuid.UUID) (*generated.Posts, error) {
//     // Custom validation logic
//     if len(title) < 3 {
//         return nil, fmt.Errorf("title too short")
//     }
//
//     // Use generated Create method (this would need to be implemented in the generator)
//     // return r.PostsQueries.CreatePost(ctx, generated.CreatePostParams{
//     //     Title: title,
//     //     Content: content,
//     //     AuthorId: authorId,
//     // })
//
//     return nil, fmt.Errorf("not implemented")
// }

// GetPostsByTag demonstrates custom query logic building on generated methods
func (r *PostRepository) GetPostsByTag(ctx context.Context, tagName string, limit int) ([]domain.PostSummary, error) {
	// In a real implementation, this would use a proper SQL query
	// For demo purposes, we'll use the generated method and filter
	posts, err := r.GetPublishedPosts(ctx, int32(limit*3))
	if err != nil {
		return nil, fmt.Errorf("failed to get posts for tag filtering: %w", err)
	}

	// Custom filtering logic (in reality, this would be in the SQL query)
	var tagged []domain.PostSummary
	for _, post := range posts {
		// Demo: filter posts that might contain the tag in content
		// In a real app, you'd have a proper tags table and join
		if len(tagged) < limit {
			tagged = append(tagged, post)
		}
	}

	return tagged, nil
}

// ArchiveOldPosts demonstrates custom business operations
func (r *PostRepository) ArchiveOldPosts(ctx context.Context, daysOld int) (int, error) {
	// Custom business logic that might use multiple generated methods
	// This would typically involve a custom SQL query or multiple operations

	// For demo purposes, we'll show the pattern without actual implementation
	// In reality, you might:
	// 1. Use a custom SQL query for bulk operations
	// 2. Or use multiple generated methods in a transaction

	return 0, fmt.Errorf("archive operation not implemented - this demonstrates the pattern")
}

// Custom validation that wraps generated methods
func (r *PostRepository) CreatePostWithValidation(ctx context.Context, title, content string, authorID uuid.UUID) (*generated.Posts, error) {
	// Custom validation logic
	if len(title) < 5 {
		return nil, fmt.Errorf("title must be at least 5 characters")
	}
	if len(content) < 20 {
		return nil, fmt.Errorf("content must be at least 20 characters")
	}

	// Use generated method for the actual database operation
	// Note: This assumes a CreatePost method exists in generated code
	// return r.PostsQueries.CreatePost(ctx, generated.CreatePostParams{
	// 	Title:    title,
	// 	Content:  content,
	// 	AuthorID: authorID,
	// })

	// For now, return error since generated code isn't available yet
	return nil, fmt.Errorf("create operation not implemented - awaiting generated code")
}

// GetPostStatistics demonstrates aggregating multiple generated queries
func (r *PostRepository) GetPostStatistics(ctx context.Context) (*domain.PostStats, error) {
	// Custom business logic that combines multiple generated queries
	// This pattern is useful for dashboard-style data aggregation

	// Example of how you might combine multiple generated methods:
	// publishedCount, err := r.GetPublishedPostCount(ctx)
	// draftCount, err := r.GetDraftPostCount(ctx)
	// etc.

	return &domain.PostStats{
		TotalPosts:     0,
		PublishedPosts: 0,
		DraftPosts:     0,
	}, fmt.Errorf("statistics not implemented - awaiting generated code")
}

// PostStats represents custom business data not directly from database
type PostStats struct {
	TotalPosts     int `json:"total_posts"`
	PublishedPosts int `json:"published_posts"`
	DraftPosts     int `json:"draft_posts"`
}
