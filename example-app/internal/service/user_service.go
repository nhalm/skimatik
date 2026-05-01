package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/nhalm/skimatik/v2/example-app/internal/models"
)

// UserRepository defines what the service layer needs from a user repository
// This interface is owned by the consumer (service), not the implementer (repository package)
// The repository should return domain types, not database-specific types
type UserRepository interface {
	// Basic generated query methods - all return domain types
	GetActiveUsers(ctx context.Context, limit int) ([]models.UserSummary, error)
	SearchUsers(ctx context.Context, query string) ([]models.UserSummary, error)
	GetUserStats(ctx context.Context, userID uuid.UUID) (*models.UserStats, error)
	DeactivateUser(ctx context.Context, userID uuid.UUID) error
	GetUser(ctx context.Context, userID uuid.UUID) (*models.UserDetail, error)
}

// UserService implements the api.UserService interface using domain types
type UserService struct {
	userRepo UserRepository
}

// NewUserService creates a new user service that implements api.UserService
func NewUserService(userRepo UserRepository) *UserService {
	return &UserService{
		userRepo: userRepo,
	}
}

// Implement api.UserService interface methods
// The service layer focuses on business logic, not data conversion

func (s *UserService) GetActiveUsers(ctx context.Context, limit int) ([]models.UserSummary, error) {
	users, err := s.userRepo.GetActiveUsers(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get active users: %w", err)
	}

	// Service layer can apply business logic here if needed
	// For now, we just pass through the domain types
	return users, nil
}

func (s *UserService) SearchUsers(ctx context.Context, query string) ([]models.UserSummary, error) {
	users, err := s.userRepo.SearchUsers(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to search users: %w", err)
	}

	// Service layer can apply business logic here if needed
	return users, nil
}

func (s *UserService) GetUserStats(ctx context.Context, userID uuid.UUID) (*models.UserStats, error) {
	stats, err := s.userRepo.GetUserStats(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user stats: %w", err)
	}

	// Service layer can apply business logic here if needed
	return stats, nil
}

func (s *UserService) DeactivateUser(ctx context.Context, userID uuid.UUID) error {
	err := s.userRepo.DeactivateUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to deactivate user: %w", err)
	}

	// Service layer can add business logic here (e.g., send notification, log activity)
	return nil
}

func (s *UserService) GetUser(ctx context.Context, userID uuid.UUID) (*models.UserDetail, error) {
	user, err := s.userRepo.GetUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Service layer can apply business logic here if needed
	return user, nil
}
