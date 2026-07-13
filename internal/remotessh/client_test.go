package remotessh

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

func TestNewClientConfigAcceptsPasswordWithExplicitInsecurePolicy(t *testing.T) {
	config, err := NewClientConfig(ClientOptions{
		Target: Target{
			Host: "example.test",
			Port: 22,
			User: "admin",
		},
		Credentials: Credentials{
			Password: "secret",
		},
		HostKeyPolicy: HostKeyPolicy{
			InsecureIgnoreHostKey: true,
		},
	})
	if err != nil {
		t.Fatalf("NewClientConfig() error = %v", err)
	}

	if config.User != "admin" {
		t.Fatalf("User = %q, want admin", config.User)
	}
	if config.Timeout != DefaultTimeout {
		t.Fatalf("Timeout = %s, want %s", config.Timeout, DefaultTimeout)
	}
	if len(config.Auth) != 1 {
		t.Fatalf("Auth method count = %d, want 1", len(config.Auth))
	}
	if config.HostKeyCallback == nil {
		t.Fatal("HostKeyCallback is nil")
	}
}

func TestNewClientConfigRejectsMissingAuth(t *testing.T) {
	_, err := NewClientConfig(ClientOptions{
		Target: Target{Host: "example.test", Port: 22, User: "admin"},
		HostKeyPolicy: HostKeyPolicy{
			InsecureIgnoreHostKey: true,
		},
	})
	if err == nil {
		t.Fatal("NewClientConfig() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "authentication") {
		t.Fatalf("NewClientConfig() error = %v, want authentication error", err)
	}
}

func TestNewClientConfigAcceptsTOFUHostKeyPolicy(t *testing.T) {
	config, err := NewClientConfig(ClientOptions{
		Target:      Target{Host: "example.test", Port: 22, User: "admin"},
		Credentials: Credentials{Password: "secret"},
	})
	if err != nil {
		t.Fatalf("NewClientConfig() error = %v", err)
	}
	if config.HostKeyCallback == nil {
		t.Fatal("HostKeyCallback is nil")
	}
}

func TestValidateTargetRejectsInvalidPortAndUser(t *testing.T) {
	err := ValidateTarget(Target{Host: "example.test", Port: 70000, User: "admin"})
	if err == nil || !strings.Contains(err.Error(), "port") {
		t.Fatalf("ValidateTarget() error = %v, want port error", err)
	}

	err = ValidateTarget(Target{Host: "example.test", Port: 22, User: "bad user"})
	if err == nil || !strings.Contains(err.Error(), "user") {
		t.Fatalf("ValidateTarget() error = %v, want user error", err)
	}
}

func TestAuthMethodsRejectsInvalidPrivateKey(t *testing.T) {
	_, err := AuthMethods(Credentials{PrivateKeyPEM: []byte("not a private key")})
	if err == nil {
		t.Fatal("AuthMethods() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "parse private key") {
		t.Fatalf("AuthMethods() error = %v, want private key parse error", err)
	}
}

func TestHostKeyCallbackRejectsMissingKnownHosts(t *testing.T) {
	_, err := HostKeyCallback(HostKeyPolicy{KnownHostsPath: "/path/that/does/not/exist"})
	if err == nil {
		t.Fatal("HostKeyCallback() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "known_hosts") {
		t.Fatalf("HostKeyCallback() error = %v, want known_hosts error", err)
	}
}

func TestHostKeyCallbackReturnsUnknownHostKeyFingerprint(t *testing.T) {
	key := testPublicKey(t)
	callback, err := HostKeyCallback(HostKeyPolicy{})
	if err != nil {
		t.Fatalf("HostKeyCallback() error = %v", err)
	}

	err = callback("example.test:22", nil, key)
	var unknown UnknownHostKeyError
	if !errors.As(err, &unknown) {
		t.Fatalf("HostKeyCallback() error = %T %[1]v, want UnknownHostKeyError", err)
	}
	if unknown.Fingerprint != ssh.FingerprintSHA256(key) {
		t.Fatalf("Fingerprint = %q, want %q", unknown.Fingerprint, ssh.FingerprintSHA256(key))
	}
}

func TestHostKeyCallbackAcceptsExpectedFingerprint(t *testing.T) {
	key := testPublicKey(t)
	callback, err := HostKeyCallback(HostKeyPolicy{ExpectedFingerprint: ssh.FingerprintSHA256(key)})
	if err != nil {
		t.Fatalf("HostKeyCallback() error = %v", err)
	}
	if err := callback("example.test:22", nil, key); err != nil {
		t.Fatalf("HostKeyCallback() verify error = %v", err)
	}
}

func TestHostKeyCallbackCapturesAcceptedFingerprint(t *testing.T) {
	key := testPublicKey(t)
	var accepted string
	callback, err := HostKeyCallback(HostKeyPolicy{AcceptNewHostKey: true, AcceptedFingerprint: &accepted})
	if err != nil {
		t.Fatalf("HostKeyCallback() error = %v", err)
	}
	if err := callback("example.test:22", nil, key); err != nil {
		t.Fatalf("HostKeyCallback() verify error = %v", err)
	}
	if accepted != ssh.FingerprintSHA256(key) {
		t.Fatalf("accepted fingerprint = %q, want %q", accepted, ssh.FingerprintSHA256(key))
	}
}

func TestHostKeyCallbackReportsAndAcceptsKnownHostsMismatch(t *testing.T) {
	knownKey := testPublicKey(t)
	actualKey := testPublicKey(t)
	knownHostsPath := filepath.Join(t.TempDir(), "known_hosts")
	line := knownhosts.Line([]string{"example.test"}, knownKey) + "\n"
	if err := os.WriteFile(knownHostsPath, []byte(line), 0o600); err != nil {
		t.Fatalf("WriteFile(known_hosts) error = %v", err)
	}
	remote := &net.TCPAddr{IP: net.ParseIP("192.0.2.10"), Port: 22}

	callback, err := HostKeyCallback(HostKeyPolicy{KnownHostsPath: knownHostsPath})
	if err != nil {
		t.Fatalf("HostKeyCallback() error = %v", err)
	}
	err = callback("example.test:22", remote, actualKey)
	var mismatch HostKeyMismatchError
	if !errors.As(err, &mismatch) {
		t.Fatalf("HostKeyCallback() error = %T %[1]v, want HostKeyMismatchError", err)
	}
	if mismatch.ExpectedFingerprint != ssh.FingerprintSHA256(knownKey) {
		t.Fatalf("ExpectedFingerprint = %q, want %q", mismatch.ExpectedFingerprint, ssh.FingerprintSHA256(knownKey))
	}
	if mismatch.ActualFingerprint != ssh.FingerprintSHA256(actualKey) {
		t.Fatalf("ActualFingerprint = %q, want %q", mismatch.ActualFingerprint, ssh.FingerprintSHA256(actualKey))
	}

	var accepted string
	callback, err = HostKeyCallback(HostKeyPolicy{
		KnownHostsPath:       knownHostsPath,
		AcceptChangedHostKey: true,
		AcceptedFingerprint:  &accepted,
	})
	if err != nil {
		t.Fatalf("HostKeyCallback(replace) error = %v", err)
	}
	if err := callback("example.test:22", remote, actualKey); err != nil {
		t.Fatalf("HostKeyCallback(replace) verify error = %v", err)
	}
	if accepted != ssh.FingerprintSHA256(actualKey) {
		t.Fatalf("accepted fingerprint = %q, want %q", accepted, ssh.FingerprintSHA256(actualKey))
	}
}

func TestAddressUsesHostPortFormatting(t *testing.T) {
	got := Address(Target{Host: "2001:db8::1", Port: 2222})
	if got != "[2001:db8::1]:2222" {
		t.Fatalf("Address() = %q, want bracketed IPv6 hostport", got)
	}
}

func testPublicKey(t *testing.T) ssh.PublicKey {
	t.Helper()
	publicKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	key, err := ssh.NewPublicKey(publicKey)
	if err != nil {
		t.Fatalf("NewPublicKey() error = %v", err)
	}
	return key
}

func TestDialHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := Dial(ctx, ClientOptions{
		Target: Target{
			Host: "203.0.113.1",
			Port: 22,
			User: "admin",
		},
		Credentials: Credentials{
			Password: "secret",
		},
		HostKeyPolicy: HostKeyPolicy{
			InsecureIgnoreHostKey: true,
		},
		Timeout: time.Millisecond,
	})
	if err == nil {
		t.Fatal("Dial() error = nil, want context/dial error")
	}
}
