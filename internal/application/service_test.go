package application

import (
	"strings"
	"testing"

	"github.com/signoredellarete/bashes/internal/domain"
)

func TestServiceAddsHostAndSubsystem(t *testing.T) {
	store := newMemoryStore()
	service := NewService(store)

	host, err := service.AddHost(EndpointInput{
		Hostname: " server-01 ",
		IP:       " 10.0.0.10 ",
		Port:     22,
		User:     " root ",
	})
	if err != nil {
		t.Fatalf("AddHost() error = %v", err)
	}
	if host.ID == "" {
		t.Fatal("AddHost() returned empty ID")
	}
	if host.Hostname != "server-01" {
		t.Fatalf("Host hostname = %q, want trimmed hostname", host.Hostname)
	}

	subsystem, err := service.AddSubsystem(host.ID, EndpointInput{
		Type:     domain.ResourceLXC,
		Hostname: "lxc-web",
		IP:       "10.0.0.20",
		Port:     2222,
		User:     "deploy",
	})
	if err != nil {
		t.Fatalf("AddSubsystem() error = %v", err)
	}
	if subsystem.Type != domain.ResourceLXC {
		t.Fatalf("Subsystem type = %q, want %q", subsystem.Type, domain.ResourceLXC)
	}

	hosts, err := service.ListHosts()
	if err != nil {
		t.Fatalf("ListHosts() error = %v", err)
	}
	if len(hosts) != 1 {
		t.Fatalf("Hosts length = %d, want 1", len(hosts))
	}
	if len(hosts[0].Subsystems) != 1 {
		t.Fatalf("Subsystems length = %d, want 1", len(hosts[0].Subsystems))
	}
}

func TestServiceRejectsInvalidSubsystemType(t *testing.T) {
	store := newMemoryStore()
	service := NewService(store)

	host, err := service.AddHost(EndpointInput{
		Hostname: "server-01",
		IP:       "10.0.0.10",
		Port:     22,
		User:     "root",
	})
	if err != nil {
		t.Fatalf("AddHost() error = %v", err)
	}

	_, err = service.AddSubsystem(host.ID, EndpointInput{
		Type:     domain.ResourceHost,
		Hostname: "bad",
		IP:       "10.0.0.20",
		Port:     22,
		User:     "root",
	})
	if err == nil {
		t.Fatal("AddSubsystem() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "invalid subsystem type") {
		t.Fatalf("AddSubsystem() error = %v, want invalid subsystem type", err)
	}
}

func TestServiceDeletesHostAndSubsystemByID(t *testing.T) {
	store := newMemoryStore()
	service := NewService(store)

	host, err := service.AddHost(EndpointInput{
		Hostname: "server-01",
		IP:       "10.0.0.10",
		Port:     22,
		User:     "root",
	})
	if err != nil {
		t.Fatalf("AddHost() error = %v", err)
	}
	subsystem, err := service.AddSubsystem(host.ID, EndpointInput{
		Type:     domain.ResourceVM,
		Hostname: "vm-app",
		IP:       "10.0.0.30",
		Port:     22,
		User:     "ubuntu",
	})
	if err != nil {
		t.Fatalf("AddSubsystem() error = %v", err)
	}

	if err := service.DeleteResource(subsystem.ID); err != nil {
		t.Fatalf("DeleteResource(subsystem) error = %v", err)
	}
	hosts, err := service.ListHosts()
	if err != nil {
		t.Fatalf("ListHosts() error = %v", err)
	}
	if len(hosts[0].Subsystems) != 0 {
		t.Fatalf("Subsystems length = %d, want 0", len(hosts[0].Subsystems))
	}

	if err := service.DeleteResource(host.ID); err != nil {
		t.Fatalf("DeleteResource(host) error = %v", err)
	}
	hosts, err = service.ListHosts()
	if err != nil {
		t.Fatalf("ListHosts() error = %v", err)
	}
	if len(hosts) != 0 {
		t.Fatalf("Hosts length = %d, want 0", len(hosts))
	}
}

func TestServiceUpdatesHostAndSubsystemByID(t *testing.T) {
	store := newMemoryStore()
	service := NewService(store)

	host, err := service.AddHost(EndpointInput{
		Hostname: "server-01",
		IP:       "10.0.0.10",
		Port:     22,
		User:     "root",
	})
	if err != nil {
		t.Fatalf("AddHost() error = %v", err)
	}
	subsystem, err := service.AddSubsystem(host.ID, EndpointInput{
		Type:     domain.ResourceVM,
		Hostname: "vm-app",
		IP:       "10.0.0.30",
		Port:     22,
		User:     "ubuntu",
	})
	if err != nil {
		t.Fatalf("AddSubsystem() error = %v", err)
	}

	if err := service.UpdateResource(host.ID, EndpointInput{
		Hostname: "server-renamed",
		IP:       "10.0.0.11",
		Port:     2200,
		User:     "admin",
	}); err != nil {
		t.Fatalf("UpdateResource(host) error = %v", err)
	}
	if err := service.UpdateResource(subsystem.ID, EndpointInput{
		Type:     domain.ResourceLXC,
		Hostname: "lxc-app",
		IP:       "10.0.0.31",
		Port:     2222,
		User:     "deploy",
	}); err != nil {
		t.Fatalf("UpdateResource(subsystem) error = %v", err)
	}

	hosts, err := service.ListHosts()
	if err != nil {
		t.Fatalf("ListHosts() error = %v", err)
	}
	if hosts[0].ID != host.ID {
		t.Fatalf("Host ID changed = %q, want %q", hosts[0].ID, host.ID)
	}
	if hosts[0].Hostname != "server-renamed" || hosts[0].Port != 2200 {
		t.Fatalf("Host was not updated: %+v", hosts[0])
	}
	if hosts[0].Subsystems[0].ID != subsystem.ID {
		t.Fatalf("Subsystem ID changed = %q, want %q", hosts[0].Subsystems[0].ID, subsystem.ID)
	}
	if hosts[0].Subsystems[0].Type != domain.ResourceLXC || hosts[0].Subsystems[0].User != "deploy" {
		t.Fatalf("Subsystem was not updated: %+v", hosts[0].Subsystems[0])
	}
}

func TestServicePropagatesValidationErrors(t *testing.T) {
	store := newMemoryStore()
	service := NewService(store)

	_, err := service.AddHost(EndpointInput{
		Hostname: "bad",
		IP:       "127.0.0.1",
		Port:     70000,
		User:     "root",
	})
	if err == nil {
		t.Fatal("AddHost() error = nil, want validation error")
	}
	if !strings.Contains(err.Error(), "port") {
		t.Fatalf("AddHost() error = %v, want port validation error", err)
	}
}

type memoryStore struct {
	data domain.Store
}

func newMemoryStore() *memoryStore {
	return &memoryStore{data: domain.NewStore()}
}

func (s *memoryStore) Load() (domain.Store, error) {
	return s.data, nil
}

func (s *memoryStore) Save(data domain.Store) error {
	data = domain.NormalizeStore(data)
	if err := data.Validate(); err != nil {
		return err
	}
	s.data = data
	return nil
}
