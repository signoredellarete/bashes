package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/signoredellarete/bashes/internal/domain"
	"github.com/signoredellarete/bashes/internal/remotessh"
	"github.com/signoredellarete/bashes/internal/webappbrowser"
	"golang.org/x/crypto/ssh"
)

type WebAppLaunchInput struct {
	WebAppID             string `json:"webAppId"`
	Password             string `json:"password,omitempty"`
	PrivateKeyPath       string `json:"privateKeyPath,omitempty"`
	KeyName              string `json:"keyName,omitempty"`
	PrivateKeyPassphrase string `json:"privateKeyPassphrase,omitempty"`
	ManagePassword       bool   `json:"managePassword,omitempty"`
	SavePassword         bool   `json:"savePassword,omitempty"`
	TrustHostKey         bool   `json:"trustHostKey,omitempty"`
	AcceptHostKey        bool   `json:"acceptHostKey,omitempty"`
	ReplaceHostKey       bool   `json:"replaceHostKey,omitempty"`
}

type WebAppRuntimeInfo struct {
	RuntimeID string `json:"runtimeId"`
	WebAppID  string `json:"webAppId"`
	BrowserID string `json:"browserId"`
	Browser   string `json:"browser"`
	SOCKSPort int    `json:"socksPort,omitempty"`
	Tunnel    bool   `json:"tunnel"`
	StartedAt string `json:"startedAt"`
}

type webAppRuntime struct {
	id          string
	groupKey    string
	browser     webappbrowser.Browser
	profileDir  string
	private     bool
	policy      webappbrowser.RetentionPolicy
	process     *exec.Cmd
	clientMu    sync.RWMutex
	client      *ssh.Client
	listener    net.Listener
	cancel      context.CancelFunc
	gateway     domain.Endpoint
	sshInput    SSHSessionInput
	reconnectMu sync.Mutex
	apps        map[string]struct{}
	info        WebAppRuntimeInfo
	stopOnce    sync.Once
}

func (a *App) LaunchWebApp(input WebAppLaunchInput) (WebAppRuntimeInfo, error) {
	app, err := a.webAppByID(input.WebAppID)
	if err != nil {
		return WebAppRuntimeInfo{}, err
	}
	browser, err := a.resolveWebAppBrowser()
	if err != nil {
		return WebAppRuntimeInfo{}, err
	}
	groupKey := webAppGroupKey(app, browser)

	a.mu.Lock()
	for _, runtime := range a.webAppRuntimes {
		if runtime.groupKey == groupKey {
			runtime.apps[app.ID] = struct{}{}
			a.webAppRuntimeIDs[app.ID] = runtime.id
			info := runtime.info
			info.WebAppID = app.ID
			a.mu.Unlock()
			if err := a.openWebAppTab(runtime, app.URL); err != nil {
				return WebAppRuntimeInfo{}, err
			}
			return info, nil
		}
	}
	a.mu.Unlock()

	runtime, err := a.startWebAppRuntime(app, browser, input, groupKey)
	if err != nil {
		return WebAppRuntimeInfo{}, err
	}
	return runtime.info, nil
}

func (a *App) ListWebAppRuntimes() []WebAppRuntimeInfo {
	a.mu.Lock()
	defer a.mu.Unlock()

	result := make([]WebAppRuntimeInfo, 0, len(a.webAppRuntimeIDs))
	for appID, runtimeID := range a.webAppRuntimeIDs {
		if runtime := a.webAppRuntimes[runtimeID]; runtime != nil {
			info := runtime.info
			info.WebAppID = appID
			result = append(result, info)
		}
	}
	return result
}

func (a *App) StopWebAppRuntime(runtimeID string) error {
	a.mu.Lock()
	runtime := a.webAppRuntimes[strings.TrimSpace(runtimeID)]
	a.mu.Unlock()
	if runtime == nil {
		return nil
	}
	a.closeWebAppRuntime(runtime, true, "Web App browser closed")
	return nil
}

func (a *App) startWebAppRuntime(app domain.WebApp, browser webappbrowser.Browser, input WebAppLaunchInput, groupKey string) (*webAppRuntime, error) {
	var client *ssh.Client
	var listener net.Listener
	var sessionInput SSHSessionInput
	var gateway domain.Endpoint
	if app.ConnectionMode == domain.WebAppConnectionTunnel {
		resource, err := a.resourceByID(app.GatewayResourceID)
		if err != nil {
			return nil, err
		}
		gateway = resource
		sessionInput = SSHSessionInput{
			ResourceID: resource.ID, Password: input.Password, PrivateKeyPath: input.PrivateKeyPath,
			KeyName: input.KeyName, PrivateKeyPassphrase: input.PrivateKeyPassphrase,
			ManagePassword: input.ManagePassword, SavePassword: input.SavePassword,
			TrustHostKey: input.TrustHostKey, AcceptHostKey: input.AcceptHostKey, ReplaceHostKey: input.ReplaceHostKey,
		}
		sessionInput, err = a.prepareSessionInput(resource, sessionInput)
		if err != nil {
			return nil, err
		}
		client, err = a.dialResource(resource, a.resolveSessionKeyPath(sessionInput), remotessh.DefaultTimeout)
		if err != nil {
			return nil, err
		}
		if err := a.persistPasswordChoice(resource.ID, sessionInput); err != nil {
			client.Close()
			return nil, err
		}
		port := app.LocalSOCKSPort
		address := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
		listener, err = net.Listen("tcp", address)
		if err != nil {
			client.Close()
			return nil, fmt.Errorf("start Web App SOCKS tunnel: %w", err)
		}
		if auth := authPreferenceFromSessionInput(sessionInput); auth != nil {
			if err := a.service.SetResourceAuth(resource.ID, *auth); err != nil {
				listener.Close()
				client.Close()
				return nil, fmt.Errorf("save SSH auth preference: %w", err)
			}
		}
	}

	profileDir, privateProfile, err := a.webAppProfileDir(groupKey, app.BrowserMode == domain.WebAppBrowserPrivate)
	if err != nil {
		closeWebAppNetwork(listener, client)
		return nil, err
	}
	policy := webappbrowser.RetentionPolicy{
		KeepCookies: app.KeepCookies, KeepLocalStorage: app.KeepLocalStorage,
		RestoreSession: app.RestoreSession, ClearCacheOnClose: app.ClearCacheOnClose,
	}
	if !privateProfile {
		_ = webappbrowser.CleanupProfile(browser.Family, profileDir, policy)
	}
	socksPort := listenerPort(listener)
	if browser.Family == webappbrowser.FamilyFirefox {
		if err := webappbrowser.WriteFirefoxPreferences(profileDir, socksPort); err != nil {
			closeWebAppNetwork(listener, client)
			if privateProfile {
				_ = os.RemoveAll(profileDir)
			}
			return nil, err
		}
	}
	command, err := webappbrowser.Start(browser, webappbrowser.LaunchOptions{
		URL: app.URL, ProfileDir: profileDir, Private: privateProfile, SOCKSPort: socksPort,
	})
	if err != nil {
		closeWebAppNetwork(listener, client)
		if privateProfile {
			_ = os.RemoveAll(profileDir)
		}
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	runtimeID := fmt.Sprintf("webapp-%d", time.Now().UnixNano())
	runtime := &webAppRuntime{
		id: runtimeID, groupKey: groupKey, browser: browser, profileDir: profileDir,
		private: privateProfile, policy: policy, process: command, client: client,
		listener: listener, cancel: cancel, apps: map[string]struct{}{app.ID: {}},
		gateway: gateway, sshInput: sessionInput,
		info: WebAppRuntimeInfo{
			RuntimeID: runtimeID, WebAppID: app.ID, BrowserID: string(browser.ID), Browser: browser.Name,
			SOCKSPort: socksPort, Tunnel: listener != nil, StartedAt: time.Now().Format(time.RFC3339),
		},
	}
	a.mu.Lock()
	a.webAppRuntimes[runtimeID] = runtime
	a.webAppRuntimeIDs[app.ID] = runtimeID
	a.mu.Unlock()

	if listener != nil {
		go func() {
			if err := remotessh.ServeSOCKS5WithDialer(ctx, listener, runtime.dial); err != nil && ctx.Err() == nil {
				a.closeWebAppRuntime(runtime, true, fmt.Sprintf("Web App tunnel closed: %v", err))
			}
		}()
		a.monitorWebAppTunnel(ctx, runtime, client)
	}
	go func() {
		_ = command.Wait()
		a.closeWebAppRuntime(runtime, false, "Web App browser closed")
	}()
	a.emitData("webapp:status", runtime.info)
	return runtime, nil
}

func (a *App) openWebAppTab(runtime *webAppRuntime, url string) error {
	command, err := webappbrowser.Start(runtime.browser, webappbrowser.LaunchOptions{
		URL: url, ProfileDir: runtime.profileDir, Private: runtime.private,
		SOCKSPort: runtime.info.SOCKSPort, NewTab: true,
	})
	if err != nil {
		return err
	}
	go func() { _ = command.Wait() }()
	return nil
}

func (a *App) closeWebAppRuntime(runtime *webAppRuntime, killBrowser bool, message string) {
	runtime.stopOnce.Do(func() {
		a.mu.Lock()
		if a.webAppRuntimes[runtime.id] == runtime {
			delete(a.webAppRuntimes, runtime.id)
		}
		for appID := range runtime.apps {
			if a.webAppRuntimeIDs[appID] == runtime.id {
				delete(a.webAppRuntimeIDs, appID)
			}
		}
		a.mu.Unlock()

		runtime.cancel()
		runtime.closeNetwork()
		if killBrowser && runtime.process != nil && runtime.process.Process != nil {
			_ = runtime.process.Process.Kill()
		}
		if runtime.private {
			_ = os.RemoveAll(runtime.profileDir)
		} else {
			_ = webappbrowser.CleanupProfile(runtime.browser.Family, runtime.profileDir, runtime.policy)
		}
		a.emitData("webapp:closed", map[string]string{"runtimeId": runtime.id, "message": message})
	})
}

func (runtime *webAppRuntime) dial(network, address string) (net.Conn, error) {
	runtime.clientMu.RLock()
	client := runtime.client
	runtime.clientMu.RUnlock()
	if client == nil {
		return nil, errors.New("Web App SSH tunnel is reconnecting")
	}
	return client.Dial(network, address)
}

func (runtime *webAppRuntime) closeNetwork() {
	if runtime.listener != nil {
		_ = runtime.listener.Close()
	}
	runtime.clientMu.Lock()
	client := runtime.client
	runtime.client = nil
	runtime.clientMu.Unlock()
	if client != nil {
		_ = client.Close()
	}
}

func (a *App) monitorWebAppTunnel(ctx context.Context, runtime *webAppRuntime, client *ssh.Client) {
	go monitorSSHKeepAlive(ctx, client, func(failure error) {
		go a.reconnectWebAppTunnel(ctx, runtime, client, failure)
	})
}

func (a *App) reconnectWebAppTunnel(ctx context.Context, runtime *webAppRuntime, failedClient *ssh.Client, failure error) {
	runtime.reconnectMu.Lock()
	defer runtime.reconnectMu.Unlock()

	runtime.clientMu.RLock()
	current := runtime.client
	runtime.clientMu.RUnlock()
	if current != failedClient || ctx.Err() != nil {
		return
	}
	_ = failedClient.Close()
	runtime.clientMu.Lock()
	if runtime.client == failedClient {
		runtime.client = nil
	}
	runtime.clientMu.Unlock()

	var lastError = failure
	for attempt, delay := range []time.Duration{0, time.Second, 2 * time.Second} {
		if delay > 0 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
			}
		}
		client, err := a.dialResource(runtime.gateway, a.resolveSessionKeyPath(runtime.sshInput), remotessh.DefaultTimeout)
		if err != nil {
			lastError = err
			continue
		}
		if ctx.Err() != nil {
			client.Close()
			return
		}
		runtime.clientMu.Lock()
		runtime.client = client
		runtime.clientMu.Unlock()
		a.emitData("webapp:status", map[string]interface{}{
			"runtimeId": runtime.id, "message": "Web App SSH tunnel reconnected", "attempt": attempt + 1,
		})
		a.monitorWebAppTunnel(ctx, runtime, client)
		return
	}
	a.closeWebAppRuntime(runtime, true, fmt.Sprintf("Web App tunnel could not reconnect: %v", lastError))
}

func (a *App) stopWebAppForID(appID string) {
	appID = strings.TrimSpace(appID)
	a.mu.Lock()
	runtimeID := a.webAppRuntimeIDs[appID]
	runtime := a.webAppRuntimes[runtimeID]
	if runtime != nil {
		delete(runtime.apps, appID)
		delete(a.webAppRuntimeIDs, appID)
	}
	remaining := runtime != nil && len(runtime.apps) > 0
	a.mu.Unlock()
	if runtime != nil && !remaining {
		a.closeWebAppRuntime(runtime, true, "Web App configuration closed")
	}
}

func (a *App) stopAllWebAppRuntimes() {
	a.mu.Lock()
	runtimes := make([]*webAppRuntime, 0, len(a.webAppRuntimes))
	for _, runtime := range a.webAppRuntimes {
		runtimes = append(runtimes, runtime)
	}
	a.mu.Unlock()
	for _, runtime := range runtimes {
		a.closeWebAppRuntime(runtime, true, "Bashes is closing")
	}
}

func (a *App) webAppByID(id string) (domain.WebApp, error) {
	workspace, err := a.service.ListWorkspace()
	if err != nil {
		return domain.WebApp{}, err
	}
	for _, app := range workspace.WebApps {
		if app.ID == strings.TrimSpace(id) {
			return app, nil
		}
	}
	return domain.WebApp{}, fmt.Errorf("web app %q not found", id)
}

func (a *App) resolveWebAppBrowser() (webappbrowser.Browser, error) {
	settings, err := a.GetWebAppBrowserSettings()
	if err != nil {
		return webappbrowser.Browser{}, err
	}
	if settings.CustomBrowserPath != "" {
		return browserFromSettings(settings)
	}
	browsers := a.ListWebAppBrowsers()
	if settings.PreferredBrowserID != "" {
		for _, browser := range browsers {
			if string(browser.ID) == settings.PreferredBrowserID {
				return browser, nil
			}
		}
		return webappbrowser.Browser{}, newAppError("webapp_browser_missing", "The selected browser is not installed", map[string]string{"browser": settings.PreferredBrowserID})
	}
	if len(browsers) == 0 {
		return webappbrowser.Browser{}, newAppError("webapp_browser_missing", "No supported browser was found", nil)
	}
	return browsers[0], nil
}

func browserFromSettings(settings WebAppBrowserSettings) (webappbrowser.Browser, error) {
	var browser webappbrowser.Browser
	switch webappbrowser.ID(settings.PreferredBrowserID) {
	case webappbrowser.Firefox:
		browser = webappbrowser.Browser{ID: webappbrowser.Firefox, Name: "Firefox", Family: webappbrowser.FamilyFirefox}
	case webappbrowser.Chrome:
		browser = webappbrowser.Browser{ID: webappbrowser.Chrome, Name: "Google Chrome", Family: webappbrowser.FamilyChromium}
	case webappbrowser.Chromium:
		browser = webappbrowser.Browser{ID: webappbrowser.Chromium, Name: "Chromium", Family: webappbrowser.FamilyChromium}
	case webappbrowser.Edge:
		browser = webappbrowser.Browser{ID: webappbrowser.Edge, Name: "Microsoft Edge", Family: webappbrowser.FamilyChromium}
	default:
		return webappbrowser.Browser{}, errors.New("select the browser type for the custom executable")
	}
	browser.Path = settings.CustomBrowserPath
	return browser, nil
}

func (a *App) webAppProfileDir(groupKey string, private bool) (string, bool, error) {
	base := filepath.Join(filepath.Dir(a.dataPath), "webapp-browser")
	if err := os.MkdirAll(base, 0o700); err != nil {
		return "", false, fmt.Errorf("create Web App browser directory: %w", err)
	}
	if private {
		dir, err := os.MkdirTemp(base, "private-*")
		if err != nil {
			return "", false, fmt.Errorf("create private browser profile: %w", err)
		}
		return dir, true, nil
	}
	sum := sha256.Sum256([]byte(groupKey))
	dir := filepath.Join(base, "profiles", hex.EncodeToString(sum[:8]))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", false, fmt.Errorf("create browser profile: %w", err)
	}
	return dir, false, nil
}

func webAppGroupKey(app domain.WebApp, browser webappbrowser.Browser) string {
	return strings.Join([]string{
		string(browser.ID), browser.Path, string(app.ConnectionMode), app.GatewayResourceID,
		string(app.PortMode), strconv.Itoa(app.LocalSOCKSPort), string(app.BrowserMode),
		strconv.FormatBool(app.KeepCookies), strconv.FormatBool(app.KeepLocalStorage),
		strconv.FormatBool(app.RestoreSession), strconv.FormatBool(app.ClearCacheOnClose),
	}, "|")
}

func listenerPort(listener net.Listener) int {
	if listener == nil {
		return 0
	}
	if address, ok := listener.Addr().(*net.TCPAddr); ok {
		return address.Port
	}
	return 0
}

func closeWebAppNetwork(listener net.Listener, client *ssh.Client) {
	if listener != nil {
		_ = listener.Close()
	}
	if client != nil {
		_ = client.Close()
	}
}
