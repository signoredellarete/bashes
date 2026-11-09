package sshconnection

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/signoredellarete/bashes/internal/apperrors"
	"github.com/signoredellarete/bashes/internal/domain"
	"github.com/signoredellarete/bashes/internal/remotessh"
	"github.com/signoredellarete/bashes/internal/sshauth"
	"golang.org/x/crypto/ssh"
)

type FingerprintStore interface {
	SetResourceHostKeyFingerprint(string, string) error
}

type Input struct {
	Authentication sshauth.Input
	TrustHostKey   bool
	AcceptHostKey  bool
	ReplaceHostKey bool
}

type Connector struct {
	auth    *sshauth.Builder
	store   FingerprintStore
	homeDir func() string
	stat    func(string) (os.FileInfo, error)
	dial    func(context.Context, remotessh.ClientOptions) (*ssh.Client, error)
}

func NewConnector(auth *sshauth.Builder, store FingerprintStore) *Connector {
	return &Connector{
		auth:    auth,
		store:   store,
		homeDir: userHomeDir,
		stat:    os.Stat,
		dial:    remotessh.Dial,
	}
}

func (c *Connector) Dial(resource domain.Endpoint, input Input, timeout time.Duration) (*ssh.Client, error) {
	authMethods, agentConn, err := c.auth.Methods(input.Authentication)
	if err != nil {
		return nil, err
	}
	if agentConn != nil {
		defer agentConn.Close()
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var acceptedFingerprint string
	client, err := c.dial(ctx, remotessh.ClientOptions{
		Target: remotessh.Target{
			Host: Host(resource),
			Port: resource.Port,
			User: resource.User,
		},
		Credentials: remotessh.Credentials{
			Password:    input.Authentication.Password,
			AuthMethods: authMethods,
		},
		HostKeyPolicy: c.hostKeyPolicy(resource, input, &acceptedFingerprint),
	})
	if err != nil {
		return nil, NormalizeError(err, resource)
	}
	if acceptedFingerprint != "" {
		if err := c.store.SetResourceHostKeyFingerprint(resource.ID, acceptedFingerprint); err != nil {
			client.Close()
			return nil, fmt.Errorf("save host key fingerprint: %w", err)
		}
	}
	return client, nil
}

func Host(resource domain.Endpoint) string {
	host := strings.TrimSpace(resource.IP)
	if host == "" {
		host = strings.TrimSpace(resource.Hostname)
	}
	return host
}

func HostKeyPolicy(resource domain.Endpoint, input Input, acceptedFingerprint *string) remotessh.HostKeyPolicy {
	return hostKeyPolicy(resource, input, acceptedFingerprint, userHomeDir(), os.Stat)
}

func (c *Connector) hostKeyPolicy(resource domain.Endpoint, input Input, acceptedFingerprint *string) remotessh.HostKeyPolicy {
	return hostKeyPolicy(resource, input, acceptedFingerprint, c.homeDir(), c.stat)
}

func hostKeyPolicy(
	resource domain.Endpoint,
	input Input,
	acceptedFingerprint *string,
	home string,
	stat func(string) (os.FileInfo, error),
) remotessh.HostKeyPolicy {
	if input.TrustHostKey {
		return remotessh.HostKeyPolicy{InsecureIgnoreHostKey: true}
	}

	policy := remotessh.HostKeyPolicy{
		ExpectedFingerprint:  strings.TrimSpace(resource.HostKeyFingerprint),
		AcceptNewHostKey:     input.AcceptHostKey,
		AcceptChangedHostKey: input.ReplaceHostKey,
		AcceptedFingerprint:  acceptedFingerprint,
	}
	knownHosts := filepath.Join(home, ".ssh", "known_hosts")
	if _, err := stat(knownHosts); err == nil {
		policy.KnownHostsPath = knownHosts
	}
	return policy
}

func NormalizeError(err error, resource domain.Endpoint) error {
	var unknown remotessh.UnknownHostKeyError
	if errors.As(err, &unknown) {
		host := unknown.Host
		if strings.TrimSpace(host) == "" {
			host = net.JoinHostPort(Host(resource), strconv.Itoa(resource.Port))
		}
		return apperrors.New("ssh_host_key_unknown", "SSH host key is unknown", map[string]string{
			"resource":    resource.ID,
			"host":        host,
			"fingerprint": unknown.Fingerprint,
		})
	}

	var mismatch remotessh.HostKeyMismatchError
	if errors.As(err, &mismatch) {
		return apperrors.New("ssh_host_key_mismatch", "SSH host key does not match", map[string]string{
			"resource": resource.ID,
			"host":     mismatch.Host,
			"expected": mismatch.ExpectedFingerprint,
			"actual":   mismatch.ActualFingerprint,
		})
	}
	return err
}

func userHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}
