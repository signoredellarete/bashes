//go:build !windows

package localterm

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/creack/pty"
	"github.com/signoredellarete/bashes/internal/remotessh"
)

type ShellOptions struct {
	Size   remotessh.TerminalSize
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	Term   string
}

type Shell struct {
	cmd  *exec.Cmd
	file *os.File
	once sync.Once
}

func StartShell(options ShellOptions) (*Shell, error) {
	shellPath := defaultShellPath()
	if shellPath == "" {
		return nil, fmt.Errorf("local shell not found")
	}

	cmd := exec.Command(shellPath)
	cmd.Env = localShellEnv(options.Term)
	if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
		cmd.Dir = home
	}

	rows := options.Size.Rows
	cols := options.Size.Cols
	if rows <= 0 {
		rows = 24
	}
	if cols <= 0 {
		cols = 80
	}

	file, err := pty.StartWithSize(cmd, &pty.Winsize{
		Rows: uint16(rows),
		Cols: uint16(cols),
	})
	if err != nil {
		return nil, fmt.Errorf("start local pty: %w", err)
	}

	shell := &Shell{cmd: cmd, file: file}
	if options.Stdin != nil {
		go func() {
			_, _ = io.Copy(file, options.Stdin)
			_ = shell.Close()
		}()
	}
	if options.Stdout != nil {
		go func() {
			_, _ = io.Copy(options.Stdout, file)
		}()
	}
	return shell, nil
}

func (s *Shell) Resize(size remotessh.TerminalSize) error {
	if s == nil || s.file == nil {
		return fmt.Errorf("local shell is not started")
	}
	if size.Rows <= 0 || size.Cols <= 0 {
		return fmt.Errorf("terminal size must be positive")
	}
	return pty.Setsize(s.file, &pty.Winsize{
		Rows: uint16(size.Rows),
		Cols: uint16(size.Cols),
	})
}

func (s *Shell) Wait() error {
	if s == nil || s.cmd == nil {
		return fmt.Errorf("local shell is not started")
	}
	err := s.cmd.Wait()
	_ = s.closeFile()
	return err
}

func (s *Shell) Close() error {
	if s == nil {
		return nil
	}
	var err error
	s.once.Do(func() {
		err = s.closeFile()
		if s.cmd != nil && s.cmd.Process != nil {
			_ = s.cmd.Process.Kill()
		}
	})
	return err
}

func (s *Shell) closeFile() error {
	if s.file == nil {
		return nil
	}
	err := s.file.Close()
	s.file = nil
	return err
}

func defaultShellPath() string {
	candidates := []string{os.Getenv("SHELL"), "/bin/bash", "/bin/zsh", "/bin/sh"}
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" || strings.ContainsAny(candidate, "\x00\r\n") {
			continue
		}
		if !filepath.IsAbs(candidate) {
			continue
		}
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
			return candidate
		}
	}
	return ""
}

func localShellEnv(term string) []string {
	if strings.TrimSpace(term) == "" {
		term = "xterm-256color"
	}
	env := os.Environ()
	env = append(env, "TERM="+term)
	env = append(env, "COLORTERM=truecolor")
	return env
}
