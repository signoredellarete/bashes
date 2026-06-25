package remotessh

import (
	"context"
	"strings"
	"testing"
	"time"
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

func TestNewClientConfigRejectsMissingHostKeyPolicy(t *testing.T) {
	_, err := NewClientConfig(ClientOptions{
		Target:      Target{Host: "example.test", Port: 22, User: "admin"},
		Credentials: Credentials{Password: "secret"},
	})
	if err == nil {
		t.Fatal("NewClientConfig() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "host key policy") {
		t.Fatalf("NewClientConfig() error = %v, want host key policy error", err)
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

func TestAddressUsesHostPortFormatting(t *testing.T) {
	got := Address(Target{Host: "2001:db8::1", Port: 2222})
	if got != "[2001:db8::1]:2222" {
		t.Fatalf("Address() = %q, want bracketed IPv6 hostport", got)
	}
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
