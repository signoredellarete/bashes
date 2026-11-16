package filetransfer

import (
	"io/fs"
	"strconv"

	"github.com/signoredellarete/bashes/internal/apperrors"
)

const (
	DefaultMaxDepth   = 128
	DefaultMaxEntries = 250_000
)

// Traversal bounds recursive file operations independently from filesystem
// implementation details.
type Traversal struct {
	maxDepth   int
	maxEntries int
	entries    int
}

func NewTraversal(maxDepth, maxEntries int) *Traversal {
	return &Traversal{maxDepth: maxDepth, maxEntries: maxEntries}
}

func NewDefaultTraversal() *Traversal {
	return NewTraversal(DefaultMaxDepth, DefaultMaxEntries)
}

func (t *Traversal) Visit(depth int, itemPath string) error {
	if depth > t.maxDepth {
		return apperrors.New("file_transfer_depth_limit", "The folder tree is too deeply nested to transfer safely.", map[string]string{
			"path":  itemPath,
			"limit": strconv.Itoa(t.maxDepth),
		})
	}
	t.entries++
	if t.entries > t.maxEntries {
		return apperrors.New("file_transfer_entry_limit", "The transfer contains too many files or folders.", map[string]string{
			"path":  itemPath,
			"limit": strconv.Itoa(t.maxEntries),
		})
	}
	return nil
}

func ValidateTransferMode(itemPath string, mode fs.FileMode) error {
	if mode&fs.ModeSymlink != 0 {
		return apperrors.New("file_transfer_symlink_unsupported", "Symbolic links are not transferred.", map[string]string{"path": itemPath})
	}
	if mode.IsDir() || mode.IsRegular() {
		return nil
	}
	return apperrors.New("file_transfer_special_file_unsupported", "Sockets, devices, pipes, and other special files are not transferred.", map[string]string{"path": itemPath})
}
