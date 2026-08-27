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

func TestNormalizeStoreBuildsTagCatalogFromNestedResources(t *testing.T) {
	store := NormalizeStore(Store{
		Version: CurrentSchemaVersion,
		Tags:    []string{"Unassigned"},
		Hosts: []Host{{
			ID:       "host-1",
			Hostname: "host",
			IP:       "127.0.0.1",
			Port:     22,
			User:     "root",
			Tags:     []string{"Prod"},
			Subsystems: []Endpoint{{
				ID:       "vm-1",
				Type:     ResourceVM,
				Hostname: "vm",
				IP:       "127.0.0.2",
				Port:     22,
				User:     "root",
				Tags:     []string{"prod", "Data"},
			}},
		}},
	})

	want := []string{"Unassigned", "Prod", "Data"}
	if len(store.Tags) != len(want) {
		t.Fatalf("Store tags = %v, want %v", store.Tags, want)
	}
	for i := range want {
		if store.Tags[i] != want[i] {
			t.Fatalf("Store tags[%d] = %q, want %q", i, store.Tags[i], want[i])
		}
	}
}
