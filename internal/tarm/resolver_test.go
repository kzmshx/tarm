package tarm

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizePath(t *testing.T) {
	root, _ := filepath.Abs("/tmp/test-root")

	tests := []struct {
		name    string
		root    string
		path    string
		wantRel string
	}{
		{name: "relative path", root: root, path: "modules/auth", wantRel: "modules/auth"},
		{name: "absolute path within root", root: root, path: filepath.Join(root, "modules/auth"), wantRel: "modules/auth"},
		{name: "path with dot segments", root: root, path: "modules/../modules/auth", wantRel: "modules/auth"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizePath(tt.root, tt.path)
			if err != nil {
				t.Fatalf("NormalizePath() error = %v", err)
			}
			if got != tt.wantRel {
				t.Errorf("NormalizePath() = %q, want %q", got, tt.wantRel)
			}
		})
	}
}

func TestResolveModuleSource(t *testing.T) {
	tests := []struct {
		name      string
		callerDir string
		source    string
		wantPath  string
		wantEmpty bool
	}{
		{name: "relative path with dot-dot", callerDir: "/repo/environments/dev/api", source: "../../../modules/network", wantPath: "/repo/modules/network"},
		{name: "current directory reference", callerDir: "/repo/modules/auth", source: "./sub", wantPath: "/repo/modules/auth/sub"},
		{name: "registry module returns empty", callerDir: "/repo", source: "terraform-aws-modules/vpc/aws", wantEmpty: true},
		{name: "git source returns empty", callerDir: "/repo", source: "git::https://github.com/example/module.git", wantEmpty: true},
		{name: "https source returns empty", callerDir: "/repo", source: "https://example.com/module.zip", wantEmpty: true},
		{name: "s3 source returns empty", callerDir: "/repo", source: "s3::https://bucket/module.zip", wantEmpty: true},
		{name: "gcs source returns empty", callerDir: "/repo", source: "gcs::https://bucket/module.zip", wantEmpty: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveModuleSource(tt.callerDir, tt.source)
			if err != nil {
				t.Fatalf("ResolveModuleSource() error = %v", err)
			}
			if tt.wantEmpty && got != "" {
				t.Errorf("got %q, want empty", got)
			} else if !tt.wantEmpty && got != tt.wantPath {
				t.Errorf("got %q, want %q", got, tt.wantPath)
			}
		})
	}
}

func TestFindParentWithTerraformFiles(t *testing.T) {
	testRoot, _ := filepath.Abs(filepath.Join("..", "..", "testdata", "terraform"))

	tests := []struct {
		name      string
		start     string
		wantDir   string
		wantEmpty bool
	}{
		{name: "file in directory with .tf files", start: filepath.Join(testRoot, "environments", "dev", "api", "main.tf"), wantDir: filepath.Join(testRoot, "environments", "dev", "api")},
		{name: "directory with .tf files", start: filepath.Join(testRoot, "environments", "dev", "api"), wantDir: filepath.Join(testRoot, "environments", "dev", "api")},
		{name: "nested non-tf file escalates to parent", start: filepath.Join(testRoot, "modules", "database", "configs", "nested", "schema.json"), wantDir: filepath.Join(testRoot, "modules", "database")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := os.Stat(tt.start); os.IsNotExist(err) {
				t.Skipf("test path does not exist: %s", tt.start)
			}
			got, err := FindParentWithTerraformFiles(tt.start, testRoot)
			if err != nil {
				t.Fatalf("error = %v", err)
			}
			if tt.wantEmpty {
				if got != "" {
					t.Errorf("got %q, want empty", got)
				}
				return
			}
			if got != tt.wantDir {
				t.Errorf("got %q, want %q", got, tt.wantDir)
			}
		})
	}
}

func TestIsWithinDirectory(t *testing.T) {
	tests := []struct {
		name string
		path string
		dir  string
		want bool
	}{
		{name: "inside", path: "a/b/c", dir: "a", want: true},
		{name: "same", path: "a/b", dir: "a/b", want: true},
		{name: "outside", path: "x/y", dir: "a", want: false},
		{name: "parent", path: "a", dir: "a/b", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsWithinDirectory(tt.path, tt.dir)
			if got != tt.want {
				t.Errorf("IsWithinDirectory(%q, %q) = %v, want %v", tt.path, tt.dir, got, tt.want)
			}
		})
	}
}
