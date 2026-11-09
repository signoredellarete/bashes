package plugin

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	DefaultHandshakeTimeout = 5 * time.Second
	DefaultRequestTimeout   = 30 * time.Second
	DefaultStopTimeout      = 3 * time.Second
	maxProtocolFrameBytes   = 1 << 20
	maxDiagnosticBytes      = 64 << 10
)

type ProcessInfo struct {
	PluginID     string
	Version      string
	Capabilities []Capability
	StartedAt    time.Time
}

type Process struct {
	info        ProcessInfo
	cmd         *exec.Cmd
	stdin       io.WriteCloser
	stdout      *bufio.Reader
	diagnostics *tailBuffer
	callMu      sync.Mutex
	writeMu     sync.Mutex
	resultMu    sync.RWMutex
	waitErr     error
	done        chan struct{}
	stopOnce    sync.Once
	requestID   atomic.Uint64
}

type commandFactory func(path string) *exec.Cmd

func StartProcess(ctx context.Context, candidate Candidate, hostVersion string, handshakeTimeout time.Duration) (*Process, error) {
	return startProcess(ctx, candidate, hostVersion, handshakeTimeout, defaultPluginCommand)
}

func startProcess(ctx context.Context, candidate Candidate, hostVersion string, handshakeTimeout time.Duration, command commandFactory) (*Process, error) {
	if !candidate.Available() {
		return nil, fmt.Errorf("plugin is unavailable: %w", candidate.Err)
	}
	if strings.TrimSpace(candidate.Executable) == "" {
		return nil, errors.New("plugin executable is required")
	}
	if command == nil {
		return nil, errors.New("plugin command factory is required")
	}
	if handshakeTimeout <= 0 {
		handshakeTimeout = DefaultHandshakeTimeout
	}

	cmd := command(candidate.Executable)
	if cmd == nil {
		return nil, errors.New("plugin command factory returned nil")
	}
	if cmd.Dir == "" {
		cmd.Dir = candidate.Directory
	}
	if cmd.Env == nil {
		cmd.Env = pluginEnvironment(os.Environ())
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("open plugin stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, fmt.Errorf("open plugin stdout: %w", err)
	}
	diagnostics := &tailBuffer{limit: maxDiagnosticBytes}
	cmd.Stderr = diagnostics
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return nil, fmt.Errorf("start plugin process: %w", err)
	}

	process := &Process{
		cmd: cmd, stdin: stdin, stdout: bufio.NewReader(stdout), diagnostics: diagnostics, done: make(chan struct{}),
	}
	hello, err := process.handshake(ctx, candidate.Manifest, hostVersion, handshakeTimeout)
	if err != nil {
		_ = process.cmd.Process.Kill()
		_ = process.stdin.Close()
		_ = process.cmd.Wait()
		return nil, err
	}
	process.info = ProcessInfo{
		PluginID: hello.PluginID, Version: hello.Version,
		Capabilities: append([]Capability(nil), hello.Capabilities...), StartedAt: time.Now(),
	}
	go process.wait()
	return process, nil
}

func (process *Process) Info() ProcessInfo {
	info := process.info
	info.Capabilities = append([]Capability(nil), process.info.Capabilities...)
	return info
}

func (process *Process) Diagnostics() string {
	return process.diagnostics.String()
}

func (process *Process) Done() <-chan struct{} {
	return process.done
}

func (process *Process) WaitError() error {
	process.resultMu.RLock()
	defer process.resultMu.RUnlock()
	return process.waitErr
}

func (process *Process) Call(ctx context.Context, method string, params any, result any) error {
	ctx, cancel := boundedContext(ctx, DefaultRequestTimeout)
	defer cancel()
	process.callMu.Lock()
	defer process.callMu.Unlock()
	select {
	case <-process.done:
		if err := process.WaitError(); err != nil {
			return fmt.Errorf("plugin process exited: %w", err)
		}
		return errors.New("plugin process exited")
	default:
	}

	id := fmt.Sprintf("request-%d", process.requestID.Add(1)+1)
	request, err := NewRequest(id, method, params)
	if err != nil {
		return err
	}
	process.writeMu.Lock()
	err = writeProtocolFrame(process.stdin, request)
	process.writeMu.Unlock()
	if err != nil {
		return fmt.Errorf("write plugin request: %w", err)
	}
	response, err := process.readResponse(ctx)
	if err != nil {
		_ = process.cmd.Process.Kill()
		return err
	}
	if response.ID != id {
		_ = process.cmd.Process.Kill()
		return fmt.Errorf("plugin response id %q does not match request id %q", response.ID, id)
	}
	return DecodeResult(response, result)
}

func (process *Process) Stop(ctx context.Context) error {
	ctx, cancel := boundedContext(ctx, DefaultStopTimeout)
	defer cancel()
	process.stopOnce.Do(func() {
		request, err := NewRequest(fmt.Sprintf("request-%d", process.requestID.Add(1)+1), "shutdown", struct{}{})
		process.writeMu.Lock()
		if err == nil {
			_ = writeProtocolFrame(process.stdin, request)
		}
		_ = process.stdin.Close()
		process.writeMu.Unlock()
	})
	select {
	case <-process.done:
		return process.WaitError()
	case <-ctx.Done():
		_ = process.cmd.Process.Kill()
		select {
		case <-process.done:
		case <-time.After(time.Second):
		}
		return ctx.Err()
	}
}

func (process *Process) handshake(ctx context.Context, manifest Manifest, hostVersion string, timeout time.Duration) (HelloResponse, error) {
	request, err := NewRequest("request-1", "hello", HelloRequest{
		APIVersion: HostAPIVersion, HostVersion: strings.TrimSpace(hostVersion),
	})
	if err != nil {
		return HelloResponse{}, err
	}
	if err := writeProtocolFrame(process.stdin, request); err != nil {
		return HelloResponse{}, fmt.Errorf("write plugin handshake: %w", err)
	}

	handshakeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	response, err := process.readResponse(handshakeCtx)
	if err != nil {
		return HelloResponse{}, fmt.Errorf("read plugin handshake: %w", err)
	}
	if response.ID != request.ID {
		return HelloResponse{}, fmt.Errorf("plugin handshake response id %q does not match %q", response.ID, request.ID)
	}
	var hello HelloResponse
	if err := DecodeResult(response, &hello); err != nil {
		return HelloResponse{}, err
	}
	if err := ValidateHello(manifest, hello); err != nil {
		return HelloResponse{}, err
	}
	return hello, nil
}

func (process *Process) readResponse(ctx context.Context) (Response, error) {
	type result struct {
		response Response
		err      error
	}
	read := make(chan result, 1)
	go func() {
		frame, err := readProtocolFrame(process.stdout)
		if err != nil {
			read <- result{err: err}
			return
		}
		var response Response
		if err := json.Unmarshal(frame, &response); err != nil {
			read <- result{err: fmt.Errorf("decode plugin response: %w", err)}
			return
		}
		read <- result{response: response}
	}()
	select {
	case <-ctx.Done():
		return Response{}, ctx.Err()
	case value := <-read:
		return value.response, value.err
	}
}

func (process *Process) wait() {
	err := process.cmd.Wait()
	process.resultMu.Lock()
	process.waitErr = err
	process.resultMu.Unlock()
	close(process.done)
}

func defaultPluginCommand(path string) *exec.Cmd {
	return exec.Command(path)
}

func writeProtocolFrame(writer io.Writer, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if len(data) > maxProtocolFrameBytes {
		return errors.New("plugin protocol frame exceeds 1 MiB")
	}
	data = append(data, '\n')
	_, err = writer.Write(data)
	return err
}

func readProtocolFrame(reader *bufio.Reader) ([]byte, error) {
	frame := make([]byte, 0, 4096)
	for {
		line, prefix, err := reader.ReadLine()
		if err != nil {
			return nil, err
		}
		if len(frame)+len(line) > maxProtocolFrameBytes {
			return nil, errors.New("plugin protocol frame exceeds 1 MiB")
		}
		frame = append(frame, line...)
		if !prefix {
			return frame, nil
		}
	}
}

func pluginEnvironment(environment []string) []string {
	allowed := map[string]struct{}{
		"HOME": {}, "LANG": {}, "PATH": {}, "SYSTEMROOT": {}, "TEMP": {}, "TMP": {}, "TMPDIR": {}, "WINDIR": {},
	}
	result := make([]string, 0, len(environment))
	for _, item := range environment {
		name, _, found := strings.Cut(item, "=")
		if !found {
			continue
		}
		upper := strings.ToUpper(name)
		if _, ok := allowed[upper]; ok || strings.HasPrefix(upper, "LC_") {
			result = append(result, item)
		}
	}
	result = append(result,
		"BASHES_PLUGIN_PROTOCOL="+ProtocolVersion,
		"BASHES_PLUGIN_PLATFORM="+runtime.GOOS+"/"+runtime.GOARCH,
	)
	return result
}

func boundedContext(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	if deadline, ok := parent.Deadline(); ok && time.Until(deadline) <= timeout {
		return context.WithCancel(parent)
	}
	return context.WithTimeout(parent, timeout)
}

type tailBuffer struct {
	mu    sync.Mutex
	limit int
	data  []byte
}

func (buffer *tailBuffer) Write(data []byte) (int, error) {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	written := len(data)
	buffer.data = append(buffer.data, data...)
	if len(buffer.data) > buffer.limit {
		buffer.data = append([]byte(nil), buffer.data[len(buffer.data)-buffer.limit:]...)
	}
	return written, nil
}

func (buffer *tailBuffer) String() string {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	return string(append([]byte(nil), buffer.data...))
}
