package plugin

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestStartProcessHandshakeCallAndStop(t *testing.T) {
	process, err := startProcess(context.Background(), processCandidate(t), "2.0.0-dev", time.Second, helperPluginCommand("success"))
	if err != nil {
		t.Fatalf("startProcess() error = %v", err)
	}
	if info := process.Info(); info.PluginID != "org.bashes.fixture" || len(info.Capabilities) != 1 {
		t.Fatalf("Info() = %+v", info)
	}
	var result struct {
		Value string `json:"value"`
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := process.Call(ctx, "diagnostic.echo", map[string]string{"value": "ping"}, &result); err != nil {
		t.Fatalf("Call() error = %v", err)
	}
	if result.Value != "ping" {
		t.Fatalf("Call() result = %+v", result)
	}
	if !strings.Contains(process.Diagnostics(), "fixture started") {
		t.Fatalf("Diagnostics() = %q", process.Diagnostics())
	}
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer stopCancel()
	if err := process.Stop(stopCtx); err != nil {
		t.Fatalf("Stop() error = %v; diagnostics=%q; wait=%v; state=%v", err, process.Diagnostics(), process.WaitError(), process.cmd.ProcessState)
	}
}

func TestStartProcessRejectsHandshakeMismatch(t *testing.T) {
	_, err := startProcess(context.Background(), processCandidate(t), "dev", time.Second, helperPluginCommand("mismatch"))
	if err == nil || !strings.Contains(err.Error(), "does not match manifest") {
		t.Fatalf("startProcess() error = %v, want manifest mismatch", err)
	}
}

func TestStartProcessTimesOutHandshake(t *testing.T) {
	_, err := startProcess(context.Background(), processCandidate(t), "dev", 20*time.Millisecond, helperPluginCommand("timeout"))
	if err == nil || !strings.Contains(err.Error(), "deadline exceeded") {
		t.Fatalf("startProcess() error = %v, want timeout", err)
	}
}

func TestProtocolFrameLimitAndEnvironmentFiltering(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader(strings.Repeat("x", maxProtocolFrameBytes+1) + "\n"))
	if _, err := readProtocolFrame(reader); err == nil {
		t.Fatal("readProtocolFrame() error = nil, want size limit")
	}
	environment := pluginEnvironment([]string{
		"PATH=/bin", "HOME=/home/test", "SSH_AUTH_SOCK=/secret/socket", "API_TOKEN=secret", "LC_ALL=C",
	})
	joined := strings.Join(environment, "\n")
	if strings.Contains(joined, "SSH_AUTH_SOCK") || strings.Contains(joined, "API_TOKEN") {
		t.Fatalf("pluginEnvironment() leaked sensitive variables: %v", environment)
	}
	if !strings.Contains(joined, "PATH=/bin") || !strings.Contains(joined, "BASHES_PLUGIN_PROTOCOL=") {
		t.Fatalf("pluginEnvironment() = %v", environment)
	}
}

func TestPluginHelperProcess(t *testing.T) {
	if os.Getenv("BASHES_TEST_PLUGIN_HELPER") != "1" {
		return
	}
	mode := os.Getenv("BASHES_TEST_PLUGIN_MODE")
	_, _ = fmt.Fprintln(os.Stderr, "fixture started")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var request Request
		if err := json.Unmarshal(scanner.Bytes(), &request); err != nil {
			os.Exit(2)
		}
		if request.Method == "hello" {
			if mode == "timeout" {
				time.Sleep(time.Second)
				continue
			}
			pluginID := "org.bashes.fixture"
			if mode == "mismatch" {
				pluginID = "org.bashes.other"
			}
			writeHelperResponse(request.ID, HelloResponse{
				APIVersion: HostAPIVersion, PluginID: pluginID, Version: "1.0.0",
				Capabilities: []Capability{CapabilityDiagnostic},
			})
			continue
		}
		if request.Method == "shutdown" {
			os.Exit(0)
		}
		var params map[string]string
		_ = json.Unmarshal(request.Params, &params)
		writeHelperResponse(request.ID, map[string]string{"value": params["value"]})
	}
	os.Exit(0)
}

func writeHelperResponse(id string, result any) {
	data, _ := json.Marshal(result)
	response := Response{JSONRPC: ProtocolVersion, ID: id, Result: data}
	encoded, _ := json.Marshal(response)
	_, _ = fmt.Fprintln(os.Stdout, string(encoded))
}

func helperPluginCommand(mode string) commandFactory {
	return func(string) *exec.Cmd {
		cmd := exec.Command(os.Args[0], "-test.run=^TestPluginHelperProcess$")
		cmd.Env = append(os.Environ(),
			"BASHES_TEST_PLUGIN_HELPER=1",
			"BASHES_TEST_PLUGIN_MODE="+mode,
		)
		return cmd
	}
}

func processCandidate(t *testing.T) Candidate {
	t.Helper()
	return Candidate{
		Directory:  t.TempDir(),
		Executable: os.Args[0],
		Manifest: Manifest{
			SchemaVersion: ManifestSchema, ID: "org.bashes.fixture", Name: "Fixture", Version: "1.0.0",
			APIVersion: HostAPIVersion, Capabilities: []Capability{CapabilityDiagnostic},
			Executables: map[string]string{"test/test": "fixture"},
		},
	}
}
