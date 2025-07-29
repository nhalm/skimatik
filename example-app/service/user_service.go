package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/nhalm/skimatik/example-app/domain"
)

var ErrUserNotFound = fmt.Errorf("user not found")

// UserRepository defines what the service layer needs from a user repository
// This interface is owned by the consumer (service), not the implementer (repository package)
// The repository should return domain types, not database-specific types
type UserRepository interface {
	// Basic generated query methods - all return domain types
	GetActiveUsers(ctx context.Context, limit int32) ([]domain.UserSummary, error)
	GetUserByEmail(ctx context.Context, email string) (*domain.UserDetail, error)
	SearchUsers(ctx context.Context, query string) ([]domain.UserSummary, error)
	GetUserStats(ctx context.Context, userID uuid.UUID) (*domain.UserStats, error)
	DeactivateUser(ctx context.Context, userID uuid.UUID) error
	GetUser(ctx context.Context, userID uuid.UUID) (*domain.UserDetail, error)
}

// userService implements the api.UserService interface using domain types
type userService struct {
	userRepo UserRepository
}

// NewUserService creates a new user service that implements api.UserService
func NewUserService(userRepo UserRepository) *userService {
	return &userService{
		userRepo: userRepo,
	}
}

// Implement api.UserService interface methods
// The service layer focuses on business logic, not data conversion

func (s *userService) GetActiveUsers(ctx context.Context, limit int) ([]domain.UserSummary, error) {
	users, err := s.userRepo.GetActiveUsers(ctx, int32(limit))
	if err != nil {
		return nil, fmt.Errorf("failed to get active users: %w", err)
	}

	// Service layer can apply business logic here if needed
	// For now, we just pass through the domain types
	return users, nil
}

func (s *userService) GetUserByEmail(ctx context.Context, email string) (*domain.UserDetail, error) {
	user, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	// Service layer can apply business logic here if needed
	return user, nil
}

func (s *userService) SearchUsers(ctx context.Context, query string) ([]domain.UserSummary, error) {
	users, err := s.userRepo.SearchUsers(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to search users: %w", err)
	}

	// Service layer can apply business logic here if needed
	return users, nil
}

func (s *userService) GetUserStats(ctx context.Context, userID uuid.UUID) (*domain.UserStats, error) {
	stats, err := s.userRepo.GetUserStats(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user stats: %w", err)
	}

	// Service layer can apply business logic here if needed
	return stats, nil
}

func (s *userService) DeactivateUser(ctx context.Context, userID uuid.UUID) error {
	err := s.userRepo.DeactivateUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to deactivate user: %w", err)
	}

	// Service layer can add business logic here (e.g., send notification, log activity)
	return nil
}

func (s *userService) GetUser(ctx context.Context, userID uuid.UUID) (*domain.UserDetail, error) {
	user, err := s.userRepo.GetUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Service layer can apply business logic here if needed
	return user, nil
}

// Temporary stub implementation - replace when repository is implemented
type stubUserService struct{}

func NewStubUserService() *stubUserService {
	return &stubUserService{}
}

func (s *stubUserService) GetActiveUsers(ctx context.Context, limit int) ([]domain.UserSummary, error) {
	return nil, fmt.Errorf("not implemented - awaiting code generation")
}

func (s *stubUserService) GetUserByEmail(ctx context.Context, email string) (*domain.UserDetail, error) {
	return nil, fmt.Errorf("not implemented - awaiting code generation")
}

func (s *stubUserService) SearchUsers(ctx context.Context, query string) ([]domain.UserSummary, error) {
	return nil, fmt.Errorf("not implemented - awaiting code generation")
}

func (s *stubUserService) GetUserStats(ctx context.Context, userID uuid.UUID) (*domain.UserStats, error) {
	return nil, fmt.Errorf("not implemented - awaiting code generation")
}

func (s *stubUserService) DeactivateUser(ctx context.Context, userID uuid.UUID) error {
	return fmt.Errorf("not implemented - awaiting code generation")
}

func (s *stubUserService) GetUser(ctx context.Context, userID uuid.UUID) (*domain.UserDetail, error) {
	return nil, fmt.Errorf("not implemented - awaiting code generation")
}
