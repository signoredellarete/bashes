package fsutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileAtomicCreatesAndReplacesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "settings.json")
	if err := WriteFileAtomic(path, []byte("first"), 0o600); err != nil {
		t.Fatalf("WriteFileAtomic(first) error = %v", err)
	}
	if err := WriteFileAtomic(path, []byte("second"), 0o600); err != nil {
		t.Fatalf("WriteFileAtomic(second) error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "second" {
		t.Fatalf("file = %q, want second", data)
	}
}

func TestWriteFileAtomicRejectsDirectoryDestination(t *testing.T) {
	path := filepath.Join(t.TempDir(), "destination")
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}
	if err := WriteFileAtomic(path, []byte("data"), 0o600); err == nil {
		t.Fatal("WriteFileAtomic() error = nil, want directory error")
	}
}
