package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/signoredellarete/bashes/internal/plugin"
)

func TestRunHandshakeEchoAndShutdown(t *testing.T) {
	hello, err := plugin.NewRequest("request-1", "hello", plugin.HelloRequest{
		APIVersion: plugin.HostAPIVersion, HostVersion: "dev",
	})
	if err != nil {
		t.Fatal(err)
	}
	echo, err := plugin.NewRequest("request-2", "diagnostic.echo", map[string]string{"value": "test"})
	if err != nil {
		t.Fatal(err)
	}
	shutdown, err := plugin.NewRequest("request-3", "shutdown", struct{}{})
	if err != nil {
		t.Fatal(err)
	}
	input := strings.Builder{}
	for _, request := range []plugin.Request{hello, echo, shutdown} {
		data, marshalErr := json.Marshal(request)
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		input.Write(data)
		input.WriteByte('\n')
	}
	output := bytes.Buffer{}
	if err := run(strings.NewReader(input.String()), &output); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	decoder := json.NewDecoder(&output)
	var helloResponse plugin.Response
	if err := decoder.Decode(&helloResponse); err != nil {
		t.Fatalf("decode hello response: %v", err)
	}
	var helloResult plugin.HelloResponse
	if err := plugin.DecodeResult(helloResponse, &helloResult); err != nil {
		t.Fatalf("hello response: %v", err)
	}
	if helloResult.PluginID != fixturePluginID {
		t.Fatalf("plugin id = %q", helloResult.PluginID)
	}
	var echoResponse plugin.Response
	if err := decoder.Decode(&echoResponse); err != nil {
		t.Fatalf("decode echo response: %v", err)
	}
	var echoResult map[string]string
	if err := plugin.DecodeResult(echoResponse, &echoResult); err != nil {
		t.Fatalf("echo response: %v", err)
	}
	if echoResult["value"] != "test" {
		t.Fatalf("echo result = %v", echoResult)
	}
}

func TestHandleRejectsUnknownMethod(t *testing.T) {
	request, err := plugin.NewRequest("request-1", "unknown", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := handle(request); err == nil {
		t.Fatal("handle() error = nil")
	}
}
