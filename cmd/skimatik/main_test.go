package main

import (
	"testing"
)

func TestGetFullVersion(t *testing.T) {
	tests := []struct {
		name           string
		version        string
		commit         string
		date           string
		expectedSubstr []string
	}{
		{
			name:    "normal commit hash",
			version: "v1.0.0",
			commit:  "abcdef1234567890",
			date:    "2025-11-03",
			expectedSubstr: []string{
				"v1.0.0",
				"commit: abcdef1",
				"built at 2025-11-03",
			},
		},
		{
			name:    "short commit hash",
			version: "v1.0.0",
			commit:  "abc",
			date:    "2025-11-03",
			expectedSubstr: []string{
				"v1.0.0",
				"commit: abc",
				"built at 2025-11-03",
			},
		},
		{
			name:    "no commit (none)",
			version: "v1.0.0",
			commit:  "none",
			date:    "2025-11-03",
			expectedSubstr: []string{
				"v1.0.0",
				"built at 2025-11-03",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalVersion := version
			originalCommit := commit
			originalDate := date
			defer func() {
				version = originalVersion
				commit = originalCommit
				date = originalDate
			}()

			version = tt.version
			commit = tt.commit
			date = tt.date

			result := getFullVersion()

			for _, substr := range tt.expectedSubstr {
				if !contains(result, substr) {
					t.Errorf("getFullVersion() = %q, should contain %q", result, substr)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
