package xserver

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync"
)

var ErrUnsupported = errors.New("embedded X server is not supported on this platform")

type Display struct {
	Address string
	Cookie  []byte
	Screen  uint32

	releaseOnce sync.Once
	release     func()
}

func (d *Display) Close() error {
	if d == nil {
		return nil
	}
	d.releaseOnce.Do(func() {
		if d.release != nil {
			d.release()
		}
	})
	return nil
}

type serverProcess struct {
	address string
	cookie  []byte
	screen  uint32
	stop    func() error
}

type launchServer func(runtimeDir string) (*serverProcess, error)

type Manager struct {
	mu         sync.Mutex
	runtimeDir string
	supported  bool
	launch     launchServer
	server     *serverProcess
	references int
	closed     bool
}

func NewManager(dataDir string) *Manager {
	return newManager(filepath.Join(dataDir, "runtime", "x11"), platformXServerSupported(), startPlatformXServer)
}

func newManager(runtimeDir string, supported bool, launch launchServer) *Manager {
	return &Manager{
		runtimeDir: runtimeDir,
		supported:  supported,
		launch:     launch,
	}
}

func (m *Manager) Supported() bool {
	return m != nil && m.supported
}

func (m *Manager) Acquire() (*Display, error) {
	if m == nil || !m.supported {
		return nil, ErrUnsupported
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil, errors.New("X server manager is closed")
	}
	if m.server == nil {
		server, err := m.launch(m.runtimeDir)
		if err != nil {
			return nil, fmt.Errorf("start embedded X server: %w", err)
		}
		m.server = server
	}

	m.references++
	server := m.server
	return &Display{
		Address: server.address,
		Cookie:  append([]byte(nil), server.cookie...),
		Screen:  server.screen,
		release: m.release,
	}, nil
}

func (m *Manager) release() {
	m.mu.Lock()
	if m.references > 0 {
		m.references--
	}
	if m.references != 0 || m.server == nil {
		m.mu.Unlock()
		return
	}
	server := m.server
	m.server = nil
	if server.stop != nil {
		_ = server.stop()
	}
	m.mu.Unlock()
}

func (m *Manager) Close() error {
	if m == nil {
		return nil
	}

	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil
	}
	m.closed = true
	m.references = 0
	server := m.server
	m.server = nil
	if server != nil && server.stop != nil {
		err := server.stop()
		m.mu.Unlock()
		return err
	}
	m.mu.Unlock()
	return nil
}
