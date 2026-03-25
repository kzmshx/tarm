package tarm

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/kzmshx/tarm/internal/git"
)

// Config holds the parameters for an analysis run.
type Config struct {
	// Root is the directory to search for Terraform files.
	Root string

	// RootModulePatterns are glob patterns identifying root modules.
	RootModulePatterns []string

	// ExcludeModulePatterns are glob patterns for modules to exclude from root module detection.
	ExcludeModulePatterns []string

	// ChangedFiles are explicitly provided paths to treat as changed.
	ChangedFiles []string

	// DetectChanges enables automatic changed file detection via the provider.
	DetectChanges bool

	// BaseRef is the base git ref for change detection.
	BaseRef string

	// HeadRef is the head git ref for change detection.
	HeadRef string

	// OutputFormat controls stdout output ("json" or "text").
	OutputFormat string
}

// Result holds the output of an analysis run.
type Result struct {
	AffectedModules []AffectedRootModule
	Cycles          [][]string
	Warnings        []string
}

// Run executes the analysis with the given config and change provider.
// It returns the result without performing any I/O side effects (no file writes, no stdout).
func Run(cfg Config, changeProvider git.ChangedFilesProvider) (*Result, error) {
	root := cfg.Root
	if root == "" {
		root = "."
	}

	if len(cfg.RootModulePatterns) == 0 {
		return nil, fmt.Errorf("at least one root module pattern must be specified")
	}

	// Resolve effective root module set.
	// When ExcludeModulePatterns is specified, we resolve patterns against the
	// filesystem to get a concrete set of root module paths, then use set lookup
	// instead of pattern matching for root module detection.
	var rootModuleSet map[string]bool
	rootModulePatterns := cfg.RootModulePatterns
	if len(cfg.ExcludeModulePatterns) > 0 {
		filtered, err := FilterPatterns(os.DirFS(root), rootModulePatterns, cfg.ExcludeModulePatterns)
		if err != nil {
			return nil, fmt.Errorf("failed to filter root module patterns: %w", err)
		}
		rootModuleSet = make(map[string]bool, len(filtered))
		for _, p := range filtered {
			rootModuleSet[p] = true
		}
	}

	// Collect changed files
	var changedFiles []string

	if cfg.DetectChanges && changeProvider != nil {
		detected, err := changeProvider.ChangedFiles()
		if err != nil {
			return nil, fmt.Errorf("failed to detect changed files: %w", err)
		}
		changedFiles = append(changedFiles, detected...)
	}

	changedFiles = append(changedFiles, cfg.ChangedFiles...)
	changedFiles = Unique(changedFiles)

	// Analyze
	a := NewAnalyzer(root)
	if err := a.Analyze(); err != nil {
		return nil, fmt.Errorf("failed to analyze modules: %w", err)
	}

	// Detect circular dependencies
	g := a.GetDependencyGraph()
	cycles := g.DetectCircularDependencies()
	for _, cycle := range cycles {
		fmt.Fprintf(os.Stderr, "WARN: Circular dependency detected: %s\n", strings.Join(cycle, " -> "))
	}

	// Build root module matcher
	var matchRootModule func(string) bool
	if rootModuleSet != nil {
		matchRootModule = func(path string) bool { return rootModuleSet[path] }
	} else {
		matchRootModule = func(path string) bool { return isRootModule(path, rootModulePatterns) }
	}

	// Get affected root modules
	affectedMap, err := a.GetAffectedRootModulesFunc(changedFiles, matchRootModule)
	if err != nil {
		return nil, fmt.Errorf("failed to get affected modules: %w", err)
	}

	// Build result
	var modules []AffectedRootModule
	for module, affectedBy := range affectedMap {
		modules = append(modules, AffectedRootModule{
			Path:       module,
			AffectedBy: Unique(affectedBy),
		})
	}

	sort.Slice(modules, func(i, j int) bool {
		return modules[i].Path < modules[j].Path
	})

	return &Result{
		AffectedModules: modules,
		Cycles:          cycles,
		Warnings:        a.Warnings(),
	}, nil
}
