package remotessh

import (
	"fmt"
	"io"

	"golang.org/x/crypto/ssh"
)

type ShellOptions struct {
	Size   TerminalSize
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	Term   string
}

type Shell struct {
	session *ssh.Session
}

func StartShell(client *ssh.Client, options ShellOptions) (*Shell, error) {
	if client == nil {
		return nil, fmt.Errorf("ssh client is required")
	}

	session, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("create ssh session: %w", err)
	}

	if options.Stdin != nil {
		session.Stdin = options.Stdin
	}
	if options.Stdout != nil {
		session.Stdout = options.Stdout
	}
	if options.Stderr != nil {
		session.Stderr = options.Stderr
	}

	term := options.Term
	if term == "" {
		term = "xterm-256color"
	}

	rows := options.Size.Rows
	cols := options.Size.Cols
	if rows <= 0 {
		rows = 24
	}
	if cols <= 0 {
		cols = 80
	}

	if err := session.RequestPty(term, rows, cols, ssh.TerminalModes{}); err != nil {
		session.Close()
		return nil, fmt.Errorf("request ssh pty: %w", err)
	}
	if err := session.Shell(); err != nil {
		session.Close()
		return nil, fmt.Errorf("start ssh shell: %w", err)
	}

	return &Shell{session: session}, nil
}

func (s *Shell) Resize(size TerminalSize) error {
	if s == nil || s.session == nil {
		return fmt.Errorf("ssh shell is not started")
	}
	if size.Rows <= 0 || size.Cols <= 0 {
		return fmt.Errorf("terminal size must be positive")
	}
	return s.session.WindowChange(size.Rows, size.Cols)
}

func (s *Shell) Wait() error {
	if s == nil || s.session == nil {
		return fmt.Errorf("ssh shell is not started")
	}
	return s.session.Wait()
}

func (s *Shell) Close() error {
	if s == nil || s.session == nil {
		return nil
	}
	return s.session.Close()
}
