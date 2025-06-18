package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kzmshx/tarm/internal/analyzer"
)

type AffectedModule struct {
	Path       string   `json:"path"`
	AffectedBy []string `json:"affected_by"`
}

type Output struct {
	AffectedModules []AffectedModule `json:"affected_modules"`
}

func main() {
	// Get environment variables
	root := os.Getenv("INPUT_ROOT")
	if root == "" {
		root = "."
	}

	entrypointsStr := os.Getenv("INPUT_ENTRYPOINTS")
	if entrypointsStr == "" {
		fmt.Fprintln(os.Stderr, "ERROR: entrypoints must be specified")
		os.Exit(1)
	}

	// Parse entrypoints (newline separated)
	entrypoints := parseMultilineInput(entrypointsStr)
	if len(entrypoints) == 0 {
		fmt.Fprintln(os.Stderr, "ERROR: at least one entrypoint must be specified")
		os.Exit(1)
	}

	// Get changed files
	var changedFiles []string
	
	// Check if we should auto-detect changed files
	if os.Getenv("INPUT_CHANGED_FILES") != "false" {
		baseRef := os.Getenv("INPUT_BASE_REF")
		if baseRef == "" {
			baseRef = "origin/main"
		}

		// Get changed files from git
		cmd := exec.Command("git", "diff", "--name-only", baseRef, "HEAD")
		output, err := cmd.Output()
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: Failed to get changed files: %v\n", err)
			os.Exit(1)
		}

		scanner := bufio.NewScanner(strings.NewReader(string(output)))
		for scanner.Scan() {
			file := scanner.Text()
			if strings.HasPrefix(file, "terraform/") {
				changedFiles = append(changedFiles, file)
			}
		}
	}

	// Get paths from input if provided
	pathsStr := os.Getenv("INPUT_PATHS")
	if pathsStr != "" {
		paths := parseMultilineInput(pathsStr)
		changedFiles = append(changedFiles, paths...)
	}

	// Initialize analyzer
	a := analyzer.New(root)

	// Analyze all modules
	if err := a.Analyze(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Failed to analyze modules: %v\n", err)
		os.Exit(1)
	}

	// Check for circular dependencies
	g := a.GetDependencyGraph()
	cycles := g.DetectCircularDependencies()
	for _, cycle := range cycles {
		fmt.Fprintf(os.Stderr, "WARN: Circular dependency detected: %s\n", strings.Join(cycle, " -> "))
	}

	// Get affected root modules
	affectedMap, err := a.GetAffectedRootModules(changedFiles, entrypoints)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Failed to get affected modules: %v\n", err)
		os.Exit(1)
	}

	// Convert to output format
	var affectedModules []AffectedModule
	affectedList := []string{}
	
	for module, affectedBy := range affectedMap {
		affectedModules = append(affectedModules, AffectedModule{
			Path:       module,
			AffectedBy: unique(affectedBy),
		})
		affectedList = append(affectedList, module)
	}

	// Set GitHub Actions outputs
	if os.Getenv("GITHUB_OUTPUT") != "" {
		outputFile, err := os.OpenFile(os.Getenv("GITHUB_OUTPUT"), os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: Failed to open GITHUB_OUTPUT file: %v\n", err)
			os.Exit(1)
		}
		defer outputFile.Close()

		// Write outputs
		fmt.Fprintf(outputFile, "affected-modules=%s\n", strings.Join(affectedList, " "))
		
		jsonOutput, _ := json.Marshal(Output{AffectedModules: affectedModules})
		fmt.Fprintf(outputFile, "affected-modules-json=%s\n", string(jsonOutput))
		
		fmt.Fprintf(outputFile, "affected-count=%d\n", len(affectedModules))

		// Create matrix for GitHub Actions
		matrix := make([]map[string]string, 0, len(affectedModules))
		for _, module := range affectedModules {
			matrix = append(matrix, map[string]string{"module": module.Path})
		}
		matrixJson, _ := json.Marshal(map[string]interface{}{"include": matrix})
		fmt.Fprintf(outputFile, "matrix=%s\n", string(matrixJson))

		// Create markdown summary
		markdown := generateMarkdownSummary(changedFiles, affectedMap)
		// Escape newlines for GitHub Actions
		escaped := strings.ReplaceAll(markdown, "\n", "%0A")
		fmt.Fprintf(outputFile, "markdown-summary=%s\n", escaped)
	}

	// Output to stdout for debugging
	if os.Getenv("INPUT_OUTPUT_FORMAT") == "json" {
		output := Output{AffectedModules: affectedModules}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		encoder.Encode(output)
	} else {
		// Markdown format
		for _, module := range affectedModules {
			fmt.Printf("## %s\n", module.Path)
			for _, affectedBy := range module.AffectedBy {
				fmt.Printf("- %s\n", affectedBy)
			}
			fmt.Println()
		}
	}
}

func parseMultilineInput(input string) []string {
	var result []string
	scanner := bufio.NewScanner(strings.NewReader(input))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}

func unique(slice []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range slice {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

func generateMarkdownSummary(changedFiles []string, affectedMap map[string][]string) string {
	var sb strings.Builder
	
	sb.WriteString("## Terraform Affected Root Modules\n\n")
	
	// Group changed files by type
	changedEnvs := []string{}
	changedMods := []string{}
	
	for _, file := range changedFiles {
		if strings.Contains(file, "/environments/") {
			parent := findParentModule(file)
			if parent != "" && !contains(changedEnvs, parent) {
				changedEnvs = append(changedEnvs, parent)
			}
		} else if strings.Contains(file, "/modules/") {
			parent := findParentModule(file)
			if parent != "" && !contains(changedMods, parent) {
				changedMods = append(changedMods, parent)
			}
		}
	}
	
	if len(changedEnvs) > 0 {
		sb.WriteString("### Changed environments\n")
		for _, env := range changedEnvs {
			sb.WriteString(fmt.Sprintf("- %s\n", env))
		}
		sb.WriteString("\n")
	}
	
	if len(changedMods) > 0 {
		sb.WriteString("### Changed modules\n")
		for _, mod := range changedMods {
			sb.WriteString(fmt.Sprintf("- %s\n", mod))
		}
		sb.WriteString("\n")
	}
	
	if len(affectedMap) > 0 {
		sb.WriteString("### Affected Root Modules\n")
		for module, affectedBy := range affectedMap {
			sb.WriteString(fmt.Sprintf("<details><summary>%s</summary>\n\n", module))
			sb.WriteString("```\nBecause of:\n")
			for _, cause := range unique(affectedBy) {
				sb.WriteString(fmt.Sprintf("- %s\n", cause))
			}
			sb.WriteString("```\n\n</details>\n")
		}
	}
	
	return sb.String()
}

func findParentModule(file string) string {
	dir := filepath.Dir(file)
	parts := strings.Split(dir, "/")
	
	// Find the index of "environments" or "modules"
	for i, part := range parts {
		if part == "environments" || part == "modules" {
			// Return path up to 2 levels after environments/modules
			if i+2 < len(parts) {
				return strings.Join(parts[:i+3], "/")
			}
			break
		}
	}
	
	return dir
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}