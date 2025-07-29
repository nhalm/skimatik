package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/nhalm/skimatik/example-app/domain"
)

var ErrPostNotFound = fmt.Errorf("post not found")

// postService implements the api.PostService interface using domain types
type postService struct {
	postRepo PostRepository
}

// NewPostService creates a new post service that implements api.PostService
func NewPostService(postRepo PostRepository) *postService {
	return &postService{
		postRepo: postRepo,
	}
}

// Implement api.PostService interface methods
// The service layer focuses on business logic, not data conversion

func (s *postService) GetPublishedPosts(ctx context.Context, limit int) ([]domain.PostSummary, error) {
	posts, err := s.postRepo.GetPublishedPosts(ctx, int32(limit))
	if err != nil {
		return nil, fmt.Errorf("failed to get published posts: %w", err)
	}

	// Service layer can apply business logic here if needed
	// For now, we just pass through the domain types
	return posts, nil
}

func (s *postService) GetPostWithAuthor(ctx context.Context, postID uuid.UUID) (*domain.PostDetail, error) {
	post, err := s.postRepo.GetPostWithAuthor(ctx, postID)
	if err != nil {
		return nil, fmt.Errorf("failed to get post with author: %w", err)
	}

	// Service layer can apply business logic here if needed
	return post, nil
}

func (s *postService) GetUserPosts(ctx context.Context, userID uuid.UUID) ([]domain.PostSummary, error) {
	posts, err := s.postRepo.GetUserPosts(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user posts: %w", err)
	}

	// Service layer can apply business logic here if needed
	return posts, nil
}

func (s *postService) GetPostsWithStats(ctx context.Context, limit int) ([]domain.PostWithStats, error) {
	posts, err := s.postRepo.GetPostsWithStats(ctx, int32(limit))
	if err != nil {
		return nil, fmt.Errorf("failed to get posts with stats: %w", err)
	}

	// Service layer can apply business logic here if needed
	return posts, nil
}

func (s *postService) PublishPost(ctx context.Context, postID uuid.UUID) error {
	err := s.postRepo.PublishPost(ctx, postID)
	if err != nil {
		return fmt.Errorf("failed to publish post: %w", err)
	}

	// Service layer can add business logic here (e.g., send notifications, update cache)
	return nil
}

// Custom business methods

func (s *postService) GetFeaturedPosts(ctx context.Context, limit int) ([]domain.PostSummary, error) {
	posts, err := s.postRepo.GetFeaturedPosts(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get featured posts: %w", err)
	}

	// Service layer can apply business logic here if needed
	return posts, nil
}

func (s *postService) GetPostsByTag(ctx context.Context, tagName string, limit int) ([]domain.PostSummary, error) {
	posts, err := s.postRepo.GetPostsByTag(ctx, tagName, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get posts by tag: %w", err)
	}

	// Service layer can apply business logic here if needed
	return posts, nil
}

func (s *postService) GetPostStatistics(ctx context.Context) (*domain.PostStats, error) {
	stats, err := s.postRepo.GetPostStatistics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get post statistics: %w", err)
	}

	// Service layer can apply business logic here if needed
	return stats, nil
}

// Temporary stub implementation - replace when repository is implemented
type stubPostService struct{}

func NewStubPostService() *stubPostService {
	return &stubPostService{}
}

func (s *stubPostService) GetPublishedPosts(ctx context.Context, limit int) ([]domain.PostSummary, error) {
	return nil, fmt.Errorf("not implemented - awaiting code generation")
}

func (s *stubPostService) GetPostWithAuthor(ctx context.Context, postID uuid.UUID) (*domain.PostDetail, error) {
	return nil, fmt.Errorf("not implemented - awaiting code generation")
}

func (s *stubPostService) GetUserPosts(ctx context.Context, userID uuid.UUID) ([]domain.PostSummary, error) {
	return nil, fmt.Errorf("not implemented - awaiting code generation")
}

func (s *stubPostService) GetPostsWithStats(ctx context.Context, limit int) ([]domain.PostWithStats, error) {
	return nil, fmt.Errorf("not implemented - awaiting code generation")
}

func (s *stubPostService) PublishPost(ctx context.Context, postID uuid.UUID) error {
	return fmt.Errorf("not implemented - awaiting code generation")
}

func (s *stubPostService) GetFeaturedPosts(ctx context.Context, limit int) ([]domain.PostSummary, error) {
	return nil, fmt.Errorf("not implemented - awaiting code generation")
}

func (s *stubPostService) GetPostsByTag(ctx context.Context, tagName string, limit int) ([]domain.PostSummary, error) {
	return nil, fmt.Errorf("not implemented - awaiting code generation")
}

func (s *stubPostService) GetPostStatistics(ctx context.Context) (*domain.PostStats, error) {
	return nil, fmt.Errorf("not implemented - awaiting code generation")
}
