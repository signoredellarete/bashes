package plugin

import (
	"encoding/json"
	"errors"
	"fmt"
)

const ProtocolVersion = "2.0"

type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      string          `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      string          `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *ResponseError  `json:"error,omitempty"`
}

type ResponseError struct {
	Code    string          `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

type HelloRequest struct {
	APIVersion  int    `json:"apiVersion"`
	HostVersion string `json:"hostVersion"`
}

type HelloResponse struct {
	APIVersion   int          `json:"apiVersion"`
	PluginID     string       `json:"pluginId"`
	Version      string       `json:"version"`
	Capabilities []Capability `json:"capabilities"`
}

func NewRequest(id, method string, params any) (Request, error) {
	if id == "" {
		return Request{}, errors.New("plugin request id is required")
	}
	if method == "" {
		return Request{}, errors.New("plugin request method is required")
	}
	encoded, err := json.Marshal(params)
	if err != nil {
		return Request{}, fmt.Errorf("encode plugin request params: %w", err)
	}
	return Request{JSONRPC: ProtocolVersion, ID: id, Method: method, Params: encoded}, nil
}

func DecodeResult(response Response, target any) error {
	if response.JSONRPC != ProtocolVersion {
		return fmt.Errorf("unsupported plugin protocol %q", response.JSONRPC)
	}
	if response.ID == "" {
		return errors.New("plugin response id is required")
	}
	if response.Error != nil {
		if response.Error.Code == "" || response.Error.Message == "" {
			return errors.New("plugin returned an invalid error response")
		}
		return fmt.Errorf("plugin error %s: %s", response.Error.Code, response.Error.Message)
	}
	if len(response.Result) == 0 {
		return errors.New("plugin response result is required")
	}
	if err := json.Unmarshal(response.Result, target); err != nil {
		return fmt.Errorf("decode plugin response result: %w", err)
	}
	return nil
}

func ValidateHello(manifest Manifest, hello HelloResponse) error {
	if hello.APIVersion != HostAPIVersion || hello.APIVersion != manifest.APIVersion {
		return fmt.Errorf("plugin handshake API version %d is incompatible", hello.APIVersion)
	}
	if hello.PluginID != manifest.ID {
		return fmt.Errorf("plugin handshake id %q does not match manifest id %q", hello.PluginID, manifest.ID)
	}
	if hello.Version != manifest.Version {
		return fmt.Errorf("plugin handshake version %q does not match manifest version %q", hello.Version, manifest.Version)
	}
	declared := make(map[Capability]struct{}, len(manifest.Capabilities))
	for _, capability := range manifest.Capabilities {
		declared[capability] = struct{}{}
	}
	seen := make(map[Capability]struct{}, len(hello.Capabilities))
	for _, capability := range hello.Capabilities {
		if _, duplicate := seen[capability]; duplicate {
			return fmt.Errorf("plugin handshake repeats capability %q", capability)
		}
		seen[capability] = struct{}{}
		if _, allowed := declared[capability]; !allowed {
			return fmt.Errorf("plugin handshake advertises undeclared capability %q", capability)
		}
	}
	return nil
}
