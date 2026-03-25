package tarm

import (
	"os"
	"path/filepath"
	"strings"
)

// NormalizePath normalizes a path to be relative to the root directory.
func NormalizePath(root, path string) (string, error) {
	if !filepath.IsAbs(path) {
		absPath, err := filepath.Abs(filepath.Join(root, path))
		if err != nil {
			return "", err
		}
		path = absPath
	}

	relPath, err := filepath.Rel(root, path)
	if err != nil {
		return "", err
	}

	return filepath.Clean(relPath), nil
}

// ResolveModuleSource resolves a module source path relative to the calling module.
// Returns empty string for non-local sources (registry, git, http, etc.).
func ResolveModuleSource(callerDir, source string) (string, error) {
	if !isLocalPath(source) {
		return "", nil
	}

	absPath := filepath.Join(callerDir, source)
	return filepath.Clean(absPath), nil
}

func isLocalPath(source string) bool {
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

	if strings.Count(source, "/") >= 2 && !strings.HasPrefix(source, "./") && !strings.HasPrefix(source, "../") && !strings.HasPrefix(source, "/") {
		return false
	}

	return true
}

// FindParentWithTerraformFiles finds the nearest parent directory containing .tf files.
func FindParentWithTerraformFiles(startPath, rootDir string) (string, error) {
	if !filepath.IsAbs(startPath) {
		absPath, err := filepath.Abs(startPath)
		if err != nil {
			return "", err
		}
		startPath = absPath
	}

	if !filepath.IsAbs(rootDir) {
		absRoot, err := filepath.Abs(rootDir)
		if err != nil {
			return "", err
		}
		rootDir = absRoot
	}

	info, err := os.Stat(startPath)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		startPath = filepath.Dir(startPath)
	}

	current := startPath
	for {
		entries, err := os.ReadDir(current)
		if err != nil {
			return "", err
		}

		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".tf") {
				return current, nil
			}
		}

		if current == rootDir || !IsWithinDirectory(current, rootDir) {
			break
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return "", nil
}

// IsWithinDirectory checks if a path is within a directory.
func IsWithinDirectory(path, dir string) bool {
	path = filepath.Clean(path)
	dir = filepath.Clean(dir)

	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}

	return !strings.HasPrefix(rel, "..")
}
