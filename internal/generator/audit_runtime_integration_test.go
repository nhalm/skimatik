//go:build !short

package generator

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nhalm/pgxkit/v2"
)

// auditedSQLRE matches the backtick-quoted query literal in a rendered audited
// Create/Update method body. Anchoring on `query := ` skips per-field struct
// tags and matches only the multi-line CTE SQL.
var auditedSQLRE = regexp.MustCompile("(?s)query := `([^`]+)`")

func renderAuditedSQL(t *testing.T, table Table, templateName string) string {
	t.Helper()

	cg := NewCodeGenerator(getTestConfig(), "test")
	if err := cg.typeMapper.MapTableColumns(&table); err != nil {
		t.Fatalf("type-map columns: %v", err)
	}

	body, err := cg.templateMgr.ExecuteTemplate(templateName, cg.prepareCRUDTemplateData(table))
	if err != nil {
		t.Fatalf("execute template %s: %v", templateName, err)
	}

	matches := auditedSQLRE.FindStringSubmatch(body)
	if len(matches) < 2 {
		t.Fatalf("could not find SQL literal in rendered template %s; body was:\n%s", templateName, body)
	}
	return matches[1]
}

// TestAuditCTE_CreateAndUpdate executes the SQL rendered by the audited
// Create/Update templates against a live Postgres and asserts audit
// invariants: row counts, valid_from/valid_to alignment, monotonic version
// numbering, and JSONB pre/post images.
func TestAuditCTE_CreateAndUpdate(t *testing.T) {
	testDB := pgxkit.RequireDB(t)
	if err := testDB.Setup(); err != nil {
		t.Fatalf("test db setup: %v", err)
	}

	db := testDB.DB
	ctx := context.Background()

	introspector := NewIntrospector(testDB.DB, "public")
	tables, err := introspector.GetTablesByName(ctx, []string{"users"})
	if err != nil {
		t.Fatalf("introspect users: %v", err)
	}
	usersTable, ok := tables["users"]
	if !ok {
		t.Fatalf("users table not found in introspection result")
	}
	usersTable.Audit = true

	createSQL := renderAuditedSQL(t, usersTable, TemplateCreateAudited)
	updateSQL := renderAuditedSQL(t, usersTable, TemplateUpdateAudited)

	for _, snippet := range []string{
		"WITH inserted AS",
		"INSERT INTO users_audit",
		"to_jsonb(inserted.*)",
		"version, snapshot, valid_from",
	} {
		if !strings.Contains(createSQL, snippet) {
			t.Fatalf("rendered CREATE SQL missing %q:\n%s", snippet, createSQL)
		}
	}
	for _, snippet := range []string{
		"WITH closed AS",
		"UPDATE users_audit",
		"SET valid_to = NOW()",
		"to_jsonb(updated.*)",
		"MAX(version)",
	} {
		if !strings.Contains(updateSQL, snippet) {
			t.Fatalf("rendered UPDATE SQL missing %q:\n%s", snippet, updateSQL)
		}
	}

	// Audit row IDs are now app-generated; no database-side UUID function
	// should appear in either rendered statement.
	if strings.Contains(createSQL, "gen_random_uuid()") {
		t.Fatalf("rendered CREATE SQL still references gen_random_uuid():\n%s", createSQL)
	}
	if strings.Contains(updateSQL, "gen_random_uuid()") {
		t.Fatalf("rendered UPDATE SQL still references gen_random_uuid():\n%s", updateSQL)
	}

	id := uuid.New()
	t.Cleanup(func() {
		_, _ = db.Exec(ctx, "DELETE FROM users_audit WHERE parent_id = $1", id)
		_, _ = db.Exec(ctx, "DELETE FROM users WHERE id = $1", id)
	})

	originalEmail := "audit-create-" + id.String() + "@example.com"

	// The rendered SQL ends with `SELECT ... FROM inserted`, so we must Query
	// (not Exec) to consume the returned row and let the audit CTE actually run.
	createArgs := buildCreateArgs(t, usersTable, id, "Audit Test User", originalEmail, "hash-placeholder")
	rows, err := db.Query(ctx, createSQL, createArgs...)
	if err != nil {
		t.Fatalf("audited CREATE failed: %v", err)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		t.Fatalf("audited CREATE rows err: %v", err)
	}

	requireAuditCount(ctx, t, db, id, 1)
	requireOpenAuditCount(ctx, t, db, id, 1)
	requireOpenAuditDataContains(ctx, t, db, id, originalEmail)
	requireOpenAuditVersion(ctx, t, db, id, 1)

	updatedEmail := "audit-update-" + id.String() + "@example.com"
	updateArgs := buildUpdateArgs(t, usersTable, id, updatedEmail, "Audit Test User")
	rows, err = db.Query(ctx, updateSQL, updateArgs...)
	if err != nil {
		t.Fatalf("audited UPDATE failed: %v", err)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		t.Fatalf("audited UPDATE rows err: %v", err)
	}

	// Closed.valid_to and open.valid_from must align exactly: both are set
	// from a single statement-level NOW() in the Update CTE.
	requireAuditCount(ctx, t, db, id, 2)

	var openStart time.Time
	var openData string
	var openVersion int
	if err := db.QueryRow(ctx,
		"SELECT valid_from, snapshot::text, version FROM users_audit WHERE parent_id = $1 AND valid_to IS NULL", id,
	).Scan(&openStart, &openData, &openVersion); err != nil {
		t.Fatalf("read open audit row: %v", err)
	}

	var closedEnd time.Time
	var closedData string
	var closedVersion int
	if err := db.QueryRow(ctx,
		"SELECT valid_to, snapshot::text, version FROM users_audit WHERE parent_id = $1 AND valid_to IS NOT NULL", id,
	).Scan(&closedEnd, &closedData, &closedVersion); err != nil {
		t.Fatalf("read closed audit row: %v", err)
	}

	if !closedEnd.Equal(openStart) {
		t.Fatalf("audit timestamp misalignment: closed.valid_to=%s open.valid_from=%s",
			closedEnd.Format(time.RFC3339Nano), openStart.Format(time.RFC3339Nano))
	}

	if closedVersion != 1 {
		t.Fatalf("closed audit version: want 1, got %d", closedVersion)
	}
	if openVersion != 2 {
		t.Fatalf("open audit version: want 2, got %d", openVersion)
	}
	if openVersion <= closedVersion {
		t.Fatalf("audit versions not monotonically increasing: closed=%d open=%d", closedVersion, openVersion)
	}

	if !strings.Contains(openData, updatedEmail) {
		t.Fatalf("open audit JSONB missing updated email %q: %s", updatedEmail, openData)
	}
	if !strings.Contains(closedData, originalEmail) {
		t.Fatalf("closed audit JSONB missing original email %q: %s", originalEmail, closedData)
	}
	if strings.Contains(closedData, updatedEmail) {
		t.Fatalf("closed audit JSONB unexpectedly contains updated email %q: %s", updatedEmail, closedData)
	}
}

// buildCreateArgs returns positional args matching prepareCRUDTemplateData's
// layout for audited CREATE: id, then every non-id column whose DefaultValue
// is empty, in declaration order, then the audit row id appended last.
func buildCreateArgs(t *testing.T, table Table, id uuid.UUID, name, email, passwordHash string) []any {
	t.Helper()
	args := []any{id}
	for _, col := range table.Columns {
		if col.Name == "id" || col.DefaultValue != "" {
			continue
		}
		switch col.Name {
		case "name":
			args = append(args, name)
		case "email":
			args = append(args, email)
		case "password_hash":
			args = append(args, passwordHash)
		default:
			if !col.IsNullable {
				t.Fatalf("buildCreateArgs has no value for NOT NULL column %q without default; users fixture changed?", col.Name)
			}
			args = append(args, nil)
		}
	}
	// Audit row id is generated app-side; uuid.New() is sufficient for the
	// runtime test since we only assert audit invariants, not id ordering.
	args = append(args, uuid.New())
	return args
}

// buildUpdateArgs returns positional args matching prepareCRUDTemplateData's
// layout for audited UPDATE: id at $1, every non-id column at $2..$N, then
// the audit row id appended last.
func buildUpdateArgs(t *testing.T, table Table, id uuid.UUID, updatedEmail, name string) []any {
	t.Helper()
	args := []any{id}
	for _, col := range table.Columns {
		if col.Name == "id" {
			continue
		}
		switch col.Name {
		case "name":
			args = append(args, name)
		case "email":
			args = append(args, updatedEmail)
		case "password_hash":
			args = append(args, "hash-placeholder")
		default:
			args = append(args, nil)
		}
	}
	args = append(args, uuid.New())
	return args
}

func requireAuditCount(ctx context.Context, t *testing.T, db *pgxkit.DB, id uuid.UUID, want int) {
	t.Helper()
	var got int
	if err := db.QueryRow(ctx,
		"SELECT count(*) FROM users_audit WHERE parent_id = $1", id,
	).Scan(&got); err != nil {
		t.Fatalf("count audit rows: %v", err)
	}
	if got != want {
		t.Fatalf("audit row count: want %d, got %d", want, got)
	}
}

func requireOpenAuditCount(ctx context.Context, t *testing.T, db *pgxkit.DB, id uuid.UUID, want int) {
	t.Helper()
	var got int
	if err := db.QueryRow(ctx,
		"SELECT count(*) FROM users_audit WHERE parent_id = $1 AND valid_to IS NULL", id,
	).Scan(&got); err != nil {
		t.Fatalf("count open audit rows: %v", err)
	}
	if got != want {
		t.Fatalf("open audit row count: want %d, got %d", want, got)
	}
}

func requireOpenAuditDataContains(ctx context.Context, t *testing.T, db *pgxkit.DB, id uuid.UUID, want string) {
	t.Helper()
	var data string
	if err := db.QueryRow(ctx,
		"SELECT snapshot::text FROM users_audit WHERE parent_id = $1 AND valid_to IS NULL", id,
	).Scan(&data); err != nil {
		t.Fatalf("read open audit data: %v", err)
	}
	if !strings.Contains(data, want) {
		t.Fatalf("open audit JSONB missing %q: %s", want, data)
	}
}

func requireOpenAuditVersion(ctx context.Context, t *testing.T, db *pgxkit.DB, id uuid.UUID, want int) {
	t.Helper()
	var got int
	if err := db.QueryRow(ctx,
		"SELECT version FROM users_audit WHERE parent_id = $1 AND valid_to IS NULL", id,
	).Scan(&got); err != nil {
		t.Fatalf("read open audit version: %v", err)
	}
	if got != want {
		t.Fatalf("open audit version: want %d, got %d", want, got)
	}
}
