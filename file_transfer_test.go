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

func TestReserveFileTransferJobIsAtomicPerResource(t *testing.T) {
	app := NewApp(filepath.Join(t.TempDir(), "hosts.json"))
	first := &fileTransferJob{
		info: FileTransferJobInfo{JobID: "one", ResourceID: "host-1", Status: "running"},
		done: make(chan struct{}),
	}
	second := &fileTransferJob{
		info: FileTransferJobInfo{JobID: "two", ResourceID: "host-1", Status: "queued"},
		done: make(chan struct{}),
	}
	if err := app.reserveFileTransferJob(first); err != nil {
		t.Fatalf("reserve first job error = %v", err)
	}
	if err := app.reserveFileTransferJob(second); err == nil {
		t.Fatal("reserve second job error = nil, want active resource conflict")
	}
}

func TestDismissFileTransferJobOnlyRemovesFinishedJobs(t *testing.T) {
	app := NewApp(filepath.Join(t.TempDir(), "hosts.json"))
	job := &fileTransferJob{
		info: FileTransferJobInfo{JobID: "job-1", ResourceID: "host-1", Status: "running"},
		done: make(chan struct{}),
	}
	app.transferJobs[job.info.JobID] = job
	if err := app.DismissFileTransferJob(FileTransferDismissJobInput{JobID: job.info.JobID}); err == nil {
		t.Fatal("DismissFileTransferJob(active) error = nil")
	}
	job.update(func(info *FileTransferJobInfo) {
		info.Status = "completed"
		info.FinishedAt = time.Now().Format(time.RFC3339)
	})
	if err := app.DismissFileTransferJob(FileTransferDismissJobInput{JobID: job.info.JobID}); err != nil {
		t.Fatalf("DismissFileTransferJob(completed) error = %v", err)
	}
	if _, exists := app.transferJobs[job.info.JobID]; exists {
		t.Fatal("completed job still retained after dismiss")
	}
}
