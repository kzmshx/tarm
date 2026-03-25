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
	args := buildDiffArgs(p.BaseRef, p.HeadRef)
	cmd := exec.Command("git", args...)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff failed: %w", err)
	}

	return parseLines(string(output)), nil
}

// buildDiffArgs constructs git diff arguments from base and head refs.
func buildDiffArgs(baseRef, headRef string) []string {
	if headRef == "" || headRef == "HEAD" {
		return []string{"diff", "--name-only", baseRef, "HEAD"}
	}
	return []string{"diff", "--name-only", fmt.Sprintf("%s...%s", baseRef, headRef)}
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
