package main

import (
	"path/filepath"
	"testing"

	"github.com/signoredellarete/bashes/internal/domain"
)

func TestApplyAuthPreferenceUsesStoredKey(t *testing.T) {
	resource := domain.Endpoint{
		ID:       "host-1",
		Hostname: "server-01",
		IP:       "10.0.0.10",
		Port:     22,
		User:     "root",
		Auth: &domain.Auth{
			Method:       domain.AuthMethodKey,
			KeyName:      "bashes-main",
			TrustHostKey: true,
		},
	}

	input := applyAuthPreference(resource, SSHSessionInput{ResourceID: resource.ID})
	if input.KeyName != "bashes-main" {
		t.Fatalf("KeyName = %q, want stored key", input.KeyName)
	}
	if !input.TrustHostKey {
		t.Fatal("TrustHostKey = false, want stored trust preference")
	}
}

func TestApplyAuthPreferenceDoesNotOverrideExplicitAuth(t *testing.T) {
	resource := domain.Endpoint{
		ID:       "host-1",
		Hostname: "server-01",
		IP:       "10.0.0.10",
		Port:     22,
		User:     "root",
		Auth: &domain.Auth{
			Method:  domain.AuthMethodKey,
			KeyName: "bashes-main",
		},
	}

	input := applyAuthPreference(resource, SSHSessionInput{
		ResourceID: resource.ID,
		Password:   "secret",
	})
	if input.KeyName != "" {
		t.Fatalf("KeyName = %q, want explicit password to keep key empty", input.KeyName)
	}
}

func TestAuthPreferenceFromSessionInput(t *testing.T) {
	auth := authPreferenceFromSessionInput(SSHSessionInput{
		KeyName:      "bashes-main",
		TrustHostKey: true,
	})
	if auth == nil || auth.Method != domain.AuthMethodKey || auth.KeyName != "bashes-main" || !auth.TrustHostKey {
		t.Fatalf("Auth preference from key input = %+v", auth)
	}

	auth = authPreferenceFromSessionInput(SSHSessionInput{Password: "secret"})
	if auth == nil || auth.Method != domain.AuthMethodPassword {
		t.Fatalf("Auth preference from password input = %+v", auth)
	}
}

func TestResolveSessionKeyPathUsesAppDataDirectory(t *testing.T) {
	app := NewApp(filepath.Join(t.TempDir(), "portable", "hosts.json"))

	input := app.resolveSessionKeyPath(SSHSessionInput{KeyName: "bashes-main"})
	want := filepath.Join(filepath.Dir(app.dataPath), "keys", "bashes-main")
	if input.PrivateKeyPath != want {
		t.Fatalf("PrivateKeyPath = %q, want %q", input.PrivateKeyPath, want)
	}
	if input.KeyName != "bashes-main" {
		t.Fatalf("KeyName = %q, want original key name preserved", input.KeyName)
	}
}

func TestDataDirForOSUsesPlatformConventions(t *testing.T) {
	env := func(values map[string]string) func(string) string {
		return func(name string) string {
			return values[name]
		}
	}

	tests := []struct {
		name string
		goos string
		home string
		env  map[string]string
		want string
	}{
		{
			name: "macos application support",
			goos: "darwin",
			home: "/Users/alice",
			env:  map[string]string{},
			want: filepath.Join("/Users/alice", "Library", "Application Support", "Bashes"),
		},
		{
			name: "windows appdata",
			goos: "windows",
			home: `C:\Users\Alice`,
			env:  map[string]string{"APPDATA": `C:\Users\Alice\AppData\Roaming`},
			want: filepath.Join(`C:\Users\Alice\AppData\Roaming`, "Bashes"),
		},
		{
			name: "linux xdg data",
			goos: "linux",
			home: "/home/alice",
			env:  map[string]string{"XDG_DATA_HOME": "/home/alice/.local/state"},
			want: filepath.Join("/home/alice/.local/state", "bashes"),
		},
		{
			name: "linux home fallback",
			goos: "linux",
			home: "/home/alice",
			env:  map[string]string{},
			want: filepath.Join("/home/alice", ".local", "share", "bashes"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dataDirForOS(tt.goos, tt.home, env(tt.env))
			if got != tt.want {
				t.Fatalf("dataDirForOS() = %q, want %q", got, tt.want)
			}
		})
	}
}
