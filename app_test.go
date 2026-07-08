package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/signoredellarete/bashes/internal/application"
	"github.com/signoredellarete/bashes/internal/domain"
)

func TestApplyAuthPreferenceUsesStoredKey(t *testing.T) {
	resource := domain.Endpoint{
		ID:       "host-1",
		Hostname: "server-01",
		IP:       "10.0.0.10",
		Port:     22,
		User:     "root",
		Auth: &domain.Auth{
			Method:       domain.AuthMethodKey,
			KeyName:      "bashes-main",
			TrustHostKey: true,
		},
	}

	input := applyAuthPreference(resource, SSHSessionInput{ResourceID: resource.ID})
	if input.KeyName != "bashes-main" {
		t.Fatalf("KeyName = %q, want stored key", input.KeyName)
	}
	if !input.TrustHostKey {
		t.Fatal("TrustHostKey = false, want stored trust preference")
	}
}

func TestApplyAuthPreferenceDoesNotOverrideExplicitAuth(t *testing.T) {
	resource := domain.Endpoint{
		ID:       "host-1",
		Hostname: "server-01",
		IP:       "10.0.0.10",
		Port:     22,
		User:     "root",
		Auth: &domain.Auth{
			Method:  domain.AuthMethodKey,
			KeyName: "bashes-main",
		},
	}

	input := applyAuthPreference(resource, SSHSessionInput{
		ResourceID: resource.ID,
		Password:   "secret",
	})
	if input.KeyName != "" {
		t.Fatalf("KeyName = %q, want explicit password to keep key empty", input.KeyName)
	}
}

func TestAuthPreferenceFromSessionInput(t *testing.T) {
	auth := authPreferenceFromSessionInput(SSHSessionInput{
		KeyName:      "bashes-main",
		TrustHostKey: true,
	})
	if auth == nil || auth.Method != domain.AuthMethodKey || auth.KeyName != "bashes-main" || !auth.TrustHostKey {
		t.Fatalf("Auth preference from key input = %+v", auth)
	}

	auth = authPreferenceFromSessionInput(SSHSessionInput{Password: "secret"})
	if auth == nil || auth.Method != domain.AuthMethodPassword {
		t.Fatalf("Auth preference from password input = %+v", auth)
	}
}

func TestResolveSessionKeyPathUsesAppDataDirectory(t *testing.T) {
	app := NewApp(filepath.Join(t.TempDir(), "portable", "hosts.json"))

	input := app.resolveSessionKeyPath(SSHSessionInput{KeyName: "bashes-main"})
	want := filepath.Join(filepath.Dir(app.dataPath), "keys", "bashes-main")
	if input.PrivateKeyPath != want {
		t.Fatalf("PrivateKeyPath = %q, want %q", input.PrivateKeyPath, want)
	}
	if input.KeyName != "bashes-main" {
		t.Fatalf("KeyName = %q, want original key name preserved", input.KeyName)
	}
}

func TestNormalizeTunnelInputDefaultsToLocalSocks(t *testing.T) {
	input := SSHTunnelInput{LocalPort: 1080}
	if err := normalizeTunnelInput(&input); err != nil {
		t.Fatalf("normalizeTunnelInput() error = %v", err)
	}
	if input.Type != "socks" {
		t.Fatalf("Type = %q, want socks", input.Type)
	}
	if input.LocalHost != "127.0.0.1" {
		t.Fatalf("LocalHost = %q, want 127.0.0.1", input.LocalHost)
	}
}

func TestNormalizeTunnelInputAcceptsLocalAndRemoteForwarding(t *testing.T) {
	tests := []struct {
		name  string
		input SSHTunnelInput
	}{
		{
			name: "local forward",
			input: SSHTunnelInput{
				Type:       "local",
				LocalHost:  "127.0.0.1",
				LocalPort:  8080,
				RemoteHost: "127.0.0.1",
				RemotePort: 80,
			},
		},
		{
			name: "remote forward",
			input: SSHTunnelInput{
				Type:       "remote",
				LocalHost:  "127.0.0.1",
				LocalPort:  8080,
				RemoteHost: "127.0.0.1",
				RemotePort: 3000,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := tt.input
			if err := normalizeTunnelInput(&input); err != nil {
				t.Fatalf("normalizeTunnelInput() error = %v", err)
			}
		})
	}
}

func TestNormalizeTunnelInputRejectsUnsupportedTypeAndPort(t *testing.T) {
	input := SSHTunnelInput{Type: "reverse", LocalPort: 1080}
	if err := normalizeTunnelInput(&input); err == nil {
		t.Fatal("normalizeTunnelInput() error = nil, want unsupported type error")
	}

	input = SSHTunnelInput{Type: "socks", LocalPort: 70000}
	if err := normalizeTunnelInput(&input); err == nil {
		t.Fatal("normalizeTunnelInput() error = nil, want invalid port error")
	}

	input = SSHTunnelInput{Type: "local", LocalPort: 8080, RemotePort: 0}
	if err := normalizeTunnelInput(&input); err == nil {
		t.Fatal("normalizeTunnelInput() error = nil, want invalid forward target port error")
	}
}

func TestResourceIDsForDeleteIncludesHostSubsystems(t *testing.T) {
	app := NewApp(filepath.Join(t.TempDir(), "hosts.json"))
	host, err := app.AddHost(applicationEndpoint("host", "10.0.0.1"))
	if err != nil {
		t.Fatalf("AddHost() error = %v", err)
	}
	subsystem, err := app.AddSubsystem(host.ID, applicationEndpoint("vm", "10.0.0.2"))
	if err != nil {
		t.Fatalf("AddSubsystem() error = %v", err)
	}

	ids, err := app.resourceIDsForDelete(host.ID)
	if err != nil {
		t.Fatalf("resourceIDsForDelete() error = %v", err)
	}
	if len(ids) != 2 || ids[0] != host.ID || ids[1] != subsystem.ID {
		t.Fatalf("resourceIDsForDelete() = %v, want host and subsystem ids", ids)
	}
}

func TestResourceIDsForDeleteIncludesNestedSubsystems(t *testing.T) {
	app := NewApp(filepath.Join(t.TempDir(), "hosts.json"))
	host, err := app.AddHost(applicationEndpoint("host", "10.0.0.1"))
	if err != nil {
		t.Fatalf("AddHost() error = %v", err)
	}
	vm, err := app.AddSubsystem(host.ID, applicationEndpoint("vm", "10.0.0.2"))
	if err != nil {
		t.Fatalf("AddSubsystem(vm) error = %v", err)
	}
	lxc, err := app.AddSubsystem(vm.ID, applicationEndpoint("lxc", "10.0.0.3"))
	if err != nil {
		t.Fatalf("AddSubsystem(lxc) error = %v", err)
	}

	ids, err := app.resourceIDsForDelete(host.ID)
	if err != nil {
		t.Fatalf("resourceIDsForDelete(host) error = %v", err)
	}
	if len(ids) != 3 || ids[0] != host.ID || ids[1] != vm.ID || ids[2] != lxc.ID {
		t.Fatalf("resourceIDsForDelete(host) = %v, want host, vm and lxc ids", ids)
	}

	ids, err = app.resourceIDsForDelete(vm.ID)
	if err != nil {
		t.Fatalf("resourceIDsForDelete(vm) error = %v", err)
	}
	if len(ids) != 2 || ids[0] != vm.ID || ids[1] != lxc.ID {
		t.Fatalf("resourceIDsForDelete(vm) = %v, want vm and lxc ids", ids)
	}
}

func TestDataDirForOSUsesPlatformConventions(t *testing.T) {
	env := func(values map[string]string) func(string) string {
		return func(name string) string {
			return values[name]
		}
	}

	tests := []struct {
		name string
		goos string
		home string
		env  map[string]string
		want string
	}{
		{
			name: "macos application support",
			goos: "darwin",
			home: "/Users/alice",
			env:  map[string]string{},
			want: filepath.Join("/Users/alice", "Library", "Application Support", "Bashes"),
		},
		{
			name: "windows appdata",
			goos: "windows",
			home: `C:\Users\Alice`,
			env:  map[string]string{"APPDATA": `C:\Users\Alice\AppData\Roaming`},
			want: filepath.Join(`C:\Users\Alice\AppData\Roaming`, "Bashes"),
		},
		{
			name: "linux xdg data",
			goos: "linux",
			home: "/home/alice",
			env:  map[string]string{"XDG_DATA_HOME": "/home/alice/.local/state"},
			want: filepath.Join("/home/alice/.local/state", "bashes"),
		},
		{
			name: "linux home fallback",
			goos: "linux",
			home: "/home/alice",
			env:  map[string]string{},
			want: filepath.Join("/home/alice", ".local", "share", "bashes"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dataDirForOS(tt.goos, tt.home, env(tt.env))
			if got != tt.want {
				t.Fatalf("dataDirForOS() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExportAndImportDatabase(t *testing.T) {
	dir := t.TempDir()
	source := NewApp(filepath.Join(dir, "source", "hosts.json"))
	host, err := source.AddHost(applicationEndpoint("source-host", "10.0.0.10"))
	if err != nil {
		t.Fatalf("AddHost() error = %v", err)
	}
	if _, err := source.AddSubsystem(host.ID, applicationEndpoint("source-vm", "10.0.0.11")); err != nil {
		t.Fatalf("AddSubsystem() error = %v", err)
	}

	exportPath := filepath.Join(dir, "export", "bashes.json")
	if err := source.ExportDatabase(exportPath); err != nil {
		t.Fatalf("ExportDatabase() error = %v", err)
	}
	if _, err := os.Stat(exportPath); err != nil {
		t.Fatalf("export file stat error = %v", err)
	}

	target := NewApp(filepath.Join(dir, "target", "hosts.json"))
	if _, err := target.AddHost(applicationEndpoint("old-host", "10.0.0.20")); err != nil {
		t.Fatalf("AddHost(old) error = %v", err)
	}
	if err := target.ImportDatabase(exportPath); err != nil {
		t.Fatalf("ImportDatabase() error = %v", err)
	}

	hosts, err := target.ListHosts()
	if err != nil {
		t.Fatalf("ListHosts() error = %v", err)
	}
	if len(hosts) != 1 || hosts[0].Hostname != "source-host" {
		t.Fatalf("imported hosts = %+v, want source-host", hosts)
	}
	if len(hosts[0].Subsystems) != 1 || hosts[0].Subsystems[0].Hostname != "source-vm" {
		t.Fatalf("imported subsystems = %+v, want source-vm", hosts[0].Subsystems)
	}
	if _, err := os.Stat(filepath.Join(dir, "target", "hosts.json.bak")); err != nil {
		t.Fatalf("backup stat error = %v", err)
	}
}

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		latest  string
		current string
		want    bool
	}{
		{latest: "v0.1.43", current: "v0.1.42", want: true},
		{latest: "v0.2.0", current: "v0.1.99", want: true},
		{latest: "v1.0.0", current: "v1.0.0", want: false},
		{latest: "v0.1.42", current: "v0.1.43", want: false},
		{latest: "v0.1.43", current: "dev", want: false},
	}

	for _, tt := range tests {
		if got := isNewerVersion(tt.latest, tt.current); got != tt.want {
			t.Fatalf("isNewerVersion(%q, %q) = %v, want %v", tt.latest, tt.current, got, tt.want)
		}
	}
}

func applicationEndpoint(hostname string, ip string) application.EndpointInput {
	return application.EndpointInput{
		Hostname: hostname,
		IP:       ip,
		Port:     22,
		User:     "root",
		Type:     domain.ResourceVM,
	}
}
