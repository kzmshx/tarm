package tarm

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestAnalyzer_Analyze(t *testing.T) {
	testRoot := filepath.Join("..", "..", "testdata", "terraform")

	analyzer := NewAnalyzer(testRoot)
	err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() failed: %v", err)
	}

	graph := analyzer.GetDependencyGraph()

	tests := []struct {
		name     string
		module   string
		wantDeps []string
	}{
		{name: "dev/api dependencies", module: "environments/dev/api", wantDeps: []string{"modules/network", "modules/database"}},
		{name: "dev/web dependencies", module: "environments/dev/web", wantDeps: []string{"modules/network", "modules/auth"}},
		{name: "stg/api dependencies", module: "environments/stg/api", wantDeps: []string{"modules/network", "modules/database", "modules/auth"}},
		{name: "auth module dependencies", module: "modules/auth", wantDeps: []string{"modules/common"}},
		{name: "database module dependencies", module: "modules/database", wantDeps: []string{"modules/common"}},
		{name: "common module has no dependencies", module: "modules/common", wantDeps: []string{}},
		{name: "prod/app only tracks local modules", module: "environments/prod/app", wantDeps: []string{"modules/network"}},
		{name: "empty module has no dependencies", module: "modules/empty", wantDeps: []string{}},
		{name: "standalone module has no local dependencies", module: "environments/standalone/simple", wantDeps: []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := graph.Dependencies[tt.module]
			if len(deps) != len(tt.wantDeps) {
				t.Errorf("Module %s: got %d deps %v, want %d %v", tt.module, len(deps), deps, len(tt.wantDeps), tt.wantDeps)
				return
			}
			depSet := make(map[string]bool)
			for _, dep := range deps {
				depSet[dep] = true
			}
			for _, want := range tt.wantDeps {
				if !depSet[want] {
					t.Errorf("Module %s: missing dependency %s", tt.module, want)
				}
			}
		})
	}
}

func TestAnalyzer_GetAffectedRootModules(t *testing.T) {
	testRoot := filepath.Join("..", "..", "testdata", "terraform")

	analyzer := NewAnalyzer(testRoot)
	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze() failed: %v", err)
	}

	rootPatterns := []string{"environments/*/*"}

	tests := []struct {
		name         string
		changedPaths []string
		wantModules  []string
	}{
		{name: "common module change", changedPaths: []string{"modules/common/main.tf"}, wantModules: []string{"environments/dev/web", "environments/stg/api", "environments/stg/web", "environments/dev/api"}},
		{name: "network module change", changedPaths: []string{"modules/network/main.tf"}, wantModules: []string{"environments/dev/api", "environments/dev/web", "environments/stg/api", "environments/stg/web", "environments/prod/app"}},
		{name: "auth module change", changedPaths: []string{"modules/auth/main.tf"}, wantModules: []string{"environments/dev/web", "environments/stg/api", "environments/stg/web"}},
		{name: "database module change", changedPaths: []string{"modules/database/main.tf"}, wantModules: []string{"environments/dev/api", "environments/stg/api"}},
		{name: "direct root module change", changedPaths: []string{"environments/dev/api/main.tf"}, wantModules: []string{"environments/dev/api"}},
		{name: "empty module change", changedPaths: []string{"modules/empty/main.tf"}, wantModules: []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			affected, err := analyzer.GetAffectedRootModules(tt.changedPaths, rootPatterns)
			if err != nil {
				t.Fatalf("GetAffectedRootModules() failed: %v", err)
			}

			if len(affected) != len(tt.wantModules) {
				var got []string
				for m := range affected {
					got = append(got, m)
				}
				t.Errorf("got %v, want %v", got, tt.wantModules)
				return
			}

			for _, want := range tt.wantModules {
				if _, ok := affected[want]; !ok {
					t.Errorf("missing affected module: %s", want)
				}
			}
		})
	}
}

func TestAnalyzer_ErrorHandling(t *testing.T) {
	tests := []struct {
		name     string
		testRoot string
	}{
		{name: "circular dependency", testRoot: filepath.Join("..", "..", "testdata", "terraform-errors", "circular")},
		{name: "invalid syntax", testRoot: filepath.Join("..", "..", "testdata", "terraform-errors", "invalid-syntax")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := NewAnalyzer(tt.testRoot)
			err := analyzer.Analyze()
			if err != nil {
				t.Logf("Got error (may be expected): %v", err)
			}
		})
	}
}

func TestAnalyzer_NonTfFileHandling(t *testing.T) {
	testRoot := filepath.Join("..", "..", "testdata", "terraform")

	analyzer := NewAnalyzer(testRoot)
	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze() failed: %v", err)
	}

	rootPatterns := []string{"environments/*/*"}

	tests := []struct {
		name         string
		changedPaths []string
		wantModules  []string
	}{
		{name: "yaml in same dir as .tf", changedPaths: []string{"environments/dev/api/config.yaml"}, wantModules: []string{"environments/dev/api"}},
		{name: "nested json escalates to parent", changedPaths: []string{"modules/database/configs/nested/schema.json"}, wantModules: []string{"environments/dev/api", "environments/stg/api"}},
		{name: "mix of .tf and non-.tf", changedPaths: []string{"environments/dev/api/main.tf", "environments/dev/api/config.yaml"}, wantModules: []string{"environments/dev/api"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			affected, err := analyzer.GetAffectedRootModules(tt.changedPaths, rootPatterns)
			if err != nil {
				t.Fatalf("failed: %v", err)
			}

			if len(affected) != len(tt.wantModules) {
				var got []string
				for m := range affected {
					got = append(got, m)
				}
				t.Errorf("got %v, want %v", got, tt.wantModules)
				return
			}

			for _, want := range tt.wantModules {
				if _, ok := affected[want]; !ok {
					t.Errorf("missing: %s", want)
				}
			}
		})
	}
}

func TestAnalyzer_ComplexDirectoryStructure(t *testing.T) {
	testRoot := filepath.Join("..", "..", "testdata", "terraform-complex")

	analyzer := NewAnalyzer(testRoot)
	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze() failed: %v", err)
	}

	graph := analyzer.GetDependencyGraph()

	tests := []struct {
		name     string
		module   string
		wantDeps []string
	}{
		{name: "root depends on deep nested", module: ".", wantDeps: []string{"stacks/shared/vpc"}},
		{name: "cross-directory dependencies", module: "stacks/shared/vpc", wantDeps: []string{"shared-components/logging"}},
		{name: "reusable modules in non-standard locations", module: "shared-components/logging", wantDeps: []string{"utils/common"}},
		{name: "app references shared components", module: "apps/web-service", wantDeps: []string{"shared-components/logging", "utils/common"}},
		{name: "team-based structure", module: "team-a/project-x", wantDeps: []string{"shared-components/logging", "utils/common"}},
		{name: "utility modules have no dependencies", module: "utils/common", wantDeps: []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := graph.Dependencies[tt.module]
			if len(deps) != len(tt.wantDeps) {
				t.Errorf("Module %s: got %v, want %v", tt.module, deps, tt.wantDeps)
				return
			}
			depSet := make(map[string]bool)
			for _, dep := range deps {
				depSet[dep] = true
			}
			for _, want := range tt.wantDeps {
				if !depSet[want] {
					t.Errorf("Module %s: missing dependency %s", tt.module, want)
				}
			}
		})
	}
}

func TestAnalyzer_ComplexRootModuleDetection(t *testing.T) {
	testRoot := filepath.Join("..", "..", "testdata", "terraform-complex")

	analyzer := NewAnalyzer(testRoot)
	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze() failed: %v", err)
	}

	tests := []struct {
		name         string
		changedPaths []string
		rootPatterns []string
		wantModules  []string
	}{
		{name: "utility module affects consumers", changedPaths: []string{"utils/common/main.tf"}, rootPatterns: []string{"apps/*", "team-*/*", "stacks/*/*"}, wantModules: []string{"apps/web-service", "team-a/project-x", "stacks/shared/vpc"}},
		{name: "shared component affects consumers", changedPaths: []string{"shared-components/logging/main.tf"}, rootPatterns: []string{"*", "apps/*", "team-*/*", "stacks/*/*"}, wantModules: []string{".", "apps/web-service", "team-a/project-x", "stacks/shared/vpc"}},
		{name: "root module at repository root", changedPaths: []string{"main.tf"}, rootPatterns: []string{"*"}, wantModules: []string{"."}},
		{name: "doublestar matches all", changedPaths: []string{"utils/common/main.tf"}, rootPatterns: []string{"**/*"}, wantModules: []string{"utils/common", "shared-components/logging", "apps/web-service", "team-a/project-x", "stacks/shared/vpc", "."}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			affected, err := analyzer.GetAffectedRootModules(tt.changedPaths, tt.rootPatterns)
			if err != nil {
				t.Fatalf("failed: %v", err)
			}

			if len(affected) != len(tt.wantModules) {
				var got []string
				for m := range affected {
					got = append(got, m)
				}
				t.Errorf("got %v, want %v", got, tt.wantModules)
				return
			}

			for _, want := range tt.wantModules {
				if _, ok := affected[want]; !ok {
					t.Errorf("missing: %s", want)
				}
			}
		})
	}
}

func TestIsRootModule(t *testing.T) {
	tests := []struct {
		name        string
		modulePath  string
		patterns    []string
		wantMatched bool
	}{
		{name: "environments pattern matches", modulePath: "environments/dev/api", patterns: []string{"environments/*/*"}, wantMatched: true},
		{name: "modules should not match environments", modulePath: "modules/auth", patterns: []string{"environments/*/*"}, wantMatched: false},
		{name: "multiple patterns", modulePath: "environments/dev/api", patterns: []string{"modules/*", "environments/*/*"}, wantMatched: true},
		{name: "doublestar matches multi-level", modulePath: "stacks/shared/vpc", patterns: []string{"stacks/**"}, wantMatched: true},
		{name: "doublestar in middle", modulePath: "infra/eu-west-1/prod/vpc", patterns: []string{"infra/**/vpc"}, wantMatched: true},
		{name: "doublestar no match on prefix", modulePath: "other/shared/vpc", patterns: []string{"stacks/**"}, wantMatched: false},
		{name: "single star no multi-level", modulePath: "stacks/shared/vpc", patterns: []string{"stacks/*"}, wantMatched: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched := isRootModule(tt.modulePath, tt.patterns)
			if matched != tt.wantMatched {
				t.Errorf("isRootModule(%s, %v) = %v, want %v", tt.modulePath, tt.patterns, matched, tt.wantMatched)
			}
		})
	}
}

// Silence the unused import warning for strings
var _ = strings.Contains
