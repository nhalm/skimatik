package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/nhalm/skimatik/example-app/repository/generated"
)

// PostService handles business logic for post operations
type PostService interface {
	GetPublishedPosts(ctx context.Context, limit int) ([]PostSummary, error)
	GetPostWithAuthor(ctx context.Context, postID uuid.UUID) (*PostDetail, error)
	GetUserPosts(ctx context.Context, userID uuid.UUID) ([]PostSummary, error)
	GetPostsWithCommentCount(ctx context.Context, limit int) ([]PostWithStats, error)
	PublishPost(ctx context.Context, postID uuid.UUID) error
}

// PostSummary represents a post summary for listings
type PostSummary struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	AuthorID    uuid.UUID `json:"author_id"`
	IsPublished bool      `json:"is_published"`
	PublishedAt *string   `json:"published_at,omitempty"`
	CreatedAt   string    `json:"created_at"`
}

// PostDetail represents detailed post information with author
type PostDetail struct {
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

// PostWithStats represents a post with engagement statistics
type PostWithStats struct {
	ID           uuid.UUID `json:"id"`
	Title        string    `json:"title"`
	AuthorID     uuid.UUID `json:"author_id"`
	AuthorName   string    `json:"author_name"`
	PublishedAt  string    `json:"published_at"`
	CreatedAt    string    `json:"created_at"`
	CommentCount int       `json:"comment_count"`
}

type postService struct {
	postQueries *generated.PostsQueries
}

// NewPostService creates a new post service
func NewPostService(postQueries *generated.PostsQueries) PostService {
	return &postService{
		postQueries: postQueries,
	}
}

// GetPublishedPosts retrieves published posts
func (s *postService) GetPublishedPosts(ctx context.Context, limit int) ([]PostSummary, error) {
	// Validate limit
	if limit <= 0 || limit > 50 {
		return nil, fmt.Errorf("limit must be between 1 and 50, got %d", limit)
	}

	posts, err := s.postQueries.GetPublishedPosts(ctx, int32(limit))
	if err != nil {
		return nil, fmt.Errorf("failed to get published posts: %w", err)
	}

	summaries := make([]PostSummary, len(posts))
	for i, post := range posts {
		summaries[i] = PostSummary{
			ID:          post.ID,
			Title:       post.Title,
			Content:     s.truncateContent(post.Content, 200),
			AuthorID:    post.AuthorID,
			IsPublished: true, // These are all published
			PublishedAt: stringPtr(post.PublishedAt.Format("2006-01-02T15:04:05Z")),
			CreatedAt:   post.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	return summaries, nil
}

// GetPostWithAuthor retrieves a post with author information
func (s *postService) GetPostWithAuthor(ctx context.Context, postID uuid.UUID) (*PostDetail, error) {
	post, err := s.postQueries.GetPostWithAuthor(ctx, postID)
	if err != nil {
		return nil, fmt.Errorf("failed to get post with author: %w", err)
	}

	var publishedAt *string
	if post.IsPublished && post.PublishedAt != nil {
		publishedAt = stringPtr(post.PublishedAt.Format("2006-01-02T15:04:05Z"))
	}

	return &PostDetail{
		ID:          post.ID,
		Title:       post.Title,
		Content:     post.Content,
		AuthorID:    post.AuthorID,
		AuthorName:  post.AuthorName,
		AuthorEmail: post.AuthorEmail,
		IsPublished: post.IsPublished,
		PublishedAt: publishedAt,
		CreatedAt:   post.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}, nil
}

// GetUserPosts retrieves all posts by a specific user
func (s *postService) GetUserPosts(ctx context.Context, userID uuid.UUID) ([]PostSummary, error) {
	posts, err := s.postQueries.GetUserPosts(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user posts: %w", err)
	}

	summaries := make([]PostSummary, len(posts))
	for i, post := range posts {
		var publishedAt *string
		if post.IsPublished && post.PublishedAt != nil {
			publishedAt = stringPtr(post.PublishedAt.Format("2006-01-02T15:04:05Z"))
		}

		summaries[i] = PostSummary{
			ID:          post.ID,
			Title:       post.Title,
			Content:     s.truncateContent(post.Content, 200),
			AuthorID:    post.AuthorID,
			IsPublished: post.IsPublished,
			PublishedAt: publishedAt,
			CreatedAt:   post.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	return summaries, nil
}

// GetPostsWithCommentCount retrieves posts with their comment counts
func (s *postService) GetPostsWithCommentCount(ctx context.Context, limit int) ([]PostWithStats, error) {
	// Validate limit
	if limit <= 0 || limit > 50 {
		return nil, fmt.Errorf("limit must be between 1 and 50, got %d", limit)
	}

	posts, err := s.postQueries.GetPostsWithCommentCount(ctx, int32(limit))
	if err != nil {
		return nil, fmt.Errorf("failed to get posts with comment count: %w", err)
	}

	stats := make([]PostWithStats, len(posts))
	for i, post := range posts {
		stats[i] = PostWithStats{
			ID:           post.ID,
			Title:        post.Title,
			AuthorID:     post.AuthorID,
			AuthorName:   post.AuthorName,
			PublishedAt:  post.PublishedAt.Format("2006-01-02T15:04:05Z"),
			CreatedAt:    post.CreatedAt.Format("2006-01-02T15:04:05Z"),
			CommentCount: int(post.CommentCount),
		}
	}

	return stats, nil
}

// PublishPost publishes a draft post
func (s *postService) PublishPost(ctx context.Context, postID uuid.UUID) error {
	err := s.postQueries.PublishPost(ctx, postID)
	if err != nil {
		return fmt.Errorf("failed to publish post: %w", err)
	}

	return nil
}

// truncateContent truncates content to a specified length
func (s *postService) truncateContent(content string, maxLength int) string {
	if len(content) <= maxLength {
		return content
	}

	// Find the last space before maxLength to avoid cutting words
	truncated := content[:maxLength]
	if lastSpace := strings.LastIndex(truncated, " "); lastSpace > maxLength/2 {
		truncated = truncated[:lastSpace]
	}

	return truncated + "..."
}

// stringPtr returns a pointer to a string
func stringPtr(s string) *string {
	return &s
}
