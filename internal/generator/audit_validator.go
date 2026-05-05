package generator

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// ValidateAuditTables checks every Audit==true parent in `parents` against its
// sibling `<parent>_audit` in `audits`, returning all failures via errors.Join
// with a copy-pasteable CREATE TABLE DDL block appended once per affected
// parent. Returns nil if all audited parents have a well-formed audit child.
// Performs no I/O.
func ValidateAuditTables(parents, audits map[string]Table) error {
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
	var affectedParents []string

	for _, parentName := range parentNames {
		parent := parents[parentName]
		parentErrs := validateAuditTableForParent(parent, audits)
		if len(parentErrs) > 0 {
			errs = append(errs, parentErrs...)
			affectedParents = append(affectedParents, parentName)
		}
	}

	for _, parentName := range affectedParents {
		parent := parents[parentName]
		errs = append(errs, fmt.Errorf(
			"expected schema for %s_audit:\n%s",
			parentName, expectedAuditDDL(parent),
		))
	}

	return errors.Join(errs...)
}

func validateAuditTableForParent(parent Table, audits map[string]Table) []error {
	auditName := parent.Name + "_audit"

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

	errs := make([]error, 0, 9)
	errs = append(errs, validateAuditColumns(audit, auditName, pkCol.Type)...)
	errs = append(errs, validateAuditFKAndIndex(audit, auditName, parent.Name, pkCol.Name)...)
	return errs
}

type auditColumnSpec struct {
	name       string
	pgType     string
	nullable   bool
	primaryKey bool
}

func validateAuditColumns(audit Table, auditName, parentPKType string) []error {
	required := []auditColumnSpec{
		{"id", "uuid", false, true},
		{"parent_id", parentPKType, false, false},
		{"version", "integer", false, false},
		{"snapshot", "jsonb", false, false},
		{"valid_from", "timestamptz", false, false},
		{"valid_to", "timestamptz", true, false},
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
	if !hasUniqueIndexOn(audit, "parent_id", "version") {
		errs = append(errs, fmt.Errorf(
			"audit: %s missing UNIQUE index on (parent_id, version)",
			auditName,
		))
	}
	return errs
}

func hasUniqueIndexOn(t Table, cols ...string) bool {
	for i := range t.Indexes {
		idx := &t.Indexes[i]
		if !idx.IsUnique || len(idx.Columns) != len(cols) {
			continue
		}
		match := true
		for j, c := range cols {
			if idx.Columns[j] != c {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

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
	b.WriteString("    version     INTEGER      NOT NULL,\n")
	b.WriteString("    snapshot    JSONB        NOT NULL,\n")
	b.WriteString("    valid_from  TIMESTAMPTZ  NOT NULL,\n")
	b.WriteString("    valid_to    TIMESTAMPTZ\n")
	b.WriteString("  );\n")
	b.WriteString("  CREATE INDEX ON ")
	b.WriteString(auditName)
	b.WriteString(" (parent_id);\n")
	b.WriteString("  CREATE UNIQUE INDEX ON ")
	b.WriteString(auditName)
	b.WriteString(" (parent_id, version);")
	return b.String()
}
