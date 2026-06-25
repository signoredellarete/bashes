package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/signoredellarete/bashes/internal/domain"
)

func TestLoadMissingStoreReturnsEmptyStore(t *testing.T) {
	repo := NewRepository(filepath.Join(t.TempDir(), "hosts.json"))

	loaded, err := repo.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.Version != domain.CurrentSchemaVersion {
		t.Fatalf("Version = %d, want %d", loaded.Version, domain.CurrentSchemaVersion)
	}
	if len(loaded.Hosts) != 0 {
		t.Fatalf("Hosts length = %d, want 0", len(loaded.Hosts))
	}
}

func TestLoadLegacyHostsMigratesGroupedSubsystems(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hosts.json")
	legacy := `[
  {
    "hostname": "proxmox-01",
    "ip": "10.0.0.10",
    "port": "22",
    "user": "root",
    "lxc": [
      {"hostname": "lxc-web", "ip": "10.0.0.20", "port": "2222", "user": "deploy"}
    ],
    "vm": [
      {"hostname": "vm-db", "ip": "10.0.0.30", "port": 22, "user": "ubuntu"}
    ],
    "docker": []
  }
]`
	if err := os.WriteFile(path, []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}

	loaded, err := NewRepository(path).Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(loaded.Hosts) != 1 {
		t.Fatalf("Hosts length = %d, want 1", len(loaded.Hosts))
	}
	host := loaded.Hosts[0]
	if host.ID == "" {
		t.Fatal("Host ID is empty")
	}
	if host.Port != 22 {
		t.Fatalf("Host port = %d, want 22", host.Port)
	}
	if len(host.Subsystems) != 2 {
		t.Fatalf("Subsystems length = %d, want 2", len(host.Subsystems))
	}
	if host.Subsystems[0].Type != domain.ResourceLXC {
		t.Fatalf("First subsystem type = %q, want %q", host.Subsystems[0].Type, domain.ResourceLXC)
	}
	if host.Subsystems[1].Type != domain.ResourceVM {
		t.Fatalf("Second subsystem type = %q, want %q", host.Subsystems[1].Type, domain.ResourceVM)
	}
}

func TestSaveWritesReadableVersionedJSONAndBackup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hosts.json")
	if err := os.WriteFile(path, []byte("[]\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	repo := NewRepository(path)
	store := domain.Store{
		Version: domain.CurrentSchemaVersion,
		Hosts: []domain.Host{
			{
				Hostname: "server-01",
				IP:       "192.168.1.10",
				Port:     22,
				User:     "admin",
			},
		},
	}

	if err := repo.Save(store); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"version": 1`) {
		t.Fatalf("Saved JSON does not include schema version:\n%s", data)
	}
	if !strings.Contains(string(data), `"hosts"`) {
		t.Fatalf("Saved JSON does not include hosts:\n%s", data)
	}

	backup, err := os.ReadFile(path + ".bak")
	if err != nil {
		t.Fatalf("Expected backup file: %v", err)
	}
	if string(backup) != "[]\n" {
		t.Fatalf("Backup = %q, want old contents", backup)
	}

	loaded, err := repo.Load()
	if err != nil {
		t.Fatalf("Load() after Save() error = %v", err)
	}
	if len(loaded.Hosts) != 1 {
		t.Fatalf("Loaded hosts length = %d, want 1", len(loaded.Hosts))
	}
	if loaded.Hosts[0].ID == "" {
		t.Fatal("Saved host ID was not normalized")
	}
}

func TestLoadRejectsInvalidLegacyPort(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hosts.json")
	legacy := `[{"hostname":"bad","ip":"127.0.0.1","port":"nope","user":"root","lxc":[],"vm":[],"docker":[]}]`
	if err := os.WriteFile(path, []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := NewRepository(path).Load()
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "decode port") {
		t.Fatalf("Load() error = %v, want decode port error", err)
	}
}

func TestSaveRejectsInvalidStore(t *testing.T) {
	repo := NewRepository(filepath.Join(t.TempDir(), "hosts.json"))
	err := repo.Save(domain.Store{
		Version: domain.CurrentSchemaVersion,
		Hosts: []domain.Host{
			{ID: "host-bad", Hostname: "bad", IP: "127.0.0.1", Port: 70000, User: "root"},
		},
	})

	if err == nil {
		t.Fatal("Save() error = nil, want validation error")
	}
	if !strings.Contains(err.Error(), "port") {
		t.Fatalf("Save() error = %v, want port validation error", err)
	}
}
