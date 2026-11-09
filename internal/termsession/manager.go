package termsession

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"

	"github.com/signoredellarete/bashes/internal/remotessh"
)

type Kind string

const (
	KindSSH   Kind = "ssh"
	KindLocal Kind = "local"
)

type Shell interface {
	Resize(remotessh.TerminalSize) error
	Wait() error
	Close() error
}

type Options struct {
	ID         string
	ResourceID string
	Kind       Kind
	Client     io.Closer
	Shell      Shell
	Stdin      io.WriteCloser
}

type Session struct {
	id         string
	resourceID string
	kind       Kind
	client     io.Closer
	shell      Shell
	stdin      io.WriteCloser
	ctx        context.Context
	cancel     context.CancelFunc
	closeOnce  sync.Once
}

type WaitResult struct {
	Kind          Kind
	Err           error
	Stopped       bool
	Authoritative bool
}

type Manager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

func NewManager() *Manager {
	return &Manager{sessions: make(map[string]*Session)}
}

func (m *Manager) Register(options Options) (*Session, error) {
	options.ID = strings.TrimSpace(options.ID)
	options.ResourceID = strings.TrimSpace(options.ResourceID)
	if options.ID == "" {
		return nil, errors.New("terminal session id is required")
	}
	if options.ResourceID == "" {
		return nil, errors.New("terminal session resource id is required")
	}
	if options.Kind != KindSSH && options.Kind != KindLocal {
		return nil, fmt.Errorf("unsupported terminal session kind %q", options.Kind)
	}
	if options.Shell == nil || options.Stdin == nil {
		return nil, errors.New("terminal session shell and stdin are required")
	}
	if options.Kind == KindSSH && options.Client == nil {
		return nil, errors.New("SSH terminal session client is required")
	}

	ctx, cancel := context.WithCancel(context.Background())
	session := &Session{
		id:         options.ID,
		resourceID: options.ResourceID,
		kind:       options.Kind,
		client:     options.Client,
		shell:      options.Shell,
		stdin:      options.Stdin,
		ctx:        ctx,
		cancel:     cancel,
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.sessions[session.id]; exists {
		cancel()
		return nil, fmt.Errorf("terminal session %q already exists", session.id)
	}
	m.sessions[session.id] = session
	return session, nil
}

func (m *Manager) Write(id string, data string) error {
	session, err := m.session(id)
	if err != nil {
		return err
	}
	_, err = io.WriteString(session.stdin, data)
	return err
}

func (m *Manager) Resize(id string, size remotessh.TerminalSize) error {
	session, err := m.session(id)
	if err != nil {
		return err
	}
	return session.shell.Resize(size)
}

func (m *Manager) Stop(id string) (Kind, bool) {
	id = strings.TrimSpace(id)
	m.mu.Lock()
	session := m.sessions[id]
	delete(m.sessions, id)
	m.mu.Unlock()
	if session == nil {
		return "", false
	}
	session.close()
	return session.kind, true
}

func (m *Manager) Wait(session *Session) WaitResult {
	if session == nil {
		return WaitResult{Err: errors.New("terminal session is nil")}
	}
	err := session.shell.Wait()

	m.mu.Lock()
	authoritative := m.sessions[session.id] == session
	if authoritative {
		delete(m.sessions, session.id)
	}
	m.mu.Unlock()

	stopped := session.ctx.Err() != nil
	session.close()
	return WaitResult{
		Kind:          session.kind,
		Err:           err,
		Stopped:       stopped,
		Authoritative: authoritative,
	}
}

func (m *Manager) Context(session *Session) context.Context {
	if session == nil {
		return context.Background()
	}
	return session.ctx
}

func (m *Manager) IDs() []string {
	m.mu.RLock()
	ids := make([]string, 0, len(m.sessions))
	for id := range m.sessions {
		ids = append(ids, id)
	}
	m.mu.RUnlock()
	sort.Strings(ids)
	return ids
}

func (m *Manager) IDsForResource(resourceID string) []string {
	resourceID = strings.TrimSpace(resourceID)
	m.mu.RLock()
	ids := make([]string, 0)
	for id, session := range m.sessions {
		if session.resourceID == resourceID {
			ids = append(ids, id)
		}
	}
	m.mu.RUnlock()
	sort.Strings(ids)
	return ids
}

func (m *Manager) session(id string) (*Session, error) {
	id = strings.TrimSpace(id)
	m.mu.RLock()
	session := m.sessions[id]
	m.mu.RUnlock()
	if session == nil {
		return nil, fmt.Errorf("terminal session %q not found", id)
	}
	return session, nil
}

func (s *Session) close() {
	s.closeOnce.Do(func() {
		s.cancel()
		_ = s.stdin.Close()
		_ = s.shell.Close()
		if s.client != nil {
			_ = s.client.Close()
		}
	})
}
