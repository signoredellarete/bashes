package filetransfer

import (
	"errors"
	"testing"

	"github.com/pkg/sftp"
	"github.com/signoredellarete/bashes/internal/apperrors"
)

func TestPresentErrorMapsSFTPStatus(t *testing.T) {
	tests := []struct {
		name        string
		code        uint32
		wantCode    string
		wantMessage string
	}{
		{name: "failure", code: 4, wantCode: "file_transfer_remote_failure", wantMessage: "The remote file operation failed. Check free space, storage quota, and permissions."},
		{name: "missing", code: 2, wantCode: "file_transfer_not_found", wantMessage: "The remote file or folder no longer exists."},
		{name: "permission", code: 3, wantCode: "file_transfer_permission_denied", wantMessage: "Permission was denied by the remote server."},
		{name: "connection lost", code: 7, wantCode: "file_transfer_connection_lost", wantMessage: "The remote file connection was lost."},
		{name: "unsupported", code: 8, wantCode: "file_transfer_unsupported", wantMessage: "The remote server does not support this file operation."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			presented := PresentError(&sftp.StatusError{Code: tt.code})
			payload, ok := apperrors.Decode(presented)
			if !ok {
				t.Fatalf("PresentError() = %v, want structured error", presented)
			}
			if payload.Code != tt.wantCode || payload.Message != tt.wantMessage {
				t.Fatalf("payload = %#v, want code %q and message %q", payload, tt.wantCode, tt.wantMessage)
			}
		})
	}
}

func TestPresentErrorPreservesPlainAndStructuredErrors(t *testing.T) {
	plain := errors.New("plain failure")
	if got := PresentError(plain); got != plain {
		t.Fatalf("PresentError(plain) = %v, want original", got)
	}
	structured := apperrors.New("existing", "Existing error", nil)
	if got := PresentError(structured); got != structured {
		t.Fatalf("PresentError(structured) = %v, want original", got)
	}
}
