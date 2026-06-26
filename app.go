package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/signoredellarete/bashes/internal/application"
	"github.com/signoredellarete/bashes/internal/domain"
	"github.com/signoredellarete/bashes/internal/remotessh"
	"github.com/signoredellarete/bashes/internal/store"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

type App struct {
	ctx      context.Context
	service  *application.Service
	mu       sync.Mutex
	sessions map[string]*sshSession
}

func NewApp(dataPath string) *App {
	if dataPath == "" {
		dataPath = filepath.Join("data", "hosts.json")
	}

	return &App{
		service:  application.NewService(store.NewRepository(dataPath)),
		sessions: make(map[string]*sshSession),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) ListHosts() ([]domain.Host, error) {
	return a.service.ListHosts()
}

func (a *App) AddHost(input application.EndpointInput) (domain.Host, error) {
	return a.service.AddHost(input)
}

func (a *App) AddSubsystem(hostID string, input application.EndpointInput) (domain.Endpoint, error) {
	return a.service.AddSubsystem(hostID, input)
}

func (a *App) DeleteResource(id string) error {
	return a.service.DeleteResource(id)
}

type SSHSessionInput struct {
	ResourceID           string `json:"resourceId"`
	Password             string `json:"password,omitempty"`
	PrivateKeyPath       string `json:"privateKeyPath,omitempty"`
	PrivateKeyPassphrase string `json:"privateKeyPassphrase,omitempty"`
	TrustHostKey         bool   `json:"trustHostKey"`
	Cols                 int    `json:"cols"`
	Rows                 int    `json:"rows"`
}

type SSHEvent struct {
	SessionID string `json:"sessionId"`
	Data      string `json:"data,omitempty"`
	Message   string `json:"message,omitempty"`
}

type sshSession struct {
	id     string
	client *ssh.Client
	shell  *remotessh.Shell
	stdin  *io.PipeWriter
	cancel context.CancelFunc
}

func (a *App) StartSSHSession(input SSHSessionInput) (string, error) {
	resource, err := a.resourceByID(input.ResourceID)
	if err != nil {
		return "", err
	}

	targetHost := strings.TrimSpace(resource.IP)
	if targetHost == "" {
		targetHost = strings.TrimSpace(resource.Hostname)
	}

	authMethods, agentConn, err := authMethods(input)
	if err != nil {
		return "", err
	}
	if agentConn != nil {
		defer agentConn.Close()
	}

	ctx, cancel := context.WithTimeout(context.Background(), remotessh.DefaultTimeout)
	client, err := remotessh.Dial(ctx, remotessh.ClientOptions{
		Target: remotessh.Target{
			Host: targetHost,
			Port: resource.Port,
			User: resource.User,
		},
		Credentials: remotessh.Credentials{
			Password:    input.Password,
			AuthMethods: authMethods,
		},
		HostKeyPolicy: hostKeyPolicy(input.TrustHostKey),
	})
	cancel()
	if err != nil {
		return "", err
	}

	stdinReader, stdinWriter := io.Pipe()
	sessionID := fmt.Sprintf("ssh-%d", time.Now().UnixNano())
	shell, err := remotessh.StartShell(client, remotessh.ShellOptions{
		Size:   remotessh.TerminalSize{Cols: input.Cols, Rows: input.Rows},
		Stdin:  stdinReader,
		Stdout: eventWriter{app: a, sessionID: sessionID},
		Stderr: eventWriter{app: a, sessionID: sessionID},
	})
	if err != nil {
		stdinWriter.Close()
		client.Close()
		return "", err
	}

	runtimeCtx, runtimeCancel := context.WithCancel(context.Background())
	session := &sshSession{
		id:     sessionID,
		client: client,
		shell:  shell,
		stdin:  stdinWriter,
		cancel: runtimeCancel,
	}

	a.mu.Lock()
	a.sessions[sessionID] = session
	a.mu.Unlock()

	a.emit("ssh:status", SSHEvent{
		SessionID: sessionID,
		Message:   fmt.Sprintf("Connected to %s@%s:%d", resource.User, targetHost, resource.Port),
	})

	go a.waitForShell(runtimeCtx, session)

	return sessionID, nil
}

func (a *App) WriteSSHSession(sessionID string, data string) error {
	session, err := a.session(sessionID)
	if err != nil {
		return err
	}
	_, err = io.WriteString(session.stdin, data)
	return err
}

func (a *App) ResizeSSHSession(sessionID string, cols int, rows int) error {
	session, err := a.session(sessionID)
	if err != nil {
		return err
	}
	return session.shell.Resize(remotessh.TerminalSize{Cols: cols, Rows: rows})
}

func (a *App) StopSSHSession(sessionID string) error {
	a.mu.Lock()
	session := a.sessions[sessionID]
	delete(a.sessions, sessionID)
	a.mu.Unlock()

	if session == nil {
		return nil
	}
	session.cancel()
	session.stdin.Close()
	session.shell.Close()
	session.client.Close()
	a.emit("ssh:closed", SSHEvent{SessionID: sessionID, Message: "SSH session closed"})
	return nil
}

func (a *App) waitForShell(ctx context.Context, session *sshSession) {
	err := session.shell.Wait()

	a.mu.Lock()
	current := a.sessions[session.id]
	if current == session {
		delete(a.sessions, session.id)
	}
	a.mu.Unlock()

	session.cancel()
	session.stdin.Close()
	session.client.Close()

	select {
	case <-ctx.Done():
		return
	default:
	}

	if err != nil && !errors.Is(err, io.EOF) {
		a.emit("ssh:closed", SSHEvent{SessionID: session.id, Message: err.Error()})
		return
	}
	a.emit("ssh:closed", SSHEvent{SessionID: session.id, Message: "SSH session closed"})
}

func (a *App) session(sessionID string) (*sshSession, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	session := a.sessions[strings.TrimSpace(sessionID)]
	if session == nil {
		return nil, fmt.Errorf("ssh session %q not found", sessionID)
	}
	return session, nil
}

func (a *App) resourceByID(id string) (domain.Endpoint, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.Endpoint{}, errors.New("resource id is required")
	}

	hosts, err := a.service.ListHosts()
	if err != nil {
		return domain.Endpoint{}, err
	}

	for _, host := range hosts {
		if host.ID == id {
			return domain.Endpoint{
				ID:       host.ID,
				Type:     domain.ResourceHost,
				Hostname: host.Hostname,
				IP:       host.IP,
				Port:     host.Port,
				User:     host.User,
			}, nil
		}
		for _, subsystem := range host.Subsystems {
			if subsystem.ID == id {
				return subsystem, nil
			}
		}
	}

	return domain.Endpoint{}, fmt.Errorf("resource %q not found", id)
}

func (a *App) emit(name string, event SSHEvent) {
	if a.ctx == nil {
		return
	}
	runtime.EventsEmit(a.ctx, name, event)
}

type eventWriter struct {
	app       *App
	sessionID string
}

func (w eventWriter) Write(data []byte) (int, error) {
	w.app.emit("ssh:output", SSHEvent{SessionID: w.sessionID, Data: string(data)})
	return len(data), nil
}

func authMethods(input SSHSessionInput) ([]ssh.AuthMethod, net.Conn, error) {
	var methods []ssh.AuthMethod

	if strings.TrimSpace(input.PrivateKeyPath) != "" {
		method, err := privateKeyAuth(input.PrivateKeyPath, input.PrivateKeyPassphrase)
		if err != nil {
			return nil, nil, err
		}
		methods = append(methods, method)
	} else {
		methods = append(methods, defaultPrivateKeyAuthMethods(input.PrivateKeyPassphrase)...)
	}

	agentMethod, agentConn := sshAgentAuthMethod()
	if agentMethod != nil {
		methods = append(methods, agentMethod)
	}

	if strings.TrimSpace(input.Password) == "" && len(methods) == 0 {
		return nil, nil, errors.New("no SSH authentication method available: enter a password or configure SSH agent/private keys")
	}

	return methods, agentConn, nil
}

func privateKeyAuth(path string, passphrase string) (ssh.AuthMethod, error) {
	key, err := os.ReadFile(expandHome(path))
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}
	signer, err := parsePrivateKey(key, passphrase)
	if err != nil {
		return nil, err
	}
	return ssh.PublicKeys(signer), nil
}

func defaultPrivateKeyAuthMethods(passphrase string) []ssh.AuthMethod {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	var signers []ssh.Signer
	for _, name := range []string{"id_ed25519", "id_ecdsa", "id_rsa"} {
		key, err := os.ReadFile(filepath.Join(home, ".ssh", name))
		if err != nil {
			continue
		}
		signer, err := parsePrivateKey(key, passphrase)
		if err == nil {
			signers = append(signers, signer)
		}
	}
	if len(signers) == 0 {
		return nil
	}
	return []ssh.AuthMethod{ssh.PublicKeys(signers...)}
}

func parsePrivateKey(key []byte, passphrase string) (ssh.Signer, error) {
	if strings.TrimSpace(passphrase) != "" {
		signer, err := ssh.ParsePrivateKeyWithPassphrase(key, []byte(passphrase))
		if err != nil {
			return nil, fmt.Errorf("parse encrypted private key: %w", err)
		}
		return signer, nil
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	return signer, nil
}

func sshAgentAuthMethod() (ssh.AuthMethod, net.Conn) {
	socket := strings.TrimSpace(os.Getenv("SSH_AUTH_SOCK"))
	if socket == "" {
		return nil, nil
	}

	conn, err := net.Dial("unix", socket)
	if err != nil {
		return nil, nil
	}

	return ssh.PublicKeysCallback(agent.NewClient(conn).Signers), conn
}

func hostKeyPolicy(trustHostKey bool) remotessh.HostKeyPolicy {
	if trustHostKey {
		return remotessh.HostKeyPolicy{InsecureIgnoreHostKey: true}
	}

	knownHosts := filepath.Join(userHomeDir(), ".ssh", "known_hosts")
	if _, err := os.Stat(knownHosts); err == nil {
		return remotessh.HostKeyPolicy{KnownHostsPath: knownHosts}
	}

	return remotessh.HostKeyPolicy{}
}

func expandHome(path string) string {
	path = strings.TrimSpace(path)
	if path == "~" {
		return userHomeDir()
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(userHomeDir(), strings.TrimPrefix(path, "~/"))
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
