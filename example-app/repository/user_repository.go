package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/nhalm/pgxkit"
	"github.com/nhalm/skimatik/example-app/domain"
	"github.com/nhalm/skimatik/example-app/repository/generated"
)

// UserRepository represents a custom repository that embeds the generated queries
// This demonstrates the recommended pattern for extending generated functionality
type UserRepository struct {
	// Embed the generated repository for basic CRUD operations
	*generated.UsersRepository
	// Embed the generated queries repository for custom queries
	*generated.UsersQueries
}

// NewUserRepository creates a new user repository with the generated repositories
func NewUserRepository(db *pgxkit.DB) *UserRepository {
	return &UserRepository{
		UsersRepository: generated.NewUsersRepository(db),
		UsersQueries:    generated.NewUsersQueries(db),
	}
}

// Implement service.UserRepository interface methods with domain type conversion

func (r *UserRepository) GetActiveUsers(ctx context.Context, limit int32) ([]domain.UserSummary, error) {
	results, err := r.UsersQueries.GetActiveUsers(ctx, fmt.Sprintf("%d", limit))
	if err != nil {
		return nil, fmt.Errorf("failed to get active users: %w", err)
	}

	users := make([]domain.UserSummary, len(results))
	for i, result := range results {
		users[i] = domain.UserSummary{
			ID:       uuid.UUID(result.Id.Bytes),
			Name:     result.Name.String,
			Email:    result.Email.String,
			IsActive: result.IsActive.Bool,
		}
	}

	return users, nil
}

func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (*domain.UserDetail, error) {
	result, err := r.UsersQueries.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	var lastLoginAt *string
	// Note: The generated result doesn't include last_login_at, so we'll leave it nil for now
	// In a real implementation, you might need to add this field to the query

	user := &domain.UserDetail{
		ID:          uuid.UUID(result.Id.Bytes),
		Name:        result.Name.String,
		Email:       result.Email.String,
		IsActive:    result.IsActive.Bool,
		PostCount:   0, // This would need to be calculated or included in the query
		CreatedAt:   result.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		LastLoginAt: lastLoginAt,
	}

	return user, nil
}

func (r *UserRepository) SearchUsers(ctx context.Context, query string) ([]domain.UserSummary, error) {
	results, err := r.UsersQueries.SearchUsers(ctx, "%"+query+"%", "50")
	if err != nil {
		return nil, fmt.Errorf("failed to search users: %w", err)
	}

	users := make([]domain.UserSummary, len(results))
	for i, result := range results {
		users[i] = domain.UserSummary{
			ID:       uuid.UUID(result.Id.Bytes),
			Name:     result.Name.String,
			Email:    result.Email.String,
			IsActive: result.IsActive.Bool,
		}
	}

	return users, nil
}

func (r *UserRepository) GetUserStats(ctx context.Context, userID uuid.UUID) (*domain.UserStats, error) {
	result, err := r.UsersQueries.GetUserStats(ctx, userID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get user stats: %w", err)
	}

	stats := &domain.UserStats{
		UserID:       userID,
		PostCount:    int(result.PostCount.Int64),
		CommentCount: int(result.CommentCount.Int64),
		LastActivity: nil, // This would need to be added to the query if needed
	}

	return stats, nil
}

func (r *UserRepository) DeactivateUser(ctx context.Context, userID uuid.UUID) error {
	err := r.UsersQueries.DeactivateUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to deactivate user: %w", err)
	}

	return nil
}

func (r *UserRepository) GetUser(ctx context.Context, userID uuid.UUID) (*domain.UserDetail, error) {
	// Use the generated Get method from UsersRepository
	user, err := r.UsersRepository.Get(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	var bio *string
	if user.Bio.Valid {
		bio = &user.Bio.String
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

	// If bio exists in user but not in domain.UserDetail, we'd need to handle it
	_ = bio // Silence unused variable warning

	return userDetail, nil
}

// Custom business methods that extend generated functionality

// GetUserWithPostCount demonstrates combining multiple generated methods
func (r *UserRepository) GetUserWithPostCount(ctx context.Context, userID uuid.UUID) (*domain.UserDetail, error) {
	// Get the basic user info
	user, err := r.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Get the stats to fill in post count
	stats, err := r.GetUserStats(ctx, userID)
	if err != nil {
		// If stats fail, still return user but with 0 post count
		return user, nil
	}

	user.PostCount = stats.PostCount
	return user, nil
}

// ActivateUser demonstrates custom business logic using generated methods
func (r *UserRepository) ActivateUser(ctx context.Context, userID uuid.UUID) error {
	// This would typically update the is_active field to true
	// For now, we'll implement this as a placeholder
	// In a real implementation, you would either:
	// 1. First fetch the user to get current values, then update
	// 2. Create a custom SQL query that only updates the is_active field
	// 3. Or modify the generated Update method to accept partial updates

	return fmt.Errorf("ActivateUser not fully implemented - would need to fetch current user data first")
}
