package credentials

import (
	"errors"
	"fmt"
	"strings"

	"github.com/zalando/go-keyring"
)

const serviceName = "Bashes SSH Passwords"

type Store interface {
	Password(resourceID string) (string, bool, error)
	SavePassword(resourceID string, password string) error
	DeletePassword(resourceID string) error
}

type backend interface {
	Set(service, user, password string) error
	Get(service, user string) (string, error)
	Delete(service, user string) error
}

type nativeBackend struct{}

func (nativeBackend) Set(service, user, password string) error {
	return keyring.Set(service, user, password)
}

func (nativeBackend) Get(service, user string) (string, error) {
	return keyring.Get(service, user)
}

func (nativeBackend) Delete(service, user string) error {
	return keyring.Delete(service, user)
}

type keyringStore struct {
	backend backend
}

func NewNativeStore() Store {
	return &keyringStore{backend: nativeBackend{}}
}

func (s *keyringStore) Password(resourceID string) (string, bool, error) {
	resourceID, err := validResourceID(resourceID)
	if err != nil {
		return "", false, err
	}
	password, err := s.backend.Get(serviceName, resourceID)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("read system keyring: %w", err)
	}
	return password, true, nil
}

func (s *keyringStore) SavePassword(resourceID string, password string) error {
	resourceID, err := validResourceID(resourceID)
	if err != nil {
		return err
	}
	if password == "" {
		return errors.New("password is required")
	}
	if err := s.backend.Set(serviceName, resourceID, password); err != nil {
		return fmt.Errorf("write system keyring: %w", err)
	}
	return nil
}

func (s *keyringStore) DeletePassword(resourceID string) error {
	resourceID, err := validResourceID(resourceID)
	if err != nil {
		return err
	}
	if err := s.backend.Delete(serviceName, resourceID); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return fmt.Errorf("delete from system keyring: %w", err)
	}
	return nil
}

func validResourceID(resourceID string) (string, error) {
	resourceID = strings.TrimSpace(resourceID)
	if resourceID == "" {
		return "", errors.New("resource id is required")
	}
	return resourceID, nil
}
