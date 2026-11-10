package sshkeyinstall

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/signoredellarete/bashes/internal/domain"
	"github.com/signoredellarete/bashes/internal/remotessh"
	"github.com/signoredellarete/bashes/internal/sshauth"
	"github.com/signoredellarete/bashes/internal/sshconnection"
	"github.com/signoredellarete/bashes/internal/sshkeys"
	"golang.org/x/crypto/ssh"
)

type ResourceStore interface {
	Endpoint(string) (domain.Endpoint, error)
	SetResourceAuth(string, domain.Auth) error
}

type ManagedKeys interface {
	ReadManagedPublic(string) (string, error)
}

type Input struct {
	ResourceID           string
	KeyName              string
	PublicKeyPath        string
	Password             string
	PrivateKeyPath       string
	PrivateKeyPassphrase string
	TrustHostKey         bool
	AcceptHostKey        bool
	ReplaceHostKey       bool
}

type Service struct {
	resources ResourceStore
	keys      ManagedKeys
	dial      func(domain.Endpoint, sshconnection.Input, time.Duration) (*ssh.Client, error)
	run       func(*ssh.Client, string) ([]byte, error)
	close     func(*ssh.Client) error
}

func NewService(resources ResourceStore, keys ManagedKeys, connector *sshconnection.Connector) *Service {
	return &Service{
		resources: resources,
		keys:      keys,
		dial:      connector.Dial,
		run:       runCommand,
		close:     func(client *ssh.Client) error { return client.Close() },
	}
}

func (s *Service) Install(input Input) error {
	publicKey, err := s.installablePublicKey(input)
	if err != nil {
		return err
	}
	resource, err := s.resources.Endpoint(input.ResourceID)
	if err != nil {
		return err
	}
	client, err := s.dial(resource, sshconnection.Input{
		Authentication: sshauth.Input{
			Password:       input.Password,
			PrivateKeyPath: input.PrivateKeyPath,
			Passphrase:     input.PrivateKeyPassphrase,
		},
		TrustHostKey:   input.TrustHostKey,
		AcceptHostKey:  input.AcceptHostKey,
		ReplaceHostKey: input.ReplaceHostKey,
	}, remotessh.DefaultTimeout)
	if err != nil {
		return err
	}
	defer s.close(client)

	output, err := s.run(client, Command(strings.TrimSpace(publicKey)))
	if err != nil {
		return fmt.Errorf("install ssh key: %w: %s", err, strings.TrimSpace(string(output)))
	}
	if strings.TrimSpace(input.PrivateKeyPath) != "" {
		return s.resources.SetResourceAuth(input.ResourceID, domain.Auth{
			Method:         domain.AuthMethodPath,
			PrivateKeyPath: sshkeys.ExpandPath(input.PrivateKeyPath),
		})
	}
	return s.resources.SetResourceAuth(input.ResourceID, domain.Auth{
		Method:  domain.AuthMethodKey,
		KeyName: sshkeys.SanitizeName(input.KeyName),
	})
}

func (s *Service) installablePublicKey(input Input) (string, error) {
	if strings.TrimSpace(input.PublicKeyPath) != "" {
		return sshkeys.ReadPublicPath(input.PublicKeyPath)
	}
	if strings.TrimSpace(input.PrivateKeyPath) != "" {
		return sshkeys.ReadPublicPath(input.PrivateKeyPath)
	}
	if strings.TrimSpace(input.KeyName) != "" {
		return s.keys.ReadManagedPublic(input.KeyName)
	}
	return "", errors.New("select or generate an SSH key before installing it")
}

func Command(authorizedKey string) string {
	quotedKey := ShellQuote(authorizedKey)
	return fmt.Sprintf(
		"mkdir -p ~/.ssh && chmod 700 ~/.ssh && touch ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys && { grep -qxF %s ~/.ssh/authorized_keys || printf '%%s\\n' %s >> ~/.ssh/authorized_keys; }",
		quotedKey,
		quotedKey,
	)
}

func ShellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func runCommand(client *ssh.Client, command string) ([]byte, error) {
	session, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("create ssh session: %w", err)
	}
	defer session.Close()
	return session.CombinedOutput(command)
}
