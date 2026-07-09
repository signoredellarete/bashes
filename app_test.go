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

	auth = authPreferenceFromSessionInput(SSHSessionInput{
		KeyName:        "bashes-main",
		PrivateKeyPath: "/home/user/.ssh/id_ed25519",
	})
	if auth == nil || auth.Method != domain.AuthMethodPath || auth.PrivateKeyPath != "/home/user/.ssh/id_ed25519" {
		t.Fatalf("Auth preference with path and key input = %+v, want path preference", auth)
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

func TestGenerateSSHKeyUsesNextDefaultName(t *testing.T) {
	app := NewApp(filepath.Join(t.TempDir(), "data", "hosts.json"))

	first, err := app.GenerateSSHKey(GenerateSSHKeyInput{})
	if err != nil {
		t.Fatalf("GenerateSSHKey(first) error = %v", err)
	}
	second, err := app.GenerateSSHKey(GenerateSSHKeyInput{})
	if err != nil {
		t.Fatalf("GenerateSSHKey(second) error = %v", err)
	}

	if first.Name != "bashes" {
		t.Fatalf("first key name = %q, want bashes", first.Name)
	}
	if second.Name != "bashes-2" {
		t.Fatalf("second key name = %q, want bashes-2", second.Name)
	}
	if _, err := os.Stat(first.PrivateKey); err != nil {
		t.Fatalf("first private key missing: %v", err)
	}
	if _, err := os.Stat(second.PrivateKey); err != nil {
		t.Fatalf("second private key missing: %v", err)
	}
}

func TestListSystemSSHKeysIncludesDefaultSSHDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		t.Fatalf("MkdirAll(.ssh) error = %v", err)
	}
	privatePath := filepath.Join(sshDir, "id_ed25519")
	publicPath := privatePath + ".pub"
	if err := os.WriteFile(privatePath, []byte("private"), 0o600); err != nil {
		t.Fatalf("WriteFile(private) error = %v", err)
	}
	if err := os.WriteFile(publicPath, []byte("ssh-ed25519 test"), 0o644); err != nil {
		t.Fatalf("WriteFile(public) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(sshDir, "known_hosts"), []byte("ignored"), 0o644); err != nil {
		t.Fatalf("WriteFile(known_hosts) error = %v", err)
	}

	keys, err := listSystemSSHKeys()
	if err != nil {
		t.Fatalf("listSystemSSHKeys() error = %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("len(keys) = %d, want 1: %+v", len(keys), keys)
	}
	if keys[0].Source != "system" || keys[0].Name != "id_ed25519" || keys[0].PrivateKey != privatePath || keys[0].PublicKey != publicPath {
		t.Fatalf("system key = %+v, want id_ed25519 paths", keys[0])
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

func TestHostsFilePathForOS(t *testing.T) {
	env := func(values map[string]string) func(string) string {
		return func(name string) string {
			return values[name]
		}
	}

	if got := hostsFilePathForOS("linux", env(nil)); got != "/etc/hosts" {
		t.Fatalf("linux hosts path = %q, want /etc/hosts", got)
	}
	if got := hostsFilePathForOS("darwin", env(nil)); got != "/etc/hosts" {
		t.Fatalf("darwin hosts path = %q, want /etc/hosts", got)
	}
	got := hostsFilePathForOS("windows", env(map[string]string{"SystemRoot": `C:\Windows`}))
	want := filepath.Join(`C:\Windows`, "System32", "drivers", "etc", "hosts")
	if got != want {
		t.Fatalf("windows hosts path = %q, want %q", got, want)
	}
	if got := sshConfigPathForOS("linux", "/home/alice"); got != filepath.Join("/home/alice", ".ssh", "config") {
		t.Fatalf("linux ssh config path = %q", got)
	}
	if got := sshConfigPathForOS("darwin", "/Users/alice"); got != filepath.Join("/Users/alice", ".ssh", "config") {
		t.Fatalf("darwin ssh config path = %q", got)
	}
	if got := sshConfigPathForOS("windows", `C:\Users\Alice`); got != "" {
		t.Fatalf("windows ssh config path = %q, want empty", got)
	}
}

func TestParseHostsFileSkipsLocalAndImportsFirstAlias(t *testing.T) {
	data := []byte(`
127.0.0.1 localhost
::1 localhost
255.255.255.255 broadcasthost
10.0.0.10 bastion bastion.local
192.168.1.20 vm1 # comment
fe80::1 link-local
`)

	entries := parseHostsFile(data)
	if len(entries) != 2 {
		t.Fatalf("parseHostsFile() returned %d entries: %+v", len(entries), entries)
	}
	if entries[0].IP != "10.0.0.10" || entries[0].Hostname != "bastion" {
		t.Fatalf("first entry = %+v, want bastion", entries[0])
	}
	if entries[1].IP != "192.168.1.20" || entries[1].Hostname != "vm1" {
		t.Fatalf("second entry = %+v, want vm1", entries[1])
	}
}

func TestParseSimpleSSHConfigMatchesExplicitHostsOnly(t *testing.T) {
	config := parseSimpleSSHConfig([]byte(`
Host bastion bastion.local
  HostName 10.0.0.10
  User admin
  Port 2202

Host *.prod
  User ignored

Host vm1
  User deploy
  Port 2222

Match host *
  User ignored
`))

	if len(config) != 2 {
		t.Fatalf("parseSimpleSSHConfig() returned %d entries: %+v", len(config), config)
	}
	match, ok := config.match(hostsFileEntry{IP: "10.0.0.10", Hostname: "bastion"})
	if !ok {
		t.Fatal("expected bastion match")
	}
	if match.user != "admin" || match.port != 2202 {
		t.Fatalf("bastion match = %+v, want admin:2202", match)
	}
	match, ok = config.match(hostsFileEntry{IP: "192.168.1.20", Hostname: "vm1"})
	if !ok {
		t.Fatal("expected vm1 match")
	}
	if match.user != "deploy" || match.port != 2222 {
		t.Fatalf("vm1 match = %+v, want deploy:2222", match)
	}
	if _, ok := config.match(hostsFileEntry{IP: "192.168.1.30", Hostname: "web.prod"}); ok {
		t.Fatal("wildcard host should not match")
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

func TestImportFromHostsFileAddsNewHostsAndSkipsDuplicates(t *testing.T) {
	dir := t.TempDir()
	app := NewApp(filepath.Join(dir, "hosts.json"))
	if _, err := app.AddHost(applicationEndpoint("existing", "10.0.0.10")); err != nil {
		t.Fatalf("AddHost(existing) error = %v", err)
	}

	hostsFile := filepath.Join(dir, "system-hosts")
	if err := os.WriteFile(hostsFile, []byte(`
127.0.0.1 localhost
10.0.0.10 duplicate-ip
10.0.0.20 imported-one
10.0.0.21 imported-two imported-two.local
`), 0o600); err != nil {
		t.Fatalf("write hosts file: %v", err)
	}

	result, err := app.importFromHostsFile(hostsFile, "deploy", "")
	if err != nil {
		t.Fatalf("importFromHostsFile() error = %v", err)
	}
	if result.Imported != 2 || result.Skipped != 1 {
		t.Fatalf("import result = %+v, want imported=2 skipped=1", result)
	}

	hosts, err := app.ListHosts()
	if err != nil {
		t.Fatalf("ListHosts() error = %v", err)
	}
	if len(hosts) != 3 {
		t.Fatalf("host count = %d, want 3: %+v", len(hosts), hosts)
	}
	if hosts[1].Hostname != "imported-one" || hosts[1].User != "deploy" || hosts[1].Port != 22 {
		t.Fatalf("imported host = %+v, want imported-one deploy:22", hosts[1])
	}
	if hosts[2].Hostname != "imported-two" {
		t.Fatalf("second imported host = %+v, want imported-two", hosts[2])
	}
}

func TestImportFromHostsFileUsesSimpleSSHConfigOverrides(t *testing.T) {
	dir := t.TempDir()
	app := NewApp(filepath.Join(dir, "hosts.json"))

	hostsFile := filepath.Join(dir, "system-hosts")
	if err := os.WriteFile(hostsFile, []byte(`
10.0.0.20 bastion
10.0.0.21 vm1
10.0.0.22 plain
`), 0o600); err != nil {
		t.Fatalf("write hosts file: %v", err)
	}
	sshConfig := filepath.Join(dir, "ssh-config")
	if err := os.WriteFile(sshConfig, []byte(`
Host bastion
  User admin
  Port 2202

Host vm-alias
  HostName vm1
  User deploy
  Port 2222
`), 0o600); err != nil {
		t.Fatalf("write ssh config: %v", err)
	}

	result, err := app.importFromHostsFile(hostsFile, "localuser", sshConfig)
	if err != nil {
		t.Fatalf("importFromHostsFile() error = %v", err)
	}
	if result.Imported != 3 {
		t.Fatalf("imported = %d, want 3", result.Imported)
	}

	hosts, err := app.ListHosts()
	if err != nil {
		t.Fatalf("ListHosts() error = %v", err)
	}
	if hosts[0].Hostname != "bastion" || hosts[0].User != "admin" || hosts[0].Port != 2202 {
		t.Fatalf("bastion host = %+v, want admin:2202", hosts[0])
	}
	if hosts[1].Hostname != "vm1" || hosts[1].User != "deploy" || hosts[1].Port != 2222 {
		t.Fatalf("vm1 host = %+v, want deploy:2222", hosts[1])
	}
	if hosts[2].Hostname != "plain" || hosts[2].User != "localuser" || hosts[2].Port != 22 {
		t.Fatalf("plain host = %+v, want localuser:22", hosts[2])
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

func TestParsePrivateKeyAcceptsPuttyPPK(t *testing.T) {
	signer, err := parsePrivateKey([]byte(testPuttyPPK), "")
	if err != nil {
		t.Fatalf("parsePrivateKey(PPK) error = %v", err)
	}
	if signer.PublicKey().Type() != "ssh-rsa" {
		t.Fatalf("PPK signer type = %q, want ssh-rsa", signer.PublicKey().Type())
	}
}

func TestIsPuttyPrivateKeyAllowsWhitespaceAndBOM(t *testing.T) {
	key := append([]byte{0xef, 0xbb, 0xbf, '\n'}, []byte("PuTTY-User-Key-File-2: ssh-rsa\n")...)
	if !isPuttyPrivateKey(key) {
		t.Fatal("isPuttyPrivateKey() = false, want true")
	}
}

const testPuttyPPK = `PuTTY-User-Key-File-2: ssh-rsa
Encryption: none
Comment: a@b
Public-Lines: 2
AAAAB3NzaC1yc2EAAAABJQAAAEEAqexbeyaaBw2rFZc2vwg4DqjOo6fQyOdfo9O2
20y96bUlHRYzRWmIDzHC5gZBzlHQ6M56dprxhCJbsIQig+sQ+w==
Private-Lines: 4
AAAAQBb2bTonz6AWmpQ3B2XsWpoyfMoB68gfREaSO04RShipjkwri4K8DmSX1+Nb
xUyFO7aS7rpsO3mitZtYt3bS3z0AAAAhANvUiZew5AgUZ3peSzSqaVch4vapHml4
7nx03dx4aS5JAAAAIQDF4bDGZq973zNxW62MVA6MsxKdNsIDILMFvhXFNc/VIwAA
ACEAgd1SYGV2aEEMQaMGQ4CnjQeiAuZL4z7OVTBTrtGap1A=
Private-MAC: 3c3a9bd98e8e912f6163be95321676b6103aaed8`

func applicationEndpoint(hostname string, ip string) application.EndpointInput {
	return application.EndpointInput{
		Hostname: hostname,
		IP:       ip,
		Port:     22,
		User:     "root",
		Type:     domain.ResourceVM,
	}
}
