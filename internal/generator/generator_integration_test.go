//go:build !short

package generator

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGenerator_AuditPreflightGate_AbortsOnMissingAuditChild exercises the
// audit pre-flight gate wired into generateTablesStage. It configures a real
// parent table (`posts`) with `audit: true` against a fixture schema that
// intentionally does NOT contain a `posts_audit` sibling. The test asserts:
//
//  1. Generate() returns a non-nil error.
//  2. The error message names the missing audit table and includes the
//     canonical CREATE TABLE DDL block produced by ValidateAuditTables — the
//     copy-pasteable schema users need to fix their database.
//  3. The output directory is empty. The gate's contract is "no file is
//     written if validation fails" — neither shared utilities nor per-table
//     generated code. This guards against a future refactor that moves any
//     write before the gate.
func TestGenerator_AuditPreflightGate_AbortsOnMissingAuditChild(t *testing.T) {
	tempDir := setupIntegrationTest(t)

	config := &Config{
		DSN:         "postgres://skimatik:skimatik_test_password@localhost:15432/skimatik_test",
		Schema:      "public",
		OutputDir:   tempDir,
		PackageName: "testgen",
		Tables:      true,
		// `posts` is in the fixture but `posts_audit` is intentionally absent;
		// `users` is included as a non-audited parent so we exercise the
		// "mixed include set" path (gate must not falsely flag non-audited
		// parents).
		Include: []string{"posts", "users"},
		TableConfigs: map[string]TableConfig{
			"posts": {
				Functions: []string{"create", "get", "update", "delete", "list", "paginate"},
				Audit:     true,
			},
			"users": {
				Functions: []string{"create", "get", "update", "delete", "list", "paginate"},
			},
		},
		Verbose: false,
	}

	gen := New(config, "test")
	ctx := context.Background()
	files, err := gen.Generate(ctx)

	if err == nil {
		t.Fatalf("expected Generate to return an error for missing posts_audit; got nil. files=%v", files)
	}

	msg := err.Error()
	// Validator error must call out the specific missing table.
	if !strings.Contains(msg, "posts_audit not found") {
		t.Errorf("expected error to mention 'posts_audit not found'; got: %s", msg)
	}
	// Validator appends a copy-pasteable DDL block for affected parents.
	if !strings.Contains(msg, "CREATE TABLE posts_audit") {
		t.Errorf("expected error to include CREATE TABLE posts_audit DDL block; got: %s", msg)
	}
	// And the leading-column index requirement should surface in the DDL block.
	if !strings.Contains(msg, "CREATE INDEX") {
		t.Errorf("expected error DDL block to include CREATE INDEX clause; got: %s", msg)
	}

	// The gate must abort before ANY file is written — shared utilities and
	// per-table code alike. A failed validation must leave the output dir
	// untouched.
	entries, readErr := os.ReadDir(tempDir)
	if readErr != nil {
		t.Fatalf("failed to read output dir %s: %v", tempDir, readErr)
	}
	if len(entries) > 0 {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, filepath.Join(tempDir, e.Name()))
		}
		t.Errorf("expected output dir to be empty after failed validation; found: %v", names)
	}
}

// TestGenerator_AuditPreflightGate_PassesWhenAuditChildIsValid confirms the
// gate's happy path: when an audited parent has a well-formed audit child in
// the schema, Generate() succeeds and emits the parent's generated file. The
// fixture defines `users` with a canonical `users_audit` sibling, so we
// configure `users` with `audit: true` and expect a successful run.
//
// This guards against a regression where the gate falsely rejects a valid
// schema (e.g. type-equivalence bug, FK-direction inversion) and prevents
// generation entirely.
func TestGenerator_AuditPreflightGate_PassesWhenAuditChildIsValid(t *testing.T) {
	tempDir := setupIntegrationTest(t)

	config := &Config{
		DSN:         "postgres://skimatik:skimatik_test_password@localhost:15432/skimatik_test",
		Schema:      "public",
		OutputDir:   tempDir,
		PackageName: "testgen",
		Tables:      true,
		Include:     []string{"users"},
		TableConfigs: map[string]TableConfig{
			"users": {
				Functions: []string{"create", "get", "update", "delete", "list", "paginate"},
				Audit:     true,
			},
		},
		Verbose: false,
	}

	gen := New(config, "test")
	ctx := context.Background()
	if _, err := gen.Generate(ctx); err != nil {
		t.Fatalf("expected Generate to succeed for users with valid users_audit; got: %v", err)
	}

	// Spot-check the per-table file was actually emitted.
	usersFile := filepath.Join(tempDir, "users_generated.go")
	if _, err := os.Stat(usersFile); err != nil {
		t.Errorf("expected %s to exist after successful Generate; stat err: %v", usersFile, err)
	}
}
