//go:build integration

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nhalm/pgxkit/v2"
)

// newTestServer connects to DATABASE_URL, builds the real handler tree, and
// wraps it with httptest.NewServer. Tests get a *httptest.Server they can hit
// like any HTTP endpoint, with cleanup wired through t.Cleanup.
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}

	db := pgxkit.NewDB()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.Connect(ctx, dsn); err != nil {
		t.Fatalf("connect to %s: %v", dsn, err)
	}
	t.Cleanup(func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = db.Shutdown(shutdownCtx)
	})

	srv := httptest.NewServer(newHandler(db))
	t.Cleanup(srv.Close)
	return srv
}

// expect200 issues a GET, asserts a 200, and drains the body. Drives the
// "all the read endpoints answer something" smoke check.
func expect200(t *testing.T, srv *httptest.Server, path string) {
	t.Helper()
	resp, err := http.Get(srv.URL + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("GET %s: status %d, body=%s", path, resp.StatusCode, body)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
}

func TestEndpointsRespond(t *testing.T) {
	srv := newTestServer(t)

	for _, path := range []string{
		"/api/health",
		"/api/users",
		"/api/posts",
		"/api/posts/with-stats",
		"/api/posts/featured",
		"/api/users/search?q=test",
	} {
		expect200(t, srv, path)
	}
}

// userResponse matches api.UserSummaryResponse closely enough for the audit flow.
type userResponse struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Email string    `json:"email"`
}

// auditEntry mirrors models.UserAuditEntry. Snapshot is JSON-encoded text in
// the DB; we decode it into snapshotPayload below to reach .Name.
type auditEntry struct {
	ID       uuid.UUID `json:"id"`
	ParentID uuid.UUID `json:"parent_id"`
	Version  int       `json:"version"`
	Snapshot string    `json:"snapshot"`
	ValidTo  *string   `json:"valid_to"`
}

type auditResponse struct {
	Audit []auditEntry `json:"audit"`
	Count int          `json:"count"`
}

type snapshotPayload struct {
	Name string `json:"name"`
}

func TestAuditFlow(t *testing.T) {
	srv := newTestServer(t)

	email := fmt.Sprintf("audit-demo-%d@example.com", time.Now().UnixNano())
	original := "Audit Demo Original"
	renamed := "Audit Demo Renamed"

	// Create the user.
	createBody, err := json.Marshal(map[string]string{"name": original, "email": email})
	if err != nil {
		t.Fatalf("marshal create: %v", err)
	}
	resp, err := http.Post(srv.URL+"/api/users", "application/json", bytes.NewReader(createBody))
	if err != nil {
		t.Fatalf("POST /api/users: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("POST /api/users: status %d, body=%s", resp.StatusCode, body)
	}
	var created userResponse
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		resp.Body.Close()
		t.Fatalf("decode create response: %v", err)
	}
	resp.Body.Close()
	if created.ID == uuid.Nil {
		t.Fatalf("created user missing id")
	}

	// PATCH the name to trigger an SCD2 close+open.
	patchBody, err := json.Marshal(map[string]string{"name": renamed})
	if err != nil {
		t.Fatalf("marshal patch: %v", err)
	}
	patchReq, err := http.NewRequest(http.MethodPatch, fmt.Sprintf("%s/api/users/%s/name", srv.URL, created.ID), bytes.NewReader(patchBody))
	if err != nil {
		t.Fatalf("build patch: %v", err)
	}
	patchReq.Header.Set("Content-Type", "application/json")
	patchResp, err := http.DefaultClient.Do(patchReq)
	if err != nil {
		t.Fatalf("PATCH /api/users/{id}/name: %v", err)
	}
	if patchResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(patchResp.Body)
		patchResp.Body.Close()
		t.Fatalf("PATCH /api/users/{id}/name: status %d, body=%s", patchResp.StatusCode, body)
	}
	patchResp.Body.Close()

	// Read the audit history and assert the SCD2 invariants.
	auditResp, err := http.Get(fmt.Sprintf("%s/api/users/%s/audit", srv.URL, created.ID))
	if err != nil {
		t.Fatalf("GET audit: %v", err)
	}
	defer auditResp.Body.Close()
	if auditResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(auditResp.Body)
		t.Fatalf("GET audit: status %d, body=%s", auditResp.StatusCode, body)
	}
	var audit auditResponse
	if err := json.NewDecoder(auditResp.Body).Decode(&audit); err != nil {
		t.Fatalf("decode audit: %v", err)
	}

	if audit.Count != 2 {
		t.Fatalf("expected 2 audit rows, got %d", audit.Count)
	}
	if len(audit.Audit) != 2 {
		t.Fatalf("expected 2 audit entries, got %d", len(audit.Audit))
	}

	var closed, open *auditEntry
	for i := range audit.Audit {
		entry := &audit.Audit[i]
		if entry.ValidTo == nil {
			if open != nil {
				t.Fatalf("expected exactly one open audit row, found another at version %d", entry.Version)
			}
			open = entry
		} else {
			if closed != nil {
				t.Fatalf("expected exactly one closed audit row, found another at version %d", entry.Version)
			}
			closed = entry
		}
	}
	if closed == nil {
		t.Fatalf("expected one closed audit row (non-nil valid_to)")
	}
	if open == nil {
		t.Fatalf("expected one open audit row (nil valid_to)")
	}

	if closed.Version != 1 {
		t.Errorf("closed row version: want 1, got %d", closed.Version)
	}
	if open.Version != 2 {
		t.Errorf("open row version: want 2, got %d", open.Version)
	}

	var closedSnap, openSnap snapshotPayload
	if err := json.Unmarshal([]byte(closed.Snapshot), &closedSnap); err != nil {
		t.Fatalf("decode closed snapshot: %v", err)
	}
	if err := json.Unmarshal([]byte(open.Snapshot), &openSnap); err != nil {
		t.Fatalf("decode open snapshot: %v", err)
	}
	if closedSnap.Name != original {
		t.Errorf("closed snapshot name: want %q, got %q", original, closedSnap.Name)
	}
	if openSnap.Name != renamed {
		t.Errorf("open snapshot name: want %q, got %q", renamed, openSnap.Name)
	}
}
