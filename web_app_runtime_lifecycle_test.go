package main

import (
	"context"
	"errors"
	"net"
	"path/filepath"
	"testing"
	"time"
)

func TestBrowserLauncherDetachedOnlyForQuickSuccessfulExit(t *testing.T) {
	tests := []struct {
		name     string
		waitErr  error
		lifetime time.Duration
		want     bool
	}{
		{name: "quick handoff", lifetime: time.Second, want: true},
		{name: "launch failure", waitErr: errors.New("failed"), lifetime: time.Second},
		{name: "normal browser close", lifetime: browserLauncherDetachWindow},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := browserLauncherDetached(test.waitErr, test.lifetime); got != test.want {
				t.Fatalf("browserLauncherDetached() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestStopWebAppTunnelKeepsRuntimeAndListener(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	runtime := &webAppRuntime{
		id: "runtime-1", listener: listener, ctx: ctx, cancel: cancel,
		apps: map[string]struct{}{"app-1": {}},
		info: WebAppRuntimeInfo{
			RuntimeID: "runtime-1", WebAppID: "app-1", Tunnel: true,
			SOCKSPort: listenerPort(listener),
		},
	}
	runtime.tunnelActive.Store(true)
	app := NewApp(filepath.Join(t.TempDir(), "hosts.json"))
	app.webAppRuntimes[runtime.id] = runtime
	app.webAppRuntimeIDs["app-1"] = runtime.id
	t.Cleanup(func() { app.closeWebAppRuntime(runtime, false, "test cleanup") })

	info, err := app.StopWebAppTunnel("app-1")
	if err != nil {
		t.Fatalf("StopWebAppTunnel() error = %v", err)
	}
	if info.TunnelActive {
		t.Fatal("TunnelActive = true after stop")
	}
	if got := app.ListWebAppRuntimes(); len(got) != 1 {
		t.Fatalf("ListWebAppRuntimes() = %+v, want retained runtime", got)
	}
	if runtime.listener.Addr() == nil || listenerPort(runtime.listener) != info.SOCKSPort {
		t.Fatal("SOCKS listener or assigned port changed after stop")
	}
}
