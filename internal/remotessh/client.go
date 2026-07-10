package remotessh

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

const DefaultTimeout = 3 * time.Second

type Target struct {
	Host string
	Port int
	User string
}

type Credentials struct {
	Password             string
	PrivateKeyPEM        []byte
	PrivateKeyPassphrase []byte
	AuthMethods          []ssh.AuthMethod
}

type HostKeyPolicy struct {
	KnownHostsPath        string
	InsecureIgnoreHostKey bool
	ExpectedFingerprint   string
	AcceptNewHostKey      bool
	AcceptedFingerprint   *string
}

type TerminalSize struct {
	Cols int
	Rows int
}

type ClientOptions struct {
	Target        Target
	Credentials   Credentials
	HostKeyPolicy HostKeyPolicy
	Timeout       time.Duration
}

func Dial(ctx context.Context, options ClientOptions) (*ssh.Client, error) {
	config, err := NewClientConfig(options)
	if err != nil {
		return nil, err
	}

	dialer := net.Dialer{Timeout: config.Timeout}
	conn, err := dialer.DialContext(ctx, "tcp", Address(options.Target))
	if err != nil {
		return nil, fmt.Errorf("dial ssh target: %w", err)
	}

	sshConn, chans, reqs, err := ssh.NewClientConn(conn, Address(options.Target), config)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("create ssh client: %w", err)
	}

	return ssh.NewClient(sshConn, chans, reqs), nil
}

func NewClientConfig(options ClientOptions) (*ssh.ClientConfig, error) {
	if err := ValidateTarget(options.Target); err != nil {
		return nil, err
	}

	authMethods, err := AuthMethods(options.Credentials)
	if err != nil {
		return nil, err
	}

	hostKeyCallback, err := HostKeyCallback(options.HostKeyPolicy)
	if err != nil {
		return nil, err
	}

	timeout := options.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	return &ssh.ClientConfig{
		User:            strings.TrimSpace(options.Target.User),
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         timeout,
	}, nil
}

func ValidateTarget(target Target) error {
	if strings.TrimSpace(target.Host) == "" {
		return errors.New("ssh host is required")
	}
	if strings.ContainsAny(target.Host, "\r\n\t") {
		return errors.New("ssh host contains invalid characters")
	}
	if target.Port < 1 || target.Port > 65535 {
		return errors.New("ssh port must be between 1 and 65535")
	}
	if strings.TrimSpace(target.User) == "" {
		return errors.New("ssh user is required")
	}
	if strings.ContainsAny(target.User, "\r\n\t ") {
		return errors.New("ssh user contains invalid characters")
	}
	return nil
}

func AuthMethods(credentials Credentials) ([]ssh.AuthMethod, error) {
	var methods []ssh.AuthMethod

	if credentials.Password != "" {
		methods = append(methods, ssh.Password(credentials.Password))
	}

	if len(credentials.PrivateKeyPEM) > 0 {
		signer, err := privateKeySigner(credentials.PrivateKeyPEM, credentials.PrivateKeyPassphrase)
		if err != nil {
			return nil, err
		}
		methods = append(methods, ssh.PublicKeys(signer))
	}

	methods = append(methods, credentials.AuthMethods...)

	if len(methods) == 0 {
		return nil, errors.New("at least one ssh authentication method is required")
	}

	return methods, nil
}

func HostKeyCallback(policy HostKeyPolicy) (ssh.HostKeyCallback, error) {
	if policy.InsecureIgnoreHostKey {
		return ssh.InsecureIgnoreHostKey(), nil
	}

	var knownHostsCallback ssh.HostKeyCallback
	if policy.KnownHostsPath != "" {
		callback, err := knownhosts.New(policy.KnownHostsPath)
		if err != nil {
			return nil, fmt.Errorf("load known_hosts: %w", err)
		}
		knownHostsCallback = callback
	}

	expectedFingerprint := strings.TrimSpace(policy.ExpectedFingerprint)
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		fingerprint := ssh.FingerprintSHA256(key)
		if expectedFingerprint != "" {
			if fingerprint == expectedFingerprint {
				return nil
			}
			return HostKeyMismatchError{
				Host:                hostname,
				ExpectedFingerprint: expectedFingerprint,
				ActualFingerprint:   fingerprint,
			}
		}

		if knownHostsCallback != nil {
			err := knownHostsCallback(hostname, remote, key)
			if err == nil {
				return nil
			}
			if !isUnknownHostKey(err) {
				return err
			}
		}

		if policy.AcceptNewHostKey {
			if policy.AcceptedFingerprint != nil {
				*policy.AcceptedFingerprint = fingerprint
			}
			return nil
		}

		return UnknownHostKeyError{Host: hostname, Fingerprint: fingerprint}
	}, nil
}

type UnknownHostKeyError struct {
	Host        string
	Fingerprint string
}

func (e UnknownHostKeyError) Error() string {
	return fmt.Sprintf("unknown host key for %s: %s", e.Host, e.Fingerprint)
}

type HostKeyMismatchError struct {
	Host                string
	ExpectedFingerprint string
	ActualFingerprint   string
}

func (e HostKeyMismatchError) Error() string {
	return fmt.Sprintf("host key mismatch for %s: expected %s, got %s", e.Host, e.ExpectedFingerprint, e.ActualFingerprint)
}

func isUnknownHostKey(err error) bool {
	var keyErr *knownhosts.KeyError
	if errors.As(err, &keyErr) {
		return len(keyErr.Want) == 0
	}
	return false
}

func Address(target Target) string {
	return net.JoinHostPort(strings.TrimSpace(target.Host), strconv.Itoa(target.Port))
}

func privateKeySigner(keyPEM, passphrase []byte) (ssh.Signer, error) {
	if len(passphrase) > 0 {
		signer, err := ssh.ParsePrivateKeyWithPassphrase(keyPEM, passphrase)
		if err != nil {
			return nil, fmt.Errorf("parse encrypted private key: %w", err)
		}
		return signer, nil
	}

	signer, err := ssh.ParsePrivateKey(keyPEM)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	return signer, nil
}
