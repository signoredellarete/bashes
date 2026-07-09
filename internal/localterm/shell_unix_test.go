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

func TestLocalShellCommandUsesLoginShellOnDarwinOnly(t *testing.T) {
	darwin := localShellCommand("/bin/zsh", "darwin")
	if darwin.Args[0] != "-zsh" {
		t.Fatalf("darwin shell argv[0] = %q, want -zsh", darwin.Args[0])
	}

	linux := localShellCommand("/bin/bash", "linux")
	if linux.Args[0] != "/bin/bash" {
		t.Fatalf("linux shell argv[0] = %q, want /bin/bash", linux.Args[0])
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
