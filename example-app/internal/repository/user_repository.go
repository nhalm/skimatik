package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nhalm/pgxkit/v2"
	"github.com/nhalm/skimatik/v2/example-app/internal/models"
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

func (r *UserRepository) GetActiveUsers(ctx context.Context, limit int) ([]models.UserSummary, error) {
	results, err := r.UsersQueries.GetActiveUsers(ctx, executorFromContext(ctx, r.db), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get active users: %w", err)
	}

	users := make([]models.UserSummary, len(results))
	for i, result := range results {
		users[i] = models.UserSummary{
			ID:       result.Id,
			Name:     result.Name,
			Email:    result.Email,
			IsActive: result.IsActive,
		}
	}

	return users, nil
}

func (r *UserRepository) SearchUsers(ctx context.Context, query string) ([]models.UserSummary, error) {
	results, err := r.UsersQueries.SearchUsers(ctx, executorFromContext(ctx, r.db), "%"+query+"%", 50)
	if err != nil {
		return nil, fmt.Errorf("failed to search users: %w", err)
	}

	users := make([]models.UserSummary, len(results))
	for i, result := range results {
		users[i] = models.UserSummary{
			ID:       result.Id,
			Name:     result.Name,
			Email:    result.Email,
			IsActive: result.IsActive,
		}
	}

	return users, nil
}

func (r *UserRepository) GetUserStats(ctx context.Context, userID uuid.UUID) (*models.UserStats, error) {
	result, err := r.UsersQueries.GetUserStats(ctx, executorFromContext(ctx, r.db), userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user stats: %w", err)
	}

	stats := &models.UserStats{
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

func (r *UserRepository) GetUser(ctx context.Context, userID uuid.UUID) (*models.UserDetail, error) {
	// Use the generated Get method from UsersRepository
	user, err := r.Get(ctx, executorFromContext(ctx, r.db), userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	userDetail := &models.UserDetail{
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

// CreateUser delegates to the audited generated Create.
func (r *UserRepository) CreateUser(ctx context.Context, name, email string, bio *string) (*models.UserSummary, error) {
	user, err := r.Create(ctx, executorFromContext(ctx, r.db), generated.CreateUsersParams{
		Name:  name,
		Email: email,
		Bio:   bio,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return &models.UserSummary{
		ID:       user.Id,
		Name:     user.Name,
		Email:    user.Email,
		IsActive: user.IsActive,
	}, nil
}

// UpdateUserName delegates to the audited generated Update, mutating only the
// name field. The current row is read first so the audit CTE captures a
// faithful post-image of the unchanged columns.
func (r *UserRepository) UpdateUserName(ctx context.Context, userID uuid.UUID, name string) (*models.UserSummary, error) {
	exec := executorFromContext(ctx, r.db)
	current, err := r.Get(ctx, exec, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to read user before update: %w", err)
	}
	user, err := r.Update(ctx, exec, userID, generated.UpdateUsersParams{
		Name:      name,
		Email:     current.Email,
		Bio:       current.Bio,
		IsActive:  current.IsActive,
		CreatedAt: current.CreatedAt,
		UpdatedAt: current.UpdatedAt,
		DeletedAt: current.DeletedAt,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}
	return &models.UserSummary{
		ID:       user.Id,
		Name:     user.Name,
		Email:    user.Email,
		IsActive: user.IsActive,
	}, nil
}

// GetUserAuditHistory returns the SCD Type 2 history rows for a user, ordered
// from oldest to newest version.
func (r *UserRepository) GetUserAuditHistory(ctx context.Context, userID uuid.UUID) ([]models.UserAuditEntry, error) {
	const q = `
		SELECT id, parent_id, version, snapshot::text, valid_from, valid_to
		FROM users_audit
		WHERE parent_id = $1
		ORDER BY version ASC
	`
	rows, err := executorFromContext(ctx, r.db).Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query users_audit: %w", err)
	}
	defer rows.Close()

	const tsLayout = "2006-01-02T15:04:05.999999999Z07:00"
	var entries []models.UserAuditEntry
	for rows.Next() {
		var (
			entry  models.UserAuditEntry
			start  time.Time
			endPtr *time.Time
		)
		if err := rows.Scan(&entry.ID, &entry.ParentID, &entry.Version, &entry.Snapshot, &start, &endPtr); err != nil {
			return nil, fmt.Errorf("failed to scan audit row: %w", err)
		}
		entry.ValidFrom = start.Format(tsLayout)
		if endPtr != nil {
			s := endPtr.Format(tsLayout)
			entry.ValidTo = &s
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("audit history rows: %w", err)
	}
	return entries, nil
}
