package analyzer

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform-config-inspect/tfconfig"
	"github.com/kzmshx/tarm/internal/graph"
	"github.com/kzmshx/tarm/internal/resolver"
)

// Analyzer analyzes Terraform module dependencies
type Analyzer struct {
	root  string
	graph *graph.DependencyGraph
}

// New creates a new analyzer
func New(root string) *Analyzer {
	// Convert to absolute path
	absRoot, err := filepath.Abs(root)
	if err != nil {
		absRoot = root
	}
	return &Analyzer{
		root:  absRoot,
		graph: graph.NewDependencyGraph(),
	}
}

// Analyze analyzes all Terraform files in the root directory and builds a dependency graph
func (a *Analyzer) Analyze() error {
	return filepath.WalkDir(a.root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip .terraform directories
		if d.IsDir() && (d.Name() == ".terraform" || strings.Contains(path, ".terragrunt-cache")) {
			return filepath.SkipDir
		}

		// Only process directories
		if !d.IsDir() {
			return nil
		}

		// Check if directory contains .tf files
		hasTfFiles, err := containsTerraformFiles(path)
		if err != nil {
			return err
		}
		if !hasTfFiles {
			return nil
		}

		// Parse the module
		module, diags := tfconfig.LoadModule(path)
		if diags.HasErrors() {
			// Log error but continue
			fmt.Fprintf(os.Stderr, "ERROR: Failed to parse %s: %s\n", path, diags.Error())
			return nil
		}

		// Get relative path from root
		relPath, err := filepath.Rel(a.root, path)
		if err != nil {
			return err
		}

		// Process module calls
		for _, call := range module.ModuleCalls {
			// Resolve the source path
			resolvedPath, err := resolver.ResolveModuleSource(path, call.Source)
			if err != nil {
				fmt.Fprintf(os.Stderr, "WARN: Failed to resolve module source %q in %s: %v\n", call.Source, relPath, err)
				continue
			}

			// Skip non-local modules
			if resolvedPath == "" {
				continue
			}

			// Make it relative to root
			relResolvedPath, err := filepath.Rel(a.root, resolvedPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "WARN: Failed to get relative path for %s: %v\n", resolvedPath, err)
				continue
			}

			// Check if the resolved path exists
			if _, err := os.Stat(resolvedPath); os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "WARN: Module source %q not found in module %q\n", call.Source, relPath)
				continue
			}

			// Add dependency
			a.graph.AddDependency(relPath, relResolvedPath)
		}

		return nil
	})
}

// GetAffectedRootModules returns root modules affected by changes in the given paths
func (a *Analyzer) GetAffectedRootModules(changedPaths []string, rootModulePatterns []string) (map[string][]string, error) {
	affectedByPath := make(map[string][]string)

	for _, changePath := range changedPaths {
		// Convert to absolute path if needed
		if !filepath.IsAbs(changePath) {
			changePath = filepath.Join(a.root, changePath)
		}

		// Find parent directory with .tf files
		tfDir, err := resolver.FindParentWithTerraformFiles(changePath, a.root)
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARN: Failed to find parent with .tf files for %s: %v\n", changePath, err)
			continue
		}

		if tfDir == "" {
			// No parent with .tf files found
			continue
		}

		// Get relative path
		relTfDir, err := filepath.Rel(a.root, tfDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARN: Failed to get relative path for %s: %v\n", tfDir, err)
			continue
		}

		// Get all affected modules
		affected := a.graph.GetAffectedModules(relTfDir)

		// Filter to only root modules
		for _, module := range affected {
			if isRootModule(module, rootModulePatterns) {
				if _, exists := affectedByPath[module]; !exists {
					affectedByPath[module] = []string{}
				}
				affectedByPath[module] = append(affectedByPath[module], relTfDir)
			}
		}
	}

	return affectedByPath, nil
}

// GetDependencyGraph returns the dependency graph
func (a *Analyzer) GetDependencyGraph() *graph.DependencyGraph {
	return a.graph
}

// containsTerraformFiles checks if a directory contains .tf files
func containsTerraformFiles(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".tf") {
			return true, nil
		}
	}

	return false, nil
}

// isRootModule checks if a module path matches any of the root module patterns
func isRootModule(modulePath string, patterns []string) bool {
	for _, pattern := range patterns {
		matched, err := filepath.Match(pattern, modulePath)
		if err != nil {
			// Invalid pattern, skip
			continue
		}
		if matched {
			return true
		}
	}
	return false
}