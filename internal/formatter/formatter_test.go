package formatter

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/kzmshx/tarm/internal/tarm"
)

func TestJSON(t *testing.T) {
	tests := []struct {
		name    string
		modules []tarm.AffectedRootModule
		want    string
	}{
		{name: "nil modules", modules: nil, want: "[]"},
		{name: "empty slice", modules: []tarm.AffectedRootModule{}, want: "[]"},
		{name: "single module", modules: []tarm.AffectedRootModule{{Path: "environments/dev/api", AffectedBy: []string{"modules/database"}}}},
		{name: "multiple modules", modules: []tarm.AffectedRootModule{
			{Path: "environments/dev/api", AffectedBy: []string{"modules/database"}},
			{Path: "environments/prod/api", AffectedBy: []string{"modules/database", "modules/common"}},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := JSON(tt.modules)
			if tt.want != "" {
				if got != tt.want {
					t.Errorf("JSON() = %q, want %q", got, tt.want)
				}
				return
			}
			var parsed []tarm.AffectedRootModule
			if err := json.Unmarshal([]byte(got), &parsed); err != nil {
				t.Fatalf("invalid JSON: %v\nOutput: %s", err, got)
			}
			if len(parsed) != len(tt.modules) {
				t.Errorf("round-trip: got %d, want %d", len(parsed), len(tt.modules))
			}
		})
	}
}

func TestMarkdown(t *testing.T) {
	tests := []struct {
		name         string
		modules      []tarm.AffectedRootModule
		wantContains []string
		wantAbsent   []string
	}{
		{
			name:         "empty",
			modules:      []tarm.AffectedRootModule{},
			wantContains: []string{"## Terraform Affected Root Modules", "No affected root modules found."},
			wantAbsent:   []string{"<details>"},
		},
		{
			name:         "single module",
			modules:      []tarm.AffectedRootModule{{Path: "environments/dev/api", AffectedBy: []string{"modules/database"}}},
			wantContains: []string{"**1** root module(s) affected:", "<details><summary>environments/dev/api</summary>", "- modules/database"},
		},
		{
			name: "multiple modules",
			modules: []tarm.AffectedRootModule{
				{Path: "environments/dev/api", AffectedBy: []string{"modules/database"}},
				{Path: "environments/prod/api", AffectedBy: []string{"modules/database"}},
			},
			wantContains: []string{"**2** root module(s) affected:", "environments/dev/api", "environments/prod/api"},
		},
		{
			name:         "deduplicates causes",
			modules:      []tarm.AffectedRootModule{{Path: "environments/dev/api", AffectedBy: []string{"modules/database", "modules/database", "modules/common"}}},
			wantContains: []string{"- modules/database", "- modules/common"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Markdown(tt.modules)
			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("missing %q in:\n%s", want, got)
				}
			}
			for _, absent := range tt.wantAbsent {
				if strings.Contains(got, absent) {
					t.Errorf("should not contain %q in:\n%s", absent, got)
				}
			}
		})
	}
}

func TestFindParentModule(t *testing.T) {
	tests := []struct {
		file string
		want string
	}{
		{"infrastructure/environments/dev/api/main.tf", "infrastructure/environments/dev/api"},
		{"infrastructure/modules/auth/main.tf", "infrastructure/modules/auth"},
		{"scripts/deploy.sh", "scripts"},
	}
	for _, tt := range tests {
		got := FindParentModule(tt.file)
		if got != tt.want {
			t.Errorf("FindParentModule(%q) = %q, want %q", tt.file, got, tt.want)
		}
	}
}
