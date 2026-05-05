package generator

import (
	"strings"
	"testing"
)

func auditFixture(parentName string) table {
	auditName := parentName + "_audit"
	return table{
		Name:   auditName,
		Schema: "public",
		Columns: []column{
			{Name: "id", Type: "uuid", IsNullable: false},
			{Name: "parent_id", Type: "uuid", IsNullable: false},
			{Name: "version", Type: "integer", IsNullable: false},
			{Name: "snapshot", Type: "jsonb", IsNullable: false},
			{Name: "valid_from", Type: "timestamptz", IsNullable: false},
			{Name: "valid_to", Type: "timestamptz", IsNullable: true},
		},
		PrimaryKey: []string{"id"},
		Indexes: []index{
			{Name: "idx_" + auditName + "_parent", Columns: []string{"parent_id"}},
			{Name: "uq_" + auditName + "_parent_version", Columns: []string{"parent_id", "version"}, IsUnique: true},
		},
		ForeignKeys: []foreignKey{{
			ConstraintName:   auditName + "_parent_fk",
			ColumnName:       "parent_id",
			ReferencedTable:  parentName,
			ReferencedColumn: "id",
		}},
	}
}

func parentFixture(name string) table {
	return table{
		Name:       name,
		Schema:     "public",
		Columns:    []column{{Name: "id", Type: "uuid", IsNullable: false}},
		PrimaryKey: []string{"id"},
		Audit:      true,
	}
}

func TestValidateAuditTables_OK(t *testing.T) {
	parents := map[string]table{"users": parentFixture("users")}
	audits := map[string]table{"users_audit": auditFixture("users")}
	if err := ValidateAuditTables(parents, audits); err != nil {
		t.Fatalf("expected nil error; got %v", err)
	}
}

func TestValidateAuditTables_Errors(t *testing.T) {
	cases := []struct {
		name         string
		parents      map[string]table
		audits       map[string]table
		wantContains []string
	}{
		{
			name:         "missing audit table",
			parents:      map[string]table{"posts": parentFixture("posts")},
			audits:       map[string]table{},
			wantContains: []string{"posts_audit not found"},
		},
		{
			name: "aggregates two parents",
			// posts_audit drops indexes AND foreign keys (covers missing index +
			// missing FK legs). users_audit retains FK/indexes but mistypes
			// the snapshot column (covers type-mismatch leg).
			parents: map[string]table{
				"posts": parentFixture("posts"),
				"users": parentFixture("users"),
			},
			audits: func() map[string]table {
				posts := auditFixture("posts")
				posts.Indexes = nil
				posts.ForeignKeys = nil
				users := auditFixture("users")
				for i := range users.Columns {
					if users.Columns[i].Name == "snapshot" {
						users.Columns[i].Type = "text"
					}
				}
				return map[string]table{"posts_audit": posts, "users_audit": users}
			}(),
			wantContains: []string{
				"posts_audit missing index on (parent_id)",
				"posts_audit missing UNIQUE index on (parent_id, version)",
				"missing foreign key",
				`users_audit column "snapshot" type mismatch`,
				"CREATE TABLE posts_audit",
				"CREATE TABLE users_audit",
				"CREATE UNIQUE INDEX",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateAuditTables(tc.parents, tc.audits)
			if err == nil {
				t.Fatal("expected error; got nil")
			}
			got := err.Error()
			for _, want := range tc.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("expected error to contain %q; got:\n%s", want, got)
				}
			}
		})
	}
}
