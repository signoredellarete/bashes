package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunValidateLegacyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.json")
	legacy := `[{"hostname":"host","ip":"127.0.0.1","port":"22","user":"root","lxc":[],"vm":[],"docker":[]}]`
	if err := os.WriteFile(path, []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := run([]string{"validate", path}); err != nil {
		t.Fatalf("run(validate) error = %v", err)
	}
}

func TestRunMigrateLegacyFile(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "legacy.json")
	output := filepath.Join(dir, "hosts.json")
	legacy := `[{"hostname":"host","ip":"127.0.0.1","port":"22","user":"root","lxc":[],"vm":[],"docker":[]}]`
	if err := os.WriteFile(input, []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := run([]string{"migrate", input, output}); err != nil {
		t.Fatalf("run(migrate) error = %v", err)
	}

	if _, err := os.Stat(output); err != nil {
		t.Fatalf("migrated output missing: %v", err)
	}
}

func TestRunRejectsInvalidArgs(t *testing.T) {
	if err := run([]string{"validate"}); err == nil {
		t.Fatal("run(validate) error = nil, want usage error")
	}
}
