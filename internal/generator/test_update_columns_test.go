package generator

import (
	"os"
	"strings"
	"testing"
)

func TestCodeGenerator_GeneratedUpdateQuery(t *testing.T) {
	config := getTestConfigWithTempDir(t)
	cg := NewCodeGenerator(config)
	table := getTestTable()

	err := cg.GenerateTableRepository(table)
	if err != nil {
		t.Fatalf("GenerateTableRepository failed: %v", err)
	}

	filename := config.GetOutputPath("users_generated.go")
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	contentStr := string(content)

	if strings.Contains(contentStr, "SET <no value>") {
		t.Fatal("Generated UPDATE query contains '<no value>' placeholder")
	}

	if !strings.Contains(contentStr, "UPDATE users") {
		t.Fatal("Generated file missing UPDATE statement")
	}

	lines := strings.Split(contentStr, "\n")
	updateFound := false
	setFound := false
	for i, line := range lines {
		if strings.Contains(line, "UPDATE users") {
			updateFound = true
			for j := i; j < i+5 && j < len(lines); j++ {
				if strings.Contains(lines[j], "SET") && strings.Contains(lines[j], "=") {
					setFound = true
					break
				}
			}
		}
	}

	if !updateFound {
		t.Fatal("UPDATE statement not found in generated code")
	}

	if !setFound {
		t.Fatal("SET clause with assignments not found in UPDATE statement")
	}
}
