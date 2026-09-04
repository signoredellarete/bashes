package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateOpenableFile(t *testing.T) {
	for _, name := range []string{"report.pdf", "diagram.SVG", "notes.txt", "sheet.xlsx", "video.mp4"} {
		if err := validateOpenableFile(name); err != nil {
			t.Fatalf("validateOpenableFile(%q) error = %v", name, err)
		}
	}
	for _, name := range []string{"script.sh", "program.exe", "README"} {
		if err := validateOpenableFile(name); err == nil {
			t.Fatalf("validateOpenableFile(%q) error = nil, want unsupported type", name)
		}
	}
}

func TestCacheFileNameSanitizesRemoteName(t *testing.T) {
	got := cacheFileName("docs/report:final?.pdf")
	if got != "report_final_.pdf" {
		t.Fatalf("cacheFileName() = %q, want %q", got, "report_final_.pdf")
	}
}

func TestStartFileTransferOpenJobOpensLocalFileDirectly(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "report.pdf")
	if err := os.WriteFile(filePath, []byte("pdf"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	app := NewApp(filepath.Join(t.TempDir(), "hosts.json"))
	app.transfers["files-1"] = &fileTransferSession{
		id:         "files-1",
		resourceID: "host-1",
		localRoot:  root,
	}
	var opened string
	app.openFile = func(path string) error {
		opened = path
		return nil
	}

	job, err := app.StartFileTransferOpenJob(FileTransferOpenInput{
		SessionID: "files-1",
		ID:        "/local/report.pdf",
	})
	if err != nil {
		t.Fatalf("StartFileTransferOpenJob() error = %v", err)
	}
	if opened != filePath {
		t.Fatalf("opened path = %q, want %q", opened, filePath)
	}
	if job.JobID != "" || job.Operation != "open" || job.Status != "completed" {
		t.Fatalf("job = %+v, want immediate completed open result", job)
	}
}

func TestStartFileTransferOpenJobRejectsUnsupportedFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "script.sh"), []byte("echo test"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	app := NewApp(filepath.Join(t.TempDir(), "hosts.json"))
	app.transfers["files-1"] = &fileTransferSession{id: "files-1", resourceID: "host-1", localRoot: root}

	_, err := app.StartFileTransferOpenJob(FileTransferOpenInput{SessionID: "files-1", ID: "/local/script.sh"})
	if err == nil || !strings.Contains(err.Error(), ".sh files") {
		t.Fatalf("StartFileTransferOpenJob() error = %v, want unsupported extension", err)
	}
}
