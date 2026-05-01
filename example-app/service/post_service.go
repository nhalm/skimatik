package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/nhalm/skimatik/v2/example-app/domain"
)

// PostService implements the api.PostService interface using domain types
type PostService struct {
	postRepo PostRepository
}

// NewPostService creates a new post service that implements api.PostService
func NewPostService(postRepo PostRepository) *PostService {
	return &PostService{
		postRepo: postRepo,
	}
}

// Implement api.PostService interface methods
// The service layer focuses on business logic, not data conversion

func (s *PostService) GetPublishedPosts(ctx context.Context, limit int) ([]domain.PostSummary, error) {
	posts, err := s.postRepo.GetPublishedPosts(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get published posts: %w", err)
	}

	// Service layer can apply business logic here if needed
	// For now, we just pass through the domain types
	return posts, nil
}

func (s *PostService) GetPostWithAuthor(ctx context.Context, postID uuid.UUID) (*domain.PostDetail, error) {
	post, err := s.postRepo.GetPostWithAuthor(ctx, postID)
	if err != nil {
		return nil, fmt.Errorf("failed to get post with author: %w", err)
	}

	// Service layer can apply business logic here if needed
	return post, nil
}

func (s *PostService) GetUserPosts(ctx context.Context, userID uuid.UUID) ([]domain.PostSummary, error) {
	posts, err := s.postRepo.GetUserPosts(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user posts: %w", err)
	}

	// Service layer can apply business logic here if needed
	return posts, nil
}

func (s *PostService) GetPostsWithStats(ctx context.Context, limit int) ([]domain.PostWithStats, error) {
	posts, err := s.postRepo.GetPostsWithStats(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get posts with stats: %w", err)
	}

	// Service layer can apply business logic here if needed
	return posts, nil
}

func (s *PostService) PublishPost(ctx context.Context, postID uuid.UUID) error {
	err := s.postRepo.PublishPost(ctx, postID)
	if err != nil {
		return fmt.Errorf("failed to publish post: %w", err)
	}

	// Service layer can add business logic here (e.g., send notifications, update cache)
	return nil
}

// Custom business methods

func (s *PostService) GetFeaturedPosts(ctx context.Context, limit int) ([]domain.PostSummary, error) {
	posts, err := s.postRepo.GetFeaturedPosts(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get featured posts: %w", err)
	}

	// Service layer can apply business logic here if needed
	return posts, nil
}

func (s *PostService) GetPostsByTag(ctx context.Context, tagName string, limit int) ([]domain.PostSummary, error) {
	posts, err := s.postRepo.GetPostsByTag(ctx, tagName, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get posts by tag: %w", err)
	}

	// Service layer can apply business logic here if needed
	return posts, nil
}

func (s *PostService) GetPostStatistics(ctx context.Context) (*domain.PostStats, error) {
	stats, err := s.postRepo.GetPostStatistics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get post statistics: %w", err)
	}

	// Service layer can apply business logic here if needed
	return stats, nil
}
