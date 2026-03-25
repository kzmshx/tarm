package tarm

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFilterPatterns(t *testing.T) {
	testRoot := filepath.Join("..", "..", "testdata", "terraform")

	tests := []struct {
		name            string
		includePatterns []string
		excludePatterns []string
		wantCount       int
		wantContains    []string
		wantAbsent      []string
	}{
		{
			name:            "include all environments",
			includePatterns: []string{"environments/*/*"},
			wantContains:    []string{"environments/dev/api", "environments/dev/web", "environments/stg/api"},
		},
		{
			name:            "exclude modules from broad pattern",
			includePatterns: []string{"environments/*/*", "modules/*"},
			excludePatterns: []string{"modules/*"},
			wantContains:    []string{"environments/dev/api"},
			wantAbsent:      []string{"modules/auth", "modules/network"},
		},
		{
			name:            "no excludes returns all includes",
			includePatterns: []string{"modules/*"},
			wantContains:    []string{"modules/auth", "modules/network"},
		},
		{
			name:            "deduplicates results",
			includePatterns: []string{"environments/dev/*", "environments/dev/*"},
			wantCount:       2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsys := os.DirFS(testRoot)
			got, err := FilterPatterns(fsys, tt.includePatterns, tt.excludePatterns)
			if err != nil {
				t.Fatalf("FilterPatterns() error = %v", err)
			}

			if tt.wantCount > 0 && len(got) != tt.wantCount {
				t.Errorf("got %d results, want %d: %v", len(got), tt.wantCount, got)
			}

			gotSet := map[string]bool{}
			for _, g := range got {
				gotSet[g] = true
			}

			for _, want := range tt.wantContains {
				if !gotSet[want] {
					t.Errorf("missing %q in result %v", want, got)
				}
			}
			for _, absent := range tt.wantAbsent {
				if gotSet[absent] {
					t.Errorf("should not contain %q in result %v", absent, got)
				}
			}
		})
	}
}
