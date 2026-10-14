package webappbrowser

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type ID string

const (
	Firefox  ID = "firefox"
	Chrome   ID = "chrome"
	Chromium ID = "chromium"
	Edge     ID = "edge"
)

type Family string

const (
	FamilyFirefox  Family = "firefox"
	FamilyChromium Family = "chromium"
)

type Browser struct {
	ID     ID     `json:"id"`
	Name   string `json:"name"`
	Path   string `json:"path"`
	Family Family `json:"family"`
}

type Detector struct {
	goos     string
	getenv   func(string) string
	lookPath func(string) (string, error)
	stat     func(string) (os.FileInfo, error)
}

func NewDetector(goos string, getenv func(string) string) Detector {
	if getenv == nil {
		getenv = os.Getenv
	}
	return Detector{goos: goos, getenv: getenv, lookPath: exec.LookPath, stat: os.Stat}
}

func (d Detector) Detect() []Browser {
	candidates := browserCandidates(d.goos, d.getenv)
	found := make([]Browser, 0, len(candidates))
	seen := map[ID]struct{}{}
	for _, candidate := range candidates {
		if _, exists := seen[candidate.ID]; exists {
			continue
		}
		path := d.resolve(candidate.Path)
		if path == "" {
			continue
		}
		candidate.Path = path
		found = append(found, candidate)
		seen[candidate.ID] = struct{}{}
	}
	return found
}

func (d Detector) resolve(candidate string) string {
	if candidate == "" {
		return ""
	}
	if filepath.IsAbs(candidate) {
		if info, err := d.stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
		return ""
	}
	path, err := d.lookPath(candidate)
	if err != nil {
		return ""
	}
	return path
}

func browserCandidates(goos string, getenv func(string) string) []Browser {
	browser := func(id ID, name string, family Family, paths ...string) []Browser {
		result := make([]Browser, 0, len(paths))
		for _, path := range paths {
			result = append(result, Browser{ID: id, Name: name, Family: family, Path: path})
		}
		return result
	}

	var result []Browser
	switch goos {
	case "windows":
		programFiles := getenv("ProgramFiles")
		programFilesX86 := getenv("ProgramFiles(x86)")
		localAppData := getenv("LOCALAPPDATA")
		result = append(result, browser(Firefox, "Firefox", FamilyFirefox,
			filepath.Join(programFiles, "Mozilla Firefox", "firefox.exe"),
			filepath.Join(programFilesX86, "Mozilla Firefox", "firefox.exe"), "firefox.exe")...)
		result = append(result, browser(Chrome, "Google Chrome", FamilyChromium,
			filepath.Join(programFiles, "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(programFilesX86, "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(localAppData, "Google", "Chrome", "Application", "chrome.exe"), "chrome.exe")...)
		result = append(result, browser(Chromium, "Chromium", FamilyChromium,
			filepath.Join(programFiles, "Chromium", "Application", "chrome.exe"),
			filepath.Join(localAppData, "Chromium", "Application", "chrome.exe"), "chromium.exe")...)
		result = append(result, browser(Edge, "Microsoft Edge", FamilyChromium,
			filepath.Join(programFiles, "Microsoft", "Edge", "Application", "msedge.exe"),
			filepath.Join(programFilesX86, "Microsoft", "Edge", "Application", "msedge.exe"), "msedge.exe")...)
	case "darwin":
		result = append(result, browser(Firefox, "Firefox", FamilyFirefox,
			"/Applications/Firefox.app/Contents/MacOS/firefox", "firefox")...)
		result = append(result, browser(Chrome, "Google Chrome", FamilyChromium,
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome", "google-chrome")...)
		result = append(result, browser(Chromium, "Chromium", FamilyChromium,
			"/Applications/Chromium.app/Contents/MacOS/Chromium", "chromium")...)
		result = append(result, browser(Edge, "Microsoft Edge", FamilyChromium,
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge", "microsoft-edge")...)
	default:
		result = append(result, browser(Firefox, "Firefox", FamilyFirefox, "firefox", "firefox-esr")...)
		result = append(result, browser(Chrome, "Google Chrome", FamilyChromium, "google-chrome", "google-chrome-stable")...)
		result = append(result, browser(Chromium, "Chromium", FamilyChromium, "chromium", "chromium-browser")...)
		result = append(result, browser(Edge, "Microsoft Edge", FamilyChromium, "microsoft-edge", "microsoft-edge-stable")...)
	}
	return result
}

type LaunchOptions struct {
	URL        string
	ProfileDir string
	Private    bool
	SOCKSPort  int
	NewTab     bool
}

type RetentionPolicy struct {
	KeepCookies       bool
	KeepLocalStorage  bool
	RestoreSession    bool
	ClearCacheOnClose bool
}

func CleanupProfile(family Family, profileDir string, policy RetentionPolicy) error {
	if strings.TrimSpace(profileDir) == "" {
		return nil
	}
	var relativePaths []string
	if policy.ClearCacheOnClose {
		switch family {
		case FamilyFirefox:
			relativePaths = append(relativePaths, "cache2", "startupCache", "shader-cache")
		case FamilyChromium:
			relativePaths = append(relativePaths, "Cache", "Code Cache", "GPUCache", filepath.Join("Default", "Cache"), filepath.Join("Default", "Code Cache"))
		}
	}
	if !policy.KeepCookies {
		switch family {
		case FamilyFirefox:
			relativePaths = append(relativePaths, "cookies.sqlite", "cookies.sqlite-shm", "cookies.sqlite-wal")
		case FamilyChromium:
			relativePaths = append(relativePaths, filepath.Join("Default", "Cookies"), filepath.Join("Default", "Network", "Cookies"))
		}
	}
	if !policy.KeepLocalStorage {
		switch family {
		case FamilyFirefox:
			relativePaths = append(relativePaths, "storage")
		case FamilyChromium:
			relativePaths = append(relativePaths, filepath.Join("Default", "Local Storage"), filepath.Join("Default", "IndexedDB"))
		}
	}
	if !policy.RestoreSession {
		switch family {
		case FamilyFirefox:
			relativePaths = append(relativePaths, "sessionstore.jsonlz4", "sessionstore-backups")
		case FamilyChromium:
			relativePaths = append(relativePaths, filepath.Join("Default", "Sessions"), filepath.Join("Default", "Current Session"), filepath.Join("Default", "Current Tabs"))
		}
	}
	var cleanupErrors []error
	for _, relativePath := range relativePaths {
		if err := os.RemoveAll(filepath.Join(profileDir, relativePath)); err != nil {
			cleanupErrors = append(cleanupErrors, err)
		}
	}
	return errors.Join(cleanupErrors...)
}

func Start(browser Browser, options LaunchOptions) (*exec.Cmd, error) {
	args, err := Arguments(browser, options)
	if err != nil {
		return nil, err
	}
	command := exec.Command(browser.Path, args...)
	if err := command.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", browser.Name, err)
	}
	return command, nil
}

func Arguments(browser Browser, options LaunchOptions) ([]string, error) {
	if strings.TrimSpace(browser.Path) == "" {
		return nil, errors.New("browser executable path is required")
	}
	if strings.TrimSpace(options.URL) == "" {
		return nil, errors.New("web app URL is required")
	}
	if strings.TrimSpace(options.ProfileDir) == "" {
		return nil, errors.New("browser profile directory is required")
	}
	if options.SOCKSPort < 0 || options.SOCKSPort > 65535 {
		return nil, errors.New("SOCKS port must be between 1 and 65535")
	}

	switch browser.Family {
	case FamilyFirefox:
		args := []string{"--profile", options.ProfileDir}
		if options.NewTab {
			args = append(args, "--new-tab")
		} else {
			args = append(args, "--new-instance")
		}
		if options.Private {
			args = append(args, "--private-window")
		}
		return append(args, options.URL), nil
	case FamilyChromium:
		args := []string{
			"--user-data-dir=" + options.ProfileDir,
			"--no-first-run",
			"--no-default-browser-check",
		}
		if options.Private {
			args = append(args, "--incognito")
		}
		if options.SOCKSPort > 0 {
			args = append(args,
				"--proxy-server=socks5://127.0.0.1:"+strconv.Itoa(options.SOCKSPort),
				"--host-resolver-rules=MAP * ~NOTFOUND, EXCLUDE 127.0.0.1",
			)
		}
		return append(args, options.URL), nil
	default:
		return nil, fmt.Errorf("unsupported browser family %q", browser.Family)
	}
}

func WriteFirefoxPreferences(profileDir string, socksPort int) error {
	if strings.TrimSpace(profileDir) == "" {
		return errors.New("browser profile directory is required")
	}
	if socksPort < 0 || socksPort > 65535 {
		return errors.New("SOCKS port must be between 1 and 65535")
	}
	if err := os.MkdirAll(profileDir, 0o700); err != nil {
		return fmt.Errorf("create Firefox profile: %w", err)
	}
	prefs := map[string]string{
		"browser.shell.checkDefaultBrowser": "false",
		"browser.startup.page":              "0",
		"network.proxy.type":                "0",
	}
	if socksPort > 0 {
		prefs["network.proxy.type"] = "1"
		prefs["network.proxy.socks"] = `"127.0.0.1"`
		prefs["network.proxy.socks_port"] = strconv.Itoa(socksPort)
		prefs["network.proxy.socks_version"] = "5"
		prefs["network.proxy.socks_remote_dns"] = "true"
	}
	keys := make([]string, 0, len(prefs))
	for key := range prefs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var content strings.Builder
	for _, key := range keys {
		fmt.Fprintf(&content, "user_pref(%q, %s);\n", key, prefs[key])
	}
	if err := os.WriteFile(filepath.Join(profileDir, "user.js"), []byte(content.String()), 0o600); err != nil {
		return fmt.Errorf("write Firefox preferences: %w", err)
	}
	return nil
}
