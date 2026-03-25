package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/kzmshx/tarm/internal/git"
	"github.com/kzmshx/tarm/internal/tarm"
)

type stringSlice []string

func (s *stringSlice) String() string { return strings.Join(*s, ", ") }
func (s *stringSlice) Set(v string) error {
	*s = append(*s, v)
	return nil
}

func main() {
	var (
		root                  string
		rootModulePatterns    stringSlice
		excludeModulePatterns stringSlice
		changedFiles          stringSlice
		detectChanges         bool
		baseRef               string
		headRef               string
		outputFormat          string
	)

	flag.StringVar(&root, "root", ".", "Root directory to search for Terraform files")
	flag.Var(&rootModulePatterns, "root-module-patterns", "Glob pattern for root modules (repeatable)")
	flag.Var(&excludeModulePatterns, "exclude-module-patterns", "Glob pattern for modules to exclude (repeatable)")
	flag.Var(&changedFiles, "changed-files", "Path to treat as changed (repeatable)")
	flag.BoolVar(&detectChanges, "detect-changes", false, "Auto-detect changed files via git diff")
	flag.StringVar(&baseRef, "base-ref", "origin/main", "Base ref for change detection")
	flag.StringVar(&headRef, "head-ref", "HEAD", "Head ref for change detection")
	flag.StringVar(&outputFormat, "output-format", "text", "Output format: text or json")
	flag.Parse()

	if len(rootModulePatterns) == 0 {
		fmt.Fprintln(os.Stderr, "error: at least one --root-module-patterns is required")
		flag.Usage()
		os.Exit(1)
	}

	cfg := tarm.Config{
		Root:                  root,
		RootModulePatterns:    rootModulePatterns,
		ExcludeModulePatterns: excludeModulePatterns,
		ChangedFiles:          changedFiles,
		DetectChanges:         detectChanges,
		BaseRef:               baseRef,
		HeadRef:               headRef,
		OutputFormat:          outputFormat,
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
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	switch cfg.OutputFormat {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(result.AffectedModules)
	default:
		for _, m := range result.AffectedModules {
			fmt.Println(m.Path)
		}
	}
}
