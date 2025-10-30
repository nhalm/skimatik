package generator

import (
	"testing"
)

func TestINClauseDebug(t *testing.T) {
	parser := NewSQLParser()
	info, err := parser.Parse("SELECT * FROM users WHERE status IN ($1)")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	t.Logf("Found %d parameters", len(info.Parameters))
	for _, param := range info.Parameters {
		t.Logf("  Param $%d: column=%s, operator=%s, InWhere=%v", 
			param.Position, param.ColumnName, param.Operator, param.IsInWhere)
	}
	if len(info.Parameters) == 0 {
		t.Error("Expected to find 1 parameter, got 0")
	}
}
