package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/kzmshx/tarm/internal/formatter"
	"github.com/kzmshx/tarm/internal/git"
	"github.com/kzmshx/tarm/internal/tarm"
)

func main() {
	cfg := tarm.Config{
		Root:                  os.Getenv("INPUT_ROOT"),
		RootModulePatterns:    tarm.ParseMultilineInput(os.Getenv("INPUT_ROOT_MODULE_PATTERNS")),
		ExcludeModulePatterns: tarm.ParseMultilineInput(os.Getenv("INPUT_EXCLUDE_MODULE_PATTERNS")),
		ChangedFiles:          tarm.ParseMultilineInput(os.Getenv("INPUT_CHANGED_FILES")),
		DetectChanges:         os.Getenv("INPUT_DETECT_CHANGES") != "false",
		BaseRef:               os.Getenv("INPUT_BASE_REF"),
		HeadRef:               os.Getenv("INPUT_HEAD_REF"),
		OutputFormat:          os.Getenv("INPUT_OUTPUT_FORMAT"),
	}

	if cfg.BaseRef == "" {
		cfg.BaseRef = "origin/main"
	}

	var provider git.ChangedFilesProvider
	if cfg.DetectChanges {
		provider = &git.DiffProvider{
			BaseRef: cfg.BaseRef,
			HeadRef: cfg.HeadRef,
		}
	}

	result, err := tarm.Run(cfg, provider)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	writeGitHubOutputs(result)
	writeStdout(cfg.OutputFormat, result)
}

func writeGitHubOutputs(r *tarm.Result) {
	outPath := os.Getenv("GITHUB_OUTPUT")
	if outPath == "" {
		return
	}

	f, err := os.OpenFile(outPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Failed to open GITHUB_OUTPUT: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	var moduleList []string
	for _, m := range r.AffectedModules {
		moduleList = append(moduleList, m.Path)
	}

	fmt.Fprintf(f, "affected-modules=%s\n", strings.Join(moduleList, " "))
	fmt.Fprintf(f, "affected-modules-json=%s\n", formatter.JSON(r.AffectedModules))
	fmt.Fprintf(f, "affected-count=%d\n", len(r.AffectedModules))
	fmt.Fprintf(f, "has-affected-modules=%t\n", len(r.AffectedModules) > 0)

	matrix := make([]map[string]string, 0, len(r.AffectedModules))
	for _, m := range r.AffectedModules {
		matrix = append(matrix, map[string]string{"module": m.Path})
	}
	matrixJSON, _ := json.Marshal(map[string]any{"include": matrix})
	fmt.Fprintf(f, "matrix=%s\n", string(matrixJSON))

	markdown := formatter.Markdown(r.AffectedModules)
	fmt.Fprintf(f, "markdown-summary=%s\n", strings.ReplaceAll(markdown, "\n", "%0A"))
}

func writeStdout(format string, r *tarm.Result) {
	if format == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(map[string]any{
			"affected_modules": r.AffectedModules,
		})
	} else {
		for _, m := range r.AffectedModules {
			fmt.Printf("## %s\n", m.Path)
			for _, cause := range m.AffectedBy {
				fmt.Printf("- %s\n", cause)
			}
			fmt.Println()
		}
	}
}
