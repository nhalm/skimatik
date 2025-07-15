package generator

// Inline pagination templates for zero-dependency code generation
const (
	// Inline pagination types and utilities template
	inlinePaginationTypesTemplate = `// PaginationParams holds parameters for cursor-based pagination
type PaginationParams struct {
	// Cursor is the base64-encoded UUID to start pagination from
	// If empty, starts from the beginning
	Cursor string ` + "`json:\"cursor,omitempty\"`" + `

	// Limit is the maximum number of items to return
	// Must be between 1 and 100, defaults to 20
	Limit int ` + "`json:\"limit,omitempty\"`" + `
}

// PaginationResult holds the result of a paginated query
type PaginationResult struct {
	// Items is the list of items returned
	Items []{{.StructName}} ` + "`json:\"items\"`" + `

	// HasMore indicates if there are more items available
	HasMore bool ` + "`json:\"has_more\"`" + `

	// NextCursor is the cursor for the next page
	// Only set if HasMore is true
	NextCursor string ` + "`json:\"next_cursor,omitempty\"`" + `

	// Total is the total count of items (optional, expensive to calculate)
	Total *int ` + "`json:\"total,omitempty\"`" + `
}

// encodeCursor encodes a UUID as a base64 cursor
func encodeCursor(id uuid.UUID) string {
	return base64.URLEncoding.EncodeToString(id[:])
}

// decodeCursor decodes a base64 cursor back to a UUID
func decodeCursor(cursor string) (uuid.UUID, error) {
	if cursor == "" {
		return uuid.Nil, fmt.Errorf("empty cursor")
	}

	cursorBytes, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid cursor format: %w", err)
	}

	if len(cursorBytes) != 16 {
		return uuid.Nil, fmt.Errorf("invalid cursor length: expected 16 bytes, got %d", len(cursorBytes))
	}

	var id uuid.UUID
	copy(id[:], cursorBytes)
	return id, nil
}

// validatePaginationParams validates pagination parameters
func validatePaginationParams(params PaginationParams) error {
	if params.Limit < 0 {
		return fmt.Errorf("limit cannot be negative")
	}
	if params.Limit > 100 {
		return fmt.Errorf("limit cannot exceed 100")
	}

	if params.Cursor != "" {
		_, err := decodeCursor(params.Cursor)
		if err != nil {
			return fmt.Errorf("invalid cursor: %w", err)
		}
	}

	return nil
}`

	// Simple List template (non-paginated)
	inlineListTemplate = `// List retrieves all {{.StructName}}s
func (r *{{.RepositoryName}}) List(ctx context.Context) ([]{{.StructName}}, error) {
	query := ` + "`" + `
		SELECT {{.SelectColumns}}
		FROM {{.TableName}}
		ORDER BY {{.IDColumn}} ASC
	` + "`" + `
	
	rows, err := r.conn.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var results []{{.StructName}}
	for rows.Next() {
		var {{.ReceiverName}} {{.StructName}}
		err := rows.Scan({{.ScanArgs}})
		if err != nil {
			return nil, err
		}
		results = append(results, {{.ReceiverName}})
	}
	
	return results, rows.Err()
}`

	// Paginated List template (with inline pagination logic)
	inlineListPaginatedTemplate = `// ListPaginated retrieves {{.StructName}}s with cursor-based pagination
func (r *{{.RepositoryName}}) ListPaginated(ctx context.Context, params PaginationParams) (*PaginationResult, error) {
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
	query := ` + "`" + `
		SELECT {{.SelectColumns}}
		FROM {{.TableName}}
		WHERE ($1::uuid IS NULL OR {{.IDColumn}} > $1)
		ORDER BY {{.IDColumn}} ASC
		LIMIT $2
	` + "`" + `
	
	rows, err := r.conn.Query(ctx, query, cursor, int32(limit+1))
	if err != nil {
		return nil, fmt.Errorf("pagination query failed: %w", err)
	}
	defer rows.Close()
	
	var items []{{.StructName}}
	for rows.Next() {
		var {{.ReceiverName}} {{.StructName}}
		err := rows.Scan({{.ScanArgs}})
		if err != nil {
			return nil, err
		}
		items = append(items, {{.ReceiverName}})
	}
	
	if err := rows.Err(); err != nil {
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

	return &PaginationResult{
		Items:      items,
		HasMore:    hasMore,
		NextCursor: nextCursor,
	}, nil
}`
)
