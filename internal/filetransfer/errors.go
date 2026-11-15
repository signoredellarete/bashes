package filetransfer

import (
	"errors"

	"github.com/pkg/sftp"
	"github.com/signoredellarete/bashes/internal/apperrors"
)

// PresentError converts opaque SFTP status failures into stable errors that the
// desktop UI can display without exposing protocol-specific status strings.
func PresentError(err error) error {
	if err == nil {
		return nil
	}
	if _, structured := apperrors.Decode(err); structured {
		return err
	}

	var status *sftp.StatusError
	if !errors.As(err, &status) {
		return err
	}

	code := "file_transfer_remote_failure"
	message := "The remote file operation failed. Check free space, storage quota, and permissions."
	switch status.FxCode() {
	case sftp.ErrSSHFxNoSuchFile:
		code = "file_transfer_not_found"
		message = "The remote file or folder no longer exists."
	case sftp.ErrSSHFxPermissionDenied:
		code = "file_transfer_permission_denied"
		message = "Permission was denied by the remote server."
	case sftp.ErrSSHFxNoConnection, sftp.ErrSSHFxConnectionLost:
		code = "file_transfer_connection_lost"
		message = "The remote file connection was lost."
	case sftp.ErrSSHFxOpUnsupported:
		code = "file_transfer_unsupported"
		message = "The remote server does not support this file operation."
	case sftp.ErrSSHFxBadMessage:
		code = "file_transfer_invalid_response"
		message = "The remote server returned an invalid SFTP response."
	}
	return apperrors.New(code, message, map[string]string{"reason": err.Error()})
}
