package transfer

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
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
}

type SessionManager[T any] struct {
	mu       sync.RWMutex
	sessions map[string]*managedSession[T]
}

func NewSessionManager[T any]() *SessionManager[T] {
	return &SessionManager[T]{sessions: make(map[string]*managedSession[T])}
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
	session := &managedSession[T]{
		id: options.ID, resourceID: options.ResourceID, value: options.Value, close: options.Close,
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.sessions[session.id]; exists {
		return fmt.Errorf("file transfer session %q already exists", session.id)
	}
	m.sessions[session.id] = session
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
	session.closeOnce.Do(func() { session.close(session.value) })
	return true
}

func (m *SessionManager[T]) IDs() []string {
	return m.idsForResource("")
}

func (m *SessionManager[T]) IDsForResource(resourceID string) []string {
	return m.idsForResource(strings.TrimSpace(resourceID))
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
