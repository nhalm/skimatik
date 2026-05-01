package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/nhalm/pgxkit/v2"
	"github.com/nhalm/skimatik/v2/example-app/domain"
	"github.com/nhalm/skimatik/v2/example-app/internal/repository/generated"
)

// UserRepository represents a custom repository that embeds the generated queries
// This demonstrates the recommended pattern for extending generated functionality
type UserRepository struct {
	db *pgxkit.DB
	// Embed the generated repository for basic CRUD operations
	*generated.UsersRepository
	// Embed the generated queries repository for custom queries
	*generated.UsersQueries
}

// NewUserRepository creates a new user repository with the generated repositories
func NewUserRepository(db *pgxkit.DB) *UserRepository {
	return &UserRepository{
		db:              db,
		UsersRepository: generated.NewUsersRepository(nil), // nil = use default UUID v7 generator
		UsersQueries:    generated.NewUsersQueries(),
	}
}

// Implement service.UserRepository interface methods with domain type conversion

func (r *UserRepository) GetActiveUsers(ctx context.Context, limit int) ([]domain.UserSummary, error) {
	results, err := r.UsersQueries.GetActiveUsers(ctx, executorFromContext(ctx, r.db), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get active users: %w", err)
	}

	users := make([]domain.UserSummary, len(results))
	for i, result := range results {
		users[i] = domain.UserSummary{
			ID:       result.Id,
			Name:     result.Name,
			Email:    result.Email,
			IsActive: result.IsActive,
		}
	}

	return users, nil
}

func (r *UserRepository) SearchUsers(ctx context.Context, query string) ([]domain.UserSummary, error) {
	results, err := r.UsersQueries.SearchUsers(ctx, executorFromContext(ctx, r.db), "%"+query+"%", 50)
	if err != nil {
		return nil, fmt.Errorf("failed to search users: %w", err)
	}

	users := make([]domain.UserSummary, len(results))
	for i, result := range results {
		users[i] = domain.UserSummary{
			ID:       result.Id,
			Name:     result.Name,
			Email:    result.Email,
			IsActive: result.IsActive,
		}
	}

	return users, nil
}

func (r *UserRepository) GetUserStats(ctx context.Context, userID uuid.UUID) (*domain.UserStats, error) {
	result, err := r.UsersQueries.GetUserStats(ctx, executorFromContext(ctx, r.db), userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user stats: %w", err)
	}

	stats := &domain.UserStats{
		UserID:       userID,
		PostCount:    result.PostCount,
		CommentCount: result.CommentCount,
		LastActivity: nil, // This would need to be added to the query if needed
	}

	return stats, nil
}

func (r *UserRepository) DeactivateUser(ctx context.Context, userID uuid.UUID) error {
	err := r.UsersQueries.DeactivateUser(ctx, executorFromContext(ctx, r.db), userID)
	if err != nil {
		return fmt.Errorf("failed to deactivate user: %w", err)
	}

	return nil
}

func (r *UserRepository) GetUser(ctx context.Context, userID uuid.UUID) (*domain.UserDetail, error) {
	// Use the generated Get method from UsersRepository
	user, err := r.Get(ctx, executorFromContext(ctx, r.db), userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	userDetail := &domain.UserDetail{
		ID:          user.Id,
		Name:        user.Name,
		Email:       user.Email,
		IsActive:    user.IsActive,
		PostCount:   0, // This would need to be calculated or fetched separately
		CreatedAt:   user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		LastLoginAt: nil, // Not available in the basic user struct
	}

	return userDetail, nil
}
