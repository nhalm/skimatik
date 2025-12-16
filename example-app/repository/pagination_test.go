package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nhalm/pgxkit"
	"github.com/nhalm/skimatik/example-app/repository/generated"
)

func getTestDB(t *testing.T) *pgxkit.DB {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		dsn = "postgres://postgres:password@localhost:15987/blog?sslmode=disable"
	}

	db := pgxkit.NewDB()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.Connect(ctx, dsn); err != nil {
		t.Skipf("Skipping test: could not connect to database: %v", err)
		return nil
	}

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		db.Shutdown(ctx)
	})

	return db
}

func TestPaginatedQuery_CursorNavigation(t *testing.T) {
	db := getTestDB(t)
	if db == nil {
		return
	}

	ctx := context.Background()

	// Clean up any existing test data
	_, err := db.Exec(ctx, "DELETE FROM posts WHERE title LIKE 'Pagination Test Post%'")
	if err != nil {
		t.Fatalf("Failed to clean up test data: %v", err)
	}

	// Get or create a test user
	var userID uuid.UUID
	err = db.QueryRow(ctx, "SELECT id FROM users LIMIT 1").Scan(&userID)
	if err != nil {
		// Create a test user if none exists
		userID = uuid.New()
		_, err = db.Exec(ctx,
			"INSERT INTO users (id, name, email, is_active) VALUES ($1, $2, $3, true)",
			userID, "Test User", "pagination-test@example.com")
		if err != nil {
			t.Fatalf("Failed to create test user: %v", err)
		}
		t.Cleanup(func() {
			db.Exec(ctx, "DELETE FROM users WHERE id = $1", userID)
		})
	}

	// Create test posts with distinct published_at times (DESC order means newest first)
	now := time.Now()
	testPosts := []struct {
		id          uuid.UUID
		title       string
		publishedAt time.Time
	}{
		{uuid.New(), "Pagination Test Post 1", now.Add(-3 * time.Hour)}, // oldest
		{uuid.New(), "Pagination Test Post 2", now.Add(-2 * time.Hour)},
		{uuid.New(), "Pagination Test Post 3", now.Add(-1 * time.Hour)}, // newest
	}

	for _, post := range testPosts {
		_, err := db.Exec(ctx,
			"INSERT INTO posts (id, title, content, author_id, is_published, published_at) VALUES ($1, $2, $3, $4, true, $5)",
			post.id, post.title, "Test content", userID, post.publishedAt)
		if err != nil {
			t.Fatalf("Failed to create test post %s: %v", post.title, err)
		}
	}

	t.Cleanup(func() {
		for _, post := range testPosts {
			db.Exec(ctx, "DELETE FROM posts WHERE id = $1", post.id)
		}
	})

	// Create the queries repository
	postsQueries := generated.NewPostsQueries(db)

	// Test: First page with limit 1 (should get newest post - Post 3)
	page1, err := postsQueries.GetPublishedPostsPaginatedPaginated(ctx, generated.PaginationParams{
		Limit: 1,
	})
	if err != nil {
		t.Fatalf("First page query failed: %v", err)
	}

	if len(page1.Items) != 1 {
		t.Errorf("Expected 1 item on first page, got %d", len(page1.Items))
	}

	if !page1.HasMore {
		t.Error("Expected HasMore=true on first page")
	}

	if page1.NextCursor == "" {
		t.Fatal("Expected NextCursor to be set on first page")
	}

	firstPageTitle := page1.Items[0].Title
	t.Logf("Page 1: %s (cursor: %s)", firstPageTitle, page1.NextCursor)

	// Test: Second page using cursor from first page (should get Post 2)
	page2, err := postsQueries.GetPublishedPostsPaginatedPaginated(ctx, generated.PaginationParams{
		Limit:      1,
		NextCursor: page1.NextCursor,
	})
	if err != nil {
		t.Fatalf("Second page query failed: %v", err)
	}

	if len(page2.Items) != 1 {
		t.Errorf("Expected 1 item on second page, got %d", len(page2.Items))
	}

	secondPageTitle := page2.Items[0].Title
	t.Logf("Page 2: %s (cursor: %s)", secondPageTitle, page2.NextCursor)

	// Verify we got different posts on each page
	if firstPageTitle == secondPageTitle {
		t.Errorf("First and second page returned the same post: %s", firstPageTitle)
	}

	// Test: Third page using cursor from second page (should get Post 1)
	page3, err := postsQueries.GetPublishedPostsPaginatedPaginated(ctx, generated.PaginationParams{
		Limit:      1,
		NextCursor: page2.NextCursor,
	})
	if err != nil {
		t.Fatalf("Third page query failed: %v", err)
	}

	if len(page3.Items) != 1 {
		t.Errorf("Expected 1 item on third page, got %d", len(page3.Items))
	}

	thirdPageTitle := page3.Items[0].Title
	t.Logf("Page 3: %s (hasMore: %v)", thirdPageTitle, page3.HasMore)

	// Verify all three pages have different posts
	titles := map[string]bool{firstPageTitle: true, secondPageTitle: true, thirdPageTitle: true}
	if len(titles) != 3 {
		t.Errorf("Expected 3 unique posts across pages, got duplicates: %s, %s, %s",
			firstPageTitle, secondPageTitle, thirdPageTitle)
	}

	// Verify DESC ordering (newest first)
	if firstPageTitle != "Pagination Test Post 3" {
		t.Errorf("Expected newest post (Post 3) on first page with DESC order, got: %s", firstPageTitle)
	}
	if secondPageTitle != "Pagination Test Post 2" {
		t.Errorf("Expected Post 2 on second page, got: %s", secondPageTitle)
	}
	if thirdPageTitle != "Pagination Test Post 1" {
		t.Errorf("Expected oldest post (Post 1) on third page, got: %s", thirdPageTitle)
	}

	t.Log("✅ Pagination cursor navigation test passed")
}

func TestPaginatedQuery_ASCOrdering(t *testing.T) {
	db := getTestDB(t)
	if db == nil {
		return
	}

	ctx := context.Background()
	postsQueries := generated.NewPostsQueries(db)

	// Test ASC ordering: GetOldestPostsPaginatedPaginated (oldest first)
	// We just verify that pagination works and ordering is correct (not specific titles)
	page1, err := postsQueries.GetOldestPostsPaginatedPaginated(ctx, generated.PaginationParams{
		Limit: 1,
	})
	if err != nil {
		t.Fatalf("First page query failed: %v", err)
	}

	if len(page1.Items) != 1 {
		t.Errorf("Expected 1 item on first page, got %d", len(page1.Items))
	}

	firstItem := page1.Items[0]
	t.Logf("Page 1 (ASC): %s (published_at: %v)", firstItem.Title, firstItem.PublishedAt)

	// Get second page
	if page1.NextCursor == "" {
		t.Fatal("Expected NextCursor to be set")
	}

	page2, err := postsQueries.GetOldestPostsPaginatedPaginated(ctx, generated.PaginationParams{
		Limit:      1,
		NextCursor: page1.NextCursor,
	})
	if err != nil {
		t.Fatalf("Second page query failed: %v", err)
	}

	if len(page2.Items) != 1 {
		t.Errorf("Expected 1 item on second page, got %d", len(page2.Items))
	}

	secondItem := page2.Items[0]
	t.Logf("Page 2 (ASC): %s (published_at: %v)", secondItem.Title, secondItem.PublishedAt)

	// Verify different items on each page
	if firstItem.Id == secondItem.Id {
		t.Errorf("First and second page returned the same post")
	}

	// Verify ASC ordering: first page item should have earlier published_at than second
	if firstItem.PublishedAt != nil && secondItem.PublishedAt != nil {
		if firstItem.PublishedAt.After(*secondItem.PublishedAt) {
			t.Errorf("ASC ordering incorrect: first item published_at (%v) should be <= second (%v)",
				firstItem.PublishedAt, secondItem.PublishedAt)
		}
	}

	t.Log("✅ ASC ordering pagination test passed")
}

func TestPaginatedQuery_WithFilterParameter(t *testing.T) {
	db := getTestDB(t)
	if db == nil {
		return
	}

	ctx := context.Background()

	// Clean up any existing test data
	_, err := db.Exec(ctx, "DELETE FROM posts WHERE title LIKE 'Filter Test Post%'")
	if err != nil {
		t.Fatalf("Failed to clean up test data: %v", err)
	}

	// Create a specific test user for this test
	testUserID := uuid.New()
	_, err = db.Exec(ctx,
		"INSERT INTO users (id, name, email, is_active) VALUES ($1, $2, $3, true)",
		testUserID, "Filter Test User", "filter-test@example.com")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	t.Cleanup(func() {
		db.Exec(ctx, "DELETE FROM users WHERE id = $1", testUserID)
	})

	// Create test posts for this specific author
	now := time.Now()
	testPosts := []struct {
		id          uuid.UUID
		title       string
		publishedAt time.Time
	}{
		{uuid.New(), "Filter Test Post 1", now.Add(-3 * time.Hour)},
		{uuid.New(), "Filter Test Post 2", now.Add(-2 * time.Hour)},
		{uuid.New(), "Filter Test Post 3", now.Add(-1 * time.Hour)},
	}

	for _, post := range testPosts {
		_, err := db.Exec(ctx,
			"INSERT INTO posts (id, title, content, author_id, is_published, published_at) VALUES ($1, $2, $3, $4, true, $5)",
			post.id, post.title, "Test content", testUserID, post.publishedAt)
		if err != nil {
			t.Fatalf("Failed to create test post %s: %v", post.title, err)
		}
	}

	t.Cleanup(func() {
		for _, post := range testPosts {
			db.Exec(ctx, "DELETE FROM posts WHERE id = $1", post.id)
		}
	})

	postsQueries := generated.NewPostsQueries(db)

	// Test: First page with author filter (should get newest post - Post 3)
	page1, err := postsQueries.GetPostsByAuthorPaginatedPaginated(ctx, testUserID, generated.PaginationParams{
		Limit: 1,
	})
	if err != nil {
		t.Fatalf("First page query failed: %v", err)
	}

	if len(page1.Items) != 1 {
		t.Errorf("Expected 1 item on first page, got %d", len(page1.Items))
	}

	if !page1.HasMore {
		t.Error("Expected HasMore=true on first page")
	}

	firstPageTitle := page1.Items[0].Title
	t.Logf("Page 1 (filtered by author): %s", firstPageTitle)

	// Verify we got our test post (DESC order, so newest first)
	if firstPageTitle != "Filter Test Post 3" {
		t.Errorf("Expected 'Filter Test Post 3' on first page, got: %s", firstPageTitle)
	}

	// Test: Second page using cursor
	if page1.NextCursor == "" {
		t.Fatal("Expected NextCursor to be set")
	}

	page2, err := postsQueries.GetPostsByAuthorPaginatedPaginated(ctx, testUserID, generated.PaginationParams{
		Limit:      1,
		NextCursor: page1.NextCursor,
	})
	if err != nil {
		t.Fatalf("Second page query failed: %v", err)
	}

	if len(page2.Items) != 1 {
		t.Errorf("Expected 1 item on second page, got %d", len(page2.Items))
	}

	secondPageTitle := page2.Items[0].Title
	t.Logf("Page 2 (filtered by author): %s", secondPageTitle)

	if secondPageTitle != "Filter Test Post 2" {
		t.Errorf("Expected 'Filter Test Post 2' on second page, got: %s", secondPageTitle)
	}

	// Test: Third page
	page3, err := postsQueries.GetPostsByAuthorPaginatedPaginated(ctx, testUserID, generated.PaginationParams{
		Limit:      1,
		NextCursor: page2.NextCursor,
	})
	if err != nil {
		t.Fatalf("Third page query failed: %v", err)
	}

	thirdPageTitle := page3.Items[0].Title
	t.Logf("Page 3 (filtered by author): %s (hasMore: %v)", thirdPageTitle, page3.HasMore)

	if thirdPageTitle != "Filter Test Post 1" {
		t.Errorf("Expected 'Filter Test Post 1' on third page, got: %s", thirdPageTitle)
	}

	// Verify author filter works: query with different author should return empty
	otherUserID := uuid.New()
	otherPage, err := postsQueries.GetPostsByAuthorPaginatedPaginated(ctx, otherUserID, generated.PaginationParams{
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("Query with other author failed: %v", err)
	}

	if len(otherPage.Items) != 0 {
		t.Errorf("Expected 0 items for non-existent author, got %d", len(otherPage.Items))
	}

	t.Log("✅ Parameterized pagination test passed")
}
