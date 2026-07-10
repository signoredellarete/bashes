package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestParseTransferID(t *testing.T) {
	tests := []struct {
		id        string
		wantScope string
		wantRel   string
	}{
		{id: "/local", wantScope: "local", wantRel: ""},
		{id: "/local/docs/readme.txt", wantScope: "local", wantRel: "docs/readme.txt"},
		{id: "/remote", wantScope: "remote", wantRel: ""},
		{id: "/remote/var/log/syslog", wantScope: "remote", wantRel: "var/log/syslog"},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			scope, rel, err := parseTransferID(tt.id)
			if err != nil {
				t.Fatalf("parseTransferID() error = %v", err)
			}
			if scope != tt.wantScope || rel != tt.wantRel {
				t.Fatalf("parseTransferID() = (%q, %q), want (%q, %q)", scope, rel, tt.wantScope, tt.wantRel)
			}
		})
	}
}

func TestValidateCopyDestinationRejectsSelfAndDescendant(t *testing.T) {
	tests := []struct {
		name      string
		sourceRel string
		targetRel string
	}{
		{name: "same path", sourceRel: "projects/app", targetRel: "projects/app"},
		{name: "inside source", sourceRel: "projects/app", targetRel: "projects/app/backup"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCopyDestination("local", tt.sourceRel, "local", tt.targetRel)
			if err == nil {
				t.Fatal("validateCopyDestination() error = nil, want self-copy error")
			}
			if !strings.Contains(err.Error(), "itself") {
				t.Fatalf("validateCopyDestination() error = %v, want itself error", err)
			}
		})
	}
}

func TestValidateCopyDestinationAllowsSiblingAndDifferentScope(t *testing.T) {
	tests := []struct {
		name        string
		sourceScope string
		sourceRel   string
		targetScope string
		targetRel   string
	}{
		{name: "sibling with shared prefix", sourceScope: "local", sourceRel: "projects/app", targetScope: "local", targetRel: "projects/app-copy"},
		{name: "different scope", sourceScope: "local", sourceRel: "projects/app", targetScope: "remote", targetRel: "projects/app"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateCopyDestination(tt.sourceScope, tt.sourceRel, tt.targetScope, tt.targetRel); err != nil {
				t.Fatalf("validateCopyDestination() error = %v", err)
			}
		})
	}
}

func TestFileTransferLocalPathStaysUnderRoot(t *testing.T) {
	root := t.TempDir()
	session := &fileTransferSession{localRoot: root}

	got, err := session.localPath("nested/file.txt")
	if err != nil {
		t.Fatalf("localPath(valid) error = %v", err)
	}
	want := filepath.Join(root, "nested", "file.txt")
	if got != want {
		t.Fatalf("localPath(valid) = %q, want %q", got, want)
	}

	got, err = session.localPath("../../etc/passwd")
	if err != nil {
		t.Fatalf("localPath(cleaned escape) error = %v", err)
	}
	if got != filepath.Join(root, "etc", "passwd") {
		t.Fatalf("localPath(cleaned escape) = %q, want path under root", got)
	}
}
