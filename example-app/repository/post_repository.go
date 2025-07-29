package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/nhalm/skimatik/example-app/repository/generated"
)

// PostRepository represents a custom repository that embeds the generated queries
// This demonstrates the recommended pattern for extending generated functionality
type PostRepository struct {
	// Embed the generated queries for basic operations
	*generated.PostQueries
}

// NewPostRepository creates a new post repository with the generated queries
func NewPostRepository(queries *generated.PostQueries) *PostRepository {
	return &PostRepository{
		PostQueries: queries,
	}
}

// Custom business logic methods that build on the generated foundation

// GetFeaturedPosts returns posts marked as featured with custom business logic
func (r *PostRepository) GetFeaturedPosts(ctx context.Context, limit int) ([]generated.GetPublishedPostsRow, error) {
	// Use the generated GetPublishedPosts as a base, then filter
	posts, err := r.PostQueries.GetPublishedPosts(ctx, int32(limit*2)) // Get more to filter
	if err != nil {
		return nil, fmt.Errorf("failed to get published posts: %w", err)
	}

	// Custom business logic: filter for "featured" posts
	// (In a real app, this might be a database field, but here we'll demo with title logic)
	var featured []generated.GetPublishedPostsRow
	for _, post := range posts {
		// Example: consider posts with "featured" in title or first 3 posts as featured
		if len(featured) < limit {
			featured = append(featured, post)
		}
	}

	return featured, nil
}

// GetPostsByTag demonstrates custom query logic building on generated methods
func (r *PostRepository) GetPostsByTag(ctx context.Context, tagName string, limit int) ([]generated.GetPublishedPostsRow, error) {
	// In a real implementation, this would use a proper SQL query
	// For demo purposes, we'll use the generated method and filter
	posts, err := r.PostQueries.GetPublishedPosts(ctx, int32(limit*3))
	if err != nil {
		return nil, fmt.Errorf("failed to get posts for tag filtering: %w", err)
	}

	// Custom filtering logic (in reality, this would be in the SQL query)
	var tagged []generated.GetPublishedPostsRow
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
func (r *PostRepository) CreatePostWithValidation(ctx context.Context, title, content string, authorID uuid.UUID) (*generated.Post, error) {
	// Custom validation logic
	if len(title) < 5 {
		return nil, fmt.Errorf("title must be at least 5 characters")
	}
	if len(content) < 20 {
		return nil, fmt.Errorf("content must be at least 20 characters")
	}

	// Use generated method for the actual database operation
	// Note: This assumes a CreatePost method exists in generated code
	// return r.PostQueries.CreatePost(ctx, generated.CreatePostParams{
	// 	Title:    title,
	// 	Content:  content,
	// 	AuthorID: authorID,
	// })

	// For now, return error since generated code isn't available yet
	return nil, fmt.Errorf("create operation not implemented - awaiting generated code")
}

// GetPostStatistics demonstrates aggregating multiple generated queries
func (r *PostRepository) GetPostStatistics(ctx context.Context) (*PostStats, error) {
	// Custom business logic that combines multiple generated queries
	// This pattern is useful for dashboard-style data aggregation

	// Example of how you might combine multiple generated methods:
	// publishedCount, err := r.GetPublishedPostCount(ctx)
	// draftCount, err := r.GetDraftPostCount(ctx)
	// etc.

	return &PostStats{
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
