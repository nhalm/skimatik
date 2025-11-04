package generator

import (
	"testing"
)

func TestSQLParser_DetectCoalesce(t *testing.T) {
	sql := `SELECT
    id,
    name,
    COALESCE(age, 0) as age_with_default,
    COALESCE(balance, 0.0) as balance_with_default,
    age as age_nullable
FROM users
WHERE id = $1`

	parser := NewSQLParser()
	info, err := parser.Parse(sql)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if len(info.SelectTargets) != 5 {
		t.Fatalf("Expected 5 select targets, got %d", len(info.SelectTargets))
	}

	// Debug: print all targets
	for i, target := range info.SelectTargets {
		t.Logf("Target %d: Alias=%s, IsCoalesce=%v, HasNonNullLiteral=%v",
			i, target.Alias, target.IsCoalesce, target.HasNonNullLiteral)
	}

	// Check age_with_default
	var ageTarget *SelectTarget
	for i := range info.SelectTargets {
		if info.SelectTargets[i].Alias == "age_with_default" {
			ageTarget = &info.SelectTargets[i]
			break
		}
	}

	if ageTarget == nil {
		t.Fatal("age_with_default target not found")
	}

	if !ageTarget.IsCoalesce {
		t.Error("age_with_default should be detected as COALESCE")
	}

	if !ageTarget.HasNonNullLiteral {
		t.Error("age_with_default should have non-null literal (0)")
	}
}
