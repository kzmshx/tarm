package graph

import (
	"fmt"
	"path/filepath"
	"strings"
)

// DependencyGraph represents the dependency relationships between Terraform modules
type DependencyGraph struct {
	// Forward dependencies: module -> modules it depends on
	Dependencies map[string][]string
	// Reverse dependencies: module -> modules that depend on it
	Dependents map[string][]string
}

// NewDependencyGraph creates a new dependency graph
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		Dependencies: make(map[string][]string),
		Dependents:   make(map[string][]string),
	}
}

// AddDependency adds a dependency relationship where 'from' depends on 'to'
func (g *DependencyGraph) AddDependency(from, to string) {
	// Normalize paths
	from = filepath.Clean(from)
	to = filepath.Clean(to)

	// Add forward dependency
	if !contains(g.Dependencies[from], to) {
		g.Dependencies[from] = append(g.Dependencies[from], to)
	}

	// Add reverse dependency
	if !contains(g.Dependents[to], from) {
		g.Dependents[to] = append(g.Dependents[to], from)
	}
}

// GetAffectedModules returns all modules that depend on the given path
func (g *DependencyGraph) GetAffectedModules(path string) []string {
	path = filepath.Clean(path)
	visited := make(map[string]bool)
	var result []string

	var dfs func(string)
	dfs = func(current string) {
		if visited[current] {
			return
		}
		visited[current] = true

		// Add this module if it's a root module (we'll filter later)
		result = append(result, current)

		// Visit all modules that depend on this one
		for _, dependent := range g.Dependents[current] {
			dfs(dependent)
		}
	}

	dfs(path)
	return result
}

// GetAllModules returns all modules in the graph
func (g *DependencyGraph) GetAllModules() []string {
	modules := make(map[string]bool)
	for module := range g.Dependencies {
		modules[module] = true
	}
	for module := range g.Dependents {
		modules[module] = true
	}

	var result []string
	for module := range modules {
		result = append(result, module)
	}
	return result
}

// DetectCircularDependencies returns any circular dependencies found
func (g *DependencyGraph) DetectCircularDependencies() [][]string {
	var cycles [][]string
	visited := make(map[string]int) // 0: unvisited, 1: visiting, 2: visited
	var path []string

	var dfs func(string) bool
	dfs = func(node string) bool {
		if visited[node] == 1 {
			// Found a cycle
			cycleStart := -1
			for i, n := range path {
				if n == node {
					cycleStart = i
					break
				}
			}
			if cycleStart >= 0 {
				cycle := make([]string, len(path)-cycleStart)
				copy(cycle, path[cycleStart:])
				cycles = append(cycles, cycle)
			}
			return true
		}
		if visited[node] == 2 {
			return false
		}

		visited[node] = 1
		path = append(path, node)

		for _, dep := range g.Dependencies[node] {
			if dfs(dep) {
				// Continue to find all cycles
			}
		}

		path = path[:len(path)-1]
		visited[node] = 2
		return false
	}

	for module := range g.Dependencies {
		if visited[module] == 0 {
			dfs(module)
		}
	}

	return cycles
}

// String returns a string representation of the graph
func (g *DependencyGraph) String() string {
	var sb strings.Builder
	sb.WriteString("Dependency Graph:\n")
	for module, deps := range g.Dependencies {
		if len(deps) > 0 {
			sb.WriteString(fmt.Sprintf("  %s -> %s\n", module, strings.Join(deps, ", ")))
		}
	}
	return sb.String()
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}