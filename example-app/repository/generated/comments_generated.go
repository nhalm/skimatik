// Code generated by skimatik. DO NOT EDIT.
// Source: table comments

package generated

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nhalm/pgxkit"
)

// Comments represents a row from the comments table
type Comments struct {
	Id         uuid.UUID `json:"id" db:"id"`
	PostId     uuid.UUID `json:"post_id" db:"post_id"`
	AuthorId   uuid.UUID `json:"author_id" db:"author_id"`
	Content    string    `json:"content" db:"content"`
	IsApproved bool      `json:"is_approved" db:"is_approved"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

// GetID returns the ID of the Comments for pagination
func (c Comments) GetID() uuid.UUID {
	return c.Id
}

// CommentsRepository provides database operations for comments
type CommentsRepository struct {
	db *pgxkit.DB
}

// NewCommentsRepository creates a new CommentsRepository
func NewCommentsRepository(db *pgxkit.DB) *CommentsRepository {
	return &CommentsRepository{
		db: db,
	}
}

// CreateCommentsParams holds parameters for creating a Comments
type CreateCommentsParams struct {
	PostId   uuid.UUID `json:"post_id" db:"post_id"`
	AuthorId uuid.UUID `json:"author_id" db:"author_id"`
	Content  string    `json:"content" db:"content"`
}

// Create creates a new Comments
func (r *CommentsRepository) Create(ctx context.Context, params CreateCommentsParams) (*Comments, error) {
	query := `
		INSERT INTO comments (post_id, author_id, content)
		VALUES ($1, $2, $3)
		RETURNING id, post_id, author_id, content, is_approved, created_at, updated_at
	`

	var c Comments
	row := ExecuteQueryRow(ctx, r.db, "create", "Comments", query, params.PostId, params.AuthorId, params.Content)
	err := row.Scan(&c.Id, &c.PostId, &c.AuthorId, &c.Content, &c.IsApproved, &c.CreatedAt, &c.UpdatedAt)
	if err := HandleQueryRowError("create", "Comments", err); err != nil {
		return nil, err
	}

	return &c, nil
}

// Get retrieves a Comments by ID
func (r *CommentsRepository) Get(ctx context.Context, id uuid.UUID) (*Comments, error) {
	query := `
		SELECT id, post_id, author_id, content, is_approved, created_at, updated_at
		FROM comments
		WHERE id = $1
	`

	var c Comments
	row := ExecuteQueryRow(ctx, r.db, "get", "Comments", query, id)
	err := row.Scan(&c.Id, &c.PostId, &c.AuthorId, &c.Content, &c.IsApproved, &c.CreatedAt, &c.UpdatedAt)
	if err := HandleQueryRowError("get", "Comments", err); err != nil {
		return nil, err
	}

	return &c, nil
}

// UpdateCommentsParams holds parameters for updating a Comments
type UpdateCommentsParams struct {
	PostId     uuid.UUID `json:"post_id" db:"post_id"`
	AuthorId   uuid.UUID `json:"author_id" db:"author_id"`
	Content    string    `json:"content" db:"content"`
	IsApproved bool      `json:"is_approved" db:"is_approved"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

// Update updates an existing Comments
func (r *CommentsRepository) Update(ctx context.Context, id uuid.UUID, params UpdateCommentsParams) (*Comments, error) {
	query := `
		UPDATE comments
		SET <no value>
		WHERE id = $7
		RETURNING id, post_id, author_id, content, is_approved, created_at, updated_at
	`

	var c Comments
	row := ExecuteQueryRow(ctx, r.db, "update", "Comments", query, params.PostId, params.AuthorId, params.Content, params.IsApproved, params.CreatedAt, params.UpdatedAt, id)
	err := row.Scan(&c.Id, &c.PostId, &c.AuthorId, &c.Content, &c.IsApproved, &c.CreatedAt, &c.UpdatedAt)
	if err := HandleQueryRowError("update", "Comments", err); err != nil {
		return nil, err
	}

	return &c, nil
}

// Delete removes a Comments by ID
func (r *CommentsRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM comments WHERE id = $1`

	rowsAffected, err := ExecuteNonQueryWithRowsAffected(ctx, r.db, "delete", "Comments", query, id)
	if err != nil {
		return err
	}

	// Check if any rows were affected
	if rowsAffected == 0 {
		return fmt.Errorf("Comments not found")
	}

	return nil
}

// List retrieves all Commentss
func (r *CommentsRepository) List(ctx context.Context) ([]Comments, error) {
	query := `
		SELECT id, post_id, author_id, content, is_approved, created_at, updated_at
		FROM comments
		ORDER BY id ASC
	`

	rows, err := ExecuteQuery(ctx, r.db, "list", "Comments", query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Comments
	for rows.Next() {
		var c Comments
		err := rows.Scan(&c.Id, &c.PostId, &c.AuthorId, &c.Content, &c.IsApproved, &c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			return nil, HandleDatabaseError("scan", "Comments", err)
		}
		results = append(results, c)
	}

	return results, HandleRowsResult("Comments", rows)
}

// ListPaginated retrieves Commentss with cursor-based pagination
func (r *CommentsRepository) ListPaginated(ctx context.Context, params PaginationParams) (*PaginationResult[Comments], error) {
	// Validate parameters
	if err := validatePaginationParams(params); err != nil {
		return nil, err
	}

	// Set default limit
	limit := params.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	// Parse cursor if provided
	var cursor *uuid.UUID
	if params.Cursor != "" {
		cursorUUID, err := decodeCursor(params.Cursor)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor format: %w", err)
		}
		cursor = &cursorUUID
	}

	// Execute query with limit + 1 to check if there are more items
	query := `
		SELECT id, post_id, author_id, content, is_approved, created_at, updated_at
		FROM comments
		WHERE ($1::uuid IS NULL OR id > $1)
		ORDER BY id ASC
		LIMIT $2
	`

	rows, err := ExecuteQuery(ctx, r.db, "list_paginated", "Comments", query, cursor, int32(limit+1))
	if err != nil {
		return nil, fmt.Errorf("pagination query failed: %w", err)
	}
	defer rows.Close()

	var items []Comments
	for rows.Next() {
		var c Comments
		err := rows.Scan(&c.Id, &c.PostId, &c.AuthorId, &c.Content, &c.IsApproved, &c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			return nil, err
		}
		items = append(items, c)
	}

	if err := HandleRowsResult("Comments", rows); err != nil {
		return nil, err
	}

	// Check if there are more items
	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit] // Remove the extra item
	}

	// Generate next cursor if there are more items
	var nextCursor string
	if hasMore && len(items) > 0 {
		lastItem := items[len(items)-1]
		nextCursor = encodeCursor(lastItem.GetID())
	}

	return &PaginationResult[Comments]{
		Items:      items,
		HasMore:    hasMore,
		NextCursor: nextCursor,
	}, nil
}

// CreateWithRetry creates a new Comments with retry logic
func (r *CommentsRepository) CreateWithRetry(ctx context.Context, params CreateCommentsParams) (*Comments, error) {
	return RetryOperation(ctx, DefaultRetryConfig, "create", func(ctx context.Context) (*Comments, error) {
		return r.Create(ctx, params)
	})
}

// GetWithRetry retrieves a Comments by ID with retry logic
func (r *CommentsRepository) GetWithRetry(ctx context.Context, id uuid.UUID) (*Comments, error) {
	return RetryOperation(ctx, DefaultRetryConfig, "get", func(ctx context.Context) (*Comments, error) {
		return r.Get(ctx, id)
	})
}

// UpdateWithRetry updates an existing Comments with retry logic
func (r *CommentsRepository) UpdateWithRetry(ctx context.Context, id uuid.UUID, params UpdateCommentsParams) (*Comments, error) {
	return RetryOperation(ctx, DefaultRetryConfig, "update", func(ctx context.Context) (*Comments, error) {
		return r.Update(ctx, id, params)
	})
}

// ListWithRetry retrieves all Commentss with retry logic
func (r *CommentsRepository) ListWithRetry(ctx context.Context) ([]Comments, error) {
	return RetryOperationSlice(ctx, DefaultRetryConfig, "list", func(ctx context.Context) ([]Comments, error) {
		return r.List(ctx)
	})
}
