package plugin

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRequestAndHelloContract(t *testing.T) {
	request, err := NewRequest("request-1", "hello", HelloRequest{APIVersion: HostAPIVersion, HostVersion: "dev"})
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	if request.JSONRPC != ProtocolVersion || request.Method != "hello" {
		t.Fatalf("NewRequest() = %#v", request)
	}

	result, err := json.Marshal(HelloResponse{
		APIVersion:   HostAPIVersion,
		PluginID:     "org.bashes.fixture",
		Version:      "1.0.0",
		Capabilities: []Capability{CapabilityDiagnostic},
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	response := Response{JSONRPC: ProtocolVersion, ID: request.ID, Result: result}
	var hello HelloResponse
	if err := DecodeResult(response, &hello); err != nil {
		t.Fatalf("DecodeResult() error = %v", err)
	}
	manifest := Manifest{
		SchemaVersion: ManifestSchema,
		ID:            "org.bashes.fixture",
		Name:          "Fixture",
		Version:       "1.0.0",
		APIVersion:    HostAPIVersion,
		Capabilities:  []Capability{CapabilityDiagnostic},
		Executables:   map[string]string{"linux/amd64": "bin/plugin"},
	}
	if err := ValidateHello(manifest, hello); err != nil {
		t.Fatalf("ValidateHello() error = %v", err)
	}
}

func TestValidateHelloRejectsUndeclaredCapability(t *testing.T) {
	manifest := Manifest{ID: "org.bashes.fixture", Version: "1.0.0", APIVersion: HostAPIVersion, Capabilities: []Capability{CapabilityDiagnostic}}
	hello := HelloResponse{APIVersion: HostAPIVersion, PluginID: manifest.ID, Version: manifest.Version, Capabilities: []Capability{CapabilityRDP}}
	if err := ValidateHello(manifest, hello); err == nil || !strings.Contains(err.Error(), "undeclared") {
		t.Fatalf("ValidateHello() error = %v, want undeclared capability", err)
	}
}
