//go:build windows

package localterm

import (
	"fmt"
	"io"

	"github.com/signoredellarete/bashes/internal/remotessh"
)

type ShellOptions struct {
	Size   remotessh.TerminalSize
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	Term   string
}

type Shell struct{}

func StartShell(options ShellOptions) (*Shell, error) {
	return nil, fmt.Errorf("local shell is not supported on Windows yet")
}

func (s *Shell) Resize(size remotessh.TerminalSize) error {
	return fmt.Errorf("local shell is not supported on Windows yet")
}

func (s *Shell) Wait() error {
	return fmt.Errorf("local shell is not supported on Windows yet")
}

func (s *Shell) Close() error {
	return nil
}
