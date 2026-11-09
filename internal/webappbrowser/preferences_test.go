package webappbrowser

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/signoredellarete/bashes/internal/apperrors"
	appsettings "github.com/signoredellarete/bashes/internal/settings"
)

func TestPreferencesServiceSaveAndResolveCustomBrowser(t *testing.T) {
	root := t.TempDir()
	settings := appsettings.NewRepository(filepath.Join(root, "settings.json"))
	service := NewPreferencesService("linux", func(string) string { return "" }, settings)
	browserPath := filepath.Join(root, "custom-firefox")
	if err := os.WriteFile(browserPath, []byte("binary"), 0o700); err != nil {
		t.Fatal(err)
	}

	saved, err := service.Save(appsettings.WebAppBrowser{
		PreferredBrowserID: string(Firefox),
		CustomBrowserPath:  browserPath,
	})
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if saved.CustomBrowserPath != browserPath {
		t.Fatalf("saved path = %q, want %q", saved.CustomBrowserPath, browserPath)
	}

	resolved, err := service.Resolve()
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved.ID != Firefox || resolved.Family != FamilyFirefox || resolved.Path != browserPath {
		t.Fatalf("resolved browser = %+v", resolved)
	}
}

func TestPreferencesServiceRejectsNonExecutableCustomBrowser(t *testing.T) {
	root := t.TempDir()
	service := NewPreferencesService(
		"linux",
		func(string) string { return "" },
		appsettings.NewRepository(filepath.Join(root, "settings.json")),
	)
	browserPath := filepath.Join(root, "browser")
	if err := os.WriteFile(browserPath, []byte("binary"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := service.Save(appsettings.WebAppBrowser{
		PreferredBrowserID: string(Firefox),
		CustomBrowserPath:  browserPath,
	})
	if err == nil {
		t.Fatal("Save() error = nil, want executable validation error")
	}
}

func TestPreferencesServiceResolveReportsMissingPreferredBrowser(t *testing.T) {
	root := t.TempDir()
	service := NewPreferencesService(
		"linux",
		func(string) string { return "" },
		appsettings.NewRepository(filepath.Join(root, "settings.json")),
	)
	service.detector.lookPath = func(string) (string, error) {
		return "", errors.New("not found")
	}
	if _, err := service.Save(appsettings.WebAppBrowser{PreferredBrowserID: string(Firefox)}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	_, err := service.Resolve()
	payload, ok := apperrors.Decode(err)
	if !ok || payload.Code != "webapp_browser_missing" {
		t.Fatalf("Resolve() error = %v, want webapp_browser_missing", err)
	}
}

func TestPreferencesServiceUsesDetectedBrowserPriority(t *testing.T) {
	root := t.TempDir()
	service := NewPreferencesService(
		"linux",
		func(string) string { return "" },
		appsettings.NewRepository(filepath.Join(root, "settings.json")),
	)
	service.detector.lookPath = func(name string) (string, error) {
		if name == "firefox" {
			return "/usr/bin/firefox", nil
		}
		return "", errors.New("not found")
	}

	resolved, err := service.Resolve()
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved.ID != Firefox || resolved.Path != "/usr/bin/firefox" {
		t.Fatalf("resolved browser = %+v", resolved)
	}
}
