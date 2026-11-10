package runtime

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

type State string

const (
	StateStarting State = "starting"
	StateRunning  State = "running"
	StateStopping State = "stopping"
	StateStopped  State = "stopped"
	StateFailed   State = "failed"
)

type Snapshot struct {
	ID         string
	ResourceID string
	Kind       string
	State      State
	StartedAt  time.Time
	UpdatedAt  time.Time
	Error      string
}

type Event struct {
	Previous State
	Current  Snapshot
}

type Observer func(Event)

type Options struct {
	ID         string
	ResourceID string
	Kind       string
	Observer   Observer
	Now        func() time.Time
}

type Lifecycle struct {
	mu       sync.RWMutex
	snapshot Snapshot
	observer Observer
	now      func() time.Time
}

func NewLifecycle(options Options) (*Lifecycle, error) {
	options.ID = strings.TrimSpace(options.ID)
	options.ResourceID = strings.TrimSpace(options.ResourceID)
	options.Kind = strings.TrimSpace(options.Kind)
	if options.ID == "" || options.ResourceID == "" || options.Kind == "" {
		return nil, errors.New("runtime id, resource id and kind are required")
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	now := options.Now()
	return &Lifecycle{
		snapshot: Snapshot{
			ID: options.ID, ResourceID: options.ResourceID, Kind: options.Kind,
			State: StateStarting, StartedAt: now, UpdatedAt: now,
		},
		observer: options.Observer,
		now:      options.Now,
	}, nil
}

func (lifecycle *Lifecycle) MarkRunning() bool {
	return lifecycle.transition(StateRunning, nil)
}

func (lifecycle *Lifecycle) RequestStop() bool {
	return lifecycle.transition(StateStopping, nil)
}

func (lifecycle *Lifecycle) Complete(err error) bool {
	if err != nil {
		return lifecycle.transition(StateFailed, err)
	}
	return lifecycle.transition(StateStopped, nil)
}

func (lifecycle *Lifecycle) Snapshot() Snapshot {
	lifecycle.mu.RLock()
	defer lifecycle.mu.RUnlock()
	return lifecycle.snapshot
}

func (lifecycle *Lifecycle) transition(next State, cause error) bool {
	lifecycle.mu.Lock()
	previous := lifecycle.snapshot.State
	if !allowed(previous, next) {
		lifecycle.mu.Unlock()
		return false
	}
	lifecycle.snapshot.State = next
	lifecycle.snapshot.UpdatedAt = lifecycle.now()
	if cause != nil {
		lifecycle.snapshot.Error = cause.Error()
	} else if next != StateFailed {
		lifecycle.snapshot.Error = ""
	}
	snapshot := lifecycle.snapshot
	observer := lifecycle.observer
	lifecycle.mu.Unlock()

	if observer != nil {
		observer(Event{Previous: previous, Current: snapshot})
	}
	return true
}

func allowed(current State, next State) bool {
	switch current {
	case StateStarting:
		return next == StateRunning || next == StateStopping || next == StateStopped || next == StateFailed
	case StateRunning:
		return next == StateStopping || next == StateStopped || next == StateFailed
	case StateStopping:
		return next == StateStopped || next == StateFailed
	case StateStopped, StateFailed:
		return false
	default:
		panic(fmt.Sprintf("unsupported runtime state %q", current))
	}
}
