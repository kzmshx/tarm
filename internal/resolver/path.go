package resolver

import (
	"os"
	"path/filepath"
	"strings"
)

// NormalizePath normalizes a path to be relative to the root directory
func NormalizePath(root, path string) (string, error) {
	// Make path absolute
	if !filepath.IsAbs(path) {
		absPath, err := filepath.Abs(filepath.Join(root, path))
		if err != nil {
			return "", err
		}
		path = absPath
	}

	// Get relative path from root
	relPath, err := filepath.Rel(root, path)
	if err != nil {
		return "", err
	}

	// Clean the path
	return filepath.Clean(relPath), nil
}

// ResolveModuleSource resolves a module source path relative to the calling module
func ResolveModuleSource(callerDir, source string) (string, error) {
	// Only handle local paths
	if !isLocalPath(source) {
		return "", nil
	}

	// Resolve the path
	absPath := filepath.Join(callerDir, source)
	return filepath.Clean(absPath), nil
}

// isLocalPath checks if a source is a local path (not a registry, git, http, etc.)
func isLocalPath(source string) bool {
	// Check for common non-local prefixes
	nonLocalPrefixes := []string{
		"git::",
		"github.com/",
		"bitbucket.org/",
		"http://",
		"https://",
		"s3::",
		"gcs::",
	}

	for _, prefix := range nonLocalPrefixes {
		if strings.HasPrefix(source, prefix) {
			return false
		}
	}

	// Check if it looks like a registry module
	if strings.Count(source, "/") >= 2 && !strings.HasPrefix(source, "./") && !strings.HasPrefix(source, "../") && !strings.HasPrefix(source, "/") {
		return false
	}

	return true
}

// FindParentWithTerraformFiles finds the nearest parent directory containing .tf files
func FindParentWithTerraformFiles(startPath, rootDir string) (string, error) {
	// Ensure startPath is absolute
	if !filepath.IsAbs(startPath) {
		absPath, err := filepath.Abs(startPath)
		if err != nil {
			return "", err
		}
		startPath = absPath
	}

	// Ensure rootDir is absolute
	if !filepath.IsAbs(rootDir) {
		absRoot, err := filepath.Abs(rootDir)
		if err != nil {
			return "", err
		}
		rootDir = absRoot
	}

	// Start from the file's directory if it's a file
	info, err := os.Stat(startPath)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		startPath = filepath.Dir(startPath)
	}

	// Walk up the directory tree
	current := startPath
	for {
		// Check if current directory contains .tf files
		entries, err := os.ReadDir(current)
		if err != nil {
			return "", err
		}

		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".tf") {
				return current, nil
			}
		}

		// Check if we've reached the root
		if current == rootDir || !strings.HasPrefix(current, rootDir) {
			break
		}

		// Move up one directory
		parent := filepath.Dir(current)
		if parent == current {
			// Reached filesystem root
			break
		}
		current = parent
	}

	return "", nil
}

// IsWithinDirectory checks if a path is within a directory
func IsWithinDirectory(path, dir string) bool {
	// Normalize both paths
	path = filepath.Clean(path)
	dir = filepath.Clean(dir)

	// Check if path starts with dir
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}

	// If the relative path starts with "..", it's outside the directory
	return !strings.HasPrefix(rel, "..")
}