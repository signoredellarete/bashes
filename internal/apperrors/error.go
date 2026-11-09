package apperrors

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const Prefix = "BASHES_ERROR:"

// Payload is the stable error contract exposed to the desktop frontend.
type Payload struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

// New creates an error that remains readable by humans and can be decoded by
// the frontend without matching the message text.
func New(code string, message string, details map[string]string) error {
	payload, err := json.Marshal(Payload{Code: code, Message: message, Details: details})
	if err != nil {
		return errors.New(message)
	}
	return fmt.Errorf("%s%s", Prefix, base64.RawURLEncoding.EncodeToString(payload))
}

// Decode extracts the first structured Bashes error embedded in err.
func Decode(err error) (Payload, bool) {
	if err == nil {
		return Payload{}, false
	}
	message := err.Error()
	index := strings.Index(message, Prefix)
	if index < 0 {
		return Payload{}, false
	}
	encoded := message[index+len(Prefix):]
	if end := strings.IndexAny(encoded, " \t\r\n"); end >= 0 {
		encoded = encoded[:end]
	}
	data, decodeErr := base64.RawURLEncoding.DecodeString(encoded)
	if decodeErr != nil {
		return Payload{}, false
	}
	var payload Payload
	if jsonErr := json.Unmarshal(data, &payload); jsonErr != nil {
		return Payload{}, false
	}
	if payload.Code == "" || payload.Message == "" {
		return Payload{}, false
	}
	return payload, true
}
