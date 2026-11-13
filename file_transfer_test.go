package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestRemoteTransferID(t *testing.T) {
	tests := []struct {
		name   string
		root   string
		target string
		want   string
	}{
		{name: "unix home", root: "/", target: "/home/alice", want: "/remote/home/alice"},
		{name: "filesystem root", root: "/", target: "/", want: "/remote"},
		{name: "chroot home", root: "/srv/sftp", target: "/srv/sftp/home/alice", want: "/remote/home/alice"},
		{name: "outside root", root: "/srv/sftp", target: "/home/alice", want: "/remote"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := remoteTransferID(tt.root, tt.target); got != tt.want {
				t.Fatalf("remoteTransferID(%q, %q) = %q, want %q", tt.root, tt.target, got, tt.want)
			}
		})
	}
}

func TestRemoteRelativePathUsesPathBoundaries(t *testing.T) {
	if _, ok := remoteRelativePath("/srv/sftp", "/srv/sftp-other/file"); ok {
		t.Fatal("remoteRelativePath() accepted a sibling with a shared prefix")
	}
	if got, ok := remoteRelativePath("/", "/work/team"); !ok || got != "work/team" {
		t.Fatalf("remoteRelativePath() = (%q, %t), want (%q, true)", got, ok, "work/team")
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

func TestAtomicDestinationUsesInjectedClock(t *testing.T) {
	root := t.TempDir()
	session := &fileTransferSession{
		localRoot: root,
		now:       func() time.Time { return time.Unix(0, 1234) },
	}
	var temporaryPath string
	err := session.withAtomicDestination("local", "result.txt", func(relative string) error {
		temporaryPath = relative
		path, pathErr := session.localPath(relative)
		if pathErr != nil {
			return pathErr
		}
		return os.WriteFile(path, []byte("result"), 0o600)
	})
	if err != nil {
		t.Fatalf("withAtomicDestination() error = %v", err)
	}
	if !strings.Contains(temporaryPath, ".result.txt.bashes-partial-1234") {
		t.Fatalf("temporary path = %q", temporaryPath)
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

func TestFileTransferLocalPathRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "outside")); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	session := &fileTransferSession{localRoot: root}
	if _, err := session.localPath("outside/file.txt"); err == nil || !strings.Contains(err.Error(), "escapes transfer root") {
		t.Fatalf("localPath() error = %v, want symlink escape rejection", err)
	}
}

func TestAtomicFileTransferRejectsConflictWithoutChangingDestination(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "source.txt"), []byte("source"), 0o600); err != nil {
		t.Fatalf("WriteFile(source) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "target.txt"), []byte("target"), 0o600); err != nil {
		t.Fatalf("WriteFile(target) error = %v", err)
	}

	session := &fileTransferSession{localRoot: root}
	err := session.copyItemAtomic(context.Background(), "local", "source.txt", "local", "target.txt")
	if err == nil || !strings.Contains(err.Error(), "destination already exists") {
		t.Fatalf("copyItemAtomic() error = %v, want conflict error", err)
	}
	data, err := os.ReadFile(filepath.Join(root, "target.txt"))
	if err != nil {
		t.Fatalf("ReadFile(target) error = %v", err)
	}
	if string(data) != "target" {
		t.Fatalf("target content = %q, want unchanged target", data)
	}
}

func TestFileTransferCopyConflictsDescribeExistingDestination(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "destination"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "source.txt"), []byte("source"), 0o600); err != nil {
		t.Fatalf("WriteFile(source) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "destination", "source.txt"), []byte("existing"), 0o600); err != nil {
		t.Fatalf("WriteFile(destination) error = %v", err)
	}

	session := &fileTransferSession{localRoot: root}
	conflicts, err := session.copyConflicts(FileTransferCopyInput{
		IDs:      []string{"/local/source.txt"},
		TargetID: "/local/destination",
	})
	if err != nil {
		t.Fatalf("copyConflicts() error = %v", err)
	}
	if len(conflicts) != 1 {
		t.Fatalf("copyConflicts() count = %d, want 1", len(conflicts))
	}
	want := FileTransferConflict{
		SourceID: "/local/source.txt",
		TargetID: "/local/destination/source.txt",
		Name:     "source.txt",
		Type:     "file",
	}
	if conflicts[0] != want {
		t.Fatalf("copyConflicts()[0] = %#v, want %#v", conflicts[0], want)
	}
}

func TestAtomicDestinationPolicySkipsExistingDestination(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target.txt")
	if err := os.WriteFile(target, []byte("existing"), 0o600); err != nil {
		t.Fatalf("WriteFile(target) error = %v", err)
	}

	session := &fileTransferSession{localRoot: root}
	writerCalled := false
	copied, err := session.withAtomicDestinationPolicy("local", "target.txt", "skip", func(string) error {
		writerCalled = true
		return nil
	})
	if err != nil {
		t.Fatalf("withAtomicDestinationPolicy() error = %v", err)
	}
	if copied || writerCalled {
		t.Fatalf("withAtomicDestinationPolicy() = (%t, writer %t), want skipped", copied, writerCalled)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile(target) error = %v", err)
	}
	if string(data) != "existing" {
		t.Fatalf("target content = %q, want unchanged destination", data)
	}
}

func TestAtomicDestinationPolicyOverwritesAndRemovesBackup(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target.txt")
	if err := os.WriteFile(target, []byte("existing"), 0o600); err != nil {
		t.Fatalf("WriteFile(target) error = %v", err)
	}

	session := &fileTransferSession{
		localRoot: root,
		now:       func() time.Time { return time.Unix(0, 1234) },
	}
	copied, err := session.withAtomicDestinationPolicy("local", "target.txt", "overwrite", func(relative string) error {
		temporary, pathErr := session.localPath(relative)
		if pathErr != nil {
			return pathErr
		}
		return os.WriteFile(temporary, []byte("replacement"), 0o600)
	})
	if err != nil {
		t.Fatalf("withAtomicDestinationPolicy() error = %v", err)
	}
	if !copied {
		t.Fatal("withAtomicDestinationPolicy() copied = false, want true")
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile(target) error = %v", err)
	}
	if string(data) != "replacement" {
		t.Fatalf("target content = %q, want replacement", data)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "target.txt" {
		t.Fatalf("destination directory = %v, want only target.txt", entries)
	}
}
