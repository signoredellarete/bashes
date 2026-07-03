package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"regexp"
	goruntime "runtime"
	"strings"
	"sync"
	"time"

	"github.com/signoredellarete/bashes/internal/application"
	"github.com/signoredellarete/bashes/internal/domain"
	"github.com/signoredellarete/bashes/internal/remotessh"
	"github.com/signoredellarete/bashes/internal/store"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

type App struct {
	ctx       context.Context
	service   *application.Service
	dataPath  string
	mu        sync.Mutex
	sessions  map[string]*sshSession
	tunnels   map[string]*sshTunnel
	transfers map[string]*fileTransferSession
}

func NewApp(dataPath string) *App {
	if dataPath == "" {
		dataPath = defaultDataPath()
	}

	return &App{
		dataPath:  dataPath,
		service:   application.NewService(store.NewRepository(dataPath)),
		sessions:  make(map[string]*sshSession),
		tunnels:   make(map[string]*sshTunnel),
		transfers: make(map[string]*fileTransferSession),
	}
}

func defaultDataPath() string {
	return filepath.Join(defaultDataDir(), "hosts.json")
}

func defaultDataDir() string {
	return dataDirForOS(goruntime.GOOS, userHomeDir(), getenv)
}

func dataDirForOS(goos string, home string, env func(string) string) string {
	switch goos {
	case "darwin":
		if home != "" {
			return filepath.Join(home, "Library", "Application Support", "Bashes")
		}
	case "windows":
		if appData := strings.TrimSpace(env("APPDATA")); appData != "" {
			return filepath.Join(appData, "Bashes")
		}
	default:
		if xdgData := strings.TrimSpace(env("XDG_DATA_HOME")); xdgData != "" {
			return filepath.Join(xdgData, "bashes")
		}
		if home != "" {
			return filepath.Join(home, ".local", "share", "bashes")
		}
	}

	if configDir, err := os.UserConfigDir(); err == nil && strings.TrimSpace(configDir) != "" {
		return filepath.Join(configDir, "Bashes")
	}
	return filepath.Join("data")
}

func getenv(name string) string {
	return os.Getenv(name)
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

func (a *App) UpdateResource(id string, input application.EndpointInput) error {
	return a.service.UpdateResource(id, input)
}

func (a *App) DeleteResource(id string) error {
	resourceIDs, err := a.resourceIDsForDelete(id)
	if err != nil {
		return err
	}
	for _, resourceID := range resourceIDs {
		a.stopTunnelsForResource(resourceID)
	}
	return a.service.DeleteResource(id)
}

type SSHKeyInfo struct {
	Name       string `json:"name"`
	PrivateKey string `json:"privateKey"`
	PublicKey  string `json:"publicKey"`
}

type GenerateSSHKeyInput struct {
	Name string `json:"name"`
}

type InstallSSHKeyInput struct {
	ResourceID           string `json:"resourceId"`
	KeyName              string `json:"keyName"`
	Password             string `json:"password,omitempty"`
	PrivateKeyPath       string `json:"privateKeyPath,omitempty"`
	PrivateKeyPassphrase string `json:"privateKeyPassphrase,omitempty"`
	TrustHostKey         bool   `json:"trustHostKey"`
}

func (a *App) ListSSHKeys() ([]SSHKeyInfo, error) {
	dir := a.keysDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []SSHKeyInfo{}, nil
		}
		return nil, err
	}

	keys := []SSHKeyInfo{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".pub") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".pub")
		keys = append(keys, SSHKeyInfo{
			Name:       name,
			PrivateKey: filepath.Join(dir, name),
			PublicKey:  filepath.Join(dir, entry.Name()),
		})
	}
	return keys, nil
}

func (a *App) GenerateSSHKey(input GenerateSSHKeyInput) (SSHKeyInfo, error) {
	name := sanitizeKeyName(input.Name)
	if name == "" {
		name = "bashes"
	}

	dir := a.keysDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return SSHKeyInfo{}, err
	}

	privatePath := filepath.Join(dir, name)
	publicPath := privatePath + ".pub"
	if _, err := os.Stat(privatePath); err == nil {
		return SSHKeyInfo{}, fmt.Errorf("ssh key %q already exists", name)
	}

	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return SSHKeyInfo{}, fmt.Errorf("generate ed25519 key: %w", err)
	}

	block, err := ssh.MarshalPrivateKey(privateKey, fmt.Sprintf("bashes:%s", name))
	if err != nil {
		return SSHKeyInfo{}, fmt.Errorf("marshal private key: %w", err)
	}
	if err := os.WriteFile(privatePath, pem.EncodeToMemory(block), 0o600); err != nil {
		return SSHKeyInfo{}, err
	}

	sshPublicKey, err := ssh.NewPublicKey(publicKey)
	if err != nil {
		return SSHKeyInfo{}, fmt.Errorf("marshal public key: %w", err)
	}
	if err := os.WriteFile(publicPath, ssh.MarshalAuthorizedKey(sshPublicKey), 0o644); err != nil {
		return SSHKeyInfo{}, err
	}

	return SSHKeyInfo{Name: name, PrivateKey: privatePath, PublicKey: publicPath}, nil
}

func (a *App) ReadSSHPublicKey(name string) (string, error) {
	key, err := os.ReadFile(a.publicKeyPath(name))
	if err != nil {
		return "", err
	}
	return string(key), nil
}

func (a *App) resolveSessionKeyPath(input SSHSessionInput) SSHSessionInput {
	if strings.TrimSpace(input.PrivateKeyPath) == "" && strings.TrimSpace(input.KeyName) != "" {
		input.PrivateKeyPath = filepath.Join(a.keysDir(), sanitizeKeyName(input.KeyName))
	}
	return input
}

func (a *App) InstallSSHKey(input InstallSSHKeyInput) error {
	publicKey, err := a.ReadSSHPublicKey(input.KeyName)
	if err != nil {
		return err
	}
	resource, err := a.resourceByID(input.ResourceID)
	if err != nil {
		return err
	}

	client, err := dialResource(resource, SSHSessionInput{
		Password:             input.Password,
		PrivateKeyPath:       input.PrivateKeyPath,
		PrivateKeyPassphrase: input.PrivateKeyPassphrase,
		TrustHostKey:         input.TrustHostKey,
	}, remotessh.DefaultTimeout)
	if err != nil {
		return err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("create ssh session: %w", err)
	}
	defer session.Close()

	authorizedKey := strings.TrimSpace(publicKey)
	command := fmt.Sprintf(
		"mkdir -p ~/.ssh && chmod 700 ~/.ssh && touch ~/.ssh/authorized_keys && grep -qxF %s ~/.ssh/authorized_keys || printf '%%s\\n' %s >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys",
		shellQuote(authorizedKey),
		shellQuote(authorizedKey),
	)
	output, err := session.CombinedOutput(command)
	if err != nil {
		return fmt.Errorf("install ssh key: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return a.service.SetResourceAuth(input.ResourceID, domain.Auth{
		Method:       domain.AuthMethodKey,
		KeyName:      sanitizeKeyName(input.KeyName),
		TrustHostKey: input.TrustHostKey,
	})
}

type SSHSessionInput struct {
	ResourceID           string `json:"resourceId"`
	Password             string `json:"password,omitempty"`
	PrivateKeyPath       string `json:"privateKeyPath,omitempty"`
	KeyName              string `json:"keyName,omitempty"`
	PrivateKeyPassphrase string `json:"privateKeyPassphrase,omitempty"`
	TrustHostKey         bool   `json:"trustHostKey"`
	Cols                 int    `json:"cols"`
	Rows                 int    `json:"rows"`
}

type SSHTunnelInput struct {
	ResourceID           string `json:"resourceId"`
	Type                 string `json:"type"`
	LocalHost            string `json:"localHost"`
	LocalPort            int    `json:"localPort"`
	RemoteHost           string `json:"remoteHost"`
	RemotePort           int    `json:"remotePort"`
	Password             string `json:"password,omitempty"`
	PrivateKeyPath       string `json:"privateKeyPath,omitempty"`
	KeyName              string `json:"keyName,omitempty"`
	PrivateKeyPassphrase string `json:"privateKeyPassphrase,omitempty"`
	TrustHostKey         bool   `json:"trustHostKey"`
}

type SSHTunnelInfo struct {
	TunnelID      string `json:"tunnelId"`
	ResourceID    string `json:"resourceId"`
	Type          string `json:"type"`
	LocalHost     string `json:"localHost"`
	LocalPort     int    `json:"localPort"`
	LocalAddress  string `json:"localAddress"`
	Target        string `json:"target"`
	ForwardTarget string `json:"forwardTarget"`
	StartedAt     string `json:"startedAt"`
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

type sshTunnel struct {
	id             string
	client         *ssh.Client
	listener       net.Listener
	cancel         context.CancelFunc
	forwardAddress string
	info           SSHTunnelInfo
}

func (a *App) StartSSHSession(input SSHSessionInput) (string, error) {
	resource, err := a.resourceByID(input.ResourceID)
	if err != nil {
		return "", err
	}
	input = applyAuthPreference(resource, input)
	dialInput := a.resolveSessionKeyPath(input)

	client, err := dialResource(resource, dialInput, remotessh.DefaultTimeout)
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
		Message:   fmt.Sprintf("Connected to %s@%s:%d", resource.User, sshHost(resource), resource.Port),
	})
	if auth := authPreferenceFromSessionInput(input); auth != nil {
		if err := a.service.SetResourceAuth(resource.ID, *auth); err != nil {
			a.emit("ssh:status", SSHEvent{
				SessionID: sessionID,
				Message:   fmt.Sprintf("Could not save SSH auth preference: %v", err),
			})
		}
	}

	go a.waitForShell(runtimeCtx, session)

	return sessionID, nil
}

func (a *App) StartSSHTunnel(input SSHTunnelInput) (SSHTunnelInfo, error) {
	resource, err := a.resourceByID(input.ResourceID)
	if err != nil {
		return SSHTunnelInfo{}, err
	}
	if err := normalizeTunnelInput(&input); err != nil {
		return SSHTunnelInfo{}, err
	}

	sessionInput := SSHSessionInput{
		ResourceID:           input.ResourceID,
		Password:             input.Password,
		PrivateKeyPath:       input.PrivateKeyPath,
		KeyName:              input.KeyName,
		PrivateKeyPassphrase: input.PrivateKeyPassphrase,
		TrustHostKey:         input.TrustHostKey,
	}
	sessionInput = applyAuthPreference(resource, sessionInput)
	dialInput := a.resolveSessionKeyPath(sessionInput)

	client, err := dialResource(resource, dialInput, remotessh.DefaultTimeout)
	if err != nil {
		return SSHTunnelInfo{}, err
	}

	listener, forwardAddress, err := a.startTunnelListener(client, input)
	if err != nil {
		client.Close()
		return SSHTunnelInfo{}, err
	}

	tunnelID := fmt.Sprintf("tunnel-%d", time.Now().UnixNano())
	ctx, cancel := context.WithCancel(context.Background())
	info := SSHTunnelInfo{
		TunnelID:      tunnelID,
		ResourceID:    resource.ID,
		Type:          input.Type,
		LocalHost:     input.LocalHost,
		LocalPort:     input.LocalPort,
		LocalAddress:  listener.Addr().String(),
		Target:        fmt.Sprintf("%s@%s:%d", resource.User, sshHost(resource), resource.Port),
		ForwardTarget: tunnelForwardTarget(input, forwardAddress),
		StartedAt:     time.Now().Format(time.RFC3339),
	}
	tunnel := &sshTunnel{
		id:             tunnelID,
		client:         client,
		listener:       listener,
		cancel:         cancel,
		forwardAddress: forwardAddress,
		info:           info,
	}

	a.mu.Lock()
	a.tunnels[tunnelID] = tunnel
	a.mu.Unlock()

	if auth := authPreferenceFromSessionInput(sessionInput); auth != nil {
		if err := a.service.SetResourceAuth(resource.ID, *auth); err != nil {
			a.emit("ssh:status", SSHEvent{SessionID: tunnelID, Message: fmt.Sprintf("Could not save SSH auth preference: %v", err)})
		}
	}

	go a.serveTunnel(ctx, tunnel)

	return info, nil
}

func (a *App) startTunnelListener(client *ssh.Client, input SSHTunnelInput) (net.Listener, string, error) {
	switch input.Type {
	case "socks":
		listener, err := net.Listen("tcp", net.JoinHostPort(input.LocalHost, fmt.Sprint(input.LocalPort)))
		if err != nil {
			return nil, "", fmt.Errorf("start SOCKS tunnel listener: %w", err)
		}
		return listener, "", nil
	case "local":
		listener, err := net.Listen("tcp", net.JoinHostPort(input.LocalHost, fmt.Sprint(input.LocalPort)))
		if err != nil {
			return nil, "", fmt.Errorf("start local tunnel listener: %w", err)
		}
		return listener, net.JoinHostPort(input.RemoteHost, fmt.Sprint(input.RemotePort)), nil
	case "remote":
		listener, err := client.Listen("tcp", net.JoinHostPort(input.LocalHost, fmt.Sprint(input.LocalPort)))
		if err != nil {
			return nil, "", fmt.Errorf("start remote tunnel listener: %w", err)
		}
		return listener, net.JoinHostPort(input.RemoteHost, fmt.Sprint(input.RemotePort)), nil
	default:
		return nil, "", fmt.Errorf("unsupported tunnel type %q", input.Type)
	}
}

func (a *App) ListSSHTunnels() []SSHTunnelInfo {
	a.mu.Lock()
	defer a.mu.Unlock()

	tunnels := make([]SSHTunnelInfo, 0, len(a.tunnels))
	for _, tunnel := range a.tunnels {
		tunnels = append(tunnels, tunnel.info)
	}
	return tunnels
}

func (a *App) StopSSHTunnel(tunnelID string) error {
	a.mu.Lock()
	tunnel := a.tunnels[strings.TrimSpace(tunnelID)]
	delete(a.tunnels, strings.TrimSpace(tunnelID))
	a.mu.Unlock()

	if tunnel == nil {
		return nil
	}
	tunnel.cancel()
	tunnel.listener.Close()
	tunnel.client.Close()
	return nil
}

func (a *App) serveTunnel(ctx context.Context, tunnel *sshTunnel) {
	var err error
	switch tunnel.info.Type {
	case "socks":
		err = remotessh.ServeSOCKS5(ctx, tunnel.listener, tunnel.client)
	case "local":
		err = remotessh.ServeLocalForward(ctx, tunnel.listener, tunnel.client, tunnel.forwardAddress)
	case "remote":
		err = remotessh.ServeRemoteForward(ctx, tunnel.listener, tunnel.forwardAddress)
	default:
		err = fmt.Errorf("unsupported tunnel type %q", tunnel.info.Type)
	}

	a.mu.Lock()
	current := a.tunnels[tunnel.id]
	if current == tunnel {
		delete(a.tunnels, tunnel.id)
	}
	a.mu.Unlock()

	tunnel.cancel()
	tunnel.client.Close()

	if err != nil && a.ctx != nil {
		a.emit("ssh:status", SSHEvent{SessionID: tunnel.id, Message: fmt.Sprintf("Tunnel closed: %v", err)})
	}
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

func (a *App) stopTunnelsForResource(resourceID string) {
	var tunnelIDs []string
	a.mu.Lock()
	for id, tunnel := range a.tunnels {
		if tunnel.info.ResourceID == resourceID {
			tunnelIDs = append(tunnelIDs, id)
		}
	}
	a.mu.Unlock()

	for _, id := range tunnelIDs {
		_ = a.StopSSHTunnel(id)
	}
}

func (a *App) resourceIDsForDelete(id string) ([]string, error) {
	id = strings.TrimSpace(id)
	hosts, err := a.service.ListHosts()
	if err != nil {
		return nil, err
	}
	for _, host := range hosts {
		if host.ID == id {
			ids := []string{host.ID}
			ids = append(ids, nestedResourceIDs(host.Subsystems)...)
			return ids, nil
		}
		if ids := nestedResourceIDsForDelete(host.Subsystems, id); len(ids) > 0 {
			return ids, nil
		}
	}
	return []string{id}, nil
}

func nestedResourceIDs(subsystems []domain.Endpoint) []string {
	var ids []string
	for _, subsystem := range subsystems {
		ids = append(ids, subsystem.ID)
		ids = append(ids, nestedResourceIDs(subsystem.Subsystems)...)
	}
	return ids
}

func nestedResourceIDsForDelete(subsystems []domain.Endpoint, id string) []string {
	for _, subsystem := range subsystems {
		if subsystem.ID == id {
			return append([]string{subsystem.ID}, nestedResourceIDs(subsystem.Subsystems)...)
		}
		if ids := nestedResourceIDsForDelete(subsystem.Subsystems, id); len(ids) > 0 {
			return ids
		}
	}
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

	wasStopped := false
	select {
	case <-ctx.Done():
		wasStopped = true
	default:
	}

	session.cancel()
	session.stdin.Close()
	session.client.Close()

	if wasStopped {
		return
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
				Auth:     host.Auth,
			}, nil
		}
		if subsystem, ok := nestedResourceByID(host.Subsystems, id); ok {
			return subsystem, nil
		}
	}

	return domain.Endpoint{}, fmt.Errorf("resource %q not found", id)
}

func nestedResourceByID(subsystems []domain.Endpoint, id string) (domain.Endpoint, bool) {
	for _, subsystem := range subsystems {
		if subsystem.ID == id {
			return subsystem, true
		}
		if child, ok := nestedResourceByID(subsystem.Subsystems, id); ok {
			return child, true
		}
	}
	return domain.Endpoint{}, false
}

func (a *App) emit(name string, event SSHEvent) {
	if a.ctx == nil {
		return
	}
	wailsruntime.EventsEmit(a.ctx, name, event)
}

type eventWriter struct {
	app       *App
	sessionID string
}

func (w eventWriter) Write(data []byte) (int, error) {
	w.app.emit("ssh:output", SSHEvent{SessionID: w.sessionID, Data: string(data)})
	return len(data), nil
}

func applyAuthPreference(resource domain.Endpoint, input SSHSessionInput) SSHSessionInput {
	if hasExplicitAuth(input) || resource.Auth == nil {
		return input
	}

	auth := resource.Auth
	if auth.TrustHostKey {
		input.TrustHostKey = true
	}
	switch auth.Method {
	case domain.AuthMethodKey:
		input.KeyName = auth.KeyName
	case domain.AuthMethodPath:
		input.PrivateKeyPath = auth.PrivateKeyPath
	case domain.AuthMethodPassword, domain.AuthMethodAgent:
	}
	return input
}

func hasExplicitAuth(input SSHSessionInput) bool {
	return strings.TrimSpace(input.Password) != "" ||
		strings.TrimSpace(input.KeyName) != "" ||
		strings.TrimSpace(input.PrivateKeyPath) != ""
}

func authPreferenceFromSessionInput(input SSHSessionInput) *domain.Auth {
	auth := domain.Auth{TrustHostKey: input.TrustHostKey}
	switch {
	case strings.TrimSpace(input.KeyName) != "":
		auth.Method = domain.AuthMethodKey
		auth.KeyName = sanitizeKeyName(input.KeyName)
	case strings.TrimSpace(input.PrivateKeyPath) != "":
		auth.Method = domain.AuthMethodPath
		auth.PrivateKeyPath = strings.TrimSpace(input.PrivateKeyPath)
	case strings.TrimSpace(input.Password) != "":
		auth.Method = domain.AuthMethodPassword
	default:
		auth.Method = domain.AuthMethodAgent
	}
	return &auth
}

func normalizeTunnelInput(input *SSHTunnelInput) error {
	input.Type = strings.ToLower(strings.TrimSpace(input.Type))
	if input.Type == "" {
		input.Type = "socks"
	}
	if input.Type != "socks" && input.Type != "local" && input.Type != "remote" {
		return fmt.Errorf("unsupported tunnel type %q", input.Type)
	}

	input.LocalHost = strings.TrimSpace(input.LocalHost)
	if input.LocalHost == "" {
		input.LocalHost = "127.0.0.1"
	}
	if invalidTunnelHost(input.LocalHost) {
		return errors.New("local bind address contains invalid characters")
	}
	if input.LocalPort < 1 || input.LocalPort > 65535 {
		return errors.New("local port must be between 1 and 65535")
	}
	if input.Type == "socks" {
		return nil
	}

	input.RemoteHost = strings.TrimSpace(input.RemoteHost)
	if input.RemoteHost == "" {
		input.RemoteHost = "127.0.0.1"
	}
	if invalidTunnelHost(input.RemoteHost) {
		return errors.New("forward target address contains invalid characters")
	}
	if input.RemotePort < 1 || input.RemotePort > 65535 {
		return errors.New("forward target port must be between 1 and 65535")
	}
	return nil
}

func invalidTunnelHost(host string) bool {
	return strings.ContainsAny(host, "\r\n\t ")
}

func tunnelForwardTarget(input SSHTunnelInput, forwardAddress string) string {
	switch input.Type {
	case "socks":
		return "dynamic"
	case "local":
		return forwardAddress
	case "remote":
		return forwardAddress
	default:
		return ""
	}
}

func dialResource(resource domain.Endpoint, input SSHSessionInput, timeout time.Duration) (*ssh.Client, error) {
	authMethods, agentConn, err := authMethods(input)
	if err != nil {
		return nil, err
	}
	if agentConn != nil {
		defer agentConn.Close()
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return remotessh.Dial(ctx, remotessh.ClientOptions{
		Target: remotessh.Target{
			Host: sshHost(resource),
			Port: resource.Port,
			User: resource.User,
		},
		Credentials: remotessh.Credentials{
			Password:    input.Password,
			AuthMethods: authMethods,
		},
		HostKeyPolicy: hostKeyPolicy(input.TrustHostKey),
	})
}

func sshHost(resource domain.Endpoint) string {
	host := strings.TrimSpace(resource.IP)
	if host == "" {
		host = strings.TrimSpace(resource.Hostname)
	}
	return host
}

func authMethods(input SSHSessionInput) ([]ssh.AuthMethod, net.Conn, error) {
	var methods []ssh.AuthMethod

	keyPath := strings.TrimSpace(input.PrivateKeyPath)
	if keyPath == "" && strings.TrimSpace(input.KeyName) != "" {
		keyPath = filepath.Join("data", "keys", sanitizeKeyName(input.KeyName))
	}

	if keyPath != "" {
		method, err := privateKeyAuth(keyPath, input.PrivateKeyPassphrase)
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

func (a *App) keysDir() string {
	return filepath.Join(filepath.Dir(a.dataPath), "keys")
}

func (a *App) publicKeyPath(name string) string {
	return filepath.Join(a.keysDir(), sanitizeKeyName(name)+".pub")
}

func sanitizeKeyName(name string) string {
	name = strings.TrimSpace(name)
	name = regexp.MustCompile(`[^A-Za-z0-9_.-]+`).ReplaceAllString(name, "-")
	return strings.Trim(name, ".-")
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
