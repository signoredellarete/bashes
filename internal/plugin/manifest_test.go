package plugin

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLoadManifestAndResolveExecutable(t *testing.T) {
	dir := t.TempDir()
	platform := runtime.GOOS + "/" + runtime.GOARCH
	executable := filepath.Join(dir, "bin", "fixture")
	if runtime.GOOS == "windows" {
		executable += ".exe"
	}
	if err := os.MkdirAll(filepath.Dir(executable), 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(executable, []byte("fixture"), 0o700); err != nil {
		t.Fatalf("WriteFile(executable) error = %v", err)
	}
	manifestJSON := `{
  "schemaVersion": 1,
  "id": "org.bashes.fixture",
  "name": "Fixture",
  "version": "1.2.3",
  "apiVersion": 1,
  "capabilities": ["diagnostic"],
  "permissions": ["runtime-data.write"],
  "executables": {"` + platform + `": "bin/` + filepath.Base(executable) + `"}
}`
	if err := os.WriteFile(filepath.Join(dir, ManifestFilename), []byte(manifestJSON), 0o600); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}

	manifest, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest() error = %v", err)
	}
	if !manifest.CompatibleWith(HostAPIVersion) || !manifest.Supports(CapabilityDiagnostic) {
		t.Fatalf("manifest capabilities = %#v", manifest)
	}
	resolved, err := manifest.Executable(dir, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Fatalf("Executable() error = %v", err)
	}
	if resolved != executable {
		t.Fatalf("Executable() = %q, want %q", resolved, executable)
	}
}

func TestManifestRejectsUnsafeOrUnknownFields(t *testing.T) {
	tests := []struct {
		name     string
		manifest string
		want     string
	}{
		{name: "path traversal", manifest: validManifest(`{"linux/amd64":"../plugin"}`), want: "inside"},
		{name: "unknown permission", manifest: strings.Replace(validManifest(`{"linux/amd64":"bin/plugin"}`), `"permissions":[]`, `"permissions":["everything"]`, 1), want: "unknown plugin permission"},
		{name: "unknown field", manifest: strings.Replace(validManifest(`{"linux/amd64":"bin/plugin"}`), `"name":"Fixture"`, `"name":"Fixture","surprise":true`, 1), want: "unknown field"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, ManifestFilename), []byte(test.manifest), 0o600); err != nil {
				t.Fatalf("WriteFile() error = %v", err)
			}
			_, err := LoadManifest(dir)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("LoadManifest() error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestExecutableRejectsSymlinkEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privileges on Windows")
	}
	pluginDir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, []byte("outside"), 0o700); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(pluginDir, "bin"), 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(pluginDir, "bin", "plugin")); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}
	manifest := Manifest{
		SchemaVersion: ManifestSchema,
		ID:            "org.bashes.fixture",
		Name:          "Fixture",
		Version:       "1.0.0",
		APIVersion:    HostAPIVersion,
		Capabilities:  []Capability{CapabilityDiagnostic},
		Executables:   map[string]string{runtime.GOOS + "/" + runtime.GOARCH: "bin/plugin"},
	}
	if _, err := manifest.Executable(pluginDir, runtime.GOOS, runtime.GOARCH); err == nil || !strings.Contains(err.Error(), "outside") {
		t.Fatalf("Executable() error = %v, want outside-directory error", err)
	}
}

func validManifest(executables string) string {
	return `{"schemaVersion":1,"id":"org.bashes.fixture","name":"Fixture","version":"1.0.0","apiVersion":1,"capabilities":["diagnostic"],"permissions":[],"executables":` + executables + `}`
}
