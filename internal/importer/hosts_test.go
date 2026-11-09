package importer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/signoredellarete/bashes/internal/domain"
)

func TestParseHostsFileSkipsLocalAndImportsFirstAlias(t *testing.T) {
	entries := parseHostsFile([]byte(`
127.0.0.1 localhost
::1 localhost
255.255.255.255 broadcasthost
10.0.0.10 bastion bastion.local
192.168.1.20 vm1 # comment
fe80::1 link-local
`))
	if len(entries) != 2 || entries[0].Hostname != "bastion" || entries[1].Hostname != "vm1" {
		t.Fatalf("parseHostsFile() = %+v", entries)
	}
}

func TestParseSSHConfigMatchesExplicitHostsOnly(t *testing.T) {
	config := parseSSHConfig([]byte(`
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
		t.Fatalf("parseSSHConfig() = %+v", config)
	}
	match, ok := config.match(hostEntry{IP: "10.0.0.10", Hostname: "bastion"})
	if !ok || match.user != "admin" || match.port != 2202 {
		t.Fatalf("bastion match = %+v, %v", match, ok)
	}
	if _, ok := config.match(hostEntry{IP: "192.168.1.30", Hostname: "web.prod"}); ok {
		t.Fatal("wildcard host should not match")
	}
}

func TestPrepareHostsFileAppliesOverridesAndSkipsExistingResources(t *testing.T) {
	dir := t.TempDir()
	hostsPath := filepath.Join(dir, "hosts")
	configPath := filepath.Join(dir, "config")
	if err := os.WriteFile(hostsPath, []byte("10.0.0.10 duplicate\n10.0.0.20 bastion\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(hosts) error = %v", err)
	}
	if err := os.WriteFile(configPath, []byte("Host bastion\n User admin\n Port 2202\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(config) error = %v", err)
	}
	store := domain.NewStore()
	store.Hosts = append(store.Hosts, domain.Host{ID: "existing", Hostname: "existing", IP: "10.0.0.10", Port: 22, User: "user"})
	store.TopLevelOrder = append(store.TopLevelOrder, "existing")

	result, updated, err := PrepareHostsFile(hostsPath, "local", configPath, store)
	if err != nil {
		t.Fatalf("PrepareHostsFile() error = %v", err)
	}
	if result.Imported != 1 || result.Skipped != 1 {
		t.Fatalf("result = %+v", result)
	}
	imported := updated.Hosts[1]
	if imported.Hostname != "bastion" || imported.User != "admin" || imported.Port != 2202 {
		t.Fatalf("imported host = %+v", imported)
	}
}
