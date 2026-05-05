//go:build !short

package generator

import (
	"context"
	"testing"
)

// TestIntrospector_GetTables_ForeignKeys exercises foreign-key introspection
// against the integration test database. The init.sql fixture defines
// well-known FKs (e.g. posts.user_id -> users.id, profiles.user_id ->
// users.id) that we can assert against without setting up our own schema.
func TestIntrospector_GetTables_ForeignKeys(t *testing.T) {
	testDB := getTestDB(t)
	ctx := context.Background()

	introspector := NewIntrospector(testDB, "public")
	tables, err := introspector.GetTables(ctx)
	if err != nil {
		t.Fatalf("GetTables failed: %v", err)
	}

	byName := make(map[string]Table, len(tables))
	for _, tbl := range tables {
		byName[tbl.Name] = tbl
	}

	cases := []struct {
		table         string
		childCol      string
		referencedTbl string
		referencedCol string
		descr         string
	}{
		{"posts", "user_id", "users", "id", "posts.user_id -> users.id"},
		{"profiles", "user_id", "users", "id", "profiles.user_id -> users.id"},
		{"comments", "post_id", "posts", "id", "comments.post_id -> posts.id"},
		{"comments", "user_id", "users", "id", "comments.user_id -> users.id"},
		{"comments", "parent_id", "comments", "id", "comments.parent_id -> comments.id (self)"},
		{"post_categories", "post_id", "posts", "id", "post_categories.post_id -> posts.id"},
		{"post_categories", "category_id", "categories", "id", "post_categories.category_id -> categories.id"},
		{"files", "user_id", "users", "id", "files.user_id -> users.id"},
	}

	for _, tc := range cases {
		t.Run(tc.descr, func(t *testing.T) {
			tbl, ok := byName[tc.table]
			if !ok {
				t.Fatalf("expected table %q in introspection results", tc.table)
			}
			if !tbl.HasForeignKeyTo(tc.childCol, tc.referencedTbl, tc.referencedCol) {
				t.Errorf("expected FK %s.%s -> %s.%s; got %+v",
					tc.table, tc.childCol, tc.referencedTbl, tc.referencedCol, tbl.ForeignKeys)
			}
		})
	}

	// Tables without FKs should not surface phantom entries.
	if usersTable, ok := byName["users"]; ok {
		if len(usersTable.ForeignKeys) != 0 {
			t.Errorf("users has no foreign keys in fixture; got %+v", usersTable.ForeignKeys)
		}
	}
}

// TestIntrospector_GetTables_IndexColumnOrder asserts that the leading-column
// order of indexes is preserved end-to-end. The fixture defines composite
// indexes (e.g. idx_users_active_created on (is_active, created_at)) where
// reordering would mask audit-table validation bugs.
func TestIntrospector_GetTables_IndexColumnOrder(t *testing.T) {
	testDB := getTestDB(t)
	ctx := context.Background()

	introspector := NewIntrospector(testDB, "public")
	tables, err := introspector.GetTables(ctx)
	if err != nil {
		t.Fatalf("GetTables failed: %v", err)
	}

	byName := make(map[string]Table, len(tables))
	for _, tbl := range tables {
		byName[tbl.Name] = tbl
	}

	// idx_users_active_created is (is_active, created_at) — the leading
	// column is is_active.
	users := byName["users"]
	if !users.HasIndexLeadingWith("is_active") {
		t.Errorf("expected users to have index leading with is_active; got indexes=%+v", users.Indexes)
	}
	// created_at is the second column, so HasIndexLeadingWith must NOT match.
	if users.HasIndexLeadingWith("created_at") {
		t.Errorf("created_at is not the leading column on any users index; got indexes=%+v", users.Indexes)
	}
	// email has its own single-column index, should match.
	if !users.HasIndexLeadingWith("email") {
		t.Errorf("expected users to have index leading with email; got indexes=%+v", users.Indexes)
	}

	// idx_posts_user_status is (user_id, status); user_id is leading.
	posts := byName["posts"]
	if !posts.HasIndexLeadingWith("user_id") {
		t.Errorf("expected posts to have index leading with user_id; got indexes=%+v", posts.Indexes)
	}

	// idx_posts_published is (published_at) WHERE status = 'published' — a
	// partial index. The introspector must surface partial indexes, not drop
	// them; the WHERE predicate should be ignored for leading-column purposes.
	if !posts.HasIndexLeadingWith("published_at") {
		t.Errorf("expected posts to have partial index leading with published_at; got indexes=%+v", posts.Indexes)
	}

	// idx_comments_post_parent is (post_id, parent_id); post_id is leading.
	comments := byName["comments"]
	if !comments.HasIndexLeadingWith("post_id") {
		t.Errorf("expected comments to have index leading with post_id; got indexes=%+v", comments.Indexes)
	}
	if comments.HasIndexLeadingWith("parent_id") {
		t.Errorf("parent_id is not the leading column on any comments index; got indexes=%+v", comments.Indexes)
	}
}

// TestIntrospector_GetTablesByName exercises the sibling-of-GetTables loader
// used by the audit pre-flight gate. The fixture defines `users_audit` —
// a well-formed audit child for the users table — that is not surfaced
// through the default GetTables filter chain in audit configurations
// (because users only configure parent tables). GetTablesByName must:
//
//  1. Return a map keyed by table name containing real schema metadata for
//     `users_audit` (columns, PK, FK to users.id, leading-column index on
//     parent_id).
//  2. Silently omit names that don't exist in the schema rather than error.
//  3. Short-circuit on empty input.
func TestIntrospector_GetTablesByName(t *testing.T) {
	testDB := getTestDB(t)
	ctx := context.Background()

	introspector := NewIntrospector(testDB, "public")

	// Empty input: no DB hit, empty map.
	empty, err := introspector.GetTablesByName(ctx, nil)
	if err != nil {
		t.Fatalf("GetTablesByName(nil) failed: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("expected empty map for nil input; got %+v", empty)
	}

	// Mix of real and made-up names. Only the real one comes back.
	got, err := introspector.GetTablesByName(ctx, []string{"users_audit", "definitely_not_a_real_table"})
	if err != nil {
		t.Fatalf("GetTablesByName failed: %v", err)
	}

	if _, ok := got["definitely_not_a_real_table"]; ok {
		t.Errorf("missing tables should be absent from the map; got %+v", got)
	}

	tbl, ok := got["users_audit"]
	if !ok {
		t.Fatalf("expected users_audit in result map; got keys %v", mapKeys(got))
	}

	// Sanity-check the introspected metadata. We don't validate audit
	// shape here — that's the validator's job in audit_validator_test.go.
	if len(tbl.PrimaryKey) != 1 || tbl.PrimaryKey[0] != "id" {
		t.Errorf("expected users_audit PK = [id]; got %v", tbl.PrimaryKey)
	}
	if !tbl.HasForeignKeyTo("parent_id", "users", "id") {
		t.Errorf("expected users_audit.parent_id -> users.id FK; got %+v", tbl.ForeignKeys)
	}
	if !tbl.HasIndexLeadingWith("parent_id") {
		t.Errorf("expected users_audit to have index leading with parent_id; got %+v", tbl.Indexes)
	}

	// Required columns should all be present.
	for _, name := range []string{"id", "parent_id", "data", "start_date", "end_date"} {
		if tbl.GetColumn(name) == nil {
			t.Errorf("expected users_audit column %q; got %+v", name, tbl.Columns)
		}
	}
}

func mapKeys(m map[string]Table) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// TestIntrospector_GetTables_CompositeForeignKey exercises the
// `position_in_unique_constraint` ordinal-alignment subquery in
// getAllTableForeignKeys. The init.sql fixture defines:
//
//	composite_uk_parent (id PK, UNIQUE (tenant_id, code))
//	composite_fk_child  (FK (tenant_id, parent_code) -> composite_uk_parent (tenant_id, code))
//
// We assert that both FK rows come back, share a single ConstraintName, and
// pair the correct (child_col, parent_col) ordinals — i.e. tenant_id maps to
// tenant_id and parent_code maps to code, not the other way around.
func TestIntrospector_GetTables_CompositeForeignKey(t *testing.T) {
	testDB := getTestDB(t)
	ctx := context.Background()

	introspector := NewIntrospector(testDB, "public")
	tables, err := introspector.GetTables(ctx)
	if err != nil {
		t.Fatalf("GetTables failed: %v", err)
	}

	var child Table
	for _, tbl := range tables {
		if tbl.Name == "composite_fk_child" {
			child = tbl
			break
		}
	}
	if child.Name == "" {
		t.Fatalf("expected composite_fk_child in introspection results")
	}

	// Collect FK rows belonging to the composite constraint.
	const wantConstraint = "composite_fk_child_parent_fkey"
	var rows []ForeignKey
	for _, fk := range child.ForeignKeys {
		if fk.ConstraintName == wantConstraint {
			rows = append(rows, fk)
		}
	}

	if len(rows) != 2 {
		t.Fatalf("expected 2 FK rows sharing ConstraintName=%q; got %d (all FKs=%+v)",
			wantConstraint, len(rows), child.ForeignKeys)
	}

	// Build (child_col -> parent_col) map and check alignment.
	pairs := make(map[string]string, len(rows))
	for _, fk := range rows {
		if fk.ReferencedTable != "composite_uk_parent" {
			t.Errorf("unexpected ReferencedTable for composite FK row: got %q", fk.ReferencedTable)
		}
		pairs[fk.ColumnName] = fk.ReferencedColumn
	}

	if got, want := pairs["tenant_id"], "tenant_id"; got != want {
		t.Errorf("tenant_id should map to %q; got %q (full pairs=%+v)", want, got, pairs)
	}
	if got, want := pairs["parent_code"], "code"; got != want {
		t.Errorf("parent_code should map to %q; got %q (full pairs=%+v)", want, got, pairs)
	}

	// HasForeignKeyTo should match each individual column independently.
	if !child.HasForeignKeyTo("tenant_id", "composite_uk_parent", "tenant_id") {
		t.Errorf("HasForeignKeyTo failed for tenant_id leg of composite FK")
	}
	if !child.HasForeignKeyTo("parent_code", "composite_uk_parent", "code") {
		t.Errorf("HasForeignKeyTo failed for parent_code leg of composite FK")
	}
}
