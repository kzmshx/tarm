package git

import (
	"testing"
)

func TestBuildDiffArgs(t *testing.T) {
	tests := []struct {
		name    string
		baseRef string
		headRef string
		want    []string
	}{
		{
			name:    "headRef empty uses HEAD",
			baseRef: "origin/main",
			headRef: "",
			want:    []string{"diff", "--name-only", "origin/main", "HEAD"},
		},
		{
			name:    "headRef HEAD uses two-arg form",
			baseRef: "origin/main",
			headRef: "HEAD",
			want:    []string{"diff", "--name-only", "origin/main", "HEAD"},
		},
		{
			name:    "explicit headRef uses three-dot form",
			baseRef: "origin/main",
			headRef: "origin/feature",
			want:    []string{"diff", "--name-only", "origin/main...origin/feature"},
		},
		{
			name:    "bare branch names without origin prefix",
			baseRef: "main",
			headRef: "feature",
			want:    []string{"diff", "--name-only", "main...feature"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildDiffArgs(tt.baseRef, tt.headRef)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("got %v, want %v", got, tt.want)
					break
				}
			}
		})
	}
}

func TestStaticProvider(t *testing.T) {
	p := &StaticProvider{Files: []string{"a.tf", "b.tf"}}
	files, err := p.ChangedFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 || files[0] != "a.tf" || files[1] != "b.tf" {
		t.Errorf("got %v, want [a.tf b.tf]", files)
	}
}

func TestStaticProvider_Empty(t *testing.T) {
	p := &StaticProvider{}
	files, err := p.ChangedFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Errorf("got %v, want empty", files)
	}
}

func TestMultiProvider(t *testing.T) {
	p := &MultiProvider{
		Providers: []ChangedFilesProvider{
			&StaticProvider{Files: []string{"a.tf", "b.tf"}},
			&StaticProvider{Files: []string{"b.tf", "c.tf"}},
		},
	}
	files, err := p.ChangedFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 3 {
		t.Errorf("got %v (len=%d), want 3 unique files", files, len(files))
	}
}

func TestMultiProvider_Empty(t *testing.T) {
	p := &MultiProvider{}
	files, err := p.ChangedFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Errorf("got %v, want empty", files)
	}
}
