package credentials

import (
	"testing"

	"github.com/zalando/go-keyring"
)

type memoryBackend struct {
	values map[string]string
}

func (m *memoryBackend) Set(service, user, password string) error {
	m.values[service+"\x00"+user] = password
	return nil
}

func (m *memoryBackend) Get(service, user string) (string, error) {
	password, ok := m.values[service+"\x00"+user]
	if !ok {
		return "", keyring.ErrNotFound
	}
	return password, nil
}

func (m *memoryBackend) Delete(service, user string) error {
	key := service + "\x00" + user
	if _, ok := m.values[key]; !ok {
		return keyring.ErrNotFound
	}
	delete(m.values, key)
	return nil
}

func TestKeyringStoreLifecycle(t *testing.T) {
	store := &keyringStore{backend: &memoryBackend{values: map[string]string{}}}

	if _, found, err := store.Password("host-1"); err != nil || found {
		t.Fatalf("Password() before save = found %v, err %v", found, err)
	}
	if err := store.SavePassword("host-1", "correct horse battery staple"); err != nil {
		t.Fatalf("SavePassword() error = %v", err)
	}
	password, found, err := store.Password("host-1")
	if err != nil || !found || password != "correct horse battery staple" {
		t.Fatalf("Password() = %q, %v, %v", password, found, err)
	}
	if err := store.DeletePassword("host-1"); err != nil {
		t.Fatalf("DeletePassword() error = %v", err)
	}
	if err := store.DeletePassword("host-1"); err != nil {
		t.Fatalf("DeletePassword() missing error = %v", err)
	}
}

func TestKeyringStoreRejectsInvalidInput(t *testing.T) {
	store := &keyringStore{backend: &memoryBackend{values: map[string]string{}}}
	if err := store.SavePassword("", "secret"); err == nil {
		t.Fatal("SavePassword() accepted an empty resource id")
	}
	if err := store.SavePassword("host-1", ""); err == nil {
		t.Fatal("SavePassword() accepted an empty password")
	}
	if _, _, err := store.Password(" "); err == nil {
		t.Fatal("Password() accepted an empty resource id")
	}
}
