package main

import (
	"path/filepath"
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
