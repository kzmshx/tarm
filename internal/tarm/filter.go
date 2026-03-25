package tarm

import (
	"io/fs"
	"slices"

	"github.com/bmatcuk/doublestar/v4"
)

// FilterPatterns returns directories matching includePatterns but not matching excludePatterns.
func FilterPatterns(fsys fs.FS, includePatterns []string, excludePatterns []string) ([]string, error) {
	var included []string
	for _, pattern := range includePatterns {
		matches, err := doublestar.Glob(fsys, pattern)
		if err != nil {
			return nil, err
		}
		included = append(included, matches...)
	}

	if len(excludePatterns) == 0 {
		return dedupeStrings(included), nil
	}

	excluded := map[string]struct{}{}
	for _, pattern := range excludePatterns {
		matches, err := doublestar.Glob(fsys, pattern)
		if err != nil {
			return nil, err
		}
		for _, m := range matches {
			excluded[m] = struct{}{}
		}
	}

	var result []string
	for _, dir := range included {
		if _, ok := excluded[dir]; !ok {
			if !slices.Contains(result, dir) {
				result = append(result, dir)
			}
		}
	}

	return result, nil
}

func dedupeStrings(s []string) []string {
	var result []string
	for _, v := range s {
		if !slices.Contains(result, v) {
			result = append(result, v)
		}
	}
	return result
}
