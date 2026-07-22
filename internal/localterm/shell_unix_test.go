//go:build !windows

package localterm

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/signoredellarete/bashes/internal/remotessh"
)

func TestStartShellRunsInteractiveCommand(t *testing.T) {
	t.Setenv("SHELL", "/bin/sh")

	stdinReader, stdinWriter := io.Pipe()
	var output lockedBuffer
	shell, err := StartShell(ShellOptions{
		Size:   remotessh.TerminalSize{Cols: 80, Rows: 24},
		Stdin:  stdinReader,
		Stdout: &output,
	})
	if err != nil {
		t.Fatalf("StartShell() error = %v", err)
	}
	defer shell.Close()

	if _, err := fmt.Fprint(stdinWriter, "printf bashes-local-ok\\r"); err != nil {
		t.Fatalf("write shell input: %v", err)
	}

	deadline := time.After(3 * time.Second)
	for {
		if strings.Contains(output.String(), "bashes-local-ok") {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("shell output %q does not contain marker", output.String())
		default:
			time.Sleep(25 * time.Millisecond)
		}
	}
}

func TestLocalShellCommandUsesLoginShellOnUnixDesktops(t *testing.T) {
	darwin := localShellCommand("/bin/zsh", "darwin")
	if darwin.Args[0] != "-zsh" {
		t.Fatalf("darwin shell argv[0] = %q, want -zsh", darwin.Args[0])
	}

	linux := localShellCommand("/bin/bash", "linux")
	if linux.Args[0] != "-bash" {
		t.Fatalf("linux shell argv[0] = %q, want -bash", linux.Args[0])
	}

	other := localShellCommand("/bin/sh", "freebsd")
	if other.Args[0] != "/bin/sh" {
		t.Fatalf("other unix shell argv[0] = %q, want /bin/sh", other.Args[0])
	}
}

func TestShellResizeAndCloseAreSafeConcurrently(t *testing.T) {
	t.Setenv("SHELL", "/bin/sh")
	shell, err := StartShell(ShellOptions{
		Size: remotessh.TerminalSize{Cols: 80, Rows: 24},
	})
	if err != nil {
		t.Fatalf("StartShell() error = %v", err)
	}

	waitDone := make(chan error, 1)
	go func() {
		waitDone <- shell.Wait()
	}()

	var resizeGroup sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		resizeGroup.Add(1)
		go func(offset int) {
			defer resizeGroup.Done()
			for iteration := 0; iteration < 100; iteration++ {
				_ = shell.Resize(remotessh.TerminalSize{
					Cols: 80 + (iteration+offset)%20,
					Rows: 24 + (iteration+offset)%10,
				})
			}
		}(worker)
	}

	if err := shell.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	resizeGroup.Wait()
	select {
	case <-waitDone:
	case <-time.After(3 * time.Second):
		t.Fatal("Wait() did not return after Close()")
	}
	if err := shell.Resize(remotessh.TerminalSize{Cols: 80, Rows: 24}); err == nil {
		t.Fatal("Resize() succeeded after Close()")
	}
}

type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(data []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(data)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}
