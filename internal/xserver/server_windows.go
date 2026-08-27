//go:build windows

package xserver

import (
	"crypto/rand"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	firstXDisplay = 100
	lastXDisplay  = 199
	xServerWait   = 5 * time.Second
)

func platformXServerSupported() bool {
	return true
}

func startPlatformXServer(runtimeDir string) (*serverProcess, error) {
	executable, err := findXServerExecutable()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(runtimeDir, 0o700); err != nil {
		return nil, fmt.Errorf("create X server runtime directory: %w", err)
	}

	display, address, err := availableDisplay()
	if err != nil {
		return nil, err
	}
	cookie := make([]byte, 16)
	if _, err := rand.Read(cookie); err != nil {
		return nil, fmt.Errorf("generate X server cookie: %w", err)
	}

	authorityPath := filepath.Join(runtimeDir, "Xauthority")
	authorityFile, err := os.OpenFile(authorityPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("create X server authority file: %w", err)
	}
	if err := writeXAuthority(authorityFile, strconv.Itoa(display), cookie); err != nil {
		_ = authorityFile.Close()
		_ = os.Remove(authorityPath)
		return nil, fmt.Errorf("write X server authority file: %w", err)
	}
	if err := authorityFile.Close(); err != nil {
		_ = os.Remove(authorityPath)
		return nil, fmt.Errorf("close X server authority file: %w", err)
	}

	logPath := filepath.Join(runtimeDir, "vcxsrv.log")
	_ = os.Remove(logPath)
	command := exec.Command(executable,
		fmt.Sprintf(":%d", display),
		"-multiwindow",
		"-clipboard",
		"-wgl",
		"-silent-dup-error",
		"-notrayicon",
		"-nohostintitle",
		"-listen", "tcp",
		"-auth", authorityPath,
		"-logfile", logPath,
		"-dpi", "auto",
	)
	command.Dir = filepath.Dir(executable)
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := command.Start(); err != nil {
		_ = os.Remove(authorityPath)
		return nil, fmt.Errorf("launch %s: %w", executable, err)
	}

	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	if err := waitForXServer(address, done); err != nil {
		_ = command.Process.Kill()
		select {
		case <-done:
		case <-time.After(time.Second):
		}
		_ = os.Remove(authorityPath)
		return nil, err
	}

	var stopOnce sync.Once
	stop := func() error {
		var stopErr error
		stopOnce.Do(func() {
			if command.Process != nil {
				if err := command.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
					stopErr = err
				}
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			_ = os.Remove(authorityPath)
		})
		return stopErr
	}

	return &serverProcess{
		address: address,
		cookie:  cookie,
		screen:  0,
		stop:    stop,
	}, nil
}

func findXServerExecutable() (string, error) {
	var candidates []string
	if configured := strings.TrimSpace(os.Getenv("BASHES_VCXSRV_PATH")); configured != "" {
		candidates = append(candidates, configured)
	}
	if executable, err := os.Executable(); err == nil {
		base := filepath.Dir(executable)
		candidates = append(candidates,
			filepath.Join(base, "xserver", "vcxsrv.exe"),
			filepath.Join(base, "vcxsrv", "vcxsrv.exe"),
		)
	}
	for _, root := range []string{os.Getenv("ProgramFiles"), os.Getenv("LOCALAPPDATA")} {
		if strings.TrimSpace(root) != "" {
			candidates = append(candidates,
				filepath.Join(root, "VcXsrv", "vcxsrv.exe"),
				filepath.Join(root, "Programs", "VcXsrv", "vcxsrv.exe"),
			)
		}
	}

	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", errors.New("VcXsrv is missing from the Bashes xserver directory")
}

func availableDisplay() (int, string, error) {
	for display := firstXDisplay; display <= lastXDisplay; display++ {
		address := net.JoinHostPort("127.0.0.1", strconv.Itoa(6000+display))
		listener, err := net.Listen("tcp4", address)
		if err != nil {
			continue
		}
		_ = listener.Close()
		return display, address, nil
	}
	return 0, "", errors.New("no free local X display is available")
}

func waitForXServer(address string, done <-chan error) error {
	deadline := time.Now().Add(xServerWait)
	for time.Now().Before(deadline) {
		select {
		case err := <-done:
			if err == nil {
				return errors.New("VcXsrv exited before accepting connections")
			}
			return fmt.Errorf("VcXsrv exited before accepting connections: %w", err)
		default:
		}

		connection, err := net.DialTimeout("tcp", address, 100*time.Millisecond)
		if err == nil {
			_ = connection.Close()
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("VcXsrv did not start within %s", xServerWait)
}
