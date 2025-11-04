//go:build !short

package generator

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestResultTypes_SimpleSelectNotNull tests that NOT NULL columns generate native Go types
func TestResultTypes_SimpleSelectNotNull(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testDB := getTestDB(t)
	ctx := context.Background()

	tempDir := t.TempDir()
	sqlDir := filepath.Join(tempDir, "queries")
	err := os.MkdirAll(sqlDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create SQL directory: %v", err)
	}

	querySQL := `-- name: GetPaymentLinkBasic :one
SELECT id, status FROM payment_links WHERE id = $1;`

	err = os.WriteFile(filepath.Join(sqlDir, "payment_links.sql"), []byte(querySQL), 0644)
	if err != nil {
		t.Fatalf("Failed to write test query: %v", err)
	}

	parser := NewQueryParser(sqlDir)
	queries, err := parser.ParseQueries()
	if err != nil {
		t.Fatalf("Failed to parse queries: %v", err)
	}

	if len(queries) != 1 {
		t.Fatalf("Expected 1 query, got %d", len(queries))
	}

	analyzer := NewQueryAnalyzer(testDB)
	err = analyzer.AnalyzeQuery(ctx, &queries[0])
	if err != nil {
		t.Fatalf("Failed to analyze query: %v", err)
	}

	query := queries[0]
	if len(query.Columns) != 2 {
		t.Fatalf("Expected 2 columns, got %d", len(query.Columns))
	}

	idCol := query.Columns[0]
	if idCol.Name != "id" {
		t.Errorf("Expected column name 'id', got '%s'", idCol.Name)
	}
	if idCol.GoType != "uuid.UUID" {
		t.Errorf("Expected id to be uuid.UUID (NOT NULL), got %s", idCol.GoType)
	}
	if idCol.IsNullable {
		t.Errorf("Expected id to be NOT NULL")
	}

	statusCol := query.Columns[1]
	if statusCol.Name != "status" {
		t.Errorf("Expected column name 'status', got '%s'", statusCol.Name)
	}
	if statusCol.GoType != "string" {
		t.Errorf("Expected status to be string (NOT NULL), got %s", statusCol.GoType)
	}
	if statusCol.IsNullable {
		t.Errorf("Expected status to be NOT NULL")
	}
}

// TestResultTypes_SelectWithNullable tests that nullable columns generate pointer types
func TestResultTypes_SelectWithNullable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testDB := getTestDB(t)
	ctx := context.Background()

	tempDir := t.TempDir()
	sqlDir := filepath.Join(tempDir, "queries")
	err := os.MkdirAll(sqlDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create SQL directory: %v", err)
	}

	querySQL := `-- name: GetPaymentLinkWithDescription :one
SELECT id, description FROM payment_links WHERE id = $1;`

	err = os.WriteFile(filepath.Join(sqlDir, "payment_links.sql"), []byte(querySQL), 0644)
	if err != nil {
		t.Fatalf("Failed to write test query: %v", err)
	}

	parser := NewQueryParser(sqlDir)
	queries, err := parser.ParseQueries()
	if err != nil {
		t.Fatalf("Failed to parse queries: %v", err)
	}

	analyzer := NewQueryAnalyzer(testDB)
	err = analyzer.AnalyzeQuery(ctx, &queries[0])
	if err != nil {
		t.Fatalf("Failed to analyze query: %v", err)
	}

	query := queries[0]
	if len(query.Columns) != 2 {
		t.Fatalf("Expected 2 columns, got %d", len(query.Columns))
	}

	idCol := query.Columns[0]
	if idCol.GoType != "uuid.UUID" {
		t.Errorf("Expected id to be uuid.UUID (NOT NULL), got %s", idCol.GoType)
	}

	descCol := query.Columns[1]
	if descCol.Name != "description" {
		t.Errorf("Expected column name 'description', got '%s'", descCol.Name)
	}
	if descCol.GoType != "*string" {
		t.Errorf("Expected description to be *string (nullable), got %s", descCol.GoType)
	}
	if !descCol.IsNullable {
		t.Errorf("Expected description to be nullable")
	}
}

// TestResultTypes_CountAggregate tests that COUNT aggregates are never nullable
func TestResultTypes_CountAggregate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testDB := getTestDB(t)
	ctx := context.Background()

	tempDir := t.TempDir()
	sqlDir := filepath.Join(tempDir, "queries")
	err := os.MkdirAll(sqlDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create SQL directory: %v", err)
	}

	querySQL := `-- name: GetPaymentStats :one
SELECT
    COUNT(*) as payment_count,
    SUM(amount) as total_amount
FROM payments
WHERE payment_link_id = $1;`

	err = os.WriteFile(filepath.Join(sqlDir, "payments.sql"), []byte(querySQL), 0644)
	if err != nil {
		t.Fatalf("Failed to write test query: %v", err)
	}

	parser := NewQueryParser(sqlDir)
	queries, err := parser.ParseQueries()
	if err != nil {
		t.Fatalf("Failed to parse queries: %v", err)
	}

	analyzer := NewQueryAnalyzer(testDB)
	err = analyzer.AnalyzeQuery(ctx, &queries[0])
	if err != nil {
		t.Fatalf("Failed to analyze query: %v", err)
	}

	query := queries[0]
	if len(query.Columns) != 2 {
		t.Fatalf("Expected 2 columns, got %d", len(query.Columns))
	}

	countCol := findColumn(query.Columns, "payment_count")
	if countCol == nil {
		t.Fatal("payment_count column not found")
	}
	if countCol.GoType != "int" {
		t.Errorf("Expected payment_count to be int (COUNT never NULL), got %s", countCol.GoType)
	}
	if countCol.IsNullable {
		t.Errorf("Expected payment_count to be NOT NULL (COUNT never returns NULL)")
	}

	sumCol := findColumn(query.Columns, "total_amount")
	if sumCol == nil {
		t.Fatal("total_amount column not found")
	}
	if sumCol.GoType != "*int" {
		t.Errorf("Expected total_amount to be *int (SUM can be NULL), got %s", sumCol.GoType)
	}
	if !sumCol.IsNullable {
		t.Errorf("Expected total_amount to be nullable (SUM can return NULL)")
	}
}

// TestResultTypes_AvgAggregate tests that AVG aggregates are nullable
func TestResultTypes_AvgAggregate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testDB := getTestDB(t)
	ctx := context.Background()

	tempDir := t.TempDir()
	sqlDir := filepath.Join(tempDir, "queries")
	err := os.MkdirAll(sqlDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create SQL directory: %v", err)
	}

	querySQL := `-- name: GetPostAverages :one
SELECT
    AVG(view_count) as avg_views,
    AVG(like_count) as avg_likes,
    COUNT(*) as total_posts
FROM posts
WHERE user_id = $1;`

	err = os.WriteFile(filepath.Join(sqlDir, "posts.sql"), []byte(querySQL), 0644)
	if err != nil {
		t.Fatalf("Failed to write test query: %v", err)
	}

	parser := NewQueryParser(sqlDir)
	queries, err := parser.ParseQueries()
	if err != nil {
		t.Fatalf("Failed to parse queries: %v", err)
	}

	analyzer := NewQueryAnalyzer(testDB)
	err = analyzer.AnalyzeQuery(ctx, &queries[0])
	if err != nil {
		t.Fatalf("Failed to analyze query: %v", err)
	}

	query := queries[0]
	if len(query.Columns) != 3 {
		t.Fatalf("Expected 3 columns, got %d", len(query.Columns))
	}

	avgViewsCol := findColumn(query.Columns, "avg_views")
	if avgViewsCol == nil {
		t.Fatal("avg_views column not found")
	}
	if avgViewsCol.GoType != "*float64" {
		t.Errorf("Expected avg_views to be *float64 (AVG returns NULL for empty set), got %s", avgViewsCol.GoType)
	}
	if !avgViewsCol.IsNullable {
		t.Errorf("Expected avg_views to be nullable (AVG can return NULL)")
	}

	avgLikesCol := findColumn(query.Columns, "avg_likes")
	if avgLikesCol == nil {
		t.Fatal("avg_likes column not found")
	}
	if avgLikesCol.GoType != "*float64" {
		t.Errorf("Expected avg_likes to be *float64 (AVG returns NULL for empty set), got %s", avgLikesCol.GoType)
	}
	if !avgLikesCol.IsNullable {
		t.Errorf("Expected avg_likes to be nullable (AVG can return NULL)")
	}

	countCol := findColumn(query.Columns, "total_posts")
	if countCol == nil {
		t.Fatal("total_posts column not found")
	}
	if countCol.GoType != "int" {
		t.Errorf("Expected total_posts to be int (COUNT never NULL), got %s", countCol.GoType)
	}
	if countCol.IsNullable {
		t.Errorf("Expected total_posts to be NOT NULL (COUNT never returns NULL)")
	}
}

// TestResultTypes_MinMaxAggregates tests that MIN/MAX aggregates are nullable
func TestResultTypes_MinMaxAggregates(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testDB := getTestDB(t)
	ctx := context.Background()

	tempDir := t.TempDir()
	sqlDir := filepath.Join(tempDir, "queries")
	err := os.MkdirAll(sqlDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create SQL directory: %v", err)
	}

	querySQL := `-- name: GetCommentStats :one
SELECT
    MIN(upvotes) as min_upvotes,
    MAX(upvotes) as max_upvotes,
    MIN(created_at) as first_comment,
    MAX(created_at) as last_comment
FROM comments
WHERE post_id = $1;`

	err = os.WriteFile(filepath.Join(sqlDir, "comments.sql"), []byte(querySQL), 0644)
	if err != nil {
		t.Fatalf("Failed to write test query: %v", err)
	}

	parser := NewQueryParser(sqlDir)
	queries, err := parser.ParseQueries()
	if err != nil {
		t.Fatalf("Failed to parse queries: %v", err)
	}

	analyzer := NewQueryAnalyzer(testDB)
	err = analyzer.AnalyzeQuery(ctx, &queries[0])
	if err != nil {
		t.Fatalf("Failed to analyze query: %v", err)
	}

	query := queries[0]
	if len(query.Columns) != 4 {
		t.Fatalf("Expected 4 columns, got %d", len(query.Columns))
	}

	minUpvotesCol := findColumn(query.Columns, "min_upvotes")
	if minUpvotesCol == nil {
		t.Fatal("min_upvotes column not found")
	}
	if minUpvotesCol.GoType != "*int" {
		t.Errorf("Expected min_upvotes to be *int (MIN returns NULL for empty set), got %s", minUpvotesCol.GoType)
	}
	if !minUpvotesCol.IsNullable {
		t.Errorf("Expected min_upvotes to be nullable (MIN can return NULL)")
	}

	maxUpvotesCol := findColumn(query.Columns, "max_upvotes")
	if maxUpvotesCol == nil {
		t.Fatal("max_upvotes column not found")
	}
	if maxUpvotesCol.GoType != "*int" {
		t.Errorf("Expected max_upvotes to be *int (MAX returns NULL for empty set), got %s", maxUpvotesCol.GoType)
	}
	if !maxUpvotesCol.IsNullable {
		t.Errorf("Expected max_upvotes to be nullable (MAX can return NULL)")
	}

	firstCommentCol := findColumn(query.Columns, "first_comment")
	if firstCommentCol == nil {
		t.Fatal("first_comment column not found")
	}
	if firstCommentCol.GoType != "*time.Time" {
		t.Errorf("Expected first_comment to be *time.Time (MIN returns NULL for empty set), got %s", firstCommentCol.GoType)
	}
	if !firstCommentCol.IsNullable {
		t.Errorf("Expected first_comment to be nullable (MIN can return NULL)")
	}

	lastCommentCol := findColumn(query.Columns, "last_comment")
	if lastCommentCol == nil {
		t.Fatal("last_comment column not found")
	}
	if lastCommentCol.GoType != "*time.Time" {
		t.Errorf("Expected last_comment to be *time.Time (MAX returns NULL for empty set), got %s", lastCommentCol.GoType)
	}
	if !lastCommentCol.IsNullable {
		t.Errorf("Expected last_comment to be nullable (MAX can return NULL)")
	}
}

// TestResultTypes_GroupByWithMixedAggregates tests GROUP BY with mixed nullable/non-nullable aggregates
func TestResultTypes_GroupByWithMixedAggregates(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testDB := getTestDB(t)
	ctx := context.Background()

	tempDir := t.TempDir()
	sqlDir := filepath.Join(tempDir, "queries")
	err := os.MkdirAll(sqlDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create SQL directory: %v", err)
	}

	querySQL := `-- name: GetUserPostStats :many
SELECT
    user_id,
    COUNT(*) as post_count,
    SUM(view_count) as total_views,
    AVG(view_count) as avg_views,
    MAX(view_count) as max_views,
    MIN(view_count) as min_views
FROM posts
GROUP BY user_id
ORDER BY post_count DESC;`

	err = os.WriteFile(filepath.Join(sqlDir, "posts.sql"), []byte(querySQL), 0644)
	if err != nil {
		t.Fatalf("Failed to write test query: %v", err)
	}

	parser := NewQueryParser(sqlDir)
	queries, err := parser.ParseQueries()
	if err != nil {
		t.Fatalf("Failed to parse queries: %v", err)
	}

	analyzer := NewQueryAnalyzer(testDB)
	err = analyzer.AnalyzeQuery(ctx, &queries[0])
	if err != nil {
		t.Fatalf("Failed to analyze query: %v", err)
	}

	query := queries[0]
	if len(query.Columns) != 6 {
		t.Fatalf("Expected 6 columns, got %d", len(query.Columns))
	}

	userIDCol := findColumn(query.Columns, "user_id")
	if userIDCol == nil {
		t.Fatal("user_id column not found")
	}
	if userIDCol.GoType != "uuid.UUID" {
		t.Errorf("Expected user_id to be uuid.UUID (NOT NULL in schema), got %s", userIDCol.GoType)
	}
	if userIDCol.IsNullable {
		t.Errorf("Expected user_id to be NOT NULL")
	}

	postCountCol := findColumn(query.Columns, "post_count")
	if postCountCol == nil {
		t.Fatal("post_count column not found")
	}
	if postCountCol.GoType != "int" {
		t.Errorf("Expected post_count to be int (COUNT never NULL), got %s", postCountCol.GoType)
	}
	if postCountCol.IsNullable {
		t.Errorf("Expected post_count to be NOT NULL (COUNT never returns NULL)")
	}

	totalViewsCol := findColumn(query.Columns, "total_views")
	if totalViewsCol == nil {
		t.Fatal("total_views column not found")
	}
	if totalViewsCol.GoType != "*int" {
		t.Errorf("Expected total_views to be *int (SUM can be NULL), got %s", totalViewsCol.GoType)
	}
	if !totalViewsCol.IsNullable {
		t.Errorf("Expected total_views to be nullable (SUM can return NULL)")
	}

	avgViewsCol := findColumn(query.Columns, "avg_views")
	if avgViewsCol == nil {
		t.Fatal("avg_views column not found")
	}
	if avgViewsCol.GoType != "*float64" {
		t.Errorf("Expected avg_views to be *float64 (AVG can be NULL), got %s", avgViewsCol.GoType)
	}
	if !avgViewsCol.IsNullable {
		t.Errorf("Expected avg_views to be nullable (AVG can return NULL)")
	}

	maxViewsCol := findColumn(query.Columns, "max_views")
	if maxViewsCol == nil {
		t.Fatal("max_views column not found")
	}
	if maxViewsCol.GoType != "*int" {
		t.Errorf("Expected max_views to be *int (MAX can be NULL), got %s", maxViewsCol.GoType)
	}
	if !maxViewsCol.IsNullable {
		t.Errorf("Expected max_views to be nullable (MAX can return NULL)")
	}

	minViewsCol := findColumn(query.Columns, "min_views")
	if minViewsCol == nil {
		t.Fatal("min_views column not found")
	}
	if minViewsCol.GoType != "*int" {
		t.Errorf("Expected min_views to be *int (MIN can be NULL), got %s", minViewsCol.GoType)
	}
	if !minViewsCol.IsNullable {
		t.Errorf("Expected min_views to be nullable (MIN can return NULL)")
	}
}

// TestResultTypes_AggregateWithHaving tests HAVING clause doesn't affect nullability
func TestResultTypes_AggregateWithHaving(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testDB := getTestDB(t)
	ctx := context.Background()

	tempDir := t.TempDir()
	sqlDir := filepath.Join(tempDir, "queries")
	err := os.MkdirAll(sqlDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create SQL directory: %v", err)
	}

	querySQL := `-- name: GetActiveUsers :many
SELECT
    user_id,
    COUNT(*) as post_count,
    AVG(like_count) as avg_likes,
    SUM(view_count) as total_views
FROM posts
GROUP BY user_id
HAVING COUNT(*) > $1 AND AVG(like_count) > $2
ORDER BY post_count DESC;`

	err = os.WriteFile(filepath.Join(sqlDir, "posts.sql"), []byte(querySQL), 0644)
	if err != nil {
		t.Fatalf("Failed to write test query: %v", err)
	}

	parser := NewQueryParser(sqlDir)
	queries, err := parser.ParseQueries()
	if err != nil {
		t.Fatalf("Failed to parse queries: %v", err)
	}

	analyzer := NewQueryAnalyzer(testDB)
	err = analyzer.AnalyzeQuery(ctx, &queries[0])
	if err != nil {
		t.Fatalf("Failed to analyze query: %v", err)
	}

	query := queries[0]
	if len(query.Columns) != 4 {
		t.Fatalf("Expected 4 columns, got %d", len(query.Columns))
	}

	postCountCol := findColumn(query.Columns, "post_count")
	if postCountCol == nil {
		t.Fatal("post_count column not found")
	}
	if postCountCol.GoType != "int" {
		t.Errorf("Expected post_count to be int (COUNT never NULL even with HAVING), got %s", postCountCol.GoType)
	}
	if postCountCol.IsNullable {
		t.Errorf("Expected post_count to be NOT NULL (COUNT never returns NULL)")
	}

	avgLikesCol := findColumn(query.Columns, "avg_likes")
	if avgLikesCol == nil {
		t.Fatal("avg_likes column not found")
	}
	if avgLikesCol.GoType != "*float64" {
		t.Errorf("Expected avg_likes to be *float64 (AVG can be NULL even with HAVING), got %s", avgLikesCol.GoType)
	}
	if !avgLikesCol.IsNullable {
		t.Errorf("Expected avg_likes to be nullable (AVG can return NULL)")
	}

	totalViewsCol := findColumn(query.Columns, "total_views")
	if totalViewsCol == nil {
		t.Fatal("total_views column not found")
	}
	if totalViewsCol.GoType != "*int" {
		t.Errorf("Expected total_views to be *int (SUM can be NULL even with HAVING), got %s", totalViewsCol.GoType)
	}
	if !totalViewsCol.IsNullable {
		t.Errorf("Expected total_views to be nullable (SUM can return NULL)")
	}
}

// TestResultTypes_ComplexAggregateExpressions tests complex aggregate expressions
func TestResultTypes_ComplexAggregateExpressions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testDB := getTestDB(t)
	ctx := context.Background()

	tempDir := t.TempDir()
	sqlDir := filepath.Join(tempDir, "queries")
	err := os.MkdirAll(sqlDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create SQL directory: %v", err)
	}

	querySQL := `-- name: GetPostEngagementStats :one
SELECT
    COUNT(*) as total_posts,
    COUNT(DISTINCT user_id) as unique_users,
    AVG(view_count * 2) as avg_double_views,
    SUM(view_count + like_count) as total_engagement,
    AVG(CASE WHEN view_count > 0 THEN like_count::float / view_count::float ELSE 0 END) as avg_engagement_rate
FROM posts
WHERE status = $1;`

	err = os.WriteFile(filepath.Join(sqlDir, "posts.sql"), []byte(querySQL), 0644)
	if err != nil {
		t.Fatalf("Failed to write test query: %v", err)
	}

	parser := NewQueryParser(sqlDir)
	queries, err := parser.ParseQueries()
	if err != nil {
		t.Fatalf("Failed to parse queries: %v", err)
	}

	analyzer := NewQueryAnalyzer(testDB)
	err = analyzer.AnalyzeQuery(ctx, &queries[0])
	if err != nil {
		t.Fatalf("Failed to analyze query: %v", err)
	}

	query := queries[0]
	if len(query.Columns) != 5 {
		t.Fatalf("Expected 5 columns, got %d", len(query.Columns))
	}

	totalPostsCol := findColumn(query.Columns, "total_posts")
	if totalPostsCol == nil {
		t.Fatal("total_posts column not found")
	}
	if totalPostsCol.GoType != "int" {
		t.Errorf("Expected total_posts to be int (COUNT(*) never NULL), got %s", totalPostsCol.GoType)
	}
	if totalPostsCol.IsNullable {
		t.Errorf("Expected total_posts to be NOT NULL")
	}

	uniqueUsersCol := findColumn(query.Columns, "unique_users")
	if uniqueUsersCol == nil {
		t.Fatal("unique_users column not found")
	}
	if uniqueUsersCol.GoType != "int" {
		t.Errorf("Expected unique_users to be int (COUNT(DISTINCT col) never NULL), got %s", uniqueUsersCol.GoType)
	}
	if uniqueUsersCol.IsNullable {
		t.Errorf("Expected unique_users to be NOT NULL (COUNT never returns NULL)")
	}

	avgDoubleViewsCol := findColumn(query.Columns, "avg_double_views")
	if avgDoubleViewsCol == nil {
		t.Fatal("avg_double_views column not found")
	}
	if avgDoubleViewsCol.GoType != "*float64" {
		t.Errorf("Expected avg_double_views to be *float64 (AVG can be NULL), got %s", avgDoubleViewsCol.GoType)
	}
	if !avgDoubleViewsCol.IsNullable {
		t.Errorf("Expected avg_double_views to be nullable (AVG can return NULL)")
	}

	totalEngagementCol := findColumn(query.Columns, "total_engagement")
	if totalEngagementCol == nil {
		t.Fatal("total_engagement column not found")
	}
	if totalEngagementCol.GoType != "*int" {
		t.Errorf("Expected total_engagement to be *int (SUM can be NULL), got %s", totalEngagementCol.GoType)
	}
	if !totalEngagementCol.IsNullable {
		t.Errorf("Expected total_engagement to be nullable (SUM can return NULL)")
	}

	avgEngagementRateCol := findColumn(query.Columns, "avg_engagement_rate")
	if avgEngagementRateCol == nil {
		t.Fatal("avg_engagement_rate column not found")
	}
	if avgEngagementRateCol.GoType != "*float64" {
		t.Errorf("Expected avg_engagement_rate to be *float64 (AVG can be NULL), got %s", avgEngagementRateCol.GoType)
	}
	if !avgEngagementRateCol.IsNullable {
		t.Errorf("Expected avg_engagement_rate to be nullable (AVG can return NULL)")
	}
}

// TestResultTypes_CoalesceExpression tests COALESCE making nullable columns effectively non-nullable
func TestResultTypes_CoalesceExpression(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testDB := getTestDB(t)
	ctx := context.Background()

	tempDir := t.TempDir()
	sqlDir := filepath.Join(tempDir, "queries")
	err := os.MkdirAll(sqlDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create SQL directory: %v", err)
	}

	querySQL := `-- name: GetUserWithDefaults :one
SELECT
    id,
    name,
    COALESCE(age, 0) as age_with_default,
    COALESCE(balance, 0.0) as balance_with_default,
    age as age_nullable
FROM users
WHERE id = $1;`

	err = os.WriteFile(filepath.Join(sqlDir, "users.sql"), []byte(querySQL), 0644)
	if err != nil {
		t.Fatalf("Failed to write test query: %v", err)
	}

	parser := NewQueryParser(sqlDir)
	queries, err := parser.ParseQueries()
	if err != nil {
		t.Fatalf("Failed to parse queries: %v", err)
	}

	analyzer := NewQueryAnalyzer(testDB)
	err = analyzer.AnalyzeQuery(ctx, &queries[0])
	if err != nil {
		t.Fatalf("Failed to analyze query: %v", err)
	}

	query := queries[0]
	if len(query.Columns) != 5 {
		t.Fatalf("Expected 5 columns, got %d", len(query.Columns))
	}

	ageDefaultCol := findColumn(query.Columns, "age_with_default")
	if ageDefaultCol == nil {
		t.Fatal("age_with_default column not found")
	}
	if ageDefaultCol.GoType != "int" {
		t.Errorf("Expected age_with_default to be int (COALESCE with literal makes non-null), got %s", ageDefaultCol.GoType)
	}
	if ageDefaultCol.IsNullable {
		t.Errorf("Expected age_with_default to be NOT NULL (COALESCE guarantees non-null)")
	}

	balanceDefaultCol := findColumn(query.Columns, "balance_with_default")
	if balanceDefaultCol == nil {
		t.Fatal("balance_with_default column not found")
	}
	if balanceDefaultCol.GoType != "float64" {
		t.Errorf("Expected balance_with_default to be float64 (COALESCE with literal), got %s", balanceDefaultCol.GoType)
	}
	if balanceDefaultCol.IsNullable {
		t.Errorf("Expected balance_with_default to be NOT NULL (COALESCE guarantees non-null)")
	}

	ageNullableCol := findColumn(query.Columns, "age_nullable")
	if ageNullableCol == nil {
		t.Fatal("age_nullable column not found")
	}
	if ageNullableCol.GoType != "*int" {
		t.Errorf("Expected age_nullable to be *int (nullable column), got %s", ageNullableCol.GoType)
	}
	if !ageNullableCol.IsNullable {
		t.Errorf("Expected age_nullable to be nullable")
	}
}

// TestResultTypes_CaseExpressionNullable tests CASE expressions without guaranteed non-null ELSE
func TestResultTypes_CaseExpressionNullable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testDB := getTestDB(t)
	ctx := context.Background()

	tempDir := t.TempDir()
	sqlDir := filepath.Join(tempDir, "queries")
	err := os.MkdirAll(sqlDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create SQL directory: %v", err)
	}

	querySQL := `-- name: GetPostStatus :one
SELECT
    id,
    title,
    CASE
        WHEN status = 'published' THEN 'Public'
        WHEN status = 'draft' THEN 'Private'
    END as status_label,
    CASE
        WHEN view_count > 1000 THEN 'Popular'
        WHEN view_count > 100 THEN 'Moderate'
    END as popularity
FROM posts
WHERE id = $1;`

	err = os.WriteFile(filepath.Join(sqlDir, "posts.sql"), []byte(querySQL), 0644)
	if err != nil {
		t.Fatalf("Failed to write test query: %v", err)
	}

	parser := NewQueryParser(sqlDir)
	queries, err := parser.ParseQueries()
	if err != nil {
		t.Fatalf("Failed to parse queries: %v", err)
	}

	analyzer := NewQueryAnalyzer(testDB)
	err = analyzer.AnalyzeQuery(ctx, &queries[0])
	if err != nil {
		t.Fatalf("Failed to analyze query: %v", err)
	}

	query := queries[0]
	if len(query.Columns) != 4 {
		t.Fatalf("Expected 4 columns, got %d", len(query.Columns))
	}

	statusLabelCol := findColumn(query.Columns, "status_label")
	if statusLabelCol == nil {
		t.Fatal("status_label column not found")
	}
	if statusLabelCol.GoType != "*string" {
		t.Errorf("Expected status_label to be *string (CASE without ELSE is nullable), got %s", statusLabelCol.GoType)
	}
	if !statusLabelCol.IsNullable {
		t.Errorf("Expected status_label to be nullable (CASE without ELSE can be NULL)")
	}

	popularityCol := findColumn(query.Columns, "popularity")
	if popularityCol == nil {
		t.Fatal("popularity column not found")
	}
	if popularityCol.GoType != "*string" {
		t.Errorf("Expected popularity to be *string (CASE without ELSE is nullable), got %s", popularityCol.GoType)
	}
	if !popularityCol.IsNullable {
		t.Errorf("Expected popularity to be nullable (CASE without ELSE can be NULL)")
	}
}

// TestResultTypes_CaseExpressionWithElse tests CASE expressions with guaranteed non-null ELSE
func TestResultTypes_CaseExpressionWithElse(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testDB := getTestDB(t)
	ctx := context.Background()

	tempDir := t.TempDir()
	sqlDir := filepath.Join(tempDir, "queries")
	err := os.MkdirAll(sqlDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create SQL directory: %v", err)
	}

	querySQL := `-- name: GetPostStatusComplete :one
SELECT
    id,
    title,
    CASE
        WHEN status = 'published' THEN 'Public'
        WHEN status = 'draft' THEN 'Private'
        ELSE 'Unknown'
    END as status_label,
    CASE
        WHEN view_count > 1000 THEN view_count * 2
        WHEN view_count > 100 THEN view_count * 1
        ELSE 0
    END as engagement_score
FROM posts
WHERE id = $1;`

	err = os.WriteFile(filepath.Join(sqlDir, "posts.sql"), []byte(querySQL), 0644)
	if err != nil {
		t.Fatalf("Failed to write test query: %v", err)
	}

	parser := NewQueryParser(sqlDir)
	queries, err := parser.ParseQueries()
	if err != nil {
		t.Fatalf("Failed to parse queries: %v", err)
	}

	analyzer := NewQueryAnalyzer(testDB)
	err = analyzer.AnalyzeQuery(ctx, &queries[0])
	if err != nil {
		t.Fatalf("Failed to analyze query: %v", err)
	}

	query := queries[0]
	if len(query.Columns) != 4 {
		t.Fatalf("Expected 4 columns, got %d", len(query.Columns))
	}

	statusLabelCol := findColumn(query.Columns, "status_label")
	if statusLabelCol == nil {
		t.Fatal("status_label column not found")
	}
	if statusLabelCol.GoType != "string" {
		t.Errorf("Expected status_label to be string (CASE with ELSE is non-null), got %s", statusLabelCol.GoType)
	}
	if statusLabelCol.IsNullable {
		t.Errorf("Expected status_label to be NOT NULL (CASE with non-null ELSE)")
	}

	engagementCol := findColumn(query.Columns, "engagement_score")
	if engagementCol == nil {
		t.Fatal("engagement_score column not found")
	}
	if engagementCol.GoType != "int" {
		t.Errorf("Expected engagement_score to be int (CASE with numeric ELSE is non-null), got %s", engagementCol.GoType)
	}
	if engagementCol.IsNullable {
		t.Errorf("Expected engagement_score to be NOT NULL (CASE with non-null ELSE)")
	}
}

// TestResultTypes_ArithmeticExpressions tests arithmetic computed columns
func TestResultTypes_ArithmeticExpressions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testDB := getTestDB(t)
	ctx := context.Background()

	tempDir := t.TempDir()
	sqlDir := filepath.Join(tempDir, "queries")
	err := os.MkdirAll(sqlDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create SQL directory: %v", err)
	}

	querySQL := `-- name: GetPostEngagement :one
SELECT
    id,
    view_count,
    like_count,
    view_count + like_count as total_engagement,
    view_count * 2 as doubled_views,
    CASE
        WHEN view_count > 0 THEN like_count::float / view_count::float
        ELSE 0.0
    END as engagement_rate
FROM posts
WHERE id = $1;`

	err = os.WriteFile(filepath.Join(sqlDir, "posts.sql"), []byte(querySQL), 0644)
	if err != nil {
		t.Fatalf("Failed to write test query: %v", err)
	}

	parser := NewQueryParser(sqlDir)
	queries, err := parser.ParseQueries()
	if err != nil {
		t.Fatalf("Failed to parse queries: %v", err)
	}

	analyzer := NewQueryAnalyzer(testDB)
	err = analyzer.AnalyzeQuery(ctx, &queries[0])
	if err != nil {
		t.Fatalf("Failed to analyze query: %v", err)
	}

	query := queries[0]
	if len(query.Columns) != 6 {
		t.Fatalf("Expected 6 columns, got %d", len(query.Columns))
	}

	totalEngagementCol := findColumn(query.Columns, "total_engagement")
	if totalEngagementCol == nil {
		t.Fatal("total_engagement column not found")
	}
	if totalEngagementCol.GoType != "*int" {
		t.Errorf("Expected total_engagement to be *int (arithmetic on nullable columns), got %s", totalEngagementCol.GoType)
	}
	if !totalEngagementCol.IsNullable {
		t.Errorf("Expected total_engagement to be nullable (view_count and like_count are nullable)")
	}

	doubledViewsCol := findColumn(query.Columns, "doubled_views")
	if doubledViewsCol == nil {
		t.Fatal("doubled_views column not found")
	}
	if doubledViewsCol.GoType != "*int" {
		t.Errorf("Expected doubled_views to be *int (arithmetic on nullable column), got %s", doubledViewsCol.GoType)
	}
	if !doubledViewsCol.IsNullable {
		t.Errorf("Expected doubled_views to be nullable (view_count is nullable)")
	}

	engagementRateCol := findColumn(query.Columns, "engagement_rate")
	if engagementRateCol == nil {
		t.Fatal("engagement_rate column not found")
	}
	if engagementRateCol.GoType != "float64" {
		t.Errorf("Expected engagement_rate to be float64 (CASE with float ELSE), got %s", engagementRateCol.GoType)
	}
	if engagementRateCol.IsNullable {
		t.Errorf("Expected engagement_rate to be NOT NULL (CASE with non-null ELSE)")
	}
}

// TestResultTypes_StringConcatenation tests string concatenation computed columns
func TestResultTypes_StringConcatenation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testDB := getTestDB(t)
	ctx := context.Background()

	tempDir := t.TempDir()
	sqlDir := filepath.Join(tempDir, "queries")
	err := os.MkdirAll(sqlDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create SQL directory: %v", err)
	}

	querySQL := `-- name: GetUserDisplay :one
SELECT
    id,
    name,
    email,
    name || ' (' || email || ')' as display_name,
    'User: ' || name as prefixed_name,
    COALESCE(profile_picture_url, 'default.png') as avatar
FROM users
WHERE id = $1;`

	err = os.WriteFile(filepath.Join(sqlDir, "users.sql"), []byte(querySQL), 0644)
	if err != nil {
		t.Fatalf("Failed to write test query: %v", err)
	}

	parser := NewQueryParser(sqlDir)
	queries, err := parser.ParseQueries()
	if err != nil {
		t.Fatalf("Failed to parse queries: %v", err)
	}

	analyzer := NewQueryAnalyzer(testDB)
	err = analyzer.AnalyzeQuery(ctx, &queries[0])
	if err != nil {
		t.Fatalf("Failed to analyze query: %v", err)
	}

	query := queries[0]
	if len(query.Columns) != 6 {
		t.Fatalf("Expected 6 columns, got %d", len(query.Columns))
	}

	displayNameCol := findColumn(query.Columns, "display_name")
	if displayNameCol == nil {
		t.Fatal("display_name column not found")
	}
	if displayNameCol.GoType != "*string" {
		t.Errorf("Expected display_name to be *string (|| operator returns computed nullable), got %s", displayNameCol.GoType)
	}
	if !displayNameCol.IsNullable {
		t.Errorf("Expected display_name to be nullable (PostgreSQL types computed expressions as nullable)")
	}

	prefixedNameCol := findColumn(query.Columns, "prefixed_name")
	if prefixedNameCol == nil {
		t.Fatal("prefixed_name column not found")
	}
	if prefixedNameCol.GoType != "*string" {
		t.Errorf("Expected prefixed_name to be *string (|| operator returns computed nullable), got %s", prefixedNameCol.GoType)
	}
	if !prefixedNameCol.IsNullable {
		t.Errorf("Expected prefixed_name to be nullable (PostgreSQL types computed expressions as nullable)")
	}

	avatarCol := findColumn(query.Columns, "avatar")
	if avatarCol == nil {
		t.Fatal("avatar column not found")
	}
	if avatarCol.GoType != "string" {
		t.Errorf("Expected avatar to be string (COALESCE with literal), got %s", avatarCol.GoType)
	}
	if avatarCol.IsNullable {
		t.Errorf("Expected avatar to be NOT NULL (COALESCE guarantees non-null)")
	}
}

// Helper functions
// TestResultTypes_InnerJoin tests that INNER JOIN maintains schema nullability for both tables
func TestResultTypes_InnerJoin(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testDB := getTestDB(t)
	ctx := context.Background()

	tempDir := t.TempDir()
	sqlDir := filepath.Join(tempDir, "queries")
	err := os.MkdirAll(sqlDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create SQL directory: %v", err)
	}

	querySQL := `-- name: GetPostsWithAuthors :many
SELECT
    p.id as post_id,
    p.title,
    p.published_at,
    u.id as author_id,
    u.name as author_name,
    u.email as author_email
FROM posts p
INNER JOIN users u ON p.user_id = u.id
WHERE p.status = 'published';`

	err = os.WriteFile(filepath.Join(sqlDir, "posts.sql"), []byte(querySQL), 0644)
	if err != nil {
		t.Fatalf("Failed to write test query: %v", err)
	}

	parser := NewQueryParser(sqlDir)
	queries, err := parser.ParseQueries()
	if err != nil {
		t.Fatalf("Failed to parse queries: %v", err)
	}

	analyzer := NewQueryAnalyzer(testDB)
	err = analyzer.AnalyzeQuery(ctx, &queries[0])
	if err != nil {
		t.Fatalf("Failed to analyze query: %v", err)
	}

	query := queries[0]
	if len(query.Columns) != 6 {
		t.Fatalf("Expected 6 columns, got %d", len(query.Columns))
	}

	postIDCol := findColumn(query.Columns, "post_id")
	if postIDCol == nil {
		t.Fatal("post_id column not found")
	}
	if postIDCol.GoType != "uuid.UUID" {
		t.Errorf("Expected post_id to be uuid.UUID (NOT NULL in schema), got %s", postIDCol.GoType)
	}
	if postIDCol.IsNullable {
		t.Errorf("Expected post_id to be NOT NULL (INNER JOIN preserves NOT NULL)")
	}

	titleCol := findColumn(query.Columns, "title")
	if titleCol == nil {
		t.Fatal("title column not found")
	}
	if titleCol.GoType != "string" {
		t.Errorf("Expected title to be string (NOT NULL in schema), got %s", titleCol.GoType)
	}
	if titleCol.IsNullable {
		t.Errorf("Expected title to be NOT NULL (INNER JOIN preserves NOT NULL)")
	}

	publishedAtCol := findColumn(query.Columns, "published_at")
	if publishedAtCol == nil {
		t.Fatal("published_at column not found")
	}
	if publishedAtCol.GoType != "*time.Time" {
		t.Errorf("Expected published_at to be *time.Time (nullable in schema), got %s", publishedAtCol.GoType)
	}
	if !publishedAtCol.IsNullable {
		t.Errorf("Expected published_at to be nullable (INNER JOIN preserves schema nullability)")
	}

	authorNameCol := findColumn(query.Columns, "author_name")
	if authorNameCol == nil {
		t.Fatal("author_name column not found")
	}
	if authorNameCol.GoType != "string" {
		t.Errorf("Expected author_name to be string (NOT NULL in schema), got %s", authorNameCol.GoType)
	}
	if authorNameCol.IsNullable {
		t.Errorf("Expected author_name to be NOT NULL (INNER JOIN preserves NOT NULL)")
	}

	authorEmailCol := findColumn(query.Columns, "author_email")
	if authorEmailCol == nil {
		t.Fatal("author_email column not found")
	}
	if authorEmailCol.GoType != "string" {
		t.Errorf("Expected author_email to be string (NOT NULL in schema), got %s", authorEmailCol.GoType)
	}
	if authorEmailCol.IsNullable {
		t.Errorf("Expected author_email to be NOT NULL (INNER JOIN preserves NOT NULL)")
	}
}

// TestResultTypes_LeftJoin tests that LEFT JOIN makes right table columns nullable
func TestResultTypes_LeftJoin(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testDB := getTestDB(t)
	ctx := context.Background()

	tempDir := t.TempDir()
	sqlDir := filepath.Join(tempDir, "queries")
	err := os.MkdirAll(sqlDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create SQL directory: %v", err)
	}

	querySQL := `-- name: GetUsersWithPostCount :many
SELECT
    u.id as user_id,
    u.name,
    u.email,
    p.id as post_id,
    p.title,
    p.status
FROM users u
LEFT JOIN posts p ON u.id = p.user_id;`

	err = os.WriteFile(filepath.Join(sqlDir, "users.sql"), []byte(querySQL), 0644)
	if err != nil {
		t.Fatalf("Failed to write test query: %v", err)
	}

	parser := NewQueryParser(sqlDir)
	queries, err := parser.ParseQueries()
	if err != nil {
		t.Fatalf("Failed to parse queries: %v", err)
	}

	analyzer := NewQueryAnalyzer(testDB)
	err = analyzer.AnalyzeQuery(ctx, &queries[0])
	if err != nil {
		t.Fatalf("Failed to analyze query: %v", err)
	}

	query := queries[0]
	if len(query.Columns) != 6 {
		t.Fatalf("Expected 6 columns, got %d", len(query.Columns))
	}

	userIDCol := findColumn(query.Columns, "user_id")
	if userIDCol == nil {
		t.Fatal("user_id column not found")
	}
	if userIDCol.GoType != "uuid.UUID" {
		t.Errorf("Expected user_id to be uuid.UUID (left table NOT NULL preserved), got %s", userIDCol.GoType)
	}
	if userIDCol.IsNullable {
		t.Errorf("Expected user_id to be NOT NULL (left table columns preserve NOT NULL)")
	}

	nameCol := findColumn(query.Columns, "name")
	if nameCol == nil {
		t.Fatal("name column not found")
	}
	if nameCol.GoType != "string" {
		t.Errorf("Expected name to be string (left table NOT NULL preserved), got %s", nameCol.GoType)
	}
	if nameCol.IsNullable {
		t.Errorf("Expected name to be NOT NULL (left table columns preserve NOT NULL)")
	}

	emailCol := findColumn(query.Columns, "email")
	if emailCol == nil {
		t.Fatal("email column not found")
	}
	if emailCol.GoType != "string" {
		t.Errorf("Expected email to be string (left table NOT NULL preserved), got %s", emailCol.GoType)
	}
	if emailCol.IsNullable {
		t.Errorf("Expected email to be NOT NULL (left table columns preserve NOT NULL)")
	}

	postIDCol := findColumn(query.Columns, "post_id")
	if postIDCol == nil {
		t.Fatal("post_id column not found")
	}
	if postIDCol.GoType != "*uuid.UUID" {
		t.Errorf("Expected post_id to be *uuid.UUID (right table becomes nullable), got %s", postIDCol.GoType)
	}
	if !postIDCol.IsNullable {
		t.Errorf("Expected post_id to be nullable (LEFT JOIN makes right table columns nullable)")
	}

	titleCol := findColumn(query.Columns, "title")
	if titleCol == nil {
		t.Fatal("title column not found")
	}
	if titleCol.GoType != "*string" {
		t.Errorf("Expected title to be *string (right table becomes nullable), got %s", titleCol.GoType)
	}
	if !titleCol.IsNullable {
		t.Errorf("Expected title to be nullable (LEFT JOIN makes right table columns nullable)")
	}

	statusCol := findColumn(query.Columns, "status")
	if statusCol == nil {
		t.Fatal("status column not found")
	}
	if statusCol.GoType != "*string" {
		t.Errorf("Expected status to be *string (right table becomes nullable), got %s", statusCol.GoType)
	}
	if !statusCol.IsNullable {
		t.Errorf("Expected status to be nullable (LEFT JOIN makes right table columns nullable)")
	}
}

// TestResultTypes_MultiTableJoin tests complex joins with 3+ tables
func TestResultTypes_MultiTableJoin(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testDB := getTestDB(t)
	ctx := context.Background()

	tempDir := t.TempDir()
	sqlDir := filepath.Join(tempDir, "queries")
	err := os.MkdirAll(sqlDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create SQL directory: %v", err)
	}

	querySQL := `-- name: GetPostsWithAuthorsAndComments :many
SELECT
    p.id as post_id,
    p.title,
    u.id as author_id,
    u.name as author_name,
    c.id as comment_id,
    c.content as comment_content,
    c.is_approved
FROM posts p
INNER JOIN users u ON p.user_id = u.id
LEFT JOIN comments c ON p.id = c.post_id
WHERE p.status = 'published';`

	err = os.WriteFile(filepath.Join(sqlDir, "complex.sql"), []byte(querySQL), 0644)
	if err != nil {
		t.Fatalf("Failed to write test query: %v", err)
	}

	parser := NewQueryParser(sqlDir)
	queries, err := parser.ParseQueries()
	if err != nil {
		t.Fatalf("Failed to parse queries: %v", err)
	}

	analyzer := NewQueryAnalyzer(testDB)
	err = analyzer.AnalyzeQuery(ctx, &queries[0])
	if err != nil {
		t.Fatalf("Failed to analyze query: %v", err)
	}

	query := queries[0]
	if len(query.Columns) != 7 {
		t.Fatalf("Expected 7 columns, got %d", len(query.Columns))
	}

	postIDCol := findColumn(query.Columns, "post_id")
	if postIDCol == nil {
		t.Fatal("post_id column not found")
	}
	if postIDCol.GoType != "uuid.UUID" {
		t.Errorf("Expected post_id to be uuid.UUID (INNER JOIN table, NOT NULL), got %s", postIDCol.GoType)
	}
	if postIDCol.IsNullable {
		t.Errorf("Expected post_id to be NOT NULL")
	}

	authorNameCol := findColumn(query.Columns, "author_name")
	if authorNameCol == nil {
		t.Fatal("author_name column not found")
	}
	if authorNameCol.GoType != "string" {
		t.Errorf("Expected author_name to be string (INNER JOIN table, NOT NULL), got %s", authorNameCol.GoType)
	}
	if authorNameCol.IsNullable {
		t.Errorf("Expected author_name to be NOT NULL")
	}

	commentIDCol := findColumn(query.Columns, "comment_id")
	if commentIDCol == nil {
		t.Fatal("comment_id column not found")
	}
	if commentIDCol.GoType != "*uuid.UUID" {
		t.Errorf("Expected comment_id to be *uuid.UUID (LEFT JOIN table becomes nullable), got %s", commentIDCol.GoType)
	}
	if !commentIDCol.IsNullable {
		t.Errorf("Expected comment_id to be nullable (LEFT JOIN makes columns nullable)")
	}

	commentContentCol := findColumn(query.Columns, "comment_content")
	if commentContentCol == nil {
		t.Fatal("comment_content column not found")
	}
	if commentContentCol.GoType != "*string" {
		t.Errorf("Expected comment_content to be *string (LEFT JOIN table becomes nullable), got %s", commentContentCol.GoType)
	}
	if !commentContentCol.IsNullable {
		t.Errorf("Expected comment_content to be nullable (LEFT JOIN makes columns nullable)")
	}

	isApprovedCol := findColumn(query.Columns, "is_approved")
	if isApprovedCol == nil {
		t.Fatal("is_approved column not found")
	}
	if isApprovedCol.GoType != "*bool" {
		t.Errorf("Expected is_approved to be *bool (LEFT JOIN table becomes nullable), got %s", isApprovedCol.GoType)
	}
	if !isApprovedCol.IsNullable {
		t.Errorf("Expected is_approved to be nullable (LEFT JOIN makes columns nullable)")
	}
}

// TestResultTypes_JoinMixedNullability tests join with mix of NOT NULL and nullable columns
func TestResultTypes_JoinMixedNullability(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testDB := getTestDB(t)
	ctx := context.Background()

	tempDir := t.TempDir()
	sqlDir := filepath.Join(tempDir, "queries")
	err := os.MkdirAll(sqlDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create SQL directory: %v", err)
	}

	querySQL := `-- name: GetTagsWithPosts :many
SELECT
    t.id as tag_id,
    t.name as tag_name,
    t.description as tag_description,
    p.id as post_id,
    p.title as post_title,
    p.published_at
FROM tags t
LEFT JOIN post_tags pt ON t.id = pt.tag_id
LEFT JOIN posts p ON pt.post_id = p.id;`

	err = os.WriteFile(filepath.Join(sqlDir, "tags.sql"), []byte(querySQL), 0644)
	if err != nil {
		t.Fatalf("Failed to write test query: %v", err)
	}

	parser := NewQueryParser(sqlDir)
	queries, err := parser.ParseQueries()
	if err != nil {
		t.Fatalf("Failed to parse queries: %v", err)
	}

	analyzer := NewQueryAnalyzer(testDB)
	err = analyzer.AnalyzeQuery(ctx, &queries[0])
	if err != nil {
		t.Fatalf("Failed to analyze query: %v", err)
	}

	query := queries[0]
	if len(query.Columns) != 6 {
		t.Fatalf("Expected 6 columns, got %d", len(query.Columns))
	}

	tagIDCol := findColumn(query.Columns, "tag_id")
	if tagIDCol == nil {
		t.Fatal("tag_id column not found")
	}
	if tagIDCol.GoType != "uuid.UUID" {
		t.Errorf("Expected tag_id to be uuid.UUID (left table NOT NULL), got %s", tagIDCol.GoType)
	}
	if tagIDCol.IsNullable {
		t.Errorf("Expected tag_id to be NOT NULL (left table preserves NOT NULL)")
	}

	tagNameCol := findColumn(query.Columns, "tag_name")
	if tagNameCol == nil {
		t.Fatal("tag_name column not found")
	}
	if tagNameCol.GoType != "string" {
		t.Errorf("Expected tag_name to be string (left table NOT NULL), got %s", tagNameCol.GoType)
	}
	if tagNameCol.IsNullable {
		t.Errorf("Expected tag_name to be NOT NULL (left table preserves NOT NULL)")
	}

	tagDescCol := findColumn(query.Columns, "tag_description")
	if tagDescCol == nil {
		t.Fatal("tag_description column not found")
	}
	if tagDescCol.GoType != "*string" {
		t.Errorf("Expected tag_description to be *string (nullable in schema), got %s", tagDescCol.GoType)
	}
	if !tagDescCol.IsNullable {
		t.Errorf("Expected tag_description to be nullable (nullable in schema, left table)")
	}

	postIDCol := findColumn(query.Columns, "post_id")
	if postIDCol == nil {
		t.Fatal("post_id column not found")
	}
	if postIDCol.GoType != "*uuid.UUID" {
		t.Errorf("Expected post_id to be *uuid.UUID (right table becomes nullable), got %s", postIDCol.GoType)
	}
	if !postIDCol.IsNullable {
		t.Errorf("Expected post_id to be nullable (LEFT JOIN makes right table nullable)")
	}

	postTitleCol := findColumn(query.Columns, "post_title")
	if postTitleCol == nil {
		t.Fatal("post_title column not found")
	}
	if postTitleCol.GoType != "*string" {
		t.Errorf("Expected post_title to be *string (right table becomes nullable), got %s", postTitleCol.GoType)
	}
	if !postTitleCol.IsNullable {
		t.Errorf("Expected post_title to be nullable (LEFT JOIN makes right table nullable)")
	}

	publishedAtCol := findColumn(query.Columns, "published_at")
	if publishedAtCol == nil {
		t.Fatal("published_at column not found")
	}
	if publishedAtCol.GoType != "*time.Time" {
		t.Errorf("Expected published_at to be *time.Time (nullable in schema + LEFT JOIN), got %s", publishedAtCol.GoType)
	}
	if !publishedAtCol.IsNullable {
		t.Errorf("Expected published_at to be nullable (nullable in schema + LEFT JOIN)")
	}
}

// findColumn finds a column by name in a slice of columns
func findColumn(columns []Column, name string) *Column {
	for i := range columns {
		if columns[i].Name == name {
			return &columns[i]
		}
	}
	return nil
}

// TestResultTypes_WindowFunction_RowNumber tests that ROW_NUMBER() is always non-nullable
func TestResultTypes_WindowFunction_RowNumber(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testDB := getTestDB(t)
	ctx := context.Background()

	tempDir := t.TempDir()
	sqlDir := filepath.Join(tempDir, "queries")
	err := os.MkdirAll(sqlDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create SQL directory: %v", err)
	}

	querySQL := `-- name: GetPostsWithRowNumber :many
SELECT
    id,
    title,
    view_count,
    ROW_NUMBER() OVER (ORDER BY view_count DESC) as row_num
FROM posts
WHERE user_id = $1;`

	err = os.WriteFile(filepath.Join(sqlDir, "posts.sql"), []byte(querySQL), 0644)
	if err != nil {
		t.Fatalf("Failed to write test query: %v", err)
	}

	parser := NewQueryParser(sqlDir)
	queries, err := parser.ParseQueries()
	if err != nil {
		t.Fatalf("Failed to parse queries: %v", err)
	}

	analyzer := NewQueryAnalyzer(testDB)
	err = analyzer.AnalyzeQuery(ctx, &queries[0])
	if err != nil {
		t.Fatalf("Failed to analyze query: %v", err)
	}

	query := queries[0]
	if len(query.Columns) != 4 {
		t.Fatalf("Expected 4 columns, got %d", len(query.Columns))
	}

	rowNumCol := findColumn(query.Columns, "row_num")
	if rowNumCol == nil {
		t.Fatal("row_num column not found")
	}
	if rowNumCol.GoType != "int" {
		t.Errorf("Expected row_num to be int (ROW_NUMBER never NULL), got %s", rowNumCol.GoType)
	}
	if rowNumCol.IsNullable {
		t.Errorf("Expected row_num to be NOT NULL (ROW_NUMBER always returns non-null)")
	}

	idCol := findColumn(query.Columns, "id")
	if idCol == nil {
		t.Fatal("id column not found")
	}
	if idCol.GoType != "uuid.UUID" {
		t.Errorf("Expected id to be uuid.UUID, got %s", idCol.GoType)
	}
	if idCol.IsNullable {
		t.Errorf("Expected id to be NOT NULL")
	}

	viewCountCol := findColumn(query.Columns, "view_count")
	if viewCountCol == nil {
		t.Fatal("view_count column not found")
	}
	if viewCountCol.GoType != "*int" {
		t.Errorf("Expected view_count to be *int (nullable in schema), got %s", viewCountCol.GoType)
	}
	if !viewCountCol.IsNullable {
		t.Errorf("Expected view_count to be nullable")
	}
}

// TestResultTypes_WindowFunction_RankDenseRank tests that RANK() and DENSE_RANK() are always non-nullable
func TestResultTypes_WindowFunction_RankDenseRank(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testDB := getTestDB(t)
	ctx := context.Background()

	tempDir := t.TempDir()
	sqlDir := filepath.Join(tempDir, "queries")
	err := os.MkdirAll(sqlDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create SQL directory: %v", err)
	}

	querySQL := `-- name: GetPostsWithRanking :many
SELECT
    id,
    title,
    like_count,
    RANK() OVER (ORDER BY like_count DESC) as rank_position,
    DENSE_RANK() OVER (ORDER BY like_count DESC) as dense_rank_position
FROM posts
WHERE status = 'published';`

	err = os.WriteFile(filepath.Join(sqlDir, "posts.sql"), []byte(querySQL), 0644)
	if err != nil {
		t.Fatalf("Failed to write test query: %v", err)
	}

	parser := NewQueryParser(sqlDir)
	queries, err := parser.ParseQueries()
	if err != nil {
		t.Fatalf("Failed to parse queries: %v", err)
	}

	analyzer := NewQueryAnalyzer(testDB)
	err = analyzer.AnalyzeQuery(ctx, &queries[0])
	if err != nil {
		t.Fatalf("Failed to analyze query: %v", err)
	}

	query := queries[0]
	if len(query.Columns) != 5 {
		t.Fatalf("Expected 5 columns, got %d", len(query.Columns))
	}

	rankCol := findColumn(query.Columns, "rank_position")
	if rankCol == nil {
		t.Fatal("rank_position column not found")
	}
	if rankCol.GoType != "int" {
		t.Errorf("Expected rank_position to be int (RANK never NULL), got %s", rankCol.GoType)
	}
	if rankCol.IsNullable {
		t.Errorf("Expected rank_position to be NOT NULL (RANK always returns non-null)")
	}

	denseRankCol := findColumn(query.Columns, "dense_rank_position")
	if denseRankCol == nil {
		t.Fatal("dense_rank_position column not found")
	}
	if denseRankCol.GoType != "int" {
		t.Errorf("Expected dense_rank_position to be int (DENSE_RANK never NULL), got %s", denseRankCol.GoType)
	}
	if denseRankCol.IsNullable {
		t.Errorf("Expected dense_rank_position to be NOT NULL (DENSE_RANK always returns non-null)")
	}
}

// TestResultTypes_WindowFunction_LagLead tests that LAG() and LEAD() are always nullable
func TestResultTypes_WindowFunction_LagLead(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testDB := getTestDB(t)
	ctx := context.Background()

	tempDir := t.TempDir()
	sqlDir := filepath.Join(tempDir, "queries")
	err := os.MkdirAll(sqlDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create SQL directory: %v", err)
	}

	querySQL := `-- name: GetPostsWithLagLead :many
SELECT
    id,
    title,
    published_at,
    LAG(published_at) OVER (ORDER BY published_at) as prev_published_at,
    LEAD(published_at) OVER (ORDER BY published_at) as next_published_at,
    LAG(view_count) OVER (ORDER BY published_at) as prev_view_count,
    LEAD(view_count) OVER (ORDER BY published_at) as next_view_count
FROM posts
WHERE status = 'published'
ORDER BY published_at;`

	err = os.WriteFile(filepath.Join(sqlDir, "posts.sql"), []byte(querySQL), 0644)
	if err != nil {
		t.Fatalf("Failed to write test query: %v", err)
	}

	parser := NewQueryParser(sqlDir)
	queries, err := parser.ParseQueries()
	if err != nil {
		t.Fatalf("Failed to parse queries: %v", err)
	}

	analyzer := NewQueryAnalyzer(testDB)
	err = analyzer.AnalyzeQuery(ctx, &queries[0])
	if err != nil {
		t.Fatalf("Failed to analyze query: %v", err)
	}

	query := queries[0]
	if len(query.Columns) != 7 {
		t.Fatalf("Expected 7 columns, got %d", len(query.Columns))
	}

	prevPublishedCol := findColumn(query.Columns, "prev_published_at")
	if prevPublishedCol == nil {
		t.Fatal("prev_published_at column not found")
	}
	if !prevPublishedCol.IsNullable {
		t.Errorf("Expected prev_published_at to be nullable (LAG can return NULL at boundaries)")
	}
	if prevPublishedCol.GoType != "*time.Time" {
		t.Errorf("Expected prev_published_at to be *time.Time, got %s", prevPublishedCol.GoType)
	}

	nextPublishedCol := findColumn(query.Columns, "next_published_at")
	if nextPublishedCol == nil {
		t.Fatal("next_published_at column not found")
	}
	if !nextPublishedCol.IsNullable {
		t.Errorf("Expected next_published_at to be nullable (LEAD can return NULL at boundaries)")
	}
	if nextPublishedCol.GoType != "*time.Time" {
		t.Errorf("Expected next_published_at to be *time.Time, got %s", nextPublishedCol.GoType)
	}

	prevViewCountCol := findColumn(query.Columns, "prev_view_count")
	if prevViewCountCol == nil {
		t.Fatal("prev_view_count column not found")
	}
	if !prevViewCountCol.IsNullable {
		t.Errorf("Expected prev_view_count to be nullable (LAG can return NULL at boundaries)")
	}
	if prevViewCountCol.GoType != "*int" {
		t.Errorf("Expected prev_view_count to be *int, got %s", prevViewCountCol.GoType)
	}

	nextViewCountCol := findColumn(query.Columns, "next_view_count")
	if nextViewCountCol == nil {
		t.Fatal("next_view_count column not found")
	}
	if !nextViewCountCol.IsNullable {
		t.Errorf("Expected next_view_count to be nullable (LEAD can return NULL at boundaries)")
	}
	if nextViewCountCol.GoType != "*int" {
		t.Errorf("Expected next_view_count to be *int, got %s", nextViewCountCol.GoType)
	}
}

// TestResultTypes_WindowFunction_Aggregates tests window aggregates maintain proper nullability
func TestResultTypes_WindowFunction_Aggregates(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testDB := getTestDB(t)
	ctx := context.Background()

	tempDir := t.TempDir()
	sqlDir := filepath.Join(tempDir, "queries")
	err := os.MkdirAll(sqlDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create SQL directory: %v", err)
	}

	querySQL := `-- name: GetPostsWithWindowAggregates :many
SELECT
    id,
    title,
    view_count,
    COUNT(*) OVER (PARTITION BY user_id) as user_post_count,
    SUM(view_count) OVER (PARTITION BY user_id) as user_total_views,
    AVG(like_count) OVER (PARTITION BY user_id) as user_avg_likes,
    MAX(published_at) OVER (PARTITION BY user_id) as user_latest_post
FROM posts
WHERE status = 'published';`

	err = os.WriteFile(filepath.Join(sqlDir, "posts.sql"), []byte(querySQL), 0644)
	if err != nil {
		t.Fatalf("Failed to write test query: %v", err)
	}

	parser := NewQueryParser(sqlDir)
	queries, err := parser.ParseQueries()
	if err != nil {
		t.Fatalf("Failed to parse queries: %v", err)
	}

	analyzer := NewQueryAnalyzer(testDB)
	err = analyzer.AnalyzeQuery(ctx, &queries[0])
	if err != nil {
		t.Fatalf("Failed to analyze query: %v", err)
	}

	query := queries[0]
	if len(query.Columns) != 7 {
		t.Fatalf("Expected 7 columns, got %d", len(query.Columns))
	}

	countCol := findColumn(query.Columns, "user_post_count")
	if countCol == nil {
		t.Fatal("user_post_count column not found")
	}
	if countCol.GoType != "int" {
		t.Errorf("Expected user_post_count to be int (COUNT never NULL), got %s", countCol.GoType)
	}
	if countCol.IsNullable {
		t.Errorf("Expected user_post_count to be NOT NULL (COUNT always returns non-null)")
	}

	sumCol := findColumn(query.Columns, "user_total_views")
	if sumCol == nil {
		t.Fatal("user_total_views column not found")
	}
	if !sumCol.IsNullable {
		t.Errorf("Expected user_total_views to be nullable (SUM can return NULL)")
	}

	avgCol := findColumn(query.Columns, "user_avg_likes")
	if avgCol == nil {
		t.Fatal("user_avg_likes column not found")
	}
	if !avgCol.IsNullable {
		t.Errorf("Expected user_avg_likes to be nullable (AVG can return NULL)")
	}

	maxCol := findColumn(query.Columns, "user_latest_post")
	if maxCol == nil {
		t.Fatal("user_latest_post column not found")
	}
	if !maxCol.IsNullable {
		t.Errorf("Expected user_latest_post to be nullable (MAX can return NULL)")
	}
}

// TestResultTypes_WindowFunction_MixedColumns tests that mixing window functions with regular columns works correctly
func TestResultTypes_WindowFunction_MixedColumns(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testDB := getTestDB(t)
	ctx := context.Background()

	tempDir := t.TempDir()
	sqlDir := filepath.Join(tempDir, "queries")
	err := os.MkdirAll(sqlDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create SQL directory: %v", err)
	}

	querySQL := `-- name: GetPostAnalytics :many
SELECT
    p.id,
    p.title,
    p.view_count,
    p.published_at,
    u.name as author_name,
    ROW_NUMBER() OVER (PARTITION BY p.user_id ORDER BY p.view_count DESC) as rank_in_user_posts,
    LAG(p.title) OVER (PARTITION BY p.user_id ORDER BY p.published_at) as prev_post_title,
    COUNT(*) OVER (PARTITION BY p.user_id) as user_total_posts,
    AVG(p.like_count) OVER (PARTITION BY p.user_id) as user_avg_likes
FROM posts p
INNER JOIN users u ON p.user_id = u.id
WHERE p.status = 'published'
ORDER BY p.user_id, p.view_count DESC;`

	err = os.WriteFile(filepath.Join(sqlDir, "posts.sql"), []byte(querySQL), 0644)
	if err != nil {
		t.Fatalf("Failed to write test query: %v", err)
	}

	parser := NewQueryParser(sqlDir)
	queries, err := parser.ParseQueries()
	if err != nil {
		t.Fatalf("Failed to parse queries: %v", err)
	}

	analyzer := NewQueryAnalyzer(testDB)
	err = analyzer.AnalyzeQuery(ctx, &queries[0])
	if err != nil {
		t.Fatalf("Failed to analyze query: %v", err)
	}

	query := queries[0]
	if len(query.Columns) != 9 {
		t.Fatalf("Expected 9 columns, got %d", len(query.Columns))
	}

	idCol := findColumn(query.Columns, "id")
	if idCol == nil {
		t.Fatal("id column not found")
	}
	if idCol.IsNullable {
		t.Errorf("Expected id to be NOT NULL")
	}
	if idCol.GoType != "uuid.UUID" {
		t.Errorf("Expected id to be uuid.UUID, got %s", idCol.GoType)
	}

	titleCol := findColumn(query.Columns, "title")
	if titleCol == nil {
		t.Fatal("title column not found")
	}
	if titleCol.IsNullable {
		t.Errorf("Expected title to be NOT NULL")
	}

	viewCountCol := findColumn(query.Columns, "view_count")
	if viewCountCol == nil {
		t.Fatal("view_count column not found")
	}
	if viewCountCol.IsNullable {
		t.Errorf("Expected view_count to be NOT NULL")
	}

	publishedAtCol := findColumn(query.Columns, "published_at")
	if publishedAtCol == nil {
		t.Fatal("published_at column not found")
	}
	if !publishedAtCol.IsNullable {
		t.Errorf("Expected published_at to be nullable")
	}

	authorNameCol := findColumn(query.Columns, "author_name")
	if authorNameCol == nil {
		t.Fatal("author_name column not found")
	}
	if authorNameCol.IsNullable {
		t.Errorf("Expected author_name to be NOT NULL (from INNER JOIN)")
	}

	rankCol := findColumn(query.Columns, "rank_in_user_posts")
	if rankCol == nil {
		t.Fatal("rank_in_user_posts column not found")
	}
	if rankCol.IsNullable {
		t.Errorf("Expected rank_in_user_posts to be NOT NULL")
	}
	if rankCol.GoType != "int" {
		t.Errorf("Expected rank_in_user_posts to be int, got %s", rankCol.GoType)
	}

	prevTitleCol := findColumn(query.Columns, "prev_post_title")
	if prevTitleCol == nil {
		t.Fatal("prev_post_title column not found")
	}
	if !prevTitleCol.IsNullable {
		t.Errorf("Expected prev_post_title to be nullable (LAG can return NULL)")
	}

	totalPostsCol := findColumn(query.Columns, "user_total_posts")
	if totalPostsCol == nil {
		t.Fatal("user_total_posts column not found")
	}
	if totalPostsCol.IsNullable {
		t.Errorf("Expected user_total_posts to be NOT NULL")
	}

	avgLikesCol := findColumn(query.Columns, "user_avg_likes")
	if avgLikesCol == nil {
		t.Fatal("user_avg_likes column not found")
	}
	if !avgLikesCol.IsNullable {
		t.Errorf("Expected user_avg_likes to be nullable (AVG can return NULL)")
	}
}

// TestResultTypes_SimpleCTE tests that CTEs maintain column nullability through WITH clauses
func TestResultTypes_SimpleCTE(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testDB := getTestDB(t)
	ctx := context.Background()

	tempDir := t.TempDir()
	sqlDir := filepath.Join(tempDir, "queries")
	err := os.MkdirAll(sqlDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create SQL directory: %v", err)
	}

	querySQL := `-- name: GetActiveUsers :many
WITH active_users AS (
    SELECT id, name, email, metadata
    FROM users
    WHERE is_active = true
)
SELECT id, name, email, metadata
FROM active_users;`

	err = os.WriteFile(filepath.Join(sqlDir, "users.sql"), []byte(querySQL), 0644)
	if err != nil {
		t.Fatalf("Failed to write test query: %v", err)
	}

	parser := NewQueryParser(sqlDir)
	queries, err := parser.ParseQueries()
	if err != nil {
		t.Fatalf("Failed to parse queries: %v", err)
	}

	if len(queries) != 1 {
		t.Fatalf("Expected 1 query, got %d", len(queries))
	}

	analyzer := NewQueryAnalyzer(testDB)
	err = analyzer.AnalyzeQuery(ctx, &queries[0])
	if err != nil {
		t.Fatalf("Failed to analyze query: %v", err)
	}

	query := queries[0]
	if len(query.Columns) != 4 {
		t.Fatalf("Expected 4 columns, got %d", len(query.Columns))
	}

	idCol := findColumn(query.Columns, "id")
	if idCol == nil {
		t.Fatal("id column not found")
	}
	if idCol.GoType != "uuid.UUID" {
		t.Errorf("Expected id to be uuid.UUID (NOT NULL through CTE), got %s", idCol.GoType)
	}
	if idCol.IsNullable {
		t.Errorf("Expected id to be NOT NULL through CTE")
	}

	nameCol := findColumn(query.Columns, "name")
	if nameCol == nil {
		t.Fatal("name column not found")
	}
	if nameCol.GoType != "string" {
		t.Errorf("Expected name to be string (NOT NULL through CTE), got %s", nameCol.GoType)
	}
	if nameCol.IsNullable {
		t.Errorf("Expected name to be NOT NULL through CTE")
	}

	metadataCol := findColumn(query.Columns, "metadata")
	if metadataCol == nil {
		t.Fatal("metadata column not found")
	}
	if !metadataCol.IsNullable {
		t.Errorf("Expected metadata to remain nullable through CTE")
	}
}

// TestResultTypes_SubqueryInFrom tests that subqueries in FROM clause maintain column metadata
func TestResultTypes_SubqueryInFrom(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testDB := getTestDB(t)
	ctx := context.Background()

	tempDir := t.TempDir()
	sqlDir := filepath.Join(tempDir, "queries")
	err := os.MkdirAll(sqlDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create SQL directory: %v", err)
	}

	querySQL := `-- name: GetUserWithPostCount :many
SELECT
    u.id,
    u.name,
    u.email,
    post_stats.post_count,
    post_stats.published_count
FROM users u
INNER JOIN (
    SELECT
        user_id,
        COUNT(*) as post_count,
        COUNT(*) FILTER (WHERE status = 'published') as published_count
    FROM posts
    GROUP BY user_id
) post_stats ON u.id = post_stats.user_id;`

	err = os.WriteFile(filepath.Join(sqlDir, "users.sql"), []byte(querySQL), 0644)
	if err != nil {
		t.Fatalf("Failed to write test query: %v", err)
	}

	parser := NewQueryParser(sqlDir)
	queries, err := parser.ParseQueries()
	if err != nil {
		t.Fatalf("Failed to parse queries: %v", err)
	}

	analyzer := NewQueryAnalyzer(testDB)
	err = analyzer.AnalyzeQuery(ctx, &queries[0])
	if err != nil {
		t.Fatalf("Failed to analyze query: %v", err)
	}

	query := queries[0]
	if len(query.Columns) != 5 {
		t.Fatalf("Expected 5 columns, got %d", len(query.Columns))
	}

	idCol := findColumn(query.Columns, "id")
	if idCol == nil {
		t.Fatal("id column not found")
	}
	if idCol.GoType != "uuid.UUID" {
		t.Errorf("Expected id to be uuid.UUID (NOT NULL through subquery), got %s", idCol.GoType)
	}
	if idCol.IsNullable {
		t.Errorf("Expected id to be NOT NULL through subquery")
	}

	postCountCol := findColumn(query.Columns, "post_count")
	if postCountCol == nil {
		t.Fatal("post_count column not found")
	}
	if postCountCol.GoType != "int" {
		t.Errorf("Expected post_count to be int (COUNT never NULL), got %s", postCountCol.GoType)
	}
	if postCountCol.IsNullable {
		t.Errorf("Expected post_count to be NOT NULL (COUNT aggregates never return NULL)")
	}

	publishedCountCol := findColumn(query.Columns, "published_count")
	if publishedCountCol == nil {
		t.Fatal("published_count column not found")
	}
	if publishedCountCol.GoType != "int" {
		t.Errorf("Expected published_count to be int (COUNT with FILTER never NULL), got %s", publishedCountCol.GoType)
	}
	if publishedCountCol.IsNullable {
		t.Errorf("Expected published_count to be NOT NULL")
	}
}

// TestResultTypes_CTEWithAggregates tests that CTEs with aggregates maintain proper nullability
func TestResultTypes_CTEWithAggregates(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testDB := getTestDB(t)
	ctx := context.Background()

	tempDir := t.TempDir()
	sqlDir := filepath.Join(tempDir, "queries")
	err := os.MkdirAll(sqlDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create SQL directory: %v", err)
	}

	querySQL := `-- name: GetPaymentLinkWithStats :one
WITH payment_stats AS (
    SELECT
        payment_link_id,
        COUNT(*) as payment_count,
        SUM(amount) as total_amount,
        AVG(amount) as avg_amount,
        MAX(amount) as max_amount
    FROM payments
    WHERE payment_link_id = $1
    GROUP BY payment_link_id
)
SELECT
    pl.id,
    pl.status,
    pl.description,
    COALESCE(ps.payment_count, 0) as payment_count,
    ps.total_amount,
    ps.avg_amount,
    ps.max_amount
FROM payment_links pl
LEFT JOIN payment_stats ps ON pl.id = ps.payment_link_id
WHERE pl.id = $1;`

	err = os.WriteFile(filepath.Join(sqlDir, "payment_links.sql"), []byte(querySQL), 0644)
	if err != nil {
		t.Fatalf("Failed to write test query: %v", err)
	}

	parser := NewQueryParser(sqlDir)
	queries, err := parser.ParseQueries()
	if err != nil {
		t.Fatalf("Failed to parse queries: %v", err)
	}

	analyzer := NewQueryAnalyzer(testDB)
	err = analyzer.AnalyzeQuery(ctx, &queries[0])
	if err != nil {
		t.Fatalf("Failed to analyze query: %v", err)
	}

	query := queries[0]
	if len(query.Columns) != 7 {
		t.Fatalf("Expected 7 columns, got %d", len(query.Columns))
	}

	idCol := findColumn(query.Columns, "id")
	if idCol == nil {
		t.Fatal("id column not found")
	}
	if idCol.GoType != "uuid.UUID" {
		t.Errorf("Expected id to be uuid.UUID (NOT NULL), got %s", idCol.GoType)
	}

	descCol := findColumn(query.Columns, "description")
	if descCol == nil {
		t.Fatal("description column not found")
	}
	if !descCol.IsNullable {
		t.Errorf("Expected description to remain nullable")
	}

	paymentCountCol := findColumn(query.Columns, "payment_count")
	if paymentCountCol == nil {
		t.Fatal("payment_count column not found")
	}
	if paymentCountCol.GoType != "int" {
		t.Errorf("Expected payment_count to be int (COALESCE makes NOT NULL), got %s", paymentCountCol.GoType)
	}
	if paymentCountCol.IsNullable {
		t.Errorf("Expected payment_count to be NOT NULL (COALESCE eliminates NULL)")
	}

	totalAmountCol := findColumn(query.Columns, "total_amount")
	if totalAmountCol == nil {
		t.Fatal("total_amount column not found")
	}
	if !totalAmountCol.IsNullable {
		t.Errorf("Expected total_amount to be nullable (SUM from LEFT JOIN)")
	}

	avgAmountCol := findColumn(query.Columns, "avg_amount")
	if avgAmountCol == nil {
		t.Fatal("avg_amount column not found")
	}
	if !avgAmountCol.IsNullable {
		t.Errorf("Expected avg_amount to be nullable (AVG from LEFT JOIN)")
	}

	maxAmountCol := findColumn(query.Columns, "max_amount")
	if maxAmountCol == nil {
		t.Fatal("max_amount column not found")
	}
	if !maxAmountCol.IsNullable {
		t.Errorf("Expected max_amount to be nullable (MAX from LEFT JOIN)")
	}
}

// TestResultTypes_MultipleCTEs tests that chained CTEs maintain column metadata correctly
func TestResultTypes_MultipleCTEs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testDB := getTestDB(t)
	ctx := context.Background()

	tempDir := t.TempDir()
	sqlDir := filepath.Join(tempDir, "queries")
	err := os.MkdirAll(sqlDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create SQL directory: %v", err)
	}

	querySQL := `-- name: GetPostWithAuthorAndCommentStats :one
WITH post_data AS (
    SELECT
        p.id,
        p.title,
        p.content,
        p.user_id as author_id,
        p.published_at
    FROM posts p
    WHERE p.id = $1
),
comment_stats AS (
    SELECT
        post_id,
        COUNT(*) as comment_count,
        COUNT(*) FILTER (WHERE is_approved = true) as approved_count
    FROM comments
    WHERE post_id = $1
    GROUP BY post_id
),
author_data AS (
    SELECT
        u.id,
        u.name as author_name,
        u.email as author_email
    FROM users u
    INNER JOIN post_data pd ON u.id = pd.author_id
)
SELECT
    pd.id,
    pd.title,
    pd.content,
    pd.published_at,
    ad.author_name,
    ad.author_email,
    COALESCE(cs.comment_count, 0) as comment_count,
    COALESCE(cs.approved_count, 0) as approved_count
FROM post_data pd
INNER JOIN author_data ad ON ad.id = pd.author_id
LEFT JOIN comment_stats cs ON cs.post_id = pd.id;`

	err = os.WriteFile(filepath.Join(sqlDir, "posts.sql"), []byte(querySQL), 0644)
	if err != nil {
		t.Fatalf("Failed to write test query: %v", err)
	}

	parser := NewQueryParser(sqlDir)
	queries, err := parser.ParseQueries()
	if err != nil {
		t.Fatalf("Failed to parse queries: %v", err)
	}

	analyzer := NewQueryAnalyzer(testDB)
	err = analyzer.AnalyzeQuery(ctx, &queries[0])
	if err != nil {
		t.Fatalf("Failed to analyze query: %v", err)
	}

	query := queries[0]
	if len(query.Columns) != 8 {
		t.Fatalf("Expected 8 columns, got %d", len(query.Columns))
	}

	idCol := findColumn(query.Columns, "id")
	if idCol == nil {
		t.Fatal("id column not found")
	}
	if idCol.GoType != "uuid.UUID" {
		t.Errorf("Expected id to be uuid.UUID (NOT NULL through multiple CTEs), got %s", idCol.GoType)
	}
	if idCol.IsNullable {
		t.Errorf("Expected id to be NOT NULL through multiple CTEs")
	}

	titleCol := findColumn(query.Columns, "title")
	if titleCol == nil {
		t.Fatal("title column not found")
	}
	if titleCol.GoType != "string" {
		t.Errorf("Expected title to be string (NOT NULL), got %s", titleCol.GoType)
	}
	if titleCol.IsNullable {
		t.Errorf("Expected title to be NOT NULL through CTE chain")
	}

	publishedAtCol := findColumn(query.Columns, "published_at")
	if publishedAtCol == nil {
		t.Fatal("published_at column not found")
	}
	if !publishedAtCol.IsNullable {
		t.Errorf("Expected published_at to remain nullable through CTE chain")
	}

	authorNameCol := findColumn(query.Columns, "author_name")
	if authorNameCol == nil {
		t.Fatal("author_name column not found")
	}
	if authorNameCol.GoType != "string" {
		t.Errorf("Expected author_name to be string (aliased from users.name NOT NULL), got %s", authorNameCol.GoType)
	}
	if authorNameCol.IsNullable {
		t.Errorf("Expected author_name to be NOT NULL through INNER JOIN")
	}

	commentCountCol := findColumn(query.Columns, "comment_count")
	if commentCountCol == nil {
		t.Fatal("comment_count column not found")
	}
	if commentCountCol.GoType != "int" {
		t.Errorf("Expected comment_count to be int (COALESCE makes NOT NULL), got %s", commentCountCol.GoType)
	}
	if commentCountCol.IsNullable {
		t.Errorf("Expected comment_count to be NOT NULL (COALESCE with 0)")
	}

	approvedCountCol := findColumn(query.Columns, "approved_count")
	if approvedCountCol == nil {
		t.Fatal("approved_count column not found")
	}
	if approvedCountCol.GoType != "int" {
		t.Errorf("Expected approved_count to be int (COALESCE makes NOT NULL), got %s", approvedCountCol.GoType)
	}
	if approvedCountCol.IsNullable {
		t.Errorf("Expected approved_count to be NOT NULL (COALESCE with 0)")
	}
}

// TestResultTypes_CTEWithJoins tests complex CTEs with JOINs maintain correct nullability
func TestResultTypes_CTEWithJoins(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testDB := getTestDB(t)
	ctx := context.Background()

	tempDir := t.TempDir()
	sqlDir := filepath.Join(tempDir, "queries")
	err := os.MkdirAll(sqlDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create SQL directory: %v", err)
	}

	querySQL := `-- name: GetUserPostsWithComments :many
WITH user_posts AS (
    SELECT
        p.id,
        p.title,
        p.content,
        p.user_id,
        p.published_at,
        u.name as author_name,
        u.email as author_email
    FROM posts p
    INNER JOIN users u ON p.user_id = u.id
    WHERE u.id = $1
)
SELECT
    up.id,
    up.title,
    up.published_at,
    up.author_name,
    c.id as comment_id,
    c.content as comment_content,
    c.is_approved
FROM user_posts up
LEFT JOIN comments c ON c.post_id = up.id
ORDER BY up.id, c.created_at;`

	err = os.WriteFile(filepath.Join(sqlDir, "posts.sql"), []byte(querySQL), 0644)
	if err != nil {
		t.Fatalf("Failed to write test query: %v", err)
	}

	parser := NewQueryParser(sqlDir)
	queries, err := parser.ParseQueries()
	if err != nil {
		t.Fatalf("Failed to parse queries: %v", err)
	}

	analyzer := NewQueryAnalyzer(testDB)
	err = analyzer.AnalyzeQuery(ctx, &queries[0])
	if err != nil {
		t.Fatalf("Failed to analyze query: %v", err)
	}

	query := queries[0]
	if len(query.Columns) != 7 {
		t.Fatalf("Expected 7 columns, got %d", len(query.Columns))
	}

	idCol := findColumn(query.Columns, "id")
	if idCol == nil {
		t.Fatal("id column not found")
	}
	if idCol.GoType != "uuid.UUID" {
		t.Errorf("Expected id to be uuid.UUID (NOT NULL from CTE with INNER JOIN), got %s", idCol.GoType)
	}
	if idCol.IsNullable {
		t.Errorf("Expected id to be NOT NULL from CTE with INNER JOIN")
	}

	titleCol := findColumn(query.Columns, "title")
	if titleCol == nil {
		t.Fatal("title column not found")
	}
	if titleCol.GoType != "string" {
		t.Errorf("Expected title to be string (NOT NULL), got %s", titleCol.GoType)
	}
	if titleCol.IsNullable {
		t.Errorf("Expected title to be NOT NULL through CTE")
	}

	publishedAtCol := findColumn(query.Columns, "published_at")
	if publishedAtCol == nil {
		t.Fatal("published_at column not found")
	}
	if !publishedAtCol.IsNullable {
		t.Errorf("Expected published_at to remain nullable through CTE")
	}

	authorNameCol := findColumn(query.Columns, "author_name")
	if authorNameCol == nil {
		t.Fatal("author_name column not found")
	}
	if authorNameCol.GoType != "string" {
		t.Errorf("Expected author_name to be string (from INNER JOIN in CTE), got %s", authorNameCol.GoType)
	}
	if authorNameCol.IsNullable {
		t.Errorf("Expected author_name to be NOT NULL (from INNER JOIN in CTE)")
	}

	commentIdCol := findColumn(query.Columns, "comment_id")
	if commentIdCol == nil {
		t.Fatal("comment_id column not found")
	}
	if !commentIdCol.IsNullable {
		t.Errorf("Expected comment_id to be nullable (from LEFT JOIN)")
	}

	commentContentCol := findColumn(query.Columns, "comment_content")
	if commentContentCol == nil {
		t.Fatal("comment_content column not found")
	}
	if !commentContentCol.IsNullable {
		t.Errorf("Expected comment_content to be nullable (from LEFT JOIN)")
	}

	isApprovedCol := findColumn(query.Columns, "is_approved")
	if isApprovedCol == nil {
		t.Fatal("is_approved column not found")
	}
	if !isApprovedCol.IsNullable {
		t.Errorf("Expected is_approved to be nullable (from LEFT JOIN)")
	}
}

// TestResultTypes_AnnotationOverridesAutoDetection tests that -- result: annotations override automatic type detection
func TestResultTypes_AnnotationOverridesAutoDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testDB := getTestDB(t)
	ctx := context.Background()

	tempDir := t.TempDir()
	sqlDir := filepath.Join(tempDir, "queries")
	err := os.MkdirAll(sqlDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create SQL directory: %v", err)
	}

	querySQL := `-- name: GetPostSummary :one
-- result: total_views int
SELECT
    COUNT(*) as post_count,
    SUM(view_count) as total_views
FROM posts
WHERE user_id = $1;`

	err = os.WriteFile(filepath.Join(sqlDir, "payments.sql"), []byte(querySQL), 0644)
	if err != nil {
		t.Fatalf("Failed to write test query: %v", err)
	}

	parser := NewQueryParser(sqlDir)
	queries, err := parser.ParseQueries()
	if err != nil {
		t.Fatalf("Failed to parse queries: %v", err)
	}

	if len(queries) != 1 {
		t.Fatalf("Expected 1 query, got %d", len(queries))
	}

	analyzer := NewQueryAnalyzer(testDB)
	err = analyzer.AnalyzeQuery(ctx, &queries[0])
	if err != nil {
		t.Fatalf("Failed to analyze query: %v", err)
	}

	query := queries[0]
	if len(query.Columns) != 2 {
		t.Fatalf("Expected 2 columns, got %d", len(query.Columns))
	}

	postCountCol := findColumn(query.Columns, "post_count")
	if postCountCol == nil {
		t.Fatal("post_count column not found")
	}
	if postCountCol.GoType != "int" {
		t.Errorf("Expected post_count to be int (COUNT auto-detection), got %s", postCountCol.GoType)
	}
	if postCountCol.IsNullable {
		t.Errorf("Expected post_count to be NOT NULL (COUNT never NULL)")
	}

	totalViewsCol := findColumn(query.Columns, "total_views")
	if totalViewsCol == nil {
		t.Fatal("total_views column not found")
	}
	if totalViewsCol.GoType != "int" {
		t.Errorf("Expected total_views to be int (overridden from *int by annotation), got %s", totalViewsCol.GoType)
	}
	if totalViewsCol.IsNullable {
		t.Errorf("Expected total_views to be NOT NULL (forced by annotation)")
	}
}

// TestResultTypes_MixAnnotatedAndUnannotated tests mixing annotated and unannotated columns
func TestResultTypes_MixAnnotatedAndUnannotated(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testDB := getTestDB(t)
	ctx := context.Background()

	tempDir := t.TempDir()
	sqlDir := filepath.Join(tempDir, "queries")
	err := os.MkdirAll(sqlDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create SQL directory: %v", err)
	}

	querySQL := `-- name: GetPublishedPosts :many
-- result: status string
SELECT status, published_at
FROM posts
WHERE user_id = $1;`

	err = os.WriteFile(filepath.Join(sqlDir, "posts.sql"), []byte(querySQL), 0644)
	if err != nil {
		t.Fatalf("Failed to write test query: %v", err)
	}

	parser := NewQueryParser(sqlDir)
	queries, err := parser.ParseQueries()
	if err != nil {
		t.Fatalf("Failed to parse queries: %v", err)
	}

	if len(queries) != 1 {
		t.Fatalf("Expected 1 query, got %d", len(queries))
	}

	analyzer := NewQueryAnalyzer(testDB)
	err = analyzer.AnalyzeQuery(ctx, &queries[0])
	if err != nil {
		t.Fatalf("Failed to analyze query: %v", err)
	}

	query := queries[0]
	if len(query.Columns) != 2 {
		t.Fatalf("Expected 2 columns, got %d", len(query.Columns))
	}

	statusCol := findColumn(query.Columns, "status")
	if statusCol == nil {
		t.Fatal("status column not found")
	}
	if statusCol.GoType != "string" {
		t.Errorf("Expected status to be string (overridden by annotation), got %s", statusCol.GoType)
	}
	if statusCol.IsNullable {
		t.Errorf("Expected status to be NOT NULL (string annotation without pointer)")
	}

	publishedAtCol := findColumn(query.Columns, "published_at")
	if publishedAtCol == nil {
		t.Fatal("published_at column not found")
	}
	if publishedAtCol.GoType != "*time.Time" {
		t.Errorf("Expected published_at to be *time.Time (auto-detected, nullable), got %s", publishedAtCol.GoType)
	}
	if !publishedAtCol.IsNullable {
		t.Errorf("Expected published_at to be nullable (no annotation, auto-detected)")
	}
}

// TestResultTypes_AnnotationNonExistentColumn tests annotation for column not in SELECT
func TestResultTypes_AnnotationNonExistentColumn(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testDB := getTestDB(t)
	ctx := context.Background()

	tempDir := t.TempDir()
	sqlDir := filepath.Join(tempDir, "queries")
	err := os.MkdirAll(sqlDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create SQL directory: %v", err)
	}

	querySQL := `-- name: GetUserBasic :one
-- result: email string
SELECT id, name
FROM users
WHERE id = $1;`

	err = os.WriteFile(filepath.Join(sqlDir, "users.sql"), []byte(querySQL), 0644)
	if err != nil {
		t.Fatalf("Failed to write test query: %v", err)
	}

	parser := NewQueryParser(sqlDir)
	queries, err := parser.ParseQueries()
	if err != nil {
		t.Fatalf("Failed to parse queries: %v", err)
	}

	if len(queries) != 1 {
		t.Fatalf("Expected 1 query, got %d", len(queries))
	}

	analyzer := NewQueryAnalyzer(testDB)
	err = analyzer.AnalyzeQuery(ctx, &queries[0])
	if err == nil {
		t.Fatal("Expected error for annotation on non-existent column, got nil")
	}

	expectedErrMsg := "email"
	if !contains(err.Error(), expectedErrMsg) {
		t.Errorf("Expected error message to contain '%s', got: %s", expectedErrMsg, err.Error())
	}
}

// TestResultTypes_IntegerTypesUseInt tests that all PostgreSQL integer types map to 'int'
// This verifies that intelligent result types use ergonomic 'int' instead of sized integers
func TestResultTypes_IntegerTypesUseInt(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testDB := getTestDB(t)
	ctx := context.Background()

	tempDir := t.TempDir()
	sqlDir := filepath.Join(tempDir, "queries")
	err := os.MkdirAll(sqlDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create SQL directory: %v", err)
	}

	// Query selecting different integer types from posts table
	// view_count is INTEGER (int4), like_count is INTEGER (int4)
	querySQL := `-- name: GetPostCounts :one
SELECT view_count, like_count FROM posts WHERE id = $1;`

	err = os.WriteFile(filepath.Join(sqlDir, "posts.sql"), []byte(querySQL), 0644)
	if err != nil {
		t.Fatalf("Failed to write test query: %v", err)
	}

	parser := NewQueryParser(sqlDir)
	queries, err := parser.ParseQueries()
	if err != nil {
		t.Fatalf("Failed to parse queries: %v", err)
	}

	if len(queries) != 1 {
		t.Fatalf("Expected 1 query, got %d", len(queries))
	}

	analyzer := NewQueryAnalyzer(testDB)
	err = analyzer.AnalyzeQuery(ctx, &queries[0])
	if err != nil {
		t.Fatalf("Failed to analyze query: %v", err)
	}

	query := queries[0]
	if len(query.Columns) != 2 {
		t.Fatalf("Expected 2 columns, got %d", len(query.Columns))
	}

	// Test: view_count should be *int (nullable integer uses pointer)
	viewCountCol := findColumn(query.Columns, "view_count")
	if viewCountCol == nil {
		t.Fatal("view_count column not found")
	}
	if viewCountCol.GoType != "*int" {
		t.Errorf("Expected view_count to be '*int' (nullable intelligent type), got '%s'", viewCountCol.GoType)
	}
	if !viewCountCol.IsNullable {
		t.Error("Expected view_count to be nullable")
	}

	// Test: like_count should be *int (nullable integer uses pointer)
	likeCountCol := findColumn(query.Columns, "like_count")
	if likeCountCol == nil {
		t.Fatal("like_count column not found")
	}
	if likeCountCol.GoType != "*int" {
		t.Errorf("Expected like_count to be '*int' (nullable intelligent type), got '%s'", likeCountCol.GoType)
	}
	if !likeCountCol.IsNullable {
		t.Error("Expected like_count to be nullable")
	}
}
