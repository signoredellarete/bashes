package xserver

import (
	"bytes"
	"errors"
	"testing"
)

func TestManagerSharesServerUntilLastDisplayCloses(t *testing.T) {
	starts := 0
	stops := 0
	manager := newManager(t.TempDir(), true, func(string) (*serverProcess, error) {
		starts++
		return &serverProcess{
			address: "127.0.0.1:6100",
			cookie:  bytes.Repeat([]byte{0x42}, 16),
			stop: func() error {
				stops++
				return nil
			},
		}, nil
	})

	first, err := manager.Acquire()
	if err != nil {
		t.Fatalf("Acquire(first) error = %v", err)
	}
	second, err := manager.Acquire()
	if err != nil {
		t.Fatalf("Acquire(second) error = %v", err)
	}
	if starts != 1 {
		t.Fatalf("server starts = %d, want 1", starts)
	}

	_ = first.Close()
	if stops != 0 {
		t.Fatalf("server stops after first close = %d, want 0", stops)
	}
	_ = second.Close()
	_ = second.Close()
	if stops != 1 {
		t.Fatalf("server stops after last close = %d, want 1", stops)
	}
}

func TestManagerCloseStopsServerAndPreventsAcquire(t *testing.T) {
	stops := 0
	manager := newManager(t.TempDir(), true, func(string) (*serverProcess, error) {
		return &serverProcess{
			address: "127.0.0.1:6100",
			cookie:  make([]byte, 16),
			stop: func() error {
				stops++
				return nil
			},
		}, nil
	})

	if _, err := manager.Acquire(); err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	if err := manager.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if stops != 1 {
		t.Fatalf("server stops = %d, want 1", stops)
	}
	if _, err := manager.Acquire(); err == nil {
		t.Fatal("Acquire() after Close error = nil")
	}
}

func TestManagerReportsUnsupportedPlatform(t *testing.T) {
	manager := newManager(t.TempDir(), false, nil)
	if _, err := manager.Acquire(); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("Acquire() error = %v, want ErrUnsupported", err)
	}
}
