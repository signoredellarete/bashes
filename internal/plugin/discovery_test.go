package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDiscoverReportsAvailableAndInvalidPlugins(t *testing.T) {
	root := t.TempDir()
	createDiscoveryFixture(t, root, "available", HostAPIVersion, true)
	createDiscoveryFixture(t, root, "incompatible", HostAPIVersion+1, true)
	if err := os.Mkdir(filepath.Join(root, "invalid"), 0o700); err != nil {
		t.Fatalf("Mkdir(invalid) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.txt"), []byte("ignored"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	candidates, err := Discover(root, runtime.GOOS, runtime.GOARCH, HostAPIVersion)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(candidates) != 3 {
		t.Fatalf("Discover() returned %d candidates, want 3", len(candidates))
	}
	byName := make(map[string]Candidate, len(candidates))
	for _, candidate := range candidates {
		byName[filepath.Base(candidate.Directory)] = candidate
	}
	if !byName["available"].Available() {
		t.Fatalf("available plugin error = %v", byName["available"].Err)
	}
	if err := byName["incompatible"].Err; err == nil || !strings.Contains(err.Error(), "incompatible") {
		t.Fatalf("incompatible plugin error = %v", err)
	}
	if byName["invalid"].Err == nil {
		t.Fatal("invalid plugin error = nil")
	}
}

func TestDiscoverMissingRootIsEmpty(t *testing.T) {
	candidates, err := Discover(filepath.Join(t.TempDir(), "missing"), runtime.GOOS, runtime.GOARCH, HostAPIVersion)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(candidates) != 0 {
		t.Fatalf("Discover() = %#v, want empty", candidates)
	}
}

func createDiscoveryFixture(t *testing.T, root, name string, apiVersion int, executable bool) {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Join(dir, "bin"), 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	path := filepath.Join(dir, "bin", "plugin")
	if executable {
		if err := os.WriteFile(path, []byte("fixture"), 0o700); err != nil {
			t.Fatalf("WriteFile(executable) error = %v", err)
		}
	}
	manifest := fmt.Sprintf(`{"schemaVersion":1,"id":"org.bashes.%s","name":"%s","version":"1.0.0","apiVersion":%d,"capabilities":["diagnostic"],"executables":{"%s/%s":"bin/plugin"}}`, name, name, apiVersion, runtime.GOOS, runtime.GOARCH)
	if err := os.WriteFile(filepath.Join(dir, ManifestFilename), []byte(manifest), 0o600); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
}
