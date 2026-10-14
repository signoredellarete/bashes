package webappbrowser

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectorUsesPriorityAndDeduplicatesCandidates(t *testing.T) {
	detector := NewDetector("linux", func(string) string { return "" })
	detector.lookPath = func(name string) (string, error) {
		switch name {
		case "firefox", "google-chrome-stable", "chromium":
			return "/usr/bin/" + name, nil
		default:
			return "", errors.New("not found")
		}
	}
	found := detector.Detect()
	if len(found) != 3 {
		t.Fatalf("Detect() = %+v, want 3 browsers", found)
	}
	if found[0].ID != Firefox || found[1].ID != Chrome || found[2].ID != Chromium {
		t.Fatalf("browser priority = %+v", found)
	}
}

func TestChromiumArgumentsConfigureIsolatedSOCKSProfile(t *testing.T) {
	args, err := Arguments(Browser{ID: Chrome, Name: "Chrome", Path: "/chrome", Family: FamilyChromium}, LaunchOptions{
		URL: "https://internal.example", ProfileDir: "/profiles/app", Private: true, SOCKSPort: 45678,
	})
	if err != nil {
		t.Fatalf("Arguments() error = %v", err)
	}
	joined := strings.Join(args, " ")
	for _, want := range []string{"--user-data-dir=/profiles/app", "--incognito", "socks5://127.0.0.1:45678", "MAP * ~NOTFOUND", "https://internal.example"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("Arguments() = %q, missing %q", joined, want)
		}
	}
}

func TestWriteFirefoxPreferencesSupportsDirectAndTunnelModes(t *testing.T) {
	dir := t.TempDir()
	if err := WriteFirefoxPreferences(dir, 40123); err != nil {
		t.Fatalf("WriteFirefoxPreferences(tunnel) error = %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "user.js"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, `network.proxy.socks_port", 40123`) || !strings.Contains(content, `network.proxy.socks_remote_dns", true`) {
		t.Fatalf("Tunnel preferences = %q", content)
	}
	if err := WriteFirefoxPreferences(dir, 0); err != nil {
		t.Fatalf("WriteFirefoxPreferences(direct) error = %v", err)
	}
	data, _ = os.ReadFile(filepath.Join(dir, "user.js"))
	if !strings.Contains(string(data), `network.proxy.type", 0`) || strings.Contains(string(data), "network.proxy.socks_port") {
		t.Fatalf("Direct preferences = %q", data)
	}
}

func TestCleanupProfileHonorsRetentionPolicy(t *testing.T) {
	dir := t.TempDir()
	for _, path := range []string{
		filepath.Join("Default", "Cache", "entry"),
		filepath.Join("Default", "Cookies"),
		filepath.Join("Default", "Local Storage", "entry"),
		filepath.Join("Default", "Sessions", "entry"),
	} {
		fullPath := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte("data"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	err := CleanupProfile(FamilyChromium, dir, RetentionPolicy{
		KeepCookies: true, KeepLocalStorage: false, RestoreSession: false, ClearCacheOnClose: true,
	})
	if err != nil {
		t.Fatalf("CleanupProfile() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "Default", "Cookies")); err != nil {
		t.Fatalf("Cookies were removed: %v", err)
	}
	for _, path := range []string{filepath.Join("Default", "Cache"), filepath.Join("Default", "Local Storage"), filepath.Join("Default", "Sessions")} {
		if _, err := os.Stat(filepath.Join(dir, path)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("%s still exists: %v", path, err)
		}
	}
}
