//go:build integration

package generator

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nhalm/pgxkit/v2"
)

// auditedSQLRE matches the backtick-quoted query literal in a rendered audited
// Create/Update method body. Anchoring on `query := ` skips per-field struct
// tags and matches only the multi-line CTE SQL.
var auditedSQLRE = regexp.MustCompile("(?s)query := `([^`]+)`")

func renderAuditedSQL(t *testing.T, table table, templateName string) string {
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
	tables, err := introspector.getTablesByName(ctx, []string{"users"})
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
	requireOpenAuditEmail(ctx, t, db, id, originalEmail)
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
	var openEmail string
	var openVersion int
	if err := db.QueryRow(ctx,
		"SELECT valid_from, snapshot->>'email', version FROM users_audit WHERE parent_id = $1 AND valid_to IS NULL", id,
	).Scan(&openStart, &openEmail, &openVersion); err != nil {
		t.Fatalf("read open audit row: %v", err)
	}

	var closedEnd time.Time
	var closedEmail string
	var closedVersion int
	if err := db.QueryRow(ctx,
		"SELECT valid_to, snapshot->>'email', version FROM users_audit WHERE parent_id = $1 AND valid_to IS NOT NULL", id,
	).Scan(&closedEnd, &closedEmail, &closedVersion); err != nil {
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

	if openEmail != updatedEmail {
		t.Fatalf("open audit snapshot.email: want %q, got %q", updatedEmail, openEmail)
	}
	if closedEmail != originalEmail {
		t.Fatalf("closed audit snapshot.email: want %q, got %q", originalEmail, closedEmail)
	}

	// Second consecutive UPDATE. Catches a future MAX(version) regression
	// that broadens the WHERE clause and pulls MAX across all parents instead
	// of scoping to this parent_id only.
	secondEmail := "audit-update2-" + id.String() + "@example.com"
	updateArgs2 := buildUpdateArgs(t, usersTable, id, secondEmail, "Audit Test User")
	rows, err = db.Query(ctx, updateSQL, updateArgs2...)
	if err != nil {
		t.Fatalf("second audited UPDATE failed: %v", err)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		t.Fatalf("second audited UPDATE rows err: %v", err)
	}

	requireAuditCount(ctx, t, db, id, 3)
	requireOpenAuditCount(ctx, t, db, id, 1)

	var openStart2 time.Time
	var openEmail2 string
	var openVersion2 int
	if err := db.QueryRow(ctx,
		"SELECT valid_from, snapshot->>'email', version FROM users_audit WHERE parent_id = $1 AND valid_to IS NULL", id,
	).Scan(&openStart2, &openEmail2, &openVersion2); err != nil {
		t.Fatalf("read open audit row after second UPDATE: %v", err)
	}
	if openVersion2 != 3 {
		t.Fatalf("open audit version after second UPDATE: want 3, got %d", openVersion2)
	}
	if openEmail2 != secondEmail {
		t.Fatalf("open audit snapshot.email after second UPDATE: want %q, got %q", secondEmail, openEmail2)
	}

	// Most-recently-closed row is the one we just closed (version = 2).
	// Its valid_to must equal the new open row's valid_from.
	var recentClosedEnd time.Time
	var recentClosedVersion int
	if err := db.QueryRow(ctx,
		`SELECT valid_to, version FROM users_audit
		 WHERE parent_id = $1 AND valid_to IS NOT NULL
		 ORDER BY valid_to DESC LIMIT 1`, id,
	).Scan(&recentClosedEnd, &recentClosedVersion); err != nil {
		t.Fatalf("read most-recently-closed audit row: %v", err)
	}
	if recentClosedVersion != 2 {
		t.Fatalf("most-recently-closed audit version: want 2, got %d", recentClosedVersion)
	}
	if !recentClosedEnd.Equal(openStart2) {
		t.Fatalf("second update timestamp misalignment: closed.valid_to=%s open.valid_from=%s",
			recentClosedEnd.Format(time.RFC3339Nano), openStart2.Format(time.RFC3339Nano))
	}

	var closedCount int
	if err := db.QueryRow(ctx,
		"SELECT count(*) FROM users_audit WHERE parent_id = $1 AND valid_to IS NOT NULL", id,
	).Scan(&closedCount); err != nil {
		t.Fatalf("count closed audit rows: %v", err)
	}
	if closedCount != 2 {
		t.Fatalf("closed audit row count after second UPDATE: want 2, got %d", closedCount)
	}
}

// buildCreateArgs returns positional args matching prepareCRUDTemplateData's
// layout for audited CREATE: id, then every non-id column in declaration
// order, then the audit row id appended last.
//
// NOTE: this helper reimplements prepareCRUDTemplateData's positional-arg
// ordering. If templates ever reorder columns, this helper silently binds
// wrong values and the test still passes. Arg-ordering correctness is
// covered end-to-end by example-app's curl flow against the real generated
// Create method, not by this in-process render-and-execute test.
func buildCreateArgs(t *testing.T, table table, id uuid.UUID, name, email, passwordHash string) []any {
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
			args = append(args, email)
		case "password_hash":
			args = append(args, passwordHash)
		default:
			if !col.IsNullable {
				t.Fatalf("buildCreateArgs has no value for NOT NULL column %q; users fixture changed?", col.Name)
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
//
// NOTE: this helper reimplements prepareCRUDTemplateData's positional-arg
// ordering. If templates ever reorder columns, this helper silently binds
// wrong values and the test still passes. Arg-ordering correctness is
// covered end-to-end by example-app's curl flow against the real generated
// Update method, not by this in-process render-and-execute test.
func buildUpdateArgs(t *testing.T, table table, id uuid.UUID, updatedEmail, name string) []any {
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

func runAuditScenario(t *testing.T, db *pgxkit.DB, usersTable table, id uuid.UUID,
	createSQL, updateSQL, createEmail, updateEmail, secondUpdateEmail string,
) {
	t.Helper()
	ctx := context.Background()

	rows, err := db.Query(ctx, createSQL, buildCreateArgs(t, usersTable, id, "Audit Test User", createEmail, "hash-placeholder")...)
	if err != nil {
		t.Fatalf("audited CREATE failed: %v", err)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		t.Fatalf("audited CREATE rows err: %v", err)
	}

	rows, err = db.Query(ctx, updateSQL, buildUpdateArgs(t, usersTable, id, updateEmail, "Audit Test User")...)
	if err != nil {
		t.Fatalf("audited UPDATE failed: %v", err)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		t.Fatalf("audited UPDATE rows err: %v", err)
	}

	rows, err = db.Query(ctx, updateSQL, buildUpdateArgs(t, usersTable, id, secondUpdateEmail, "Audit Test User")...)
	if err != nil {
		t.Fatalf("second audited UPDATE failed: %v", err)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		t.Fatalf("second audited UPDATE rows err: %v", err)
	}
}

func loadAuditTemplates(t *testing.T, testDB *pgxkit.TestDB) (table, string, string) {
	t.Helper()

	introspector := NewIntrospector(testDB.DB, "public")
	tables, err := introspector.getTablesByName(context.Background(), []string{"users"})
	if err != nil {
		t.Fatalf("introspect users: %v", err)
	}
	usersTable, ok := tables["users"]
	if !ok {
		t.Fatalf("users table not found in introspection result")
	}
	usersTable.Audit = true

	return usersTable,
		renderAuditedSQL(t, usersTable, TemplateCreateAudited),
		renderAuditedSQL(t, usersTable, TemplateUpdateAudited)
}

func preCleanAuditFixture(t *testing.T, db *pgxkit.DB, emails []string) {
	t.Helper()
	ctx := context.Background()
	if _, err := db.Exec(ctx,
		`DELETE FROM users_audit
		 WHERE parent_id IN (SELECT id FROM users WHERE email = ANY($1))`, emails); err != nil {
		t.Fatalf("pre-clean users_audit: %v", err)
	}
	if _, err := db.Exec(ctx, `DELETE FROM users WHERE email = ANY($1)`, emails); err != nil {
		t.Fatalf("pre-clean users: %v", err)
	}
}

func TestAuditCTE_Golden(t *testing.T) {
	testDB := pgxkit.RequireDB(t)
	if err := testDB.Setup(); err != nil {
		t.Fatalf("test db setup: %v", err)
	}

	usersTable, createSQL, updateSQL := loadAuditTemplates(t, testDB)

	const (
		createEmail       = "audit-cte-golden-create@example.com"
		updateEmail       = "audit-cte-golden-update@example.com"
		secondUpdateEmail = "audit-cte-golden-update2@example.com"
	)
	emails := []string{createEmail, updateEmail, secondUpdateEmail}
	preCleanAuditFixture(t, testDB.DB, emails)
	t.Cleanup(func() { preCleanAuditFixture(t, testDB.DB, emails) })

	golden := testDB.EnableGolden("TestAuditCTE_Golden")
	runAuditScenario(t, golden, usersTable, uuid.New(), createSQL, updateSQL, createEmail, updateEmail, secondUpdateEmail)
	golden.AssertGolden(t, "TestAuditCTE_Golden")
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

func requireOpenAuditEmail(ctx context.Context, t *testing.T, db *pgxkit.DB, id uuid.UUID, want string) {
	t.Helper()
	var got string
	if err := db.QueryRow(ctx,
		"SELECT snapshot->>'email' FROM users_audit WHERE parent_id = $1 AND valid_to IS NULL", id,
	).Scan(&got); err != nil {
		t.Fatalf("read open audit email: %v", err)
	}
	if got != want {
		t.Fatalf("open audit email: want %q, got %q", want, got)
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
