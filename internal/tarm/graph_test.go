package tarm

import (
	"sort"
	"strings"
	"testing"
)

func TestAddDependency(t *testing.T) {
	tests := []struct {
		name     string
		edges    [][2]string
		wantDeps map[string][]string
		wantRevs map[string][]string
	}{
		{
			name:  "single dependency",
			edges: [][2]string{{"a", "b"}},
			wantDeps: map[string][]string{"a": {"b"}},
			wantRevs: map[string][]string{"b": {"a"}},
		},
		{
			name:  "multiple dependencies from one module",
			edges: [][2]string{{"a", "b"}, {"a", "c"}},
			wantDeps: map[string][]string{"a": {"b", "c"}},
			wantRevs: map[string][]string{"b": {"a"}, "c": {"a"}},
		},
		{
			name:  "duplicate dependency is not added twice",
			edges: [][2]string{{"a", "b"}, {"a", "b"}},
			wantDeps: map[string][]string{"a": {"b"}},
			wantRevs: map[string][]string{"b": {"a"}},
		},
		{
			name:  "path normalization cleans trailing slash",
			edges: [][2]string{{"a/b/", "c/d/"}},
			wantDeps: map[string][]string{"a/b": {"c/d"}},
			wantRevs: map[string][]string{"c/d": {"a/b"}},
		},
		{
			name:  "chain dependency a -> b -> c",
			edges: [][2]string{{"a", "b"}, {"b", "c"}},
			wantDeps: map[string][]string{"a": {"b"}, "b": {"c"}},
			wantRevs: map[string][]string{"b": {"a"}, "c": {"b"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewDependencyGraph()
			for _, e := range tt.edges {
				g.AddDependency(e[0], e[1])
			}

			for mod, wantDeps := range tt.wantDeps {
				gotDeps := g.Dependencies[mod]
				if len(gotDeps) != len(wantDeps) {
					t.Errorf("Dependencies[%s]: got %v, want %v", mod, gotDeps, wantDeps)
					continue
				}
				sort.Strings(gotDeps)
				sort.Strings(wantDeps)
				for i := range gotDeps {
					if gotDeps[i] != wantDeps[i] {
						t.Errorf("Dependencies[%s]: got %v, want %v", mod, gotDeps, wantDeps)
						break
					}
				}
			}

			for mod, wantRevs := range tt.wantRevs {
				gotRevs := g.Dependents[mod]
				if len(gotRevs) != len(wantRevs) {
					t.Errorf("Dependents[%s]: got %v, want %v", mod, gotRevs, wantRevs)
					continue
				}
				sort.Strings(gotRevs)
				sort.Strings(wantRevs)
				for i := range gotRevs {
					if gotRevs[i] != wantRevs[i] {
						t.Errorf("Dependents[%s]: got %v, want %v", mod, gotRevs, wantRevs)
						break
					}
				}
			}
		})
	}
}

func TestGetAffectedModules(t *testing.T) {
	tests := []struct {
		name        string
		edges       [][2]string
		changedPath string
		wantModules []string
	}{
		{name: "direct dependent", edges: [][2]string{{"a", "b"}}, changedPath: "b", wantModules: []string{"a", "b"}},
		{name: "transitive dependents", edges: [][2]string{{"a", "b"}, {"b", "c"}}, changedPath: "c", wantModules: []string{"a", "b", "c"}},
		{name: "no dependents", edges: [][2]string{{"a", "b"}}, changedPath: "a", wantModules: []string{"a"}},
		{name: "multiple dependents on same module", edges: [][2]string{{"a", "c"}, {"b", "c"}}, changedPath: "c", wantModules: []string{"a", "b", "c"}},
		{name: "diamond dependency", edges: [][2]string{{"a", "b"}, {"a", "c"}, {"b", "d"}, {"c", "d"}}, changedPath: "d", wantModules: []string{"a", "b", "c", "d"}},
		{name: "unknown module returns only itself", edges: [][2]string{{"a", "b"}}, changedPath: "unknown", wantModules: []string{"unknown"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewDependencyGraph()
			for _, e := range tt.edges {
				g.AddDependency(e[0], e[1])
			}

			got := g.GetAffectedModules(tt.changedPath)
			sort.Strings(got)
			sort.Strings(tt.wantModules)

			if len(got) != len(tt.wantModules) {
				t.Errorf("got %v, want %v", got, tt.wantModules)
				return
			}
			for i := range got {
				if got[i] != tt.wantModules[i] {
					t.Errorf("got %v, want %v", got, tt.wantModules)
					return
				}
			}
		})
	}
}

func TestGetAllModules(t *testing.T) {
	g := NewDependencyGraph()
	g.AddDependency("a", "b")
	g.AddDependency("b", "c")
	g.AddDependency("d", "c")

	got := g.GetAllModules()
	sort.Strings(got)

	want := []string{"a", "b", "c", "d"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestDetectCircularDependencies(t *testing.T) {
	tests := []struct {
		name       string
		edges      [][2]string
		wantCycles bool
	}{
		{name: "no cycles", edges: [][2]string{{"a", "b"}, {"b", "c"}}, wantCycles: false},
		{name: "direct cycle", edges: [][2]string{{"a", "b"}, {"b", "a"}}, wantCycles: true},
		{name: "indirect cycle", edges: [][2]string{{"a", "b"}, {"b", "c"}, {"c", "a"}}, wantCycles: true},
		{name: "self-loop", edges: [][2]string{{"a", "a"}}, wantCycles: true},
		{name: "empty graph", edges: nil, wantCycles: false},
		{
			name:       "cycle in dependents-only module",
			edges:      [][2]string{{"a", "b"}, {"b", "c"}, {"c", "b"}},
			wantCycles: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewDependencyGraph()
			for _, e := range tt.edges {
				g.AddDependency(e[0], e[1])
			}

			cycles := g.DetectCircularDependencies()
			hasCycles := len(cycles) > 0
			if hasCycles != tt.wantCycles {
				t.Errorf("DetectCircularDependencies() found cycles = %v, want %v (cycles: %v)", hasCycles, tt.wantCycles, cycles)
			}
		})
	}
}

func TestGraphString(t *testing.T) {
	g := NewDependencyGraph()
	g.AddDependency("a", "b")

	s := g.String()
	if s == "" {
		t.Error("String() returned empty string")
	}
	if !strings.Contains(s, "a") || !strings.Contains(s, "b") {
		t.Errorf("String() = %q, expected to contain 'a' and 'b'", s)
	}
}
