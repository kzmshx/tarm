package tarm

import (
	"path/filepath"
	"sort"
	"testing"

	"github.com/kzmshx/tarm/internal/git"
)

func TestRun(t *testing.T) {
	testRoot := filepath.Join("..", "..", "testdata", "terraform")

	tests := []struct {
		name        string
		cfg         Config
		detected    []string
		wantModules []string
		wantErr     bool
	}{
		{
			name:        "explicit changed files",
			cfg:         Config{Root: testRoot, RootModulePatterns: []string{"environments/*/*"}, ChangedFiles: []string{"modules/common/main.tf"}},
			wantModules: []string{"environments/dev/api", "environments/dev/web", "environments/stg/api", "environments/stg/web"},
		},
		{
			name:        "direct root module change",
			cfg:         Config{Root: testRoot, RootModulePatterns: []string{"environments/*/*"}, ChangedFiles: []string{"environments/dev/api/main.tf"}},
			wantModules: []string{"environments/dev/api"},
		},
		{
			name: "no changed files",
			cfg:  Config{Root: testRoot, RootModulePatterns: []string{"environments/*/*"}},
		},
		{
			name:        "exclude module patterns",
			cfg:         Config{Root: testRoot, RootModulePatterns: []string{"environments/*/*"}, ExcludeModulePatterns: []string{"environments/dev/*"}, ChangedFiles: []string{"modules/network/main.tf"}},
			wantModules: []string{"environments/stg/api", "environments/stg/web", "environments/prod/app"},
		},
		{
			name:    "missing patterns returns error",
			cfg:     Config{Root: testRoot},
			wantErr: true,
		},
		{
			name:        "combines detected and explicit",
			cfg:         Config{Root: testRoot, RootModulePatterns: []string{"environments/*/*"}, ChangedFiles: []string{"modules/auth/main.tf"}, DetectChanges: true},
			detected:    []string{"modules/database/main.tf"},
			wantModules: []string{"environments/dev/api", "environments/dev/web", "environments/stg/api", "environments/stg/web"},
		},
		{
			name:        "detect changes with provider",
			cfg:         Config{Root: testRoot, RootModulePatterns: []string{"environments/*/*"}, DetectChanges: true},
			detected:    []string{"modules/network/main.tf"},
			wantModules: []string{"environments/dev/api", "environments/dev/web", "environments/stg/api", "environments/stg/web", "environments/prod/app"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &git.StaticProvider{Files: tt.detected}

			result, err := Run(tt.cfg, provider)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}

			var gotModules []string
			for _, m := range result.AffectedModules {
				gotModules = append(gotModules, m.Path)
			}
			sort.Strings(gotModules)
			sort.Strings(tt.wantModules)

			if len(gotModules) != len(tt.wantModules) {
				t.Errorf("got %v, want %v", gotModules, tt.wantModules)
				return
			}
			for i := range gotModules {
				if gotModules[i] != tt.wantModules[i] {
					t.Errorf("got %v, want %v", gotModules, tt.wantModules)
					return
				}
			}
		})
	}
}
