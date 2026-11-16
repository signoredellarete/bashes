package filetransfer

import (
	"io/fs"
	"testing"

	"github.com/signoredellarete/bashes/internal/apperrors"
)

func TestTraversalEnforcesDepthAndEntryLimits(t *testing.T) {
	depth := NewTraversal(1, 10)
	if err := depth.Visit(2, "deep"); errorCode(err) != "file_transfer_depth_limit" {
		t.Fatalf("Visit(depth) error = %v", err)
	}

	entries := NewTraversal(10, 1)
	if err := entries.Visit(0, "first"); err != nil {
		t.Fatalf("Visit(first) error = %v", err)
	}
	if err := entries.Visit(0, "second"); errorCode(err) != "file_transfer_entry_limit" {
		t.Fatalf("Visit(second) error = %v", err)
	}
}

func TestValidateTransferModeRejectsLinksAndSpecialFiles(t *testing.T) {
	if err := ValidateTransferMode("link", fs.ModeSymlink); errorCode(err) != "file_transfer_symlink_unsupported" {
		t.Fatalf("ValidateTransferMode(link) error = %v", err)
	}
	if err := ValidateTransferMode("socket", fs.ModeSocket); errorCode(err) != "file_transfer_special_file_unsupported" {
		t.Fatalf("ValidateTransferMode(socket) error = %v", err)
	}
	if err := ValidateTransferMode("directory", fs.ModeDir); err != nil {
		t.Fatalf("ValidateTransferMode(directory) error = %v", err)
	}
	if err := ValidateTransferMode("file", 0); err != nil {
		t.Fatalf("ValidateTransferMode(file) error = %v", err)
	}
}

func errorCode(err error) string {
	payload, _ := apperrors.Decode(err)
	return payload.Code
}
