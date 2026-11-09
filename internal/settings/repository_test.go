package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRepositoryLoadsLegacyFlatSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	legacy := `{"customDirectory":" /keys ","preferredBrowserId":" firefox ","customBrowserPath":" /browser "}`
	if err := os.WriteFile(path, []byte(legacy), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	data, err := NewRepository(path).Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if data.Version != CurrentVersion || data.CustomDirectory != "/keys" || data.PreferredBrowserID != "firefox" || data.CustomBrowserPath != "/browser" {
		t.Fatalf("Load() = %#v", data)
	}
}

func TestRepositoryUpdatePreservesUnrelatedSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	repository := NewRepository(path)
	if _, err := repository.Update(func(data *Data) error {
		data.CustomDirectory = "/keys"
		data.PreferredBrowserID = "firefox"
		return nil
	}); err != nil {
		t.Fatalf("Update(first) error = %v", err)
	}
	if _, err := repository.Update(func(data *Data) error {
		data.CustomBrowserPath = "/browser"
		return nil
	}); err != nil {
		t.Fatalf("Update(second) error = %v", err)
	}

	data, err := repository.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if data.CustomDirectory != "/keys" || data.PreferredBrowserID != "firefox" || data.CustomBrowserPath != "/browser" {
		t.Fatalf("Load() = %#v", data)
	}
}
