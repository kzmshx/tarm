package formatter

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/kzmshx/tarm/internal/tarm"
)

// JSON marshals the affected root modules to a JSON string.
func JSON(modules []tarm.AffectedRootModule) string {
	if modules == nil {
		modules = []tarm.AffectedRootModule{}
	}
	b, err := json.Marshal(modules)
	if err != nil {
		return "[]"
	}
	return string(b)
}

// Markdown generates a GitHub-flavored markdown summary of the affected root modules.
func Markdown(modules []tarm.AffectedRootModule) string {
	var sb strings.Builder

	sb.WriteString("## Terraform Affected Root Modules\n\n")

	if len(modules) == 0 {
		sb.WriteString("No affected root modules found.\n")
		return sb.String()
	}

	sb.WriteString(fmt.Sprintf("**%d** root module(s) affected:\n\n", len(modules)))

	for _, module := range modules {
		sb.WriteString(fmt.Sprintf("<details><summary>%s</summary>\n\n", module.Path))
		sb.WriteString("```\nBecause of:\n")
		for _, cause := range tarm.Unique(module.AffectedBy) {
			sb.WriteString(fmt.Sprintf("- %s\n", cause))
		}
		sb.WriteString("```\n\n</details>\n\n")
	}

	return sb.String()
}

// FindParentModule extracts the module path from a file path based on known directory conventions.
func FindParentModule(file string) string {
	dir := filepath.Dir(file)
	parts := strings.Split(dir, "/")

	for i, part := range parts {
		if part == "environments" || part == "modules" {
			if i+2 < len(parts) {
				return strings.Join(parts[:i+3], "/")
			}
			break
		}
	}

	return dir
}
