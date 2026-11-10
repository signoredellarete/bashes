package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"sync"
	"time"

	"github.com/signoredellarete/bashes/internal/apperrors"
	"github.com/signoredellarete/bashes/internal/application"
	"github.com/signoredellarete/bashes/internal/credentials"
	"github.com/signoredellarete/bashes/internal/database"
	"github.com/signoredellarete/bashes/internal/desktop"
	"github.com/signoredellarete/bashes/internal/domain"
	"github.com/signoredellarete/bashes/internal/importer"
	"github.com/signoredellarete/bashes/internal/localterm"
	"github.com/signoredellarete/bashes/internal/plugin"
	"github.com/signoredellarete/bashes/internal/remotessh"
	appsettings "github.com/signoredellarete/bashes/internal/settings"
	"github.com/signoredellarete/bashes/internal/sshauth"
	"github.com/signoredellarete/bashes/internal/sshconnection"
	"github.com/signoredellarete/bashes/internal/sshkeyinstall"
	"github.com/signoredellarete/bashes/internal/sshkeys"
	"github.com/signoredellarete/bashes/internal/sshtunnel"
	"github.com/signoredellarete/bashes/internal/store"
	"github.com/signoredellarete/bashes/internal/termsession"
	"github.com/signoredellarete/bashes/internal/transfer"
	"github.com/signoredellarete/bashes/internal/webappbrowser"
	"github.com/signoredellarete/bashes/internal/webappruntime"
	"golang.org/x/crypto/ssh"
)

var version = "dev"

const (
	repositoryURL = "https://github.com/signoredellarete/bashes"
	readmeURL     = repositoryURL + "#readme"
	releasesURL   = repositoryURL + "/releases"
)

type App struct {
	ctx              context.Context
	lifecycleMu      sync.RWMutex
	shutdownOnce     sync.Once
	service          *application.Service
	database         *database.Service
	passwords        credentials.Store
	dataPath         string
	sshKeys          *sshkeys.Service
	sshConnector     *sshconnection.Connector
	sshKeyInstaller  *sshkeyinstall.Service
	webAppBrowsers   *webappbrowser.PreferencesService
	sessions         *termsession.Manager
	tunnels          *sshtunnel.Manager
	transfers        *transfer.SessionManager[*fileTransferSession]
	transferJobs     *transfer.Manager
	webAppRuntimes   *webappruntime.Registry[*webAppRuntime]
	plugins          *plugin.Supervisor
	desktopRuntime   desktop.Runtime
	openFile         func(string) error
	openFileCacheDir string
}

func NewApp(dataPath string) *App {
	return newAppWithPasswordStore(dataPath, credentials.NewNativeStore())
}

func newAppWithPasswordStore(dataPath string, passwords credentials.Store) *App {
	if dataPath == "" {
		dataPath = defaultDataPath()
	}

	settingsRepository := appsettings.NewRepository(filepath.Join(filepath.Dir(dataPath), "settings.json"))
	applicationService := application.NewService(store.NewRepository(dataPath))
	databaseService := database.NewService(applicationService)
	sshAuthBuilder := sshauth.NewBuilder()
	sshKeyService := sshkeys.NewService(
		filepath.Join(filepath.Dir(dataPath), "keys"),
		settingsRepository,
		listWSLSSHKeyInventory,
	)
	webAppBrowserService := webappbrowser.NewPreferencesService(goruntime.GOOS, getenv, settingsRepository)
	sshConnector := sshconnection.NewConnector(sshAuthBuilder, applicationService)
	sshKeyInstaller := sshkeyinstall.NewService(applicationService, sshKeyService, sshConnector)

	return &App{
		dataPath:         dataPath,
		service:          applicationService,
		database:         databaseService,
		sshKeys:          sshKeyService,
		sshConnector:     sshConnector,
		sshKeyInstaller:  sshKeyInstaller,
		webAppBrowsers:   webAppBrowserService,
		passwords:        passwords,
		sessions:         termsession.NewManager(),
		tunnels:          sshtunnel.NewManager(),
		transfers:        transfer.NewSessionManager[*fileTransferSession](),
		transferJobs:     transfer.NewManager(),
		webAppRuntimes:   webappruntime.NewRegistry[*webAppRuntime](),
		plugins:          plugin.NewSupervisor(plugin.SupervisorOptions{HostVersion: version}),
		desktopRuntime:   desktop.NewWailsRuntime(),
		openFile:         openFileWithDefaultApplication,
		openFileCacheDir: defaultOpenFileCacheDir(dataPath),
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

func hostsFilePathForOS(goos string, env func(string) string) string {
	if goos == "windows" {
		root := strings.TrimSpace(env("SystemRoot"))
		if root == "" {
			root = strings.TrimSpace(env("WINDIR"))
		}
		if root == "" {
			root = `C:\Windows`
		}
		return filepath.Join(root, "System32", "drivers", "etc", "hosts")
	}
	return "/etc/hosts"
}

func defaultSSHUserForOS(goos string, env func(string) string) string {
	names := []string{"USER", "LOGNAME", "USERNAME"}
	if goos == "windows" {
		names = []string{"USERNAME", "USER", "LOGNAME"}
	}
	for _, name := range names {
		if value := strings.TrimSpace(env(name)); value != "" {
			return value
		}
	}
	if goos == "windows" {
		return "Administrator"
	}
	return "root"
}

func sshConfigPathForOS(goos string, home string) string {
	if goos == "windows" || strings.TrimSpace(home) == "" {
		return ""
	}
	return filepath.Join(home, ".ssh", "config")
}

func getenv(name string) string {
	return os.Getenv(name)
}

func (a *App) startup(ctx context.Context) {
	a.lifecycleMu.Lock()
	defer a.lifecycleMu.Unlock()
	a.ctx = ctx
}

func (a *App) beforeClose(context.Context) bool {
	a.beginShutdown()
	return false
}

func (a *App) shutdown(context.Context) {
	a.beginShutdown()
}

func (a *App) beginShutdown() {
	a.shutdownOnce.Do(func() {
		a.lifecycleMu.Lock()
		a.ctx = nil
		a.lifecycleMu.Unlock()
		a.stopAllRuntimeConnections()
	})
}

func (a *App) ListHosts() ([]domain.Host, error) {
	return a.service.ListHosts()
}

func (a *App) ListWorkspace() (application.Workspace, error) {
	return a.service.ListWorkspace()
}

func (a *App) ListTags() ([]string, error) {
	return a.service.ListTags()
}

func (a *App) CreateTag(name string) (string, error) {
	return a.service.CreateTag(name)
}

func (a *App) RenameTag(currentName, newName string) error {
	return a.service.RenameTag(currentName, newName)
}

func (a *App) DeleteTag(name string) error {
	return a.service.DeleteTag(name)
}

func (a *App) AddHost(input application.EndpointInput) (domain.Host, error) {
	return a.service.AddHost(input)
}

func (a *App) AddWebApp(input application.WebAppInput) (domain.WebApp, error) {
	return a.service.AddWebApp(input)
}

func (a *App) UpdateWebApp(id string, input application.WebAppInput) error {
	if err := a.service.UpdateWebApp(id, input); err != nil {
		return err
	}
	a.stopWebAppForID(id)
	return nil
}

func (a *App) DeleteWebApp(id string) error {
	if err := a.service.DeleteWebApp(id); err != nil {
		return err
	}
	a.stopWebAppForID(id)
	return nil
}

func (a *App) AddSubsystem(hostID string, input application.EndpointInput) (domain.Endpoint, error) {
	return a.service.AddSubsystem(hostID, input)
}

func (a *App) UpdateResource(id string, input application.EndpointInput) error {
	return a.service.UpdateResource(id, input)
}

func (a *App) SetResourceTags(id string, tags []string) error {
	return a.service.SetResourceTags(id, tags)
}

func (a *App) ReorderHosts(order []string) error {
	return a.service.ReorderHosts(order)
}

func (a *App) ReorderTopLevel(order []string) error {
	return a.service.ReorderTopLevel(order)
}

func (a *App) DeleteResource(id string) error {
	resourceIDs, err := a.resourceIDsForDelete(id)
	if err != nil {
		return err
	}
	workspace, err := a.service.ListWorkspace()
	if err != nil {
		return err
	}
	resourceSet := make(map[string]struct{}, len(resourceIDs))
	for _, resourceID := range resourceIDs {
		resourceSet[resourceID] = struct{}{}
	}
	for _, webApp := range workspace.WebApps {
		if _, referenced := resourceSet[webApp.GatewayResourceID]; referenced {
			return fmt.Errorf("resource is used as gateway by web app %q", webApp.Name)
		}
	}
	for _, resourceID := range resourceIDs {
		a.stopSessionsForResource(resourceID)
		a.stopTunnelsForResource(resourceID)
		a.closeTransfersForResource(resourceID)
	}
	if err := a.service.DeleteResource(id); err != nil {
		return err
	}
	for _, resourceID := range resourceIDs {
		_ = a.passwords.DeletePassword(resourceID)
	}
	return nil
}

type SSHKeyInfo = sshkeys.Key

type SSHKeySettings = appsettings.SSHKeys

type WebAppBrowserSettings = appsettings.WebAppBrowser

type SSHKeySourceInfo = sshkeys.Source

type SSHKeyInventory = sshkeys.Inventory

type GenerateSSHKeyInput = sshkeys.GenerateInput

type ImportSSHKeyPairInput = sshkeys.ImportInput

type InstallSSHKeyInput struct {
	ResourceID           string `json:"resourceId"`
	KeyName              string `json:"keyName"`
	PublicKeyPath        string `json:"publicKeyPath,omitempty"`
	Password             string `json:"password,omitempty"`
	PrivateKeyPath       string `json:"privateKeyPath,omitempty"`
	PrivateKeyPassphrase string `json:"privateKeyPassphrase,omitempty"`
	TrustHostKey         bool   `json:"trustHostKey"`
	AcceptHostKey        bool   `json:"acceptHostKey,omitempty"`
	ReplaceHostKey       bool   `json:"replaceHostKey,omitempty"`
}

type AppInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Platform    string `json:"platform"`
	Arch        string `json:"arch"`
	DataPath    string `json:"dataPath"`
	RepoURL     string `json:"repoUrl"`
	ReadmeURL   string `json:"readmeUrl"`
	ReleasesURL string `json:"releasesUrl"`
}

type UpdateInfo struct {
	CurrentVersion  string `json:"currentVersion"`
	LatestVersion   string `json:"latestVersion"`
	UpdateAvailable bool   `json:"updateAvailable"`
	ReleaseURL      string `json:"releaseUrl"`
	RepoURL         string `json:"repoUrl"`
	Message         string `json:"message"`
}

type HostsFileImportResult = importer.HostsFileResult

type HostsFileImportTarget = importer.HostsFileTarget

func (a *App) ExportDatabase(path string) error {
	return a.database.Export(path)
}

func (a *App) ImportFromHostsFile() (HostsFileImportResult, error) {
	return a.importFromHostsFile(
		hostsFilePathForOS(goruntime.GOOS, getenv),
		defaultSSHUserForOS(goruntime.GOOS, getenv),
		sshConfigPathForOS(goruntime.GOOS, userHomeDir()),
	)
}

func (a *App) PreviewImportFromHostsFile() (HostsFileImportResult, error) {
	result, _, err := a.prepareHostsFileImport(
		hostsFilePathForOS(goruntime.GOOS, getenv),
		defaultSSHUserForOS(goruntime.GOOS, getenv),
		sshConfigPathForOS(goruntime.GOOS, userHomeDir()),
	)
	return result, err
}

func (a *App) importFromHostsFile(path string, user string, sshConfigPath string) (HostsFileImportResult, error) {
	return a.database.ImportHostsFile(path, user, sshConfigPath)
}

func (a *App) prepareHostsFileImport(path string, user string, sshConfigPath string) (HostsFileImportResult, domain.Store, error) {
	return a.database.PreviewHostsFile(path, user, sshConfigPath)
}

func (a *App) prepareHostsFileImportWithStore(path string, user string, sshConfigPath string, data domain.Store) (HostsFileImportResult, domain.Store, error) {
	return database.PrepareHostsFile(path, user, sshConfigPath, data)
}

func (a *App) ImportDatabase(path string) error {
	if err := a.database.Import(path); err != nil {
		return err
	}
	a.stopAllRuntimeConnections()
	a.emitData(desktop.EventDatabaseImported, map[string]string{
		"message": "Database imported.",
		"path":    a.dataPath,
	})
	return nil
}

func (a *App) ListSSHKeys() ([]SSHKeyInfo, error) {
	inventory, err := a.GetSSHKeyInventory()
	if err != nil {
		return nil, err
	}
	return inventory.Keys, nil
}

func (a *App) GetSSHKeyInventory() (SSHKeyInventory, error) {
	return a.sshKeys.Inventory()
}

func (a *App) GetSSHKeySettings() (SSHKeySettings, error) {
	return a.sshKeys.Settings()
}

func (a *App) SaveSSHKeySettings(input SSHKeySettings) (SSHKeySettings, error) {
	return a.sshKeys.SaveSettings(input)
}

func (a *App) ListWebAppBrowsers() []webappbrowser.Browser {
	return a.webAppBrowsers.List()
}

func (a *App) GetWebAppBrowserSettings() (WebAppBrowserSettings, error) {
	return a.webAppBrowsers.Load()
}

func (a *App) SaveWebAppBrowserSettings(input WebAppBrowserSettings) (WebAppBrowserSettings, error) {
	return a.webAppBrowsers.Save(input)
}

func (a *App) ChooseSSHKeyDirectory() (string, error) {
	ctx := a.runtimeContext()
	if ctx == nil {
		return "", errors.New("application window is not ready")
	}
	return a.desktopRuntime.OpenDirectoryDialog(ctx, desktop.OpenDialogOptions{
		Title: "Choose SSH Keys Directory",
	})
}

func listSystemSSHKeys() ([]SSHKeyInfo, error) {
	home := userHomeDir()
	if home == "" {
		return []SSHKeyInfo{}, nil
	}
	return listSSHKeysInDirectory(filepath.Join(home, ".ssh"), "system", true)
}

func listSSHKeysInDirectory(dir string, source string, ignoreMissing bool) ([]SSHKeyInfo, error) {
	return sshkeys.ListDirectory(dir, source, ignoreMissing)
}

func (a *App) GenerateSSHKey(input GenerateSSHKeyInput) (SSHKeyInfo, error) {
	return a.sshKeys.Generate(input)
}

func (a *App) ImportSSHKeyPair(input ImportSSHKeyPairInput) (SSHKeyInfo, error) {
	return a.sshKeys.ImportPair(input)
}

func (a *App) DeleteSSHKey(name string) error {
	return a.sshKeys.Delete(name)
}

func (a *App) ReadSSHPublicKey(name string) (string, error) {
	return a.sshKeys.ReadManagedPublic(name)
}

func (a *App) ReadSSHPublicKeyPath(path string) (string, error) {
	return sshkeys.ReadPublicPath(path)
}

func (a *App) ReadSSHPrivateKey(name string) (string, error) {
	return a.sshKeys.ReadManagedPrivate(name)
}

func (a *App) ReadSSHPrivateKeyPath(path string) (string, error) {
	return sshkeys.ReadPrivatePath(path)
}

func (a *App) resolveSessionKeyPath(input SSHSessionInput) SSHSessionInput {
	if strings.TrimSpace(input.PrivateKeyPath) == "" && strings.TrimSpace(input.KeyName) != "" {
		input.PrivateKeyPath = a.sshKeys.ManagedPrivatePath(input.KeyName)
	}
	return input
}

func (a *App) InstallSSHKey(input InstallSSHKeyInput) error {
	return a.sshKeyInstaller.Install(sshKeyInstallInput(input))
}

func installSSHKeyCommand(authorizedKey string) string {
	return sshkeyinstall.Command(authorizedKey)
}

func sshKeyInstallInput(input InstallSSHKeyInput) sshkeyinstall.Input {
	return sshkeyinstall.Input{
		ResourceID:           input.ResourceID,
		KeyName:              input.KeyName,
		PublicKeyPath:        input.PublicKeyPath,
		Password:             input.Password,
		PrivateKeyPath:       input.PrivateKeyPath,
		PrivateKeyPassphrase: input.PrivateKeyPassphrase,
		TrustHostKey:         input.TrustHostKey,
		AcceptHostKey:        input.AcceptHostKey,
		ReplaceHostKey:       input.ReplaceHostKey,
	}
}

type SSHSessionInput struct {
	ResourceID           string `json:"resourceId"`
	Password             string `json:"password,omitempty"`
	PrivateKeyPath       string `json:"privateKeyPath,omitempty"`
	KeyName              string `json:"keyName,omitempty"`
	PrivateKeyPassphrase string `json:"privateKeyPassphrase,omitempty"`
	ManagePassword       bool   `json:"managePassword,omitempty"`
	SavePassword         bool   `json:"savePassword,omitempty"`
	TrustHostKey         bool   `json:"trustHostKey"`
	AcceptHostKey        bool   `json:"acceptHostKey,omitempty"`
	ReplaceHostKey       bool   `json:"replaceHostKey,omitempty"`
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
	ManagePassword       bool   `json:"managePassword,omitempty"`
	SavePassword         bool   `json:"savePassword,omitempty"`
	TrustHostKey         bool   `json:"trustHostKey"`
	AcceptHostKey        bool   `json:"acceptHostKey,omitempty"`
	ReplaceHostKey       bool   `json:"replaceHostKey,omitempty"`
	AllowPublicBind      bool   `json:"allowPublicBind,omitempty"`
}

type SSHTunnelInfo = sshtunnel.Info

type LocalSessionInput struct {
	Cols int `json:"cols"`
	Rows int `json:"rows"`
}

type SSHEvent struct {
	SessionID string `json:"sessionId"`
	Data      string `json:"data,omitempty"`
	Bytes     string `json:"bytes,omitempty"`
	Message   string `json:"message,omitempty"`
}

func (a *App) StartSSHSession(input SSHSessionInput) (string, error) {
	resource, err := a.resourceByID(input.ResourceID)
	if err != nil {
		return "", err
	}
	input, err = a.prepareSessionInput(resource, input)
	if err != nil {
		return "", err
	}
	dialInput := a.resolveSessionKeyPath(input)

	client, err := a.dialResource(resource, dialInput, remotessh.DefaultTimeout)
	if err != nil {
		return "", err
	}
	if err := a.persistPasswordChoice(resource.ID, input); err != nil {
		client.Close()
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

	session, err := a.sessions.Register(termsession.Options{
		ID: sessionID, ResourceID: resource.ID, Kind: termsession.KindSSH,
		Client: client, Shell: shell, Stdin: stdinWriter,
	})
	if err != nil {
		_ = stdinWriter.Close()
		_ = shell.Close()
		_ = client.Close()
		return "", err
	}
	runtimeCtx := a.sessions.Context(session)

	a.emit(desktop.EventSSHStatus, SSHEvent{
		SessionID: sessionID,
		Message:   fmt.Sprintf("Connected to %s@%s:%d", resource.User, sshHost(resource), resource.Port),
	})
	if auth := authPreferenceFromSessionInput(input); auth != nil {
		if err := a.service.SetResourceAuth(resource.ID, *auth); err != nil {
			a.emit(desktop.EventSSHStatus, SSHEvent{
				SessionID: sessionID,
				Message:   fmt.Sprintf("Could not save SSH auth preference: %v", err),
			})
		}
	}

	go remotessh.MonitorKeepAlive(runtimeCtx, client, remotessh.KeepAliveOptions{}, func(err error) {
		_ = shell.Close()
		_ = client.Close()
	})
	go a.waitForShell(sessionID, session)

	return sessionID, nil
}

func (a *App) SupportsLocalShell() bool {
	return goruntime.GOOS != "windows"
}

func (a *App) StartLocalSession(input LocalSessionInput) (string, error) {
	if !a.SupportsLocalShell() {
		return "", errors.New("local shell is not supported on Windows yet")
	}

	stdinReader, stdinWriter := io.Pipe()
	sessionID := fmt.Sprintf("local-%d", time.Now().UnixNano())
	shell, err := localterm.StartShell(localterm.ShellOptions{
		Size:   remotessh.TerminalSize{Cols: input.Cols, Rows: input.Rows},
		Stdin:  stdinReader,
		Stdout: eventWriter{app: a, sessionID: sessionID},
		Stderr: eventWriter{app: a, sessionID: sessionID},
	})
	if err != nil {
		stdinWriter.Close()
		return "", err
	}

	session, err := a.sessions.Register(termsession.Options{
		ID: sessionID, ResourceID: "__bashes_localhost__", Kind: termsession.KindLocal,
		Shell: shell, Stdin: stdinWriter,
	})
	if err != nil {
		_ = stdinWriter.Close()
		_ = shell.Close()
		return "", err
	}
	a.emit(desktop.EventSSHStatus, SSHEvent{
		SessionID: sessionID,
		Message:   "Local shell started",
	})
	go a.waitForShell(sessionID, session)

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
		ManagePassword:       input.ManagePassword,
		SavePassword:         input.SavePassword,
		TrustHostKey:         input.TrustHostKey,
		AcceptHostKey:        input.AcceptHostKey,
		ReplaceHostKey:       input.ReplaceHostKey,
	}
	sessionInput, err = a.prepareSessionInput(resource, sessionInput)
	if err != nil {
		return SSHTunnelInfo{}, err
	}
	dialInput := a.resolveSessionKeyPath(sessionInput)

	client, err := a.dialResource(resource, dialInput, remotessh.DefaultTimeout)
	if err != nil {
		return SSHTunnelInfo{}, err
	}
	if err := a.persistPasswordChoice(resource.ID, sessionInput); err != nil {
		client.Close()
		return SSHTunnelInfo{}, err
	}

	listener, forwardAddress, err := a.startTunnelListener(client, input)
	if err != nil {
		client.Close()
		return SSHTunnelInfo{}, err
	}

	tunnelID := fmt.Sprintf("tunnel-%d", time.Now().UnixNano())
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
	serve := func(ctx context.Context) error {
		switch info.Type {
		case "socks":
			return remotessh.ServeSOCKS5(ctx, listener, client)
		case "local":
			return remotessh.ServeLocalForward(ctx, listener, client, forwardAddress)
		case "remote":
			return remotessh.ServeRemoteForward(ctx, listener, forwardAddress)
		default:
			return fmt.Errorf("unsupported tunnel type %q", info.Type)
		}
	}
	tunnel, err := a.tunnels.Register(sshtunnel.Options{
		Info: info, Client: client, Listener: listener, Serve: serve,
	})
	if err != nil {
		_ = listener.Close()
		_ = client.Close()
		return SSHTunnelInfo{}, err
	}
	ctx := a.tunnels.Context(tunnel)

	if auth := authPreferenceFromSessionInput(sessionInput); auth != nil {
		if err := a.service.SetResourceAuth(resource.ID, *auth); err != nil {
			a.emit(desktop.EventSSHStatus, SSHEvent{SessionID: tunnelID, Message: fmt.Sprintf("Could not save SSH auth preference: %v", err)})
		}
	}

	go remotessh.MonitorKeepAlive(ctx, client, remotessh.KeepAliveOptions{}, func(err error) {
		if a.tunnels.Fail(tunnel) {
			a.emit(desktop.EventSSHTunnelClosed, SSHEvent{SessionID: tunnelID, Message: fmt.Sprintf("Tunnel closed: SSH keepalive failed: %v", err)})
		}
	})
	go a.waitForTunnel(tunnel)

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
	return a.tunnels.Infos()
}

func (a *App) StopSSHTunnel(tunnelID string) error {
	a.tunnels.Stop(tunnelID)
	return nil
}

func (a *App) waitForTunnel(tunnel *sshtunnel.Tunnel) {
	result := a.tunnels.Run(tunnel)
	if result.Stopped || !result.Authoritative {
		return
	}
	if result.Err != nil {
		a.emit(desktop.EventSSHTunnelClosed, SSHEvent{SessionID: result.ID, Message: fmt.Sprintf("Tunnel closed: %v", result.Err)})
	}
}

func (a *App) WriteSSHSession(sessionID string, data string) error {
	return a.sessions.Write(sessionID, data)
}

func (a *App) ResizeSSHSession(sessionID string, cols int, rows int) error {
	return a.sessions.Resize(sessionID, remotessh.TerminalSize{Cols: cols, Rows: rows})
}

func (a *App) StopSSHSession(sessionID string) error {
	kind, stopped := a.sessions.Stop(sessionID)
	if !stopped {
		return nil
	}
	a.emit(desktop.EventSSHClosed, SSHEvent{SessionID: sessionID, Message: sessionClosedMessage(kind)})
	return nil
}

func (a *App) stopTunnelsForResource(resourceID string) {
	for _, id := range a.tunnels.IDsForResource(resourceID) {
		_ = a.StopSSHTunnel(id)
	}
}

func (a *App) stopSessionsForResource(resourceID string) {
	for _, id := range a.sessions.IDsForResource(resourceID) {
		_ = a.StopSSHSession(id)
	}
}

func (a *App) closeTransfersForResource(resourceID string) {
	for _, id := range a.transfers.IDsForResource(resourceID) {
		_ = a.CloseFileTransfer(id)
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

func (a *App) waitForShell(sessionID string, session *termsession.Session) {
	result := a.sessions.Wait(session)
	if result.Stopped || !result.Authoritative {
		return
	}
	if result.Err != nil && !errors.Is(result.Err, io.EOF) {
		a.emit(desktop.EventSSHClosed, SSHEvent{SessionID: sessionID, Message: result.Err.Error()})
		return
	}
	a.emit(desktop.EventSSHClosed, SSHEvent{SessionID: sessionID, Message: sessionClosedMessage(result.Kind)})
}

func sessionClosedMessage(kind termsession.Kind) string {
	if kind == termsession.KindLocal {
		return "Local shell closed"
	}
	return "SSH session closed"
}

func (a *App) stopAllRuntimeConnections() {
	for _, id := range a.sessions.IDs() {
		_ = a.StopSSHSession(id)
	}
	for _, id := range a.tunnels.IDs() {
		_ = a.StopSSHTunnel(id)
	}
	for _, id := range a.transfers.IDs() {
		_ = a.CloseFileTransfer(id)
	}
	a.stopAllWebAppRuntimes()
	ctx, cancel := context.WithTimeout(context.Background(), plugin.DefaultStopTimeout)
	defer cancel()
	_ = a.plugins.Shutdown(ctx)
}

func (a *App) resourceByID(id string) (domain.Endpoint, error) {
	return a.service.Endpoint(id)
}

func (a *App) emit(name string, event SSHEvent) {
	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	if a.ctx == nil {
		return
	}
	a.desktopRuntime.Emit(a.ctx, name, event)
}

func (a *App) emitData(name string, data interface{}) {
	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	if a.ctx == nil {
		return
	}
	a.desktopRuntime.Emit(a.ctx, name, data)
}

func (a *App) runtimeContext() context.Context {
	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	return a.ctx
}

func (a *App) exportDatabaseFromMenu() {
	ctx := a.runtimeContext()
	if ctx == nil {
		return
	}
	path, err := a.desktopRuntime.SaveFileDialog(ctx, desktop.SaveDialogOptions{
		Title:           "Export Bashes Database",
		DefaultFilename: fmt.Sprintf("bashes-hosts-%s.json", time.Now().Format("20060102")),
		Filters: []desktop.FileFilter{
			{DisplayName: "JSON files (*.json)", Pattern: "*.json"},
		},
	})
	if err != nil {
		a.showErrorDialog("Export Database", err)
		return
	}
	if strings.TrimSpace(path) == "" {
		return
	}
	if err := a.ExportDatabase(path); err != nil {
		a.showErrorDialog("Export Database", err)
		return
	}
	a.showInfoDialog("Export Database", fmt.Sprintf("Database exported to:\n%s", path))
	a.emitData(desktop.EventDatabaseExported, map[string]string{"message": "Database exported.", "path": path})
}

func (a *App) importDatabaseFromMenu() {
	ctx := a.runtimeContext()
	if ctx == nil {
		return
	}
	path, err := a.desktopRuntime.OpenFileDialog(ctx, desktop.OpenDialogOptions{
		Title: "Import Bashes Database",
		Filters: []desktop.FileFilter{
			{DisplayName: "JSON files (*.json)", Pattern: "*.json"},
		},
	})
	if err != nil {
		a.showErrorDialog("Import Database", err)
		return
	}
	if strings.TrimSpace(path) == "" {
		return
	}

	choice, err := a.desktopRuntime.MessageDialog(ctx, desktop.MessageDialogOptions{
		Type:          desktop.MessageQuestion,
		Title:         "Import Database",
		Message:       "Importing a database replaces the current hosts.json after creating hosts.json.bak. Active sessions, tunnels and file transfers will be closed.",
		Buttons:       []string{"Import", "Cancel"},
		DefaultButton: "Import",
		CancelButton:  "Cancel",
	})
	if err != nil {
		a.showErrorDialog("Import Database", err)
		return
	}
	if !databaseImportConfirmed(choice) {
		return
	}

	if err := a.ImportDatabase(path); err != nil {
		a.showErrorDialog("Import Database", err)
		return
	}
	a.showInfoDialog("Import Database", "Database imported.")
}

func databaseImportConfirmed(choice string) bool {
	// Wails GTK and Windows question dialogs return native labels even when custom labels are requested.
	switch strings.ToLower(strings.TrimSpace(choice)) {
	case "import", "yes":
		return true
	default:
		return false
	}
}

func (a *App) importHostsFileFromMenu() {
	result, err := a.PreviewImportFromHostsFile()
	if err != nil {
		a.showErrorDialog("Import from hosts file", err)
		return
	}
	a.emitData(desktop.EventHostsFilePreview, result)
}

func (a *App) openSettingsFromMenu() {
	a.emitData(desktop.EventOpenSettings, map[string]bool{"open": true})
}

func (a *App) openSSHKeysFromMenu() {
	a.emitData(desktop.EventOpenSSHKeys, map[string]bool{"allowInstall": false})
}

func (a *App) showAboutFromMenu() {
	a.emitData(desktop.EventOpenAbout, a.GetAppInfo())
}

func (a *App) checkForUpdatesFromMenu() {
	info, err := a.CheckForUpdate()
	if err != nil {
		a.emitData(desktop.EventUpdateCheck, map[string]interface{}{
			"manual": true,
			"error":  err.Error(),
		})
		return
	}
	a.emitData(desktop.EventUpdateCheck, map[string]interface{}{
		"manual": true,
		"info":   info,
	})
}

func (a *App) openReadmeFromMenu() {
	if ctx := a.runtimeContext(); ctx != nil {
		a.desktopRuntime.OpenURL(ctx, readmeURL)
	}
}

func (a *App) openReleasesFromMenu() {
	if ctx := a.runtimeContext(); ctx != nil {
		a.desktopRuntime.OpenURL(ctx, releasesURL)
	}
}

func (a *App) showInfoDialog(title string, message string) {
	ctx := a.runtimeContext()
	if ctx == nil {
		return
	}
	_, _ = a.desktopRuntime.MessageDialog(ctx, desktop.MessageDialogOptions{
		Type:    desktop.MessageInfo,
		Title:   title,
		Message: message,
	})
}

func (a *App) showErrorDialog(title string, err error) {
	ctx := a.runtimeContext()
	if ctx == nil {
		return
	}
	_, _ = a.desktopRuntime.MessageDialog(ctx, desktop.MessageDialogOptions{
		Type:    desktop.MessageError,
		Title:   title,
		Message: err.Error(),
	})
}

type eventWriter struct {
	app       *App
	sessionID string
}

func (w eventWriter) Write(data []byte) (int, error) {
	if len(data) > 0 {
		w.app.emit(desktop.EventSSHOutput, sshOutputEvent(w.sessionID, data))
	}
	return len(data), nil
}

func sshOutputEvent(sessionID string, data []byte) SSHEvent {
	return SSHEvent{
		SessionID: sessionID,
		Bytes:     base64.StdEncoding.EncodeToString(data),
	}
}

func applyAuthPreference(resource domain.Endpoint, input SSHSessionInput) SSHSessionInput {
	if hasExplicitAuth(input) || resource.Auth == nil {
		return input
	}

	auth := resource.Auth
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
	auth := domain.Auth{}
	switch {
	case strings.TrimSpace(input.PrivateKeyPath) != "":
		auth.Method = domain.AuthMethodPath
		auth.PrivateKeyPath = strings.TrimSpace(input.PrivateKeyPath)
	case strings.TrimSpace(input.KeyName) != "":
		auth.Method = domain.AuthMethodKey
		auth.KeyName = sanitizeKeyName(input.KeyName)
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
	if !input.AllowPublicBind && !isLoopbackBind(input.LocalHost) {
		return apperrors.New("public_tunnel_bind", "public tunnel bind requires confirmation", map[string]string{
			"host": input.LocalHost,
			"type": input.Type,
		})
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

func isLoopbackBind(host string) bool {
	host = strings.TrimSpace(strings.Trim(host, "[]"))
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
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

func (a *App) dialResource(resource domain.Endpoint, input SSHSessionInput, timeout time.Duration) (*ssh.Client, error) {
	return a.sshConnector.Dial(resource, sshConnectionInput(input), timeout)
}

func sshHost(resource domain.Endpoint) string {
	return sshconnection.Host(resource)
}

func authMethods(input SSHSessionInput) ([]ssh.AuthMethod, net.Conn, error) {
	return sshauth.NewBuilder().Methods(sshAuthInput(input))
}

func sshAuthInput(input SSHSessionInput) sshauth.Input {
	return sshauth.Input{
		Password:       input.Password,
		PrivateKeyPath: input.PrivateKeyPath,
		KeyName:        input.KeyName,
		Passphrase:     input.PrivateKeyPassphrase,
	}
}

func sshConnectionInput(input SSHSessionInput) sshconnection.Input {
	return sshconnection.Input{
		Authentication: sshAuthInput(input),
		TrustHostKey:   input.TrustHostKey,
		AcceptHostKey:  input.AcceptHostKey,
		ReplaceHostKey: input.ReplaceHostKey,
	}
}

func parsePrivateKey(key []byte, passphrase string) (ssh.Signer, error) {
	return sshauth.ParsePrivateKey(key, passphrase)
}

func isPuttyPrivateKey(key []byte) bool {
	return sshauth.IsPuttyPrivateKey(key)
}

func hostKeyPolicy(resource domain.Endpoint, input SSHSessionInput, acceptedFingerprint *string) remotessh.HostKeyPolicy {
	return sshconnection.HostKeyPolicy(resource, sshConnectionInput(input), acceptedFingerprint)
}

func hostKeyError(err error, resource domain.Endpoint) error {
	return sshconnection.NormalizeError(err, resource)
}

func writeSSHKeyPairAtomic(privatePath string, privateData []byte, publicPath string, publicData []byte) error {
	return sshkeys.WritePairAtomic(privatePath, privateData, publicPath, publicData)
}

func publishSSHKeyFileWithLink(
	tempPath string,
	path string,
	data []byte,
	perm os.FileMode,
	link func(string, string) error,
) error {
	return sshkeys.PublishFileWithLink(tempPath, path, data, perm, link)
}

func expandHome(path string) string {
	path = strings.TrimSpace(path)
	if path == "~" {
		return userHomeDir()
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(userHomeDir(), strings.TrimPrefix(path, "~/"))
	}
	if strings.HasPrefix(path, `~\`) {
		return filepath.Join(userHomeDir(), strings.TrimPrefix(path, `~\`))
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
	return a.sshKeys.ManagedDir()
}

func sanitizeKeyName(name string) string {
	return sshkeys.SanitizeName(name)
}

func shellQuote(value string) string {
	return sshkeyinstall.ShellQuote(value)
}
