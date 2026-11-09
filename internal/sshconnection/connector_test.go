package sshconnection

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/signoredellarete/bashes/internal/apperrors"
	"github.com/signoredellarete/bashes/internal/domain"
	"github.com/signoredellarete/bashes/internal/remotessh"
	"github.com/signoredellarete/bashes/internal/sshauth"
)

type fingerprintStore struct{}

func (fingerprintStore) SetResourceHostKeyFingerprint(string, string) error { return nil }

func TestConnectorHostKeyPolicyUsesSavedFingerprintAndKnownHosts(t *testing.T) {
	home := t.TempDir()
	knownHosts := filepath.Join(home, ".ssh", "known_hosts")
	if err := os.MkdirAll(filepath.Dir(knownHosts), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(knownHosts, []byte("host key"), 0o600); err != nil {
		t.Fatal(err)
	}
	connector := NewConnector(sshauth.NewBuilder(), fingerprintStore{})
	connector.homeDir = func() string { return home }

	var accepted string
	policy := connector.hostKeyPolicy(
		domain.Endpoint{HostKeyFingerprint: "SHA256:known"},
		Input{AcceptHostKey: true},
		&accepted,
	)

	if policy.ExpectedFingerprint != "SHA256:known" || !policy.AcceptNewHostKey {
		t.Fatalf("policy = %+v", policy)
	}
	if policy.KnownHostsPath != knownHosts || policy.AcceptedFingerprint == nil {
		t.Fatalf("policy known hosts = %+v", policy)
	}
}

func TestConnectorHostKeyPolicySupportsExplicitInsecureMode(t *testing.T) {
	policy := HostKeyPolicy(domain.Endpoint{}, Input{TrustHostKey: true}, nil)
	if !policy.InsecureIgnoreHostKey {
		t.Fatalf("policy = %+v, want insecure mode", policy)
	}
}

func TestNormalizeErrorPreservesStructuredUnknownHostKeyDetails(t *testing.T) {
	resource := domain.Endpoint{ID: "host-1", IP: "192.0.2.10", Port: 2222}
	err := NormalizeError(remotessh.UnknownHostKeyError{Fingerprint: "SHA256:new"}, resource)
	payload, ok := apperrors.Decode(err)
	if !ok || payload.Code != "ssh_host_key_unknown" {
		t.Fatalf("NormalizeError() = %v", err)
	}
	if payload.Details["resource"] != resource.ID || payload.Details["host"] != "192.0.2.10:2222" {
		t.Fatalf("details = %+v", payload.Details)
	}
}

func TestNormalizeErrorLeavesUnrelatedErrorsUntouched(t *testing.T) {
	want := errors.New("connection reset")
	if got := NormalizeError(want, domain.Endpoint{}); got != want {
		t.Fatalf("NormalizeError() = %v, want original error", got)
	}
}
