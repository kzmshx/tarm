package tarm

import (
	"bufio"
	"strings"
)

// AffectedRootModule represents a root module affected by changes.
type AffectedRootModule struct {
	Path       string   `json:"path"`
	AffectedBy []string `json:"affected_by"`
}

// Unique returns a new slice with duplicate elements removed, preserving order.
func Unique[T comparable](slice []T) []T {
	seen := make(map[T]bool)
	var result []T
	for _, v := range slice {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}

// ParseMultilineInput parses a newline-separated string into a slice of trimmed, non-empty lines.
func ParseMultilineInput(input string) []string {
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
