package analyzer

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestAnalyzer_Analyze(t *testing.T) {
	testRoot := filepath.Join("..", "..", "testdata", "terraform")
	
	analyzer := New(testRoot)
	err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() failed: %v", err)
	}

	graph := analyzer.GetDependencyGraph()

	// Expected dependencies based on actual testdata structure:
	// environments/dev/api -> modules/network, modules/database
	// environments/dev/web -> modules/network, modules/auth
	// environments/stg/api -> modules/network, modules/database, modules/auth
	// environments/stg/web -> modules/network, modules/auth
	// modules/auth -> modules/common
	// modules/database -> modules/common

	tests := []struct {
		name       string
		module     string
		wantDeps   []string
	}{
		{
			name:     "dev/api dependencies",
			module:   "environments/dev/api",
			wantDeps: []string{"modules/network", "modules/database"},
		},
		{
			name:     "dev/web dependencies",
			module:   "environments/dev/web",
			wantDeps: []string{"modules/network", "modules/auth"},
		},
		{
			name:     "stg/api dependencies",
			module:   "environments/stg/api",
			wantDeps: []string{"modules/network", "modules/database", "modules/auth"},
		},
		{
			name:     "auth module dependencies", 
			module:   "modules/auth",
			wantDeps: []string{"modules/common"},
		},
		{
			name:     "database module dependencies",
			module:   "modules/database", 
			wantDeps: []string{"modules/common"},
		},
		{
			name:     "common module has no dependencies",
			module:   "modules/common",
			wantDeps: []string{},
		},
		{
			name:     "prod/app only tracks local modules, ignores external",
			module:   "environments/prod/app",
			wantDeps: []string{"modules/network"},
		},
		{
			name:     "empty module has no dependencies",
			module:   "modules/empty",
			wantDeps: []string{},
		},
		{
			name:     "standalone module has no local dependencies",
			module:   "environments/standalone/simple",
			wantDeps: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := graph.Dependencies[tt.module]
			if len(deps) != len(tt.wantDeps) {
				t.Errorf("Module %s: got %d dependencies, want %d", tt.module, len(deps), len(tt.wantDeps))
				t.Errorf("Got: %v", deps)
				t.Errorf("Want: %v", tt.wantDeps)
				return
			}

			depSet := make(map[string]bool)
			for _, dep := range deps {
				depSet[dep] = true
			}

			for _, wantDep := range tt.wantDeps {
				if !depSet[wantDep] {
					t.Errorf("Module %s: missing dependency %s", tt.module, wantDep)
				}
			}
		})
	}
}

func TestAnalyzer_GetAffectedRootModules(t *testing.T) {
	testRoot := filepath.Join("..", "..", "testdata", "terraform")
	
	analyzer := New(testRoot)
	err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() failed: %v", err)
	}

	rootPatterns := []string{"environments/*/*"}

	tests := []struct {
		name         string
		changedPaths []string
		wantModules  []string
	}{
		{
			name:         "common module change affects all modules using auth/database",
			changedPaths: []string{"modules/common/main.tf"},
			wantModules:  []string{"environments/dev/web", "environments/stg/api", "environments/stg/web", "environments/dev/api"},
		},
		{
			name:         "network module change affects all environments including prod",
			changedPaths: []string{"modules/network/main.tf"},
			wantModules:  []string{"environments/dev/api", "environments/dev/web", "environments/stg/api", "environments/stg/web", "environments/prod/app"},
		},
		{
			name:         "auth module change affects web and stg environments",
			changedPaths: []string{"modules/auth/main.tf"},
			wantModules:  []string{"environments/dev/web", "environments/stg/api", "environments/stg/web"},
		},
		{
			name:         "database module change affects api environments",
			changedPaths: []string{"modules/database/main.tf"},
			wantModules:  []string{"environments/dev/api", "environments/stg/api"},
		},
		{
			name:         "direct root module change",
			changedPaths: []string{"environments/dev/api/main.tf"},
			wantModules:  []string{"environments/dev/api"},
		},
		{
			name:         "standalone module is not affected by common module changes",
			changedPaths: []string{"modules/common/main.tf"},
			wantModules:  []string{"environments/dev/web", "environments/stg/api", "environments/stg/web", "environments/dev/api"},
		},
		{
			name:         "empty module change affects nothing",
			changedPaths: []string{"modules/empty/main.tf"},
			wantModules:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			affected, err := analyzer.GetAffectedRootModules(tt.changedPaths, rootPatterns)
			if err != nil {
				t.Fatalf("GetAffectedRootModules() failed: %v", err)
			}

			var gotModules []string
			for module := range affected {
				gotModules = append(gotModules, module)
			}

			if len(gotModules) != len(tt.wantModules) {
				t.Errorf("Got %d affected modules, want %d", len(gotModules), len(tt.wantModules))
				t.Errorf("Got: %v", gotModules)
				t.Errorf("Want: %v", tt.wantModules)
				return
			}

			moduleSet := make(map[string]bool)
			for _, module := range gotModules {
				moduleSet[module] = true
			}

			for _, wantModule := range tt.wantModules {
				if !moduleSet[wantModule] {
					t.Errorf("Missing affected module: %s", wantModule)
				}
			}
		})
	}
}

func TestAnalyzer_ErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		testRoot    string
		expectError bool
		errorContains string
	}{
		{
			name:        "circular dependency should be handled gracefully",
			testRoot:    filepath.Join("..", "..", "testdata", "terraform-errors", "circular"),
			expectError: false, // Should not crash, but may detect cycle
		},
		{
			name:        "invalid syntax should be handled gracefully (logs error but continues)",
			testRoot:    filepath.Join("..", "..", "testdata", "terraform-errors", "invalid-syntax"),
			expectError: false, // Analyzer continues despite parse errors
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := New(tt.testRoot)
			err := analyzer.Analyze()
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Logf("Got error (may be expected for edge cases): %v", err)
				}
			}
		})
	}
}

func TestAnalyzer_NonTfFileHandling(t *testing.T) {
	testRoot := filepath.Join("..", "..", "testdata", "terraform")
	
	analyzer := New(testRoot)
	err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() failed: %v", err)
	}

	rootPatterns := []string{"environments/*/*"}

	tests := []struct {
		name         string
		changedPaths []string
		wantModules  []string
	}{
		{
			name:         "yaml file in same directory as .tf file",
			changedPaths: []string{"environments/dev/api/config.yaml"},
			wantModules:  []string{"environments/dev/api"},
		},
		{
			name:         "json file in nested directory escalates to parent with .tf",
			changedPaths: []string{"modules/database/configs/nested/schema.json"},
			wantModules:  []string{"environments/dev/api", "environments/stg/api"}, // Modules that depend on database
		},
		{
			name:         "mix of .tf and non-.tf files",
			changedPaths: []string{
				"environments/dev/api/main.tf",
				"environments/dev/api/config.yaml",
			},
			wantModules:  []string{"environments/dev/api"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			affected, err := analyzer.GetAffectedRootModules(tt.changedPaths, rootPatterns)
			if err != nil {
				t.Fatalf("GetAffectedRootModules() failed: %v", err)
			}

			var gotModules []string
			for module := range affected {
				gotModules = append(gotModules, module)
			}

			if len(gotModules) != len(tt.wantModules) {
				t.Errorf("Got %d affected modules, want %d", len(gotModules), len(tt.wantModules))
				t.Errorf("Got: %v", gotModules)
				t.Errorf("Want: %v", tt.wantModules)
				return
			}

			moduleSet := make(map[string]bool)
			for _, module := range gotModules {
				moduleSet[module] = true
			}

			for _, wantModule := range tt.wantModules {
				if !moduleSet[wantModule] {
					t.Errorf("Missing affected module: %s", wantModule)
				}
			}
		})
	}
}

func TestAnalyzer_ComplexDirectoryStructure(t *testing.T) {
	testRoot := filepath.Join("..", "..", "testdata", "terraform-complex")
	
	analyzer := New(testRoot)
	err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() failed: %v", err)
	}

	graph := analyzer.GetDependencyGraph()

	tests := []struct {
		name       string
		module     string
		wantDeps   []string
	}{
		{
			name:     "root module can depend on deep nested modules",
			module:   ".",
			wantDeps: []string{"stacks/shared/vpc"},
		},
		{
			name:     "deep nested root module with cross-directory dependencies",
			module:   "stacks/shared/vpc",
			wantDeps: []string{"shared-components/logging"},
		},
		{
			name:     "reusable modules in non-standard locations",
			module:   "shared-components/logging",
			wantDeps: []string{"utils/common"},
		},
		{
			name:     "app-style modules can reference shared components",
			module:   "apps/web-service",
			wantDeps: []string{"shared-components/logging", "utils/common"},
		},
		{
			name:     "team-based structure with shared dependencies",
			module:   "team-a/project-x",
			wantDeps: []string{"shared-components/logging", "utils/common"},
		},
		{
			name:     "utility modules have no dependencies",
			module:   "utils/common",
			wantDeps: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := graph.Dependencies[tt.module]
			if len(deps) != len(tt.wantDeps) {
				t.Errorf("Module %s: got %d dependencies, want %d", tt.module, len(deps), len(tt.wantDeps))
				t.Errorf("Got: %v", deps)
				t.Errorf("Want: %v", tt.wantDeps)
				return
			}

			depSet := make(map[string]bool)
			for _, dep := range deps {
				depSet[dep] = true
			}

			for _, wantDep := range tt.wantDeps {
				if !depSet[wantDep] {
					t.Errorf("Module %s: missing dependency %s", tt.module, wantDep)
				}
			}
		})
	}
}

func TestAnalyzer_ComplexRootModuleDetection(t *testing.T) {
	testRoot := filepath.Join("..", "..", "testdata", "terraform-complex")
	
	analyzer := New(testRoot)
	err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() failed: %v", err)
	}

	tests := []struct {
		name         string
		changedPaths []string
		rootPatterns []string
		wantModules  []string
	}{
		{
			name:         "utility module change affects multiple consumers",
			changedPaths: []string{"utils/common/main.tf"},
			rootPatterns: []string{"apps/*", "team-*/*", "stacks/*/*"},
			wantModules:  []string{"apps/web-service", "team-a/project-x", "stacks/shared/vpc"},
		},
		{
			name:         "shared component change affects direct consumers",
			changedPaths: []string{"shared-components/logging/main.tf"},
			rootPatterns: []string{"*", "apps/*", "team-*/*", "stacks/*/*"},
			wantModules:  []string{".", "apps/web-service", "team-a/project-x", "stacks/shared/vpc"},
		},
		{
			name:         "root module at repository root",
			changedPaths: []string{"main.tf"},
			rootPatterns: []string{"*"},
			wantModules:  []string{"."},
		},
		{
			name:         "flexible patterns can match various structures",
			changedPaths: []string{"utils/common/main.tf"},
			rootPatterns: []string{"**/*"},
			wantModules:  []string{"utils/common", "shared-components/logging", "apps/web-service", "team-a/project-x"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			affected, err := analyzer.GetAffectedRootModules(tt.changedPaths, tt.rootPatterns)
			if err != nil {
				t.Fatalf("GetAffectedRootModules() failed: %v", err)
			}

			var gotModules []string
			for module := range affected {
				gotModules = append(gotModules, module)
			}

			if len(gotModules) != len(tt.wantModules) {
				t.Errorf("Got %d affected modules, want %d", len(gotModules), len(tt.wantModules))
				t.Errorf("Got: %v", gotModules)
				t.Errorf("Want: %v", tt.wantModules)
				return
			}

			moduleSet := make(map[string]bool)
			for _, module := range gotModules {
				moduleSet[module] = true
			}

			for _, wantModule := range tt.wantModules {
				if !moduleSet[wantModule] {
					t.Errorf("Missing affected module: %s", wantModule)
				}
			}
		})
	}
}

func TestAnalyzer_RootModulePatternMatching(t *testing.T) {
	tests := []struct {
		name        string
		modulePath  string
		patterns    []string
		wantMatched bool
	}{
		{
			name:        "environments pattern matches dev/api",
			modulePath:  "environments/dev/api",
			patterns:    []string{"environments/*/*"},
			wantMatched: true,
		},
		{
			name:        "environments pattern matches stg/web",
			modulePath:  "environments/stg/web", 
			patterns:    []string{"environments/*/*"},
			wantMatched: true,
		},
		{
			name:        "modules should not match environments pattern",
			modulePath:  "modules/auth",
			patterns:    []string{"environments/*/*"},
			wantMatched: false,
		},
		{
			name:        "multiple patterns with first match",
			modulePath:  "environments/dev/api",
			patterns:    []string{"modules/*", "environments/*/*"},
			wantMatched: true,
		},
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