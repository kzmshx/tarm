package git

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
)

// ChangedFilesProvider detects changed files between two refs.
type ChangedFilesProvider interface {
	ChangedFiles() ([]string, error)
}

// DiffProvider detects changed files using git diff.
type DiffProvider struct {
	BaseRef string
	HeadRef string
}

// ChangedFiles returns the list of files changed between BaseRef and HeadRef.
func (p *DiffProvider) ChangedFiles() ([]string, error) {
	var cmd *exec.Cmd
	if p.HeadRef == "" || p.HeadRef == "HEAD" {
		cmd = exec.Command("git", "diff", "--name-only", p.BaseRef, "HEAD")
	} else {
		cmd = exec.Command("git", "diff", "--name-only", fmt.Sprintf("origin/%s...origin/%s", p.BaseRef, p.HeadRef))
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff failed: %w", err)
	}

	return parseLines(string(output)), nil
}

// StaticProvider returns a fixed list of changed files.
type StaticProvider struct {
	Files []string
}

// ChangedFiles returns the static file list.
func (p *StaticProvider) ChangedFiles() ([]string, error) {
	return p.Files, nil
}

// MultiProvider combines multiple providers, deduplicating results.
type MultiProvider struct {
	Providers []ChangedFilesProvider
}

// ChangedFiles collects files from all providers and deduplicates.
func (p *MultiProvider) ChangedFiles() ([]string, error) {
	seen := make(map[string]bool)
	var all []string
	for _, provider := range p.Providers {
		files, err := provider.ChangedFiles()
		if err != nil {
			return nil, err
		}
		for _, f := range files {
			if !seen[f] {
				seen[f] = true
				all = append(all, f)
			}
		}
	}
	return all, nil
}

func parseLines(s string) []string {
	var result []string
	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}
