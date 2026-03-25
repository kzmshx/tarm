package tarm

import (
	"testing"
)

func TestUnique(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{name: "no duplicates", input: []string{"a", "b", "c"}, want: []string{"a", "b", "c"}},
		{name: "with duplicates", input: []string{"a", "b", "a", "c", "b"}, want: []string{"a", "b", "c"}},
		{name: "all same", input: []string{"a", "a", "a"}, want: []string{"a"}},
		{name: "empty slice", input: []string{}, want: nil},
		{name: "nil slice", input: nil, want: nil},
		{name: "single element", input: []string{"a"}, want: []string{"a"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Unique(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("Unique() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("Unique() = %v, want %v", got, tt.want)
					return
				}
			}
		})
	}
}

func TestUniqueInt(t *testing.T) {
	got := Unique([]int{1, 2, 1, 3, 2})
	want := []int{1, 2, 3}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestParseMultilineInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{name: "single line", input: "environments/*/*", want: []string{"environments/*/*"}},
		{name: "multiple lines", input: "environments/*/*\nstacks/*/*", want: []string{"environments/*/*", "stacks/*/*"}},
		{name: "lines with whitespace", input: "  environments/*/*  \n  stacks/*/*  ", want: []string{"environments/*/*", "stacks/*/*"}},
		{name: "empty lines ignored", input: "environments/*/*\n\n\nstacks/*/*\n", want: []string{"environments/*/*", "stacks/*/*"}},
		{name: "empty string", input: "", want: nil},
		{name: "only whitespace", input: "  \n  \n  ", want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseMultilineInput(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("ParseMultilineInput() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ParseMultilineInput() = %v, want %v", got, tt.want)
					return
				}
			}
		})
	}
}
