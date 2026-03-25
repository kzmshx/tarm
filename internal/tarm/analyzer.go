package tarm

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/hashicorp/terraform-config-inspect/tfconfig"
)

// Analyzer analyzes Terraform module dependencies.
type Analyzer struct {
	root     string
	graph    *DependencyGraph
	warnings []string
}

// NewAnalyzer creates a new analyzer for the given root directory.
func NewAnalyzer(root string) *Analyzer {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		absRoot = root
	}
	return &Analyzer{
		root:  absRoot,
		graph: NewDependencyGraph(),
	}
}

// Analyze walks the root directory and builds a dependency graph.
func (a *Analyzer) Analyze() error {
	return filepath.WalkDir(a.root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() && (d.Name() == ".terraform" || strings.Contains(path, ".terragrunt-cache")) {
			return filepath.SkipDir
		}

		if !d.IsDir() {
			return nil
		}

		hasTfFiles, err := containsTerraformFiles(path)
		if err != nil {
			return err
		}
		if !hasTfFiles {
			return nil
		}

		relPath, err := filepath.Rel(a.root, path)
		if err != nil {
			return err
		}

		module, diags := tfconfig.LoadModule(path)
		if diags.HasErrors() {
			msg := fmt.Sprintf("failed to parse %s: %s", relPath, diags.Error())
			fmt.Fprintf(os.Stderr, "WARN: %s\n", msg)
			a.warnings = append(a.warnings, msg)
			return nil
		}

		for _, call := range module.ModuleCalls {
			resolvedPath, err := ResolveModuleSource(path, call.Source)
			if err != nil {
				fmt.Fprintf(os.Stderr, "WARN: Failed to resolve module source %q in %s: %v\n", call.Source, relPath, err)
				continue
			}

			if resolvedPath == "" {
				continue
			}

			relResolvedPath, err := filepath.Rel(a.root, resolvedPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "WARN: Failed to get relative path for %s: %v\n", resolvedPath, err)
				continue
			}

			if _, err := os.Stat(resolvedPath); os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "WARN: Module source %q not found in module %q\n", call.Source, relPath)
				continue
			}

			a.graph.AddDependency(relPath, relResolvedPath)
		}

		return nil
	})
}

// GetAffectedRootModules returns root modules affected by changes in the given paths,
// using glob patterns to identify root modules.
func (a *Analyzer) GetAffectedRootModules(changedPaths []string, rootModulePatterns []string) (map[string][]string, error) {
	return a.GetAffectedRootModulesFunc(changedPaths, func(path string) bool {
		return isRootModule(path, rootModulePatterns)
	})
}

// GetAffectedRootModulesFunc returns root modules affected by changes in the given paths,
// using a custom matcher function to identify root modules.
func (a *Analyzer) GetAffectedRootModulesFunc(changedPaths []string, isRoot func(string) bool) (map[string][]string, error) {
	affectedByPath := make(map[string][]string)

	for _, changePath := range changedPaths {
		if !filepath.IsAbs(changePath) {
			changePath = filepath.Join(a.root, changePath)
		}

		tfDir, err := FindParentWithTerraformFiles(changePath, a.root)
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARN: Failed to find parent with .tf files for %s: %v\n", changePath, err)
			continue
		}

		if tfDir == "" {
			continue
		}

		relTfDir, err := filepath.Rel(a.root, tfDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARN: Failed to get relative path for %s: %v\n", tfDir, err)
			continue
		}

		affected := a.graph.GetAffectedModules(relTfDir)

		for _, module := range affected {
			if isRoot(module) {
				if _, exists := affectedByPath[module]; !exists {
					affectedByPath[module] = []string{}
				}
				affectedByPath[module] = append(affectedByPath[module], relTfDir)
			}
		}
	}

	return affectedByPath, nil
}

// GetDependencyGraph returns the dependency graph.
func (a *Analyzer) GetDependencyGraph() *DependencyGraph {
	return a.graph
}

// Warnings returns warnings collected during analysis.
func (a *Analyzer) Warnings() []string {
	return a.warnings
}

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

func isRootModule(modulePath string, patterns []string) bool {
	for _, pattern := range patterns {
		matched, err := doublestar.Match(pattern, modulePath)
		if err != nil {
			continue
		}
		if matched {
			return true
		}
	}
	return false
}
