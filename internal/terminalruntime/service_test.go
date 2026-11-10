package terminalruntime

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/signoredellarete/bashes/internal/connectionauth"
	"github.com/signoredellarete/bashes/internal/domain"
	"github.com/signoredellarete/bashes/internal/localterm"
	"github.com/signoredellarete/bashes/internal/remotessh"
	"github.com/signoredellarete/bashes/internal/sshconnection"
	"github.com/signoredellarete/bashes/internal/termsession"
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

type memoryPasswords struct {
	values map[string]string
}

func (passwords *memoryPasswords) Password(resourceID string) (string, bool, error) {
	password, found := passwords.values[resourceID]
	return password, found, nil
}

func (passwords *memoryPasswords) SavePassword(resourceID string, password string) error {
	passwords.values[resourceID] = password
	return nil
}

func (passwords *memoryPasswords) DeletePassword(resourceID string) error {
	delete(passwords.values, resourceID)
	return nil
}

type managedKeys struct{ path string }

func (keys managedKeys) ManagedPrivatePath(string) string { return keys.path }

type fakeClient struct {
	mu     sync.Mutex
	closes int
}

func (client *fakeClient) Close() error {
	client.mu.Lock()
	client.closes++
	client.mu.Unlock()
	return nil
}

type fakeShell struct {
	done      chan error
	closeOnce sync.Once
}

func newFakeShell() *fakeShell { return &fakeShell{done: make(chan error, 1)} }

func (shell *fakeShell) Resize(remotessh.TerminalSize) error { return nil }
func (shell *fakeShell) Wait() error                         { return <-shell.done }
func (shell *fakeShell) Close() error {
	shell.closeOnce.Do(func() { shell.done <- nil })
	return nil
}

func TestSSHSessionLifecycleUsesServicesAndEmitsTypedEvents(t *testing.T) {
	resources := &memoryResources{endpoint: domain.Endpoint{
		ID: "host-1", Hostname: "server", IP: "192.0.2.8", Port: 22, User: "deploy",
		Auth: &domain.Auth{Method: domain.AuthMethodKey, KeyName: "managed"},
	}}
	passwords := &memoryPasswords{values: map[string]string{}}
	manager := termsession.NewManager()
	var eventsMu sync.Mutex
	events := make([]Event, 0)
	service := &Service{
		resources: resources,
		auth:      connectionauth.NewService(passwords),
		keys:      managedKeys{path: "/managed/id_ed25519"},
		manager:   manager,
		goos:      "linux",
		now:       func() time.Time { return time.Unix(0, 42) },
		emit: func(event Event) {
			eventsMu.Lock()
			events = append(events, event)
			eventsMu.Unlock()
		},
	}
	client := &fakeClient{}
	service.dial = func(_ domain.Endpoint, input sshconnection.Input, _ time.Duration) (remoteClient, error) {
		if input.Authentication.PrivateKeyPath != "/managed/id_ed25519" {
			t.Fatalf("private key path = %q", input.Authentication.PrivateKeyPath)
		}
		return client, nil
	}
	shell := newFakeShell()
	var stdout io.Writer
	service.startRemote = func(_ remoteClient, options remotessh.ShellOptions) (termsession.Shell, error) {
		stdout = options.Stdout
		return shell, nil
	}
	service.keepAlive = func(ctx context.Context, _ remoteClient, _ func(error)) { <-ctx.Done() }

	sessionID, err := service.StartSSH(SSHInput{ResourceID: "host-1", Cols: 120, Rows: 40})
	if err != nil {
		t.Fatalf("StartSSH() error = %v", err)
	}
	if sessionID != "ssh-42" {
		t.Fatalf("session id = %q", sessionID)
	}
	if _, err := stdout.Write([]byte("hello")); err != nil {
		t.Fatalf("stdout.Write() error = %v", err)
	}
	if !service.Stop(sessionID) {
		t.Fatal("Stop() = false")
	}
	if service.Stop(sessionID) {
		t.Fatal("second Stop() = true")
	}

	eventsMu.Lock()
	defer eventsMu.Unlock()
	if len(events) != 3 || events[0].Kind != EventStatus || events[1].Kind != EventOutput || events[2].Kind != EventClosed {
		t.Fatalf("events = %+v", events)
	}
	if string(events[1].Data) != "hello" {
		t.Fatalf("output = %q", events[1].Data)
	}
	if resources.auth.Method != domain.AuthMethodKey || resources.auth.KeyName != "managed" {
		t.Fatalf("saved auth = %+v", resources.auth)
	}
}

func TestLocalSessionSupportAndNaturalClose(t *testing.T) {
	manager := termsession.NewManager()
	closed := make(chan Event, 1)
	service := &Service{
		manager: manager,
		goos:    "linux",
		now:     func() time.Time { return time.Unix(0, 84) },
		emit: func(event Event) {
			if event.Kind == EventClosed {
				closed <- event
			}
		},
	}
	shell := newFakeShell()
	service.startLocal = func(localterm.ShellOptions) (termsession.Shell, error) { return shell, nil }

	sessionID, err := service.StartLocal(LocalInput{Cols: 80, Rows: 24})
	if err != nil {
		t.Fatalf("StartLocal() error = %v", err)
	}
	if sessionID != "local-84" {
		t.Fatalf("session id = %q", sessionID)
	}
	shell.done <- io.EOF
	select {
	case event := <-closed:
		if event.Message != "Local shell closed" {
			t.Fatalf("closed event = %+v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for close event")
	}
}

func TestLocalSessionIsUnavailableOnWindows(t *testing.T) {
	service := &Service{goos: "windows"}
	if service.SupportsLocalShell() {
		t.Fatal("SupportsLocalShell() = true")
	}
	if _, err := service.StartLocal(LocalInput{}); err == nil {
		t.Fatal("StartLocal() error = nil")
	}
}
