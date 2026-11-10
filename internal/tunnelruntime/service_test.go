package tunnelruntime

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/signoredellarete/bashes/internal/connectionauth"
	"github.com/signoredellarete/bashes/internal/domain"
	"github.com/signoredellarete/bashes/internal/sshconnection"
	"github.com/signoredellarete/bashes/internal/sshtunnel"
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

type memoryPasswords struct{ values map[string]string }

func (passwords *memoryPasswords) Password(id string) (string, bool, error) {
	password, found := passwords.values[id]
	return password, found, nil
}
func (passwords *memoryPasswords) SavePassword(id string, password string) error {
	passwords.values[id] = password
	return nil
}
func (passwords *memoryPasswords) DeletePassword(id string) error {
	delete(passwords.values, id)
	return nil
}

type managedKeys struct{ path string }

func (keys managedKeys) ManagedPrivatePath(string) string { return keys.path }

type fakeClient struct {
	mu       sync.Mutex
	closes   int
	listener net.Listener
}

func (connection *fakeClient) Close() error {
	connection.mu.Lock()
	connection.closes++
	connection.mu.Unlock()
	return nil
}
func (connection *fakeClient) Listen(string, string) (net.Listener, error) {
	return connection.listener, nil
}

func TestServiceStartsAndStopsManagedKeyTunnel(t *testing.T) {
	resources := &memoryResources{endpoint: domain.Endpoint{
		ID: "host-1", IP: "192.0.2.10", Port: 22, User: "deploy",
		Auth: &domain.Auth{Method: domain.AuthMethodKey, KeyName: "managed"},
	}}
	manager := sshtunnel.NewManager()
	service := &Service{
		resources: resources,
		auth:      connectionauth.NewService(&memoryPasswords{values: map[string]string{}}),
		keys:      managedKeys{path: "/managed/id_ed25519"},
		manager:   manager,
		now:       func() time.Time { return time.Unix(0, 42) },
	}
	connection := &fakeClient{}
	service.dial = func(_ domain.Endpoint, input sshconnection.Input, _ time.Duration) (client, error) {
		if input.Authentication.PrivateKeyPath != "/managed/id_ed25519" {
			t.Fatalf("private key path = %q", input.Authentication.PrivateKeyPath)
		}
		return connection, nil
	}
	service.listen = func(string, string) (net.Listener, error) {
		return net.Listen("tcp", "127.0.0.1:0")
	}
	service.serve = func(ctx context.Context, _ string, _ net.Listener, _ client, _ string) error {
		<-ctx.Done()
		return nil
	}
	service.keepAlive = func(ctx context.Context, _ client, _ func(error)) { <-ctx.Done() }

	info, err := service.Start(Input{ResourceID: "host-1", Type: "socks", LocalPort: 1080})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if info.TunnelID != "tunnel-42" || info.ForwardTarget != "dynamic" {
		t.Fatalf("tunnel info = %+v", info)
	}
	if resources.auth.Method != domain.AuthMethodKey || resources.auth.KeyName != "managed" {
		t.Fatalf("saved auth = %+v", resources.auth)
	}
	service.Stop(info.TunnelID)
	if len(service.Infos()) != 0 {
		t.Fatalf("Infos() = %+v", service.Infos())
	}
}

func TestServiceEmitsKeepaliveFailure(t *testing.T) {
	resources := &memoryResources{endpoint: domain.Endpoint{ID: "host-1", IP: "192.0.2.10", Port: 22, User: "deploy"}}
	events := make(chan Event, 1)
	service := &Service{
		resources: resources,
		auth:      connectionauth.NewService(&memoryPasswords{values: map[string]string{}}),
		keys:      managedKeys{},
		manager:   sshtunnel.NewManager(),
		now:       time.Now,
		emit:      func(event Event) { events <- event },
	}
	service.dial = func(domain.Endpoint, sshconnection.Input, time.Duration) (client, error) {
		return &fakeClient{}, nil
	}
	service.listen = func(string, string) (net.Listener, error) {
		return net.Listen("tcp", "127.0.0.1:0")
	}
	service.serve = func(ctx context.Context, _ string, _ net.Listener, _ client, _ string) error {
		<-ctx.Done()
		return nil
	}
	service.keepAlive = func(_ context.Context, _ client, failed func(error)) {
		failed(errors.New("connection lost"))
	}

	if _, err := service.Start(Input{ResourceID: "host-1", Type: "socks", LocalPort: 1080}); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	select {
	case event := <-events:
		if !event.Closed || event.Message != "Tunnel closed: SSH keepalive failed: connection lost" {
			t.Fatalf("event = %+v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for tunnel failure event")
	}
}

func TestNormalizeInputRejectsPublicBindWithoutConfirmation(t *testing.T) {
	input := Input{Type: "socks", LocalHost: "0.0.0.0", LocalPort: 1080}
	if err := NormalizeInput(&input); err == nil {
		t.Fatal("NormalizeInput() error = nil")
	}
	input.AllowPublicBind = true
	if err := NormalizeInput(&input); err != nil {
		t.Fatalf("NormalizeInput(confirmed) error = %v", err)
	}
}
