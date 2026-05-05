package generator

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// ValidateAuditTables enforces the audit pre-flight contract documented in
// issue #144: every parent in `parents` flagged with Audit==true must have a
// sibling `<parent>_audit` table in `audits` with the canonical shape:
//
//	id          UUID PRIMARY KEY
//	parent_id   <parent-PK-type> NOT NULL  REFERENCES <parent>(<pk>)
//	data        JSONB NOT NULL
//	start_date  TIMESTAMPTZ NOT NULL
//	end_date    TIMESTAMPTZ NULL
//	+ an index leading with parent_id
//
// The function is permissive about extra columns — only the five required
// ones are validated. All errors across all audited tables are aggregated and
// returned via errors.Join so the user can fix everything in one pass; each
// affected audit table's expected DDL is appended once at the end of the
// joined message so users can copy-paste a working schema.
//
// Parents are processed in lexical order so the joined error message is
// stable; this matters for golden tests and for users diffing CI output.
//
// The function performs no I/O. All metadata is supplied by the caller (this
// keeps the validator unit-testable with constructed Table fixtures).
func ValidateAuditTables(parents, audits map[string]Table) error {
	// Stable iteration order — required for deterministic error output.
	parentNames := make([]string, 0, len(parents))
	for name := range parents {
		t := parents[name]
		if !t.Audit {
			continue
		}
		parentNames = append(parentNames, name)
	}
	sort.Strings(parentNames)

	var errs []error
	// affectedParents tracks parents whose audit table had at least one
	// failure, in iteration (sorted) order, so we can append the canonical
	// DDL block at the end of the joined message — once per audit table.
	var affectedParents []string

	for _, parentName := range parentNames {
		parent := parents[parentName]
		parentErrs := validateAuditTableForParent(parent, audits)
		if len(parentErrs) > 0 {
			errs = append(errs, parentErrs...)
			affectedParents = append(affectedParents, parentName)
		}
	}

	// Append one DDL block per affected parent so users can copy-paste.
	for _, parentName := range affectedParents {
		parent := parents[parentName]
		errs = append(errs, fmt.Errorf(
			"expected schema for %s_audit:\n%s",
			parentName, expectedAuditDDL(parent),
		))
	}

	return errors.Join(errs...)
}

// validateAuditTableForParent validates the audit child for a single audited
// parent and returns the collected error list. Pulled out of
// ValidateAuditTables so the top-level function stays under the gocyclo
// threshold and so the per-parent flow is easier to reason about in
// isolation.
func validateAuditTableForParent(parent Table, audits map[string]Table) []error {
	auditName := parent.Name + "_audit"

	// Parent must have a single-column PK so we can describe
	// `parent_id` precisely. Validators upstream catch composite PKs
	// separately; here we degrade gracefully so the user sees the
	// audit-shape gap instead of a silent skip.
	pkCol := parent.GetPrimaryKeyColumn()
	if pkCol == nil {
		return []error{fmt.Errorf(
			"audit: %s has no single-column primary key; cannot validate %s",
			parent.Name, auditName,
		)}
	}

	audit, ok := audits[auditName]
	if !ok {
		return []error{fmt.Errorf("audit: %s not found", auditName)}
	}

	// Five required columns + up to two FK/index issues = 7 worst-case
	// failures. Preallocate to silence prealloc and avoid repeated
	// growth in the (common) multi-failure path.
	errs := make([]error, 0, 7)
	errs = append(errs, validateAuditColumns(audit, auditName, pkCol.Type)...)
	errs = append(errs, validateAuditFKAndIndex(audit, auditName, parent.Name, pkCol.Name)...)
	return errs
}

// auditColumnSpec describes a required column on an audit child table.
type auditColumnSpec struct {
	name       string
	pgType     string
	nullable   bool
	primaryKey bool
}

// validateAuditColumns checks the five canonical audit columns. parentPKType
// is the PostgreSQL type of the parent table's primary key column, which
// drives the expected type of `parent_id`.
func validateAuditColumns(audit Table, auditName, parentPKType string) []error {
	required := []auditColumnSpec{
		{"id", "uuid", false, true},
		{"parent_id", parentPKType, false, false},
		{"data", "jsonb", false, false},
		{"start_date", "timestamptz", false, false},
		{"end_date", "timestamptz", true, false},
	}

	errs := make([]error, 0, len(required))
	for _, req := range required {
		errs = append(errs, validateAuditColumn(audit, auditName, req)...)
	}
	return errs
}

func validateAuditColumn(audit Table, auditName string, req auditColumnSpec) []error {
	col := audit.GetColumn(req.name)
	if col == nil {
		return []error{fmt.Errorf(
			"audit: %s missing column %q (expected %s %s)",
			auditName, req.name, req.pgType, nullabilityWord(req.nullable),
		)}
	}

	var errs []error
	if !pgTypeEquivalent(col.Type, req.pgType) {
		errs = append(errs, fmt.Errorf(
			"audit: %s column %q type mismatch (have %s, expected %s)",
			auditName, req.name, col.Type, req.pgType,
		))
	}
	if col.IsNullable != req.nullable {
		errs = append(errs, fmt.Errorf(
			"audit: %s column %q nullability mismatch (have %s, expected %s)",
			auditName, req.name,
			nullabilityWord(col.IsNullable), nullabilityWord(req.nullable),
		))
	}
	if req.primaryKey && !columnIsPrimaryKey(audit, req.name) {
		errs = append(errs, fmt.Errorf(
			"audit: %s column %q must be the primary key",
			auditName, req.name,
		))
	}
	return errs
}

// validateAuditFKAndIndex enforces the FK constraint from parent_id to the
// parent's primary key column and the leading-column index requirement on
// parent_id. If parent_id is missing entirely the column-level error from
// validateAuditColumns is enough; we don't pile on with FK/index complaints
// for a column that doesn't exist.
func validateAuditFKAndIndex(audit Table, auditName, parentName, parentPKCol string) []error {
	if audit.GetColumn("parent_id") == nil {
		return nil
	}

	var errs []error
	if !audit.HasForeignKeyTo("parent_id", parentName, parentPKCol) {
		errs = append(errs, fmt.Errorf(
			"audit: %s missing foreign key on parent_id referencing %s(%s)",
			auditName, parentName, parentPKCol,
		))
	}
	if !audit.HasIndexLeadingWith("parent_id") {
		errs = append(errs, fmt.Errorf(
			"audit: %s missing index on (parent_id)",
			auditName,
		))
	}
	return errs
}

// pgTypeEquivalent reports whether two PostgreSQL type names refer to the
// same underlying type, accounting for the canonical-vs-alias forms users
// may have in their schema. The introspector normalizes timestamps to
// `timestamptz` already, but a fixture or future loader change could surface
// the long form, so we treat both as equivalent here.
//
// Comparison is case-insensitive and trims surrounding whitespace, which
// matches PostgreSQL's identifier handling for unquoted type names.
func pgTypeEquivalent(a, b string) bool {
	na := normalizePgType(a)
	nb := normalizePgType(b)
	return na == nb
}

func normalizePgType(t string) string {
	t = strings.ToLower(strings.TrimSpace(t))
	switch t {
	case "timestamp with time zone":
		return "timestamptz"
	case "timestamp without time zone":
		return "timestamp"
	case "character varying":
		return "varchar"
	}
	return t
}

func nullabilityWord(nullable bool) string {
	if nullable {
		return "NULL"
	}
	return "NOT NULL"
}

func columnIsPrimaryKey(t Table, name string) bool {
	for _, pk := range t.PrimaryKey {
		if pk == name {
			return true
		}
	}
	return false
}

// expectedAuditDDL returns the canonical DDL block for a parent table's audit
// child. Used as the copy-paste payload appended to the aggregated error
// message so users can fix their schema in one round trip.
func expectedAuditDDL(parent Table) string {
	pkCol := parent.GetPrimaryKeyColumn()
	pkType := "UUID"
	pkName := "id"
	if pkCol != nil {
		pkType = strings.ToUpper(pkCol.Type)
		pkName = pkCol.Name
	}
	auditName := parent.Name + "_audit"
	var b strings.Builder
	b.WriteString("  CREATE TABLE ")
	b.WriteString(auditName)
	b.WriteString(" (\n")
	b.WriteString("    id          UUID         PRIMARY KEY,\n")
	b.WriteString("    parent_id   ")
	b.WriteString(pkType)
	b.WriteString(" NOT NULL REFERENCES ")
	b.WriteString(parent.Name)
	b.WriteString("(")
	b.WriteString(pkName)
	b.WriteString("),\n")
	b.WriteString("    data        JSONB        NOT NULL,\n")
	b.WriteString("    start_date  TIMESTAMPTZ  NOT NULL,\n")
	b.WriteString("    end_date    TIMESTAMPTZ\n")
	b.WriteString("  );\n")
	b.WriteString("  CREATE INDEX ON ")
	b.WriteString(auditName)
	b.WriteString(" (parent_id);")
	return b.String()
}
