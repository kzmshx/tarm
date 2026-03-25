package git

import (
	"testing"
)

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
