package generator

import (
	"strings"
	"testing"
)

func auditFixture(parentName string) Table {
	auditName := parentName + "_audit"
	return Table{
		Name:   auditName,
		Schema: "public",
		Columns: []Column{
			{Name: "id", Type: "uuid", IsNullable: false},
			{Name: "parent_id", Type: "uuid", IsNullable: false},
			{Name: "version", Type: "integer", IsNullable: false},
			{Name: "snapshot", Type: "jsonb", IsNullable: false},
			{Name: "valid_from", Type: "timestamptz", IsNullable: false},
			{Name: "valid_to", Type: "timestamptz", IsNullable: true},
		},
		PrimaryKey: []string{"id"},
		Indexes: []Index{
			{Name: "idx_" + auditName + "_parent", Columns: []string{"parent_id"}},
			{Name: "uq_" + auditName + "_parent_version", Columns: []string{"parent_id", "version"}, IsUnique: true},
		},
		ForeignKeys: []ForeignKey{{
			ConstraintName:   auditName + "_parent_fk",
			ColumnName:       "parent_id",
			ReferencedTable:  parentName,
			ReferencedColumn: "id",
		}},
	}
}

func parentFixture(name string) Table {
	return Table{
		Name:       name,
		Schema:     "public",
		Columns:    []Column{{Name: "id", Type: "uuid", IsNullable: false}},
		PrimaryKey: []string{"id"},
		Audit:      true,
	}
}

func TestValidateAuditTables_OK(t *testing.T) {
	parents := map[string]Table{"users": parentFixture("users")}
	audits := map[string]Table{"users_audit": auditFixture("users")}
	if err := ValidateAuditTables(parents, audits); err != nil {
		t.Fatalf("expected nil error; got %v", err)
	}
}

func TestValidateAuditTables_Errors(t *testing.T) {
	cases := []struct {
		name         string
		parents      map[string]Table
		audits       map[string]Table
		wantContains []string
	}{
		{
			name:         "missing audit table",
			parents:      map[string]Table{"posts": parentFixture("posts")},
			audits:       map[string]Table{},
			wantContains: []string{"posts_audit not found"},
		},
		{
			name:    "wrong column type",
			parents: map[string]Table{"posts": parentFixture("posts")},
			audits: func() map[string]Table {
				a := auditFixture("posts")
				for i := range a.Columns {
					if a.Columns[i].Name == "snapshot" {
						a.Columns[i].Type = "text"
					}
				}
				return map[string]Table{"posts_audit": a}
			}(),
			wantContains: []string{"type mismatch"},
		},
		{
			name:    "missing FK",
			parents: map[string]Table{"posts": parentFixture("posts")},
			audits: func() map[string]Table {
				a := auditFixture("posts")
				a.ForeignKeys = nil
				return map[string]Table{"posts_audit": a}
			}(),
			wantContains: []string{"missing foreign key"},
		},
		{
			name:    "missing unique index on (parent_id, version)",
			parents: map[string]Table{"posts": parentFixture("posts")},
			audits: func() map[string]Table {
				a := auditFixture("posts")
				kept := a.Indexes[:0]
				for _, idx := range a.Indexes {
					if !idx.IsUnique {
						kept = append(kept, idx)
					}
				}
				a.Indexes = kept
				return map[string]Table{"posts_audit": a}
			}(),
			wantContains: []string{"missing UNIQUE index on (parent_id, version)"},
		},
		{
			name: "aggregates two parents",
			parents: map[string]Table{
				"posts": parentFixture("posts"),
				"users": parentFixture("users"),
			},
			audits: func() map[string]Table {
				posts := auditFixture("posts")
				posts.Indexes = nil
				users := auditFixture("users")
				for i := range users.Columns {
					if users.Columns[i].Name == "snapshot" {
						users.Columns[i].Type = "text"
					}
				}
				return map[string]Table{"posts_audit": posts, "users_audit": users}
			}(),
			wantContains: []string{
				"posts_audit missing index on (parent_id)",
				"posts_audit missing UNIQUE index on (parent_id, version)",
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
