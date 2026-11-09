package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/signoredellarete/bashes/internal/plugin"
)

const (
	fixturePluginID      = "org.bashes.diagnostic-fixture"
	fixturePluginVersion = "1.0.0"
	fixtureMaxFrameBytes = 1 << 20
)

var errShutdown = errors.New("plugin shutdown requested")

func main() {
	if err := run(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "diagnostic fixture:", err)
		os.Exit(1)
	}
}

func run(input io.Reader, output io.Writer) error {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 4096), fixtureMaxFrameBytes)
	encoder := json.NewEncoder(output)
	for scanner.Scan() {
		var request plugin.Request
		if err := json.Unmarshal(scanner.Bytes(), &request); err != nil {
			return fmt.Errorf("decode request: %w", err)
		}
		response, err := handle(request)
		if errors.Is(err, errShutdown) {
			return nil
		}
		if err != nil {
			response = plugin.Response{
				JSONRPC: plugin.ProtocolVersion, ID: request.ID,
				Error: &plugin.ResponseError{Code: "invalid_request", Message: err.Error()},
			}
		}
		if err := encoder.Encode(response); err != nil {
			return fmt.Errorf("encode response: %w", err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read request: %w", err)
	}
	return nil
}

func handle(request plugin.Request) (plugin.Response, error) {
	if request.JSONRPC != plugin.ProtocolVersion || request.ID == "" {
		return plugin.Response{}, errors.New("unsupported protocol or missing request id")
	}
	var result any
	switch request.Method {
	case "hello":
		var hello plugin.HelloRequest
		if err := json.Unmarshal(request.Params, &hello); err != nil {
			return plugin.Response{}, fmt.Errorf("decode hello: %w", err)
		}
		if hello.APIVersion != plugin.HostAPIVersion {
			return plugin.Response{}, fmt.Errorf("unsupported host API version %d", hello.APIVersion)
		}
		result = plugin.HelloResponse{
			APIVersion: plugin.HostAPIVersion, PluginID: fixturePluginID,
			Version: fixturePluginVersion, Capabilities: []plugin.Capability{plugin.CapabilityDiagnostic},
		}
	case "diagnostic.echo":
		var params map[string]any
		if err := json.Unmarshal(request.Params, &params); err != nil {
			return plugin.Response{}, fmt.Errorf("decode echo: %w", err)
		}
		result = params
	case "shutdown":
		return plugin.Response{}, errShutdown
	default:
		return plugin.Response{}, fmt.Errorf("unsupported method %q", request.Method)
	}
	data, err := json.Marshal(result)
	if err != nil {
		return plugin.Response{}, err
	}
	return plugin.Response{JSONRPC: plugin.ProtocolVersion, ID: request.ID, Result: data}, nil
}
