package application

import (
	"strings"
	"sync"
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

func TestServiceAddsNestedSubsystem(t *testing.T) {
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
	vm, err := service.AddSubsystem(host.ID, EndpointInput{
		Type:     domain.ResourceVM,
		Hostname: "vm-app",
		IP:       "10.0.0.30",
		Port:     22,
		User:     "ubuntu",
	})
	if err != nil {
		t.Fatalf("AddSubsystem(vm) error = %v", err)
	}
	lxc, err := service.AddSubsystem(vm.ID, EndpointInput{
		Type:     domain.ResourceLXC,
		Hostname: "lxc-app",
		IP:       "10.0.0.31",
		Port:     22,
		User:     "deploy",
	})
	if err != nil {
		t.Fatalf("AddSubsystem(lxc) error = %v", err)
	}

	hosts, err := service.ListHosts()
	if err != nil {
		t.Fatalf("ListHosts() error = %v", err)
	}
	if len(hosts[0].Subsystems) != 1 {
		t.Fatalf("Host subsystem length = %d, want 1", len(hosts[0].Subsystems))
	}
	if len(hosts[0].Subsystems[0].Subsystems) != 1 {
		t.Fatalf("Nested subsystem length = %d, want 1", len(hosts[0].Subsystems[0].Subsystems))
	}
	if hosts[0].Subsystems[0].Subsystems[0].ID != lxc.ID {
		t.Fatalf("Nested subsystem ID = %q, want %q", hosts[0].Subsystems[0].Subsystems[0].ID, lxc.ID)
	}
}

func TestServiceSerializesConcurrentHostAdds(t *testing.T) {
	store := newMemoryStore()
	service := NewService(store)
	const count = 50

	var wg sync.WaitGroup
	start := make(chan struct{})
	errs := make(chan error, count)

	for i := 0; i < count; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := service.AddHost(EndpointInput{
				Hostname: "host",
				IP:       "10.0.0.1",
				Port:     2200 + i,
				User:     "root",
			})
			errs <- err
		}()
	}

	close(start)
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("AddHost() concurrent error = %v", err)
		}
	}
	hosts, err := service.ListHosts()
	if err != nil {
		t.Fatalf("ListHosts() error = %v", err)
	}
	if len(hosts) != count {
		t.Fatalf("Hosts length = %d, want %d", len(hosts), count)
	}
	seen := make(map[string]struct{}, count)
	for _, host := range hosts {
		if _, exists := seen[host.ID]; exists {
			t.Fatalf("duplicate host ID %q", host.ID)
		}
		seen[host.ID] = struct{}{}
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

func TestServiceUpdatesDeletesAndSetsAuthOnNestedSubsystem(t *testing.T) {
	store := newMemoryStore()
	service := NewService(store)

	host, err := service.AddHost(EndpointInput{Hostname: "host", IP: "10.0.0.1", Port: 22, User: "root"})
	if err != nil {
		t.Fatalf("AddHost() error = %v", err)
	}
	vm, err := service.AddSubsystem(host.ID, EndpointInput{Type: domain.ResourceVM, Hostname: "vm", IP: "10.0.0.2", Port: 22, User: "ubuntu"})
	if err != nil {
		t.Fatalf("AddSubsystem(vm) error = %v", err)
	}
	lxc, err := service.AddSubsystem(vm.ID, EndpointInput{Type: domain.ResourceLXC, Hostname: "lxc", IP: "10.0.0.3", Port: 22, User: "deploy"})
	if err != nil {
		t.Fatalf("AddSubsystem(lxc) error = %v", err)
	}

	if err := service.UpdateResource(lxc.ID, EndpointInput{Type: domain.ResourceDocker, Hostname: "docker", IP: "10.0.0.4", Port: 2222, User: "app"}); err != nil {
		t.Fatalf("UpdateResource(nested) error = %v", err)
	}
	if err := service.SetResourceAuth(lxc.ID, domain.Auth{Method: domain.AuthMethodPath, PrivateKeyPath: " ~/.ssh/nested "}); err != nil {
		t.Fatalf("SetResourceAuth(nested) error = %v", err)
	}

	hosts, err := service.ListHosts()
	if err != nil {
		t.Fatalf("ListHosts() error = %v", err)
	}
	nested := hosts[0].Subsystems[0].Subsystems[0]
	if nested.Type != domain.ResourceDocker || nested.Hostname != "docker" || nested.Port != 2222 {
		t.Fatalf("Nested subsystem was not updated: %+v", nested)
	}
	if nested.Auth == nil || nested.Auth.PrivateKeyPath != "~/.ssh/nested" {
		t.Fatalf("Nested auth was not saved: %+v", nested.Auth)
	}

	if err := service.DeleteResource(vm.ID); err != nil {
		t.Fatalf("DeleteResource(parent subsystem) error = %v", err)
	}
	hosts, err = service.ListHosts()
	if err != nil {
		t.Fatalf("ListHosts() error = %v", err)
	}
	if len(hosts[0].Subsystems) != 0 {
		t.Fatalf("Host subsystem length after deleting parent = %d, want 0", len(hosts[0].Subsystems))
	}
}

func TestServiceReordersHostBlocks(t *testing.T) {
	store := newMemoryStore()
	service := NewService(store)

	hostA, err := service.AddHost(EndpointInput{Hostname: "host-a", IP: "10.0.0.1", Port: 22, User: "root"})
	if err != nil {
		t.Fatalf("AddHost(host-a) error = %v", err)
	}
	subA, err := service.AddSubsystem(hostA.ID, EndpointInput{Type: domain.ResourceVM, Hostname: "vm-a", IP: "10.0.0.11", Port: 22, User: "ubuntu"})
	if err != nil {
		t.Fatalf("AddSubsystem(host-a) error = %v", err)
	}
	hostB, err := service.AddHost(EndpointInput{Hostname: "host-b", IP: "10.0.0.2", Port: 22, User: "root"})
	if err != nil {
		t.Fatalf("AddHost(host-b) error = %v", err)
	}
	hostC, err := service.AddHost(EndpointInput{Hostname: "host-c", IP: "10.0.0.3", Port: 22, User: "root"})
	if err != nil {
		t.Fatalf("AddHost(host-c) error = %v", err)
	}

	if err := service.ReorderHosts([]string{hostC.ID, hostA.ID, hostB.ID}); err != nil {
		t.Fatalf("ReorderHosts() error = %v", err)
	}

	hosts, err := service.ListHosts()
	if err != nil {
		t.Fatalf("ListHosts() error = %v", err)
	}
	if got := []string{hosts[0].ID, hosts[1].ID, hosts[2].ID}; got[0] != hostC.ID || got[1] != hostA.ID || got[2] != hostB.ID {
		t.Fatalf("Host order = %v, want [%s %s %s]", got, hostC.ID, hostA.ID, hostB.ID)
	}
	if len(hosts[1].Subsystems) != 1 || hosts[1].Subsystems[0].ID != subA.ID {
		t.Fatalf("Subsystem block was not preserved: %+v", hosts[1].Subsystems)
	}
}

func TestServiceRejectsInvalidHostOrder(t *testing.T) {
	store := newMemoryStore()
	service := NewService(store)

	hostA, err := service.AddHost(EndpointInput{Hostname: "host-a", IP: "10.0.0.1", Port: 22, User: "root"})
	if err != nil {
		t.Fatalf("AddHost(host-a) error = %v", err)
	}
	hostB, err := service.AddHost(EndpointInput{Hostname: "host-b", IP: "10.0.0.2", Port: 22, User: "root"})
	if err != nil {
		t.Fatalf("AddHost(host-b) error = %v", err)
	}

	if err := service.ReorderHosts([]string{hostA.ID}); err == nil {
		t.Fatal("ReorderHosts(short order) error = nil, want error")
	}
	if err := service.ReorderHosts([]string{hostA.ID, hostA.ID}); err == nil {
		t.Fatal("ReorderHosts(duplicate) error = nil, want error")
	}
	if err := service.ReorderHosts([]string{hostA.ID, "missing-host"}); err == nil {
		t.Fatal("ReorderHosts(missing host) error = nil, want error")
	}

	hosts, err := service.ListHosts()
	if err != nil {
		t.Fatalf("ListHosts() error = %v", err)
	}
	if hosts[0].ID != hostA.ID || hosts[1].ID != hostB.ID {
		t.Fatalf("Host order changed after invalid reorder: %+v", hosts)
	}
}

func TestServiceSetsResourceAuth(t *testing.T) {
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

	if err := service.SetResourceAuth(host.ID, domain.Auth{
		Method:  domain.AuthMethodKey,
		KeyName: " bashes-main ",
	}); err != nil {
		t.Fatalf("SetResourceAuth(host) error = %v", err)
	}
	if err := service.SetResourceAuth(subsystem.ID, domain.Auth{
		Method:         domain.AuthMethodPath,
		PrivateKeyPath: " ~/.ssh/custom ",
	}); err != nil {
		t.Fatalf("SetResourceAuth(subsystem) error = %v", err)
	}
	if err := service.SetResourceHostKeyFingerprint(host.ID, " SHA256:hostfingerprint "); err != nil {
		t.Fatalf("SetResourceHostKeyFingerprint(host) error = %v", err)
	}
	if err := service.SetResourceHostKeyFingerprint(subsystem.ID, "SHA256:subfingerprint"); err != nil {
		t.Fatalf("SetResourceHostKeyFingerprint(subsystem) error = %v", err)
	}

	hosts, err := service.ListHosts()
	if err != nil {
		t.Fatalf("ListHosts() error = %v", err)
	}
	if hosts[0].Auth == nil || hosts[0].Auth.Method != domain.AuthMethodKey || hosts[0].Auth.KeyName != "bashes-main" {
		t.Fatalf("Host auth was not saved: %+v", hosts[0].Auth)
	}
	if hosts[0].HostKeyFingerprint != "SHA256:hostfingerprint" {
		t.Fatalf("HostKeyFingerprint = %q", hosts[0].HostKeyFingerprint)
	}
	if hosts[0].Subsystems[0].Auth == nil || hosts[0].Subsystems[0].Auth.Method != domain.AuthMethodPath || hosts[0].Subsystems[0].Auth.PrivateKeyPath != "~/.ssh/custom" {
		t.Fatalf("Subsystem auth was not saved: %+v", hosts[0].Subsystems[0].Auth)
	}
	if hosts[0].Subsystems[0].HostKeyFingerprint != "SHA256:subfingerprint" {
		t.Fatalf("Subsystem HostKeyFingerprint = %q", hosts[0].Subsystems[0].HostKeyFingerprint)
	}
}

func TestServiceClearsHostKeyFingerprintWhenEndpointChanges(t *testing.T) {
	service := NewService(newMemoryStore())
	host, err := service.AddHost(EndpointInput{Hostname: "server", IP: "10.0.0.1", Port: 22, User: "root"})
	if err != nil {
		t.Fatalf("AddHost() error = %v", err)
	}
	if err := service.SetResourceHostKeyFingerprint(host.ID, "SHA256:known"); err != nil {
		t.Fatalf("SetResourceHostKeyFingerprint() error = %v", err)
	}

	if err := service.UpdateResource(host.ID, EndpointInput{Hostname: "server", IP: "10.0.0.2", Port: 22, User: "root"}); err != nil {
		t.Fatalf("UpdateResource() error = %v", err)
	}
	hosts, err := service.ListHosts()
	if err != nil {
		t.Fatalf("ListHosts() error = %v", err)
	}
	if hosts[0].HostKeyFingerprint != "" {
		t.Fatalf("HostKeyFingerprint = %q, want cleared fingerprint", hosts[0].HostKeyFingerprint)
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
