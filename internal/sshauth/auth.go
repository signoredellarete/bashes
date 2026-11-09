package sshauth

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kayrus/putty"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

type Input struct {
	Password       string
	PrivateKeyPath string
	KeyName        string
	Passphrase     string
}

type Builder struct {
	homeDir  func() string
	readFile func(string) ([]byte, error)
	getenv   func(string) string
	dial     func(string, string) (net.Conn, error)
}

func NewBuilder() *Builder {
	return &Builder{
		homeDir:  userHomeDir,
		readFile: os.ReadFile,
		getenv:   os.Getenv,
		dial:     net.Dial,
	}
}

func (b *Builder) Methods(input Input) ([]ssh.AuthMethod, net.Conn, error) {
	var methods []ssh.AuthMethod

	keyPath := strings.TrimSpace(input.PrivateKeyPath)
	if keyPath == "" && strings.TrimSpace(input.KeyName) != "" {
		return nil, nil, fmt.Errorf("ssh key %q was not resolved to a private key path", sanitizeKeyName(input.KeyName))
	}

	if keyPath != "" {
		method, err := b.privateKeyAuth(keyPath, input.Passphrase)
		if err != nil {
			return nil, nil, err
		}
		methods = append(methods, method)
	} else {
		methods = append(methods, b.defaultPrivateKeyAuthMethods(input.Passphrase)...)
	}

	agentMethod, agentConn := b.agentAuthMethod()
	if agentMethod != nil {
		methods = append(methods, agentMethod)
	}

	if strings.TrimSpace(input.Password) == "" && len(methods) == 0 {
		return nil, nil, errors.New("no SSH authentication method available: enter a password or configure SSH agent/private keys")
	}
	return methods, agentConn, nil
}

func (b *Builder) privateKeyAuth(path string, passphrase string) (ssh.AuthMethod, error) {
	key, err := b.readFile(expandHome(path, b.homeDir()))
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}
	signer, err := ParsePrivateKey(key, passphrase)
	if err != nil {
		return nil, err
	}
	return ssh.PublicKeys(signer), nil
}

func (b *Builder) defaultPrivateKeyAuthMethods(passphrase string) []ssh.AuthMethod {
	home := b.homeDir()
	if strings.TrimSpace(home) == "" {
		return nil
	}

	var signers []ssh.Signer
	for _, name := range []string{"id_ed25519", "id_ecdsa", "id_rsa"} {
		key, err := b.readFile(filepath.Join(home, ".ssh", name))
		if err != nil {
			continue
		}
		signer, err := ParsePrivateKey(key, passphrase)
		if err == nil {
			signers = append(signers, signer)
		}
	}
	if len(signers) == 0 {
		return nil
	}
	return []ssh.AuthMethod{ssh.PublicKeys(signers...)}
}

func (b *Builder) agentAuthMethod() (ssh.AuthMethod, net.Conn) {
	socket := strings.TrimSpace(b.getenv("SSH_AUTH_SOCK"))
	if socket == "" {
		return nil, nil
	}
	conn, err := b.dial("unix", socket)
	if err != nil {
		return nil, nil
	}
	return ssh.PublicKeysCallback(agent.NewClient(conn).Signers), conn
}

func ParsePrivateKey(key []byte, passphrase string) (ssh.Signer, error) {
	if IsPuttyPrivateKey(key) {
		return parsePuttyPrivateKey(key, passphrase)
	}
	if IsPublicKey(key) {
		return nil, errors.New("parse private key: the selected file is an SSH public key; choose the matching private key file without the .pub extension")
	}

	if strings.TrimSpace(passphrase) != "" {
		signer, err := ssh.ParsePrivateKeyWithPassphrase(key, []byte(passphrase))
		if err != nil {
			return nil, fmt.Errorf("parse encrypted private key: %w. Choose an OpenSSH/PEM private key or PuTTY PPK private key, not the .pub file", err)
		}
		return signer, nil
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w. Choose an OpenSSH/PEM private key or PuTTY PPK private key, not the .pub file", err)
	}
	return signer, nil
}

func PublicKeyFromPrivateData(privateData []byte) (ssh.PublicKey, bool, error) {
	if IsPuttyPrivateKey(privateData) {
		key, err := putty.New(privateData)
		if err != nil {
			return nil, false, fmt.Errorf("parse PuTTY PPK private key: %w", err)
		}
		rawPublicKey, err := key.ParseRawPublicKey()
		if err != nil {
			return nil, false, fmt.Errorf("parse PuTTY PPK public key: %w", err)
		}
		publicKey, err := ssh.NewPublicKey(rawPublicKey)
		if err != nil {
			return nil, false, fmt.Errorf("create public key from PuTTY PPK: %w", err)
		}
		return publicKey, true, nil
	}

	signer, err := ssh.ParsePrivateKey(privateData)
	if err == nil {
		return signer.PublicKey(), true, nil
	}
	var passphraseMissing *ssh.PassphraseMissingError
	if errors.As(err, &passphraseMissing) {
		if passphraseMissing.PublicKey != nil {
			return passphraseMissing.PublicKey, true, nil
		}
		return nil, false, nil
	}
	return nil, false, fmt.Errorf("parse private key: %w", err)
}

func IsPuttyPrivateKey(key []byte) bool {
	normalized := bytes.TrimSpace(bytes.TrimPrefix(key, []byte{0xef, 0xbb, 0xbf}))
	return bytes.HasPrefix(normalized, []byte("PuTTY-User-Key-File-"))
}

func IsPublicKey(key []byte) bool {
	normalized := strings.TrimSpace(string(bytes.TrimPrefix(key, []byte{0xef, 0xbb, 0xbf})))
	return strings.HasPrefix(normalized, "ssh-rsa ") ||
		strings.HasPrefix(normalized, "ssh-ed25519 ") ||
		strings.HasPrefix(normalized, "ecdsa-sha2-") ||
		strings.HasPrefix(normalized, "sk-ssh-ed25519@openssh.com ") ||
		strings.HasPrefix(normalized, "sk-ecdsa-sha2-nistp256@openssh.com ")
}

func parsePuttyPrivateKey(key []byte, passphrase string) (ssh.Signer, error) {
	puttyKey, err := putty.New(key)
	if err != nil {
		return nil, fmt.Errorf("parse PuTTY PPK private key: %w", err)
	}
	rawKey, err := puttyKey.ParseRawPrivateKey([]byte(passphrase))
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "expecting password") {
			return nil, errors.New("parse PuTTY PPK private key: key is encrypted; enter the key passphrase")
		}
		return nil, fmt.Errorf("parse PuTTY PPK private key: %w", err)
	}
	signer, err := ssh.NewSignerFromKey(rawKey)
	if err != nil {
		return nil, fmt.Errorf("create signer from PuTTY PPK private key: %w", err)
	}
	return signer, nil
}

func expandHome(path string, home string) string {
	path = strings.TrimSpace(path)
	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, strings.TrimPrefix(path, "~/"))
	}
	if strings.HasPrefix(path, `~\`) {
		return filepath.Join(home, strings.TrimPrefix(path, `~\`))
	}
	return path
}

func userHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}

func sanitizeKeyName(name string) string {
	name = strings.TrimSpace(name)
	name = regexp.MustCompile(`[^A-Za-z0-9_.-]+`).ReplaceAllString(name, "-")
	return strings.Trim(name, ".-")
}
