package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/signoredellarete/bashes/internal/domain"
)

type memoryPasswordStore struct {
	values map[string]string
}

func newMemoryPasswordStore() *memoryPasswordStore {
	return &memoryPasswordStore{values: map[string]string{}}
}

func (s *memoryPasswordStore) Password(resourceID string) (string, bool, error) {
	password, found := s.values[resourceID]
	return password, found, nil
}

func (s *memoryPasswordStore) SavePassword(resourceID string, password string) error {
	s.values[resourceID] = password
	return nil
}

func (s *memoryPasswordStore) DeletePassword(resourceID string) error {
	delete(s.values, resourceID)
	return nil
}

func TestSavedPasswordLifecycleIsSeparateFromDatabaseExport(t *testing.T) {
	dir := t.TempDir()
	passwords := newMemoryPasswordStore()
	app := newAppWithPasswordStore(filepath.Join(dir, "hosts.json"), passwords)
	host, err := app.AddHost(applicationEndpoint("saved-password-host", "10.0.0.15"))
	if err != nil {
		t.Fatalf("AddHost() error = %v", err)
	}
	if err := app.service.SetResourceAuth(host.ID, domain.Auth{Method: domain.AuthMethodPassword}); err != nil {
		t.Fatalf("SetResourceAuth() error = %v", err)
	}

	const secret = "not-in-the-portable-json"
	if err := passwords.SavePassword(host.ID, secret); err != nil {
		t.Fatalf("SavePassword() error = %v", err)
	}
	resource, err := app.resourceByID(host.ID)
	if err != nil {
		t.Fatalf("resourceByID() error = %v", err)
	}
	input, err := app.prepareSessionInput(resource, SSHSessionInput{ResourceID: host.ID})
	if err != nil {
		t.Fatalf("prepareSessionInput() error = %v", err)
	}
	if input.Password != secret {
		t.Fatalf("prepared password = %q, want saved password", input.Password)
	}

	exportPath := filepath.Join(dir, "export.json")
	if err := app.ExportDatabase(exportPath); err != nil {
		t.Fatalf("ExportDatabase() error = %v", err)
	}
	exported, err := os.ReadFile(exportPath)
	if err != nil {
		t.Fatalf("ReadFile(export) error = %v", err)
	}
	if strings.Contains(string(exported), secret) || strings.Contains(strings.ToLower(string(exported)), "savepassword") {
		t.Fatalf("database export contains password storage data: %s", exported)
	}
	if !strings.Contains(string(exported), `"method": "password"`) {
		t.Fatalf("database export lost the non-secret auth preference: %s", exported)
	}
}

func TestPersistPasswordChoiceSavesAndRemovesCredential(t *testing.T) {
	passwords := newMemoryPasswordStore()
	app := newAppWithPasswordStore(filepath.Join(t.TempDir(), "hosts.json"), passwords)

	if err := app.persistPasswordChoice("host-1", SSHSessionInput{
		Password:       "secret",
		ManagePassword: true,
		SavePassword:   true,
	}); err != nil {
		t.Fatalf("persistPasswordChoice(save) error = %v", err)
	}
	if password := passwords.values["host-1"]; password != "secret" {
		t.Fatalf("saved password = %q", password)
	}
	if err := app.persistPasswordChoice("host-1", SSHSessionInput{ManagePassword: true}); err != nil {
		t.Fatalf("persistPasswordChoice(delete) error = %v", err)
	}
	if _, found := passwords.values["host-1"]; found {
		t.Fatal("saved password was not removed")
	}
}

func TestPersistPasswordChoiceDoesNotTouchUnmanagedCredential(t *testing.T) {
	passwords := newMemoryPasswordStore()
	passwords.values["host-1"] = "saved-secret"
	app := newAppWithPasswordStore(filepath.Join(t.TempDir(), "hosts.json"), passwords)

	if err := app.persistPasswordChoice("host-1", SSHSessionInput{Password: "session-secret"}); err != nil {
		t.Fatalf("persistPasswordChoice() error = %v", err)
	}
	if password := passwords.values["host-1"]; password != "saved-secret" {
		t.Fatalf("unmanaged password changed to %q", password)
	}
}

func TestExplicitPasswordTakesPrecedenceOverSavedPassword(t *testing.T) {
	passwords := newMemoryPasswordStore()
	app := newAppWithPasswordStore(filepath.Join(t.TempDir(), "hosts.json"), passwords)
	host, err := app.AddHost(applicationEndpoint("explicit-password-host", "10.0.0.16"))
	if err != nil {
		t.Fatalf("AddHost() error = %v", err)
	}
	if err := app.service.SetResourceAuth(host.ID, domain.Auth{Method: domain.AuthMethodPassword}); err != nil {
		t.Fatalf("SetResourceAuth() error = %v", err)
	}
	if err := passwords.SavePassword(host.ID, "saved-secret"); err != nil {
		t.Fatalf("SavePassword() error = %v", err)
	}

	resource, err := app.resourceByID(host.ID)
	if err != nil {
		t.Fatalf("resourceByID() error = %v", err)
	}
	input, err := app.prepareSessionInput(resource, SSHSessionInput{
		ResourceID: host.ID,
		Password:   "explicit-secret",
	})
	if err != nil {
		t.Fatalf("prepareSessionInput() error = %v", err)
	}
	if input.Password != "explicit-secret" {
		t.Fatalf("prepared password = %q, want explicit password", input.Password)
	}

	found, err := app.HasSavedPassword(host.ID)
	if err != nil {
		t.Fatalf("HasSavedPassword() error = %v", err)
	}
	if !found {
		t.Fatal("HasSavedPassword() = false, want true")
	}
}
