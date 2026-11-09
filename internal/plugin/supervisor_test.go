package plugin

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestSupervisorStartsCallsAndStopsPlugin(t *testing.T) {
	supervisor := newSupervisor(SupervisorOptions{
		HostVersion: "2.0.0-dev", HandshakeTimeout: time.Second,
	}, helperPluginCommand("success"))
	candidate := processCandidate(t)
	info, err := supervisor.Start(context.Background(), candidate)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if info.PluginID != candidate.Manifest.ID || len(supervisor.Infos()) != 1 {
		t.Fatalf("Start() info = %+v, running = %+v", info, supervisor.Infos())
	}
	if _, err := supervisor.Start(context.Background(), candidate); err == nil || !strings.Contains(err.Error(), "already running") {
		t.Fatalf("duplicate Start() error = %v", err)
	}

	var result map[string]string
	callCtx, callCancel := context.WithTimeout(context.Background(), time.Second)
	defer callCancel()
	if err := supervisor.Call(callCtx, candidate.Manifest.ID, "diagnostic.echo", map[string]string{"value": "test"}, &result); err != nil {
		t.Fatalf("Call() error = %v", err)
	}
	if result["value"] != "test" {
		t.Fatalf("Call() result = %v", result)
	}
	if diagnostics, found := supervisor.Diagnostics(candidate.Manifest.ID); !found || !strings.Contains(diagnostics, "fixture started") {
		t.Fatalf("Diagnostics() = %q, %v", diagnostics, found)
	}

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer stopCancel()
	if err := supervisor.Stop(stopCtx, candidate.Manifest.ID); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if len(supervisor.Infos()) != 0 {
		t.Fatalf("Infos() after stop = %+v", supervisor.Infos())
	}
}

func TestSupervisorRejectsCallsToStoppedPlugin(t *testing.T) {
	supervisor := NewSupervisor(SupervisorOptions{HostVersion: "dev"})
	if err := supervisor.Call(context.Background(), "missing", "diagnostic.echo", nil, nil); err == nil {
		t.Fatal("Call() error = nil")
	}
}
