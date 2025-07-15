package generator

// CRUD operation templates for code generation
const (
	// GetByID template
	getByIDTemplate = `// GetByID retrieves a {{.StructName}} by its ID
func (r *{{.RepositoryName}}) GetByID(ctx context.Context, id uuid.UUID) (*{{.StructName}}, error) {
	query := ` + "`" + `
		SELECT {{.SelectColumns}}
		FROM {{.TableName}}
		WHERE {{.IDColumn}} = $1
	` + "`" + `
	
	var {{.ReceiverName}} {{.StructName}}
	err := r.conn.QueryRow(ctx, query, id).Scan({{.ScanArgs}})
	if err != nil {
		return nil, err
	}
	
	return &{{.ReceiverName}}, nil
}`

	// Create template
	createTemplate = `// Create{{.StructName}}Params holds parameters for creating a {{.StructName}}
type Create{{.StructName}}Params struct {
{{range .CreateFields}}	{{.Name}} {{.Type}} ` + "`{{.Tag}}`" + `
{{end}}}

// Create creates a new {{.StructName}}
func (r *{{.RepositoryName}}) Create(ctx context.Context, params Create{{.StructName}}Params) (*{{.StructName}}, error) {
	query := ` + "`" + `
		INSERT INTO {{.TableName}} ({{.InsertColumns}})
		VALUES ({{.InsertPlaceholders}})
		RETURNING {{.SelectColumns}}
	` + "`" + `
	
	var {{.ReceiverName}} {{.StructName}}
	err := r.conn.QueryRow(ctx, query, {{.InsertArgs}}).Scan({{.ScanArgs}})
	if err != nil {
		return nil, err
	}
	
	return &{{.ReceiverName}}, nil
}`

	// Update template
	updateTemplate = `// Update{{.StructName}}Params holds parameters for updating a {{.StructName}}
type Update{{.StructName}}Params struct {
{{range .UpdateFields}}	{{.Name}} {{.Type}} ` + "`{{.Tag}}`" + `
{{end}}}

// Update updates a {{.StructName}} by ID
func (r *{{.RepositoryName}}) Update(ctx context.Context, id uuid.UUID, params Update{{.StructName}}Params) (*{{.StructName}}, error) {
	query := ` + "`" + `
		UPDATE {{.TableName}}
		SET {{.UpdateAssignments}}
		WHERE {{.IDColumn}} = ${{.IDParamIndex}}
		RETURNING {{.SelectColumns}}
	` + "`" + `
	
	var {{.ReceiverName}} {{.StructName}}
	err := r.conn.QueryRow(ctx, query, {{.UpdateArgs}}).Scan({{.ScanArgs}})
	if err != nil {
		return nil, err
	}
	
	return &{{.ReceiverName}}, nil
}`

	// Delete template
	deleteTemplate = `// Delete deletes a {{.StructName}} by ID
func (r *{{.RepositoryName}}) Delete(ctx context.Context, id uuid.UUID) error {
	query := ` + "`" + `
		DELETE FROM {{.TableName}}
		WHERE {{.IDColumn}} = $1
	` + "`" + `
	
	_, err := r.conn.Exec(ctx, query, id)
	return err
}`

	// List template (simple non-paginated version)
	listTemplate = `// List retrieves all {{.StructName}}s
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
)
