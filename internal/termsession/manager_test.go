package termsession

import (
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/signoredellarete/bashes/internal/remotessh"
)

func TestManagerLifecycle(t *testing.T) {
	shell := newFakeShell()
	stdinReader, stdinWriter := io.Pipe()
	defer stdinReader.Close()
	manager := NewManager()
	session, err := manager.Register(Options{
		ID: "session-1", ResourceID: "host-1", Kind: KindLocal,
		Shell: shell, Stdin: stdinWriter,
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if got := manager.IDsForResource("host-1"); len(got) != 1 || got[0] != "session-1" {
		t.Fatalf("IDsForResource() = %v", got)
	}

	waited := make(chan WaitResult, 1)
	go func() { waited <- manager.Wait(session) }()
	if kind, stopped := manager.Stop("session-1"); !stopped || kind != KindLocal {
		t.Fatalf("Stop() = %q, %v", kind, stopped)
	}
	result := <-waited
	if !result.Stopped || result.Authoritative {
		t.Fatalf("Wait() = %+v, want stopped non-authoritative result", result)
	}
	if shell.closeCount() != 1 {
		t.Fatalf("shell close count = %d, want 1", shell.closeCount())
	}
	if _, stopped := manager.Stop("session-1"); stopped {
		t.Fatal("second Stop() reported an active session")
	}
}

func TestManagerWaitReportsNaturalExit(t *testing.T) {
	shell := newFakeShell()
	stdinReader, stdinWriter := io.Pipe()
	defer stdinReader.Close()
	manager := NewManager()
	session, err := manager.Register(Options{ID: "session-1", ResourceID: "host-1", Kind: KindLocal, Shell: shell, Stdin: stdinWriter})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	shell.finish(io.EOF)
	result := manager.Wait(session)
	if result.Stopped || !result.Authoritative || !errors.Is(result.Err, io.EOF) {
		t.Fatalf("Wait() = %+v", result)
	}
}

func TestManagerRejectsDuplicateIDs(t *testing.T) {
	manager := NewManager()
	register := func() error {
		_, writer := io.Pipe()
		_, err := manager.Register(Options{ID: "duplicate", ResourceID: "host", Kind: KindLocal, Shell: newFakeShell(), Stdin: writer})
		return err
	}
	if err := register(); err != nil {
		t.Fatalf("first Register() error = %v", err)
	}
	if err := register(); err == nil {
		t.Fatal("second Register() error = nil")
	}
	manager.Stop("duplicate")
}

type fakeShell struct {
	mu       sync.Mutex
	done     chan error
	closes   int
	closeOne sync.Once
}

func newFakeShell() *fakeShell {
	return &fakeShell{done: make(chan error, 1)}
}

func (s *fakeShell) Resize(remotessh.TerminalSize) error { return nil }

func (s *fakeShell) Wait() error { return <-s.done }

func (s *fakeShell) Close() error {
	s.mu.Lock()
	s.closes++
	s.mu.Unlock()
	s.closeOne.Do(func() { s.finish(nil) })
	return nil
}

func (s *fakeShell) finish(err error) {
	select {
	case s.done <- err:
	case <-time.After(time.Second):
	}
}

func (s *fakeShell) closeCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closes
}
