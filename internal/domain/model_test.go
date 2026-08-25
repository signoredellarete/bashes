package domain

import (
	"strings"
	"testing"
)

func TestNormalizeTagsTrimsAndDeduplicates(t *testing.T) {
	got := NormalizeTags([]string{" prod ", "HPC", "PROD", "", " hpc "})
	want := []string{"prod", "HPC"}
	if len(got) != len(want) {
		t.Fatalf("NormalizeTags() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("NormalizeTags()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestValidateTagsRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name string
		tags []string
		want string
	}{
		{name: "control", tags: []string{"prod\nwork"}, want: "control"},
		{name: "too long", tags: []string{strings.Repeat("a", MaxTagLength+1)}, want: "exceeds"},
		{name: "duplicate", tags: []string{"prod", "PROD"}, want: "duplicate"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateTags(test.tags)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("ValidateTags(%v) error = %v, want %q", test.tags, err, test.want)
			}
		})
	}
}
