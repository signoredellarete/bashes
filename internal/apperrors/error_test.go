package apperrors

import (
	"fmt"
	"reflect"
	"testing"
)

func TestNewAndDecodeRoundTrip(t *testing.T) {
	want := Payload{
		Code:    "ssh_host_key_unknown",
		Message: "SSH host key is unknown",
		Details: map[string]string{"host": "example.test:22", "fingerprint": "SHA256:test"},
	}

	err := New(want.Code, want.Message, want.Details)
	got, ok := Decode(fmt.Errorf("connect failed: %w", err))
	if !ok {
		t.Fatalf("Decode() did not recognize %q", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Decode() = %#v, want %#v", got, want)
	}
}

func TestDecodeRejectsMalformedErrors(t *testing.T) {
	tests := []error{
		nil,
		fmt.Errorf("plain error"),
		fmt.Errorf("%snot-base64", Prefix),
		fmt.Errorf("%se30", Prefix),
	}
	for _, err := range tests {
		if payload, ok := Decode(err); ok {
			t.Fatalf("Decode(%v) = %#v, true; want false", err, payload)
		}
	}
}
