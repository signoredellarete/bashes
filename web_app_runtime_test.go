//go:build !windows

package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/signoredellarete/bashes/internal/application"
	"github.com/signoredellarete/bashes/internal/domain"
)

func TestLaunchAndStopDirectWebAppRuntime(t *testing.T) {
	root := t.TempDir()
	browser := filepath.Join(root, "fake-firefox")
	if err := os.WriteFile(browser, []byte("#!/bin/sh\nsleep 30\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	app := NewApp(filepath.Join(root, "data", "hosts.json"))
	if _, err := app.SaveWebAppBrowserSettings(WebAppBrowserSettings{
		PreferredBrowserID: "firefox", CustomBrowserPath: browser,
	}); err != nil {
		t.Fatalf("SaveWebAppBrowserSettings() error = %v", err)
	}
	webApp, err := app.AddWebApp(application.WebAppInput{
		Name: "Dashboard", URL: "https://dashboard.internal",
		ConnectionMode: domain.WebAppConnectionDirect,
		BrowserMode:    domain.WebAppBrowserStandard,
		KeepCookies:    true, KeepLocalStorage: true, RestoreSession: true, ClearCacheOnClose: true,
	})
	if err != nil {
		t.Fatalf("AddWebApp() error = %v", err)
	}
	runtime, err := app.LaunchWebApp(WebAppLaunchInput{WebAppID: webApp.ID})
	if err != nil {
		t.Fatalf("LaunchWebApp() error = %v", err)
	}
	if runtime.RuntimeID == "" || runtime.Tunnel || runtime.BrowserID != "firefox" {
		t.Fatalf("Runtime = %+v", runtime)
	}
	if got := app.ListWebAppRuntimes(); len(got) != 1 || got[0].WebAppID != webApp.ID {
		t.Fatalf("ListWebAppRuntimes() = %+v", got)
	}
	if err := app.StopWebAppRuntime(runtime.RuntimeID); err != nil {
		t.Fatalf("StopWebAppRuntime() error = %v", err)
	}
	if got := app.ListWebAppRuntimes(); len(got) != 0 {
		t.Fatalf("Runtimes after stop = %+v", got)
	}
}

func TestQuickBrowserLauncherExitKeepsManagedRuntime(t *testing.T) {
	root := t.TempDir()
	browser := filepath.Join(root, "fake-firefox")
	if err := os.WriteFile(browser, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	app := NewApp(filepath.Join(root, "data", "hosts.json"))
	if _, err := app.SaveWebAppBrowserSettings(WebAppBrowserSettings{
		PreferredBrowserID: "firefox", CustomBrowserPath: browser,
	}); err != nil {
		t.Fatalf("SaveWebAppBrowserSettings() error = %v", err)
	}
	webApp, err := app.AddWebApp(application.WebAppInput{
		Name: "Dashboard", URL: "https://dashboard.internal",
		ConnectionMode: domain.WebAppConnectionDirect,
		BrowserMode:    domain.WebAppBrowserStandard,
	})
	if err != nil {
		t.Fatalf("AddWebApp() error = %v", err)
	}
	runtime, err := app.LaunchWebApp(WebAppLaunchInput{WebAppID: webApp.ID})
	if err != nil {
		t.Fatalf("LaunchWebApp() error = %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	if got := app.ListWebAppRuntimes(); len(got) != 1 {
		t.Fatalf("Runtimes after launcher handoff = %+v, want one", got)
	}
	if err := app.StopWebAppRuntime(runtime.RuntimeID); err != nil {
		t.Fatalf("StopWebAppRuntime() error = %v", err)
	}
}
