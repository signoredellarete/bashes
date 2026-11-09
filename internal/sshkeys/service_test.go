package sshkeys

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	appsettings "github.com/signoredellarete/bashes/internal/settings"
	"golang.org/x/crypto/ssh"
)

func TestServiceGenerateUsesProgressiveDefaultNames(t *testing.T) {
	service := newTestService(t, nil)

	first, err := service.Generate(GenerateInput{})
	if err != nil {
		t.Fatalf("Generate(first) error = %v", err)
	}
	second, err := service.Generate(GenerateInput{})
	if err != nil {
		t.Fatalf("Generate(second) error = %v", err)
	}

	if first.Name != "bashes" || second.Name != "bashes-2" {
		t.Fatalf("generated names = %q, %q", first.Name, second.Name)
	}
	for _, path := range []string{first.PrivateKey, first.PublicKey, second.PrivateKey, second.PublicKey} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("generated key %q missing: %v", path, err)
		}
	}
}

func TestServiceImportRejectsMismatchedPair(t *testing.T) {
	service := newTestService(t, nil)
	privateKey, _ := testKeyPair(t)
	_, otherPublicKey := testKeyPair(t)

	_, err := service.ImportPair(ImportInput{PrivateKey: privateKey, PublicKey: otherPublicKey})
	if err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("ImportPair() error = %v, want mismatch", err)
	}
}

func TestServiceInventoryCombinesManagedSystemCustomAndAdditionalSources(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	service := newTestService(t, func() ([]Key, []Source) {
		return []Key{{Name: "wsl-key", PrivateKey: `C:\\wsl\\id_wsl`, Source: "wsl"}},
			[]Source{{Source: "wsl", Label: "WSL", Count: 1}}
	})
	if _, err := service.Generate(GenerateInput{Name: "managed"}); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	systemDir := filepath.Join(home, ".ssh")
	writeTestKeyFiles(t, systemDir, "id_system")
	customDir := t.TempDir()
	writeTestKeyFiles(t, customDir, "id_custom")
	if _, err := service.SaveSettings(appsettings.SSHKeys{CustomDirectory: customDir}); err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}

	inventory, err := service.Inventory()
	if err != nil {
		t.Fatalf("Inventory() error = %v", err)
	}
	sources := make(map[string]Source, len(inventory.Sources))
	for _, source := range inventory.Sources {
		sources[source.Source] = source
	}
	for _, source := range []string{"bashes", "system", "custom", "wsl"} {
		if sources[source].Count != 1 {
			t.Fatalf("source %q = %+v, want one key", source, sources[source])
		}
	}
}

func TestServiceDeleteCannotEscapeManagedDirectory(t *testing.T) {
	root := t.TempDir()
	service := NewService(
		filepath.Join(root, "keys"),
		appsettings.NewRepository(filepath.Join(root, "settings.json")),
		nil,
	)
	externalPath := filepath.Join(root, "external")
	if err := os.WriteFile(externalPath, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := service.Delete("../external"); err == nil {
		t.Fatal("Delete() error = nil, want invalid name")
	}
	data, err := os.ReadFile(externalPath)
	if err != nil || string(data) != "keep" {
		t.Fatalf("external key changed: data=%q error=%v", data, err)
	}
}

func TestWritePairAtomicPreservesExistingPrivateKey(t *testing.T) {
	dir := t.TempDir()
	privatePath := filepath.Join(dir, "existing")
	publicPath := privatePath + ".pub"
	if err := os.WriteFile(privatePath, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := WritePairAtomic(privatePath, []byte("replacement"), publicPath, []byte("public"))
	if err == nil {
		t.Fatal("WritePairAtomic() error = nil, want existing key error")
	}
	data, readErr := os.ReadFile(privatePath)
	if readErr != nil || string(data) != "original" {
		t.Fatalf("existing private key changed: data=%q error=%v", data, readErr)
	}
	if _, statErr := os.Stat(publicPath); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("public key unexpectedly created: %v", statErr)
	}
}

func newTestService(t *testing.T, additional AdditionalInventory) *Service {
	t.Helper()
	root := t.TempDir()
	return NewService(
		filepath.Join(root, "keys"),
		appsettings.NewRepository(filepath.Join(root, "settings.json")),
		additional,
	)
}

func writeTestKeyFiles(t *testing.T, dir string, name string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte("private"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+".pub"), []byte("ssh-ed25519 test"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func testKeyPair(t *testing.T) (string, string) {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	privateBlock, err := ssh.MarshalPrivateKey(privateKey, "bashes-test")
	if err != nil {
		t.Fatal(err)
	}
	sshPublicKey, err := ssh.NewPublicKey(publicKey)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(privateBlock)), string(ssh.MarshalAuthorizedKey(sshPublicKey))
}
