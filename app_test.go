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
