package connectionauth

import (
	"errors"
	"testing"

	"github.com/signoredellarete/bashes/internal/domain"
)

type memoryPasswordStore struct {
	values map[string]string
	err    error
}

func (store *memoryPasswordStore) Password(resourceID string) (string, bool, error) {
	if store.err != nil {
		return "", false, store.err
	}
	password, found := store.values[resourceID]
	return password, found, nil
}

func (store *memoryPasswordStore) SavePassword(resourceID string, password string) error {
	if store.err != nil {
		return store.err
	}
	store.values[resourceID] = password
	return nil
}

func (store *memoryPasswordStore) DeletePassword(resourceID string) error {
	if store.err != nil {
		return store.err
	}
	delete(store.values, resourceID)
	return nil
}

func TestPrepareLoadsSavedPasswordForPasswordPreference(t *testing.T) {
	passwords := &memoryPasswordStore{values: map[string]string{"host-1": "saved-secret"}}
	service := NewService(passwords)
	resource := domain.Endpoint{ID: "host-1", Auth: &domain.Auth{Method: domain.AuthMethodPassword}}

	input, err := service.Prepare(resource, Input{})
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if input.Password != "saved-secret" {
		t.Fatalf("Password = %q, want saved-secret", input.Password)
	}
}

func TestPreparePreservesExplicitAuthentication(t *testing.T) {
	passwords := &memoryPasswordStore{values: map[string]string{"host-1": "saved-secret"}}
	service := NewService(passwords)
	resource := domain.Endpoint{
		ID:   "host-1",
		Auth: &domain.Auth{Method: domain.AuthMethodKey, KeyName: "stored-key"},
	}

	input, err := service.Prepare(resource, Input{Password: "session-secret"})
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if input.Password != "session-secret" || input.KeyName != "" {
		t.Fatalf("Prepare() = %+v", input)
	}
}

func TestPersistPasswordChoiceOnlyMutatesManagedPasswords(t *testing.T) {
	passwords := &memoryPasswordStore{values: map[string]string{"host-1": "saved-secret"}}
	service := NewService(passwords)

	if err := service.PersistPasswordChoice("host-1", Input{Password: "ignored"}); err != nil {
		t.Fatalf("PersistPasswordChoice(unmanaged) error = %v", err)
	}
	if passwords.values["host-1"] != "saved-secret" {
		t.Fatalf("unmanaged password = %q", passwords.values["host-1"])
	}
	if err := service.PersistPasswordChoice("host-1", Input{
		Password: "replacement", ManagePassword: true, SavePassword: true,
	}); err != nil {
		t.Fatalf("PersistPasswordChoice(save) error = %v", err)
	}
	if passwords.values["host-1"] != "replacement" {
		t.Fatalf("saved password = %q", passwords.values["host-1"])
	}
	if err := service.PersistPasswordChoice("host-1", Input{ManagePassword: true}); err != nil {
		t.Fatalf("PersistPasswordChoice(delete) error = %v", err)
	}
	if _, found := passwords.values["host-1"]; found {
		t.Fatal("password was not deleted")
	}
}

func TestCredentialErrorsHaveOperationContext(t *testing.T) {
	service := NewService(&memoryPasswordStore{values: map[string]string{}, err: errors.New("keyring unavailable")})
	resource := domain.Endpoint{ID: "host-1", Auth: &domain.Auth{Method: domain.AuthMethodPassword}}

	if _, err := service.Prepare(resource, Input{}); err == nil || err.Error() != "read saved SSH password: keyring unavailable" {
		t.Fatalf("Prepare() error = %v", err)
	}
	if err := service.PersistPasswordChoice("host-1", Input{ManagePassword: true}); err == nil || err.Error() != "remove saved SSH password: keyring unavailable" {
		t.Fatalf("PersistPasswordChoice() error = %v", err)
	}
}

func TestPreferenceUsesPathKeyPasswordAndAgentPrecedence(t *testing.T) {
	tests := []struct {
		name  string
		input Input
		want  domain.Auth
	}{
		{name: "path", input: Input{PrivateKeyPath: " /keys/id ", KeyName: "managed", Password: "secret"}, want: domain.Auth{Method: domain.AuthMethodPath, PrivateKeyPath: "/keys/id"}},
		{name: "key", input: Input{KeyName: " managed key ", Password: "secret"}, want: domain.Auth{Method: domain.AuthMethodKey, KeyName: "managed-key"}},
		{name: "password", input: Input{Password: "secret"}, want: domain.Auth{Method: domain.AuthMethodPassword}},
		{name: "agent", input: Input{}, want: domain.Auth{Method: domain.AuthMethodAgent}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := Preference(test.input)
			if got.Method != test.want.Method || got.PrivateKeyPath != test.want.PrivateKeyPath || got.KeyName != test.want.KeyName {
				t.Fatalf("Preference() = %+v, want %+v", got, test.want)
			}
		})
	}
}
