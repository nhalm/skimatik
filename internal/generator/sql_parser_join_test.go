package generator

import (
	"testing"
)

func TestSQLParser_DetectLeftJoin(t *testing.T) {
	sql := `SELECT u.id, u.name, p.title, p.status
FROM users u
LEFT JOIN posts p ON u.id = p.user_id`

	parser := NewSQLParser()
	info, err := parser.Parse(sql)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if len(info.Joins) != 1 {
		t.Fatalf("Expected 1 join, got %d", len(info.Joins))
	}

	join := info.Joins[0]
	if join.Type != JoinTypeLeft {
		t.Errorf("Expected LEFT JOIN, got %v", join.Type)
	}
	if join.LeftTable != "u" {
		t.Errorf("Expected left table 'u', got '%s'", join.LeftTable)
	}
	if join.RightTable != "p" {
		t.Errorf("Expected right table 'p', got '%s'", join.RightTable)
	}
}

func TestSQLParser_DetectMultipleJoins(t *testing.T) {
	sql := `SELECT u.id, p.title, c.content
FROM users u
LEFT JOIN posts p ON u.id = p.user_id
INNER JOIN comments c ON p.id = c.post_id`

	parser := NewSQLParser()
	info, err := parser.Parse(sql)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if len(info.Joins) != 2 {
		t.Fatalf("Expected 2 joins, got %d", len(info.Joins))
	}

	// First join should be LEFT
	if info.Joins[0].Type != JoinTypeLeft {
		t.Errorf("Expected first join to be LEFT, got %v", info.Joins[0].Type)
	}

	// Second join should be INNER
	if info.Joins[1].Type != JoinTypeInner {
		t.Errorf("Expected second join to be INNER, got %v", info.Joins[1].Type)
	}
}
