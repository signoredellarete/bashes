package sshauth

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestBuilderRejectsUnresolvedManagedKey(t *testing.T) {
	builder := newTestBuilder(t)
	_, _, err := builder.Methods(Input{KeyName: "managed key"})
	if err == nil || !strings.Contains(err.Error(), `ssh key "managed-key" was not resolved`) {
		t.Fatalf("Methods() error = %v", err)
	}
}

func TestBuilderLoadsDefaultSystemKey(t *testing.T) {
	builder := newTestBuilder(t)
	privateKey, _ := testKeyPair(t)
	sshDir := filepath.Join(builder.homeDir(), ".ssh")
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sshDir, "id_ed25519"), privateKey, 0o600); err != nil {
		t.Fatal(err)
	}

	methods, connection, err := builder.Methods(Input{})
	if err != nil {
		t.Fatalf("Methods() error = %v", err)
	}
	if connection != nil {
		t.Fatal("agent connection is not nil")
	}
	if len(methods) != 1 {
		t.Fatalf("len(methods) = %d, want 1", len(methods))
	}
}

func TestBuilderAcceptsPasswordWithoutLocalKeys(t *testing.T) {
	builder := newTestBuilder(t)
	methods, connection, err := builder.Methods(Input{Password: "secret"})
	if err != nil {
		t.Fatalf("Methods() error = %v", err)
	}
	if len(methods) != 0 || connection != nil {
		t.Fatalf("methods=%d connection=%v, want password-only auth", len(methods), connection)
	}
}

func TestParsePrivateKeyRejectsPublicKeyText(t *testing.T) {
	_, publicKey := testKeyPair(t)
	_, err := ParsePrivateKey(publicKey, "")
	if err == nil || !strings.Contains(err.Error(), "selected file is an SSH public key") {
		t.Fatalf("ParsePrivateKey() error = %v", err)
	}
}

func TestPublicKeyFromPrivateDataMatchesGeneratedPair(t *testing.T) {
	privateKey, publicKeyText := testKeyPair(t)
	publicKey, comparable, err := PublicKeyFromPrivateData(privateKey)
	if err != nil {
		t.Fatalf("PublicKeyFromPrivateData() error = %v", err)
	}
	expected, _, _, _, err := ssh.ParseAuthorizedKey(publicKeyText)
	if err != nil {
		t.Fatal(err)
	}
	if !comparable || string(publicKey.Marshal()) != string(expected.Marshal()) {
		t.Fatal("derived public key does not match generated key")
	}
}

func newTestBuilder(t *testing.T) *Builder {
	t.Helper()
	builder := NewBuilder()
	home := t.TempDir()
	builder.homeDir = func() string { return home }
	builder.getenv = func(string) string { return "" }
	return builder
}

func testKeyPair(t *testing.T) ([]byte, []byte) {
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
	return pem.EncodeToMemory(privateBlock), ssh.MarshalAuthorizedKey(sshPublicKey)
}
