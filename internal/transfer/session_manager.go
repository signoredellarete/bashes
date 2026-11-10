package transfer

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	runtimestate "github.com/signoredellarete/bashes/internal/runtime"
)

type SessionOptions[T any] struct {
	ID         string
	ResourceID string
	Value      T
	Close      func(T)
}

type managedSession[T any] struct {
	id         string
	resourceID string
	value      T
	close      func(T)
	closeOnce  sync.Once
	lifecycle  *runtimestate.Lifecycle
}

type SessionManager[T any] struct {
	mu       sync.RWMutex
	sessions map[string]*managedSession[T]
	observer runtimestate.Observer
}

func NewSessionManager[T any]() *SessionManager[T] {
	return NewSessionManagerWithObserver[T](nil)
}

func NewSessionManagerWithObserver[T any](observer runtimestate.Observer) *SessionManager[T] {
	return &SessionManager[T]{sessions: make(map[string]*managedSession[T]), observer: observer}
}

func (m *SessionManager[T]) Register(options SessionOptions[T]) error {
	options.ID = strings.TrimSpace(options.ID)
	options.ResourceID = strings.TrimSpace(options.ResourceID)
	if options.ID == "" {
		return errors.New("file transfer session id is required")
	}
	if options.ResourceID == "" {
		return errors.New("file transfer session resource id is required")
	}
	if options.Close == nil {
		return errors.New("file transfer session close function is required")
	}
	lifecycle, err := runtimestate.NewLifecycle(runtimestate.Options{
		ID: options.ID, ResourceID: options.ResourceID, Kind: "file-transfer-session", Observer: m.observer,
	})
	if err != nil {
		return err
	}
	session := &managedSession[T]{
		id: options.ID, resourceID: options.ResourceID, value: options.Value,
		close: options.Close, lifecycle: lifecycle,
	}
	m.mu.Lock()
	if _, exists := m.sessions[session.id]; exists {
		m.mu.Unlock()
		return fmt.Errorf("file transfer session %q already exists", session.id)
	}
	m.sessions[session.id] = session
	m.mu.Unlock()
	session.lifecycle.MarkRunning()
	return nil
}

func (m *SessionManager[T]) Get(id string) (T, bool) {
	id = strings.TrimSpace(id)
	m.mu.RLock()
	session := m.sessions[id]
	m.mu.RUnlock()
	if session == nil {
		var zero T
		return zero, false
	}
	return session.value, true
}

func (m *SessionManager[T]) Stop(id string) bool {
	id = strings.TrimSpace(id)
	m.mu.Lock()
	session := m.sessions[id]
	delete(m.sessions, id)
	m.mu.Unlock()
	if session == nil {
		return false
	}
	session.lifecycle.RequestStop()
	session.closeOnce.Do(func() { session.close(session.value) })
	session.lifecycle.Complete(nil)
	return true
}

func (m *SessionManager[T]) IDs() []string {
	return m.idsForResource("")
}

func (m *SessionManager[T]) IDsForResource(resourceID string) []string {
	return m.idsForResource(strings.TrimSpace(resourceID))
}

func (m *SessionManager[T]) Snapshots() []runtimestate.Snapshot {
	m.mu.RLock()
	snapshots := make([]runtimestate.Snapshot, 0, len(m.sessions))
	for _, session := range m.sessions {
		snapshots = append(snapshots, session.lifecycle.Snapshot())
	}
	m.mu.RUnlock()
	sort.Slice(snapshots, func(i, j int) bool { return snapshots[i].ID < snapshots[j].ID })
	return snapshots
}

func (m *SessionManager[T]) idsForResource(resourceID string) []string {
	m.mu.RLock()
	ids := make([]string, 0, len(m.sessions))
	for id, session := range m.sessions {
		if resourceID == "" || session.resourceID == resourceID {
			ids = append(ids, id)
		}
	}
	m.mu.RUnlock()
	sort.Strings(ids)
	return ids
}
