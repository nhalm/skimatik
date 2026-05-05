//go:build !short

package generator

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGenerator_AuditPreflightGate_AbortsOnMissingAuditChild verifies that
// when an audited parent is missing its `<parent>_audit` sibling, Generate
// returns an error containing the canonical CREATE TABLE DDL block and
// leaves the output directory empty.
func TestGenerator_AuditPreflightGate_AbortsOnMissingAuditChild(t *testing.T) {
	tempDir := setupIntegrationTest(t)

	config := &Config{
		DSN:         "postgres://skimatik:skimatik_test_password@localhost:15432/skimatik_test",
		Schema:      "public",
		OutputDir:   tempDir,
		PackageName: "testgen",
		Tables:      true,
		// `posts_audit` is intentionally absent from the fixture; `users` is
		// included as a non-audited parent to exercise the mixed include set.
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
	if !strings.Contains(msg, "posts_audit not found") {
		t.Errorf("expected error to mention 'posts_audit not found'; got: %s", msg)
	}
	if !strings.Contains(msg, "CREATE TABLE posts_audit") {
		t.Errorf("expected error to include CREATE TABLE posts_audit DDL block; got: %s", msg)
	}
	if !strings.Contains(msg, "CREATE INDEX") {
		t.Errorf("expected error DDL block to include CREATE INDEX clause; got: %s", msg)
	}

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

// TestGenerator_AuditPreflightGate_PassesWhenAuditChildIsValid verifies the
// gate's happy path: when an audited parent has a well-formed audit child,
// Generate succeeds and emits the per-table file.
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

	usersFile := filepath.Join(tempDir, "users_generated.go")
	if _, err := os.Stat(usersFile); err != nil {
		t.Errorf("expected %s to exist after successful Generate; stat err: %v", usersFile, err)
	}
}
