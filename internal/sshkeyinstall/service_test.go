package sshkeyinstall

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/signoredellarete/bashes/internal/domain"
	"github.com/signoredellarete/bashes/internal/sshconnection"
	"golang.org/x/crypto/ssh"
)

type memoryResources struct {
	endpoint domain.Endpoint
	auth     domain.Auth
}

func (resources *memoryResources) Endpoint(id string) (domain.Endpoint, error) {
	if id != resources.endpoint.ID {
		return domain.Endpoint{}, errors.New("resource not found")
	}
	return resources.endpoint, nil
}

func (resources *memoryResources) SetResourceAuth(id string, auth domain.Auth) error {
	if id != resources.endpoint.ID {
		return errors.New("resource not found")
	}
	resources.auth = auth
	return nil
}

type memoryKeys struct {
	publicKey string
}

func (keys memoryKeys) ReadManagedPublic(string) (string, error) {
	return keys.publicKey, nil
}

func TestServiceInstallsManagedKeyAndPersistsPreference(t *testing.T) {
	resources := &memoryResources{endpoint: domain.Endpoint{
		ID: "host-1", Hostname: "server", IP: "192.0.2.10", Port: 22, User: "deploy",
	}}
	service := &Service{
		resources: resources,
		keys:      memoryKeys{publicKey: "ssh-ed25519 AAAAtest user@example"},
	}
	var dialInput sshconnection.Input
	service.dial = func(_ domain.Endpoint, input sshconnection.Input, timeout time.Duration) (*ssh.Client, error) {
		dialInput = input
		if timeout <= 0 {
			t.Fatal("dial timeout is not positive")
		}
		return nil, nil
	}
	var command string
	service.run = func(_ *ssh.Client, value string) ([]byte, error) {
		command = value
		return nil, nil
	}
	service.close = func(*ssh.Client) error { return nil }

	err := service.Install(Input{
		ResourceID:   "host-1",
		KeyName:      "managed-key",
		Password:     "secret",
		TrustHostKey: true,
	})
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if dialInput.Authentication.Password != "secret" || !dialInput.TrustHostKey {
		t.Fatalf("dial input = %+v", dialInput)
	}
	if !strings.Contains(command, "authorized_keys") || !strings.Contains(command, "ssh-ed25519 AAAAtest") {
		t.Fatalf("command = %q", command)
	}
	if resources.auth.Method != domain.AuthMethodKey || resources.auth.KeyName != "managed-key" {
		t.Fatalf("saved auth = %+v", resources.auth)
	}
}

func TestServiceReportsRemoteCommandOutput(t *testing.T) {
	resources := &memoryResources{endpoint: domain.Endpoint{ID: "host-1"}}
	service := &Service{
		resources: resources,
		keys:      memoryKeys{publicKey: "ssh-ed25519 AAAAtest"},
		dial: func(domain.Endpoint, sshconnection.Input, time.Duration) (*ssh.Client, error) {
			return nil, nil
		},
		run: func(*ssh.Client, string) ([]byte, error) {
			return []byte("permission denied"), errors.New("exit status 1")
		},
		close: func(*ssh.Client) error { return nil },
	}

	err := service.Install(Input{ResourceID: "host-1", KeyName: "managed-key"})
	if err == nil || !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("Install() error = %v", err)
	}
}

func TestServiceRequiresAKeySelection(t *testing.T) {
	service := &Service{resources: &memoryResources{}, keys: memoryKeys{}}
	if err := service.Install(Input{}); err == nil || !strings.Contains(err.Error(), "select or generate") {
		t.Fatalf("Install() error = %v", err)
	}
}

func TestCommandQuotesAuthorizedKey(t *testing.T) {
	key := "ssh-ed25519 AAAA'quoted comment"
	command := Command(key)
	if strings.Count(command, ShellQuote(key)) != 2 {
		t.Fatalf("Command() = %q", command)
	}
}
