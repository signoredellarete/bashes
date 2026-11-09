package plugin

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type SupervisorOptions struct {
	HostVersion      string
	HandshakeTimeout time.Duration
}

type Supervisor struct {
	mu        sync.RWMutex
	options   SupervisorOptions
	command   commandFactory
	processes map[string]*Process
	starting  map[string]struct{}
}

func NewSupervisor(options SupervisorOptions) *Supervisor {
	return newSupervisor(options, defaultPluginCommand)
}

func newSupervisor(options SupervisorOptions, command commandFactory) *Supervisor {
	if options.HandshakeTimeout <= 0 {
		options.HandshakeTimeout = DefaultHandshakeTimeout
	}
	return &Supervisor{
		options: options, command: command,
		processes: make(map[string]*Process), starting: make(map[string]struct{}),
	}
}

func (supervisor *Supervisor) Start(ctx context.Context, candidate Candidate) (ProcessInfo, error) {
	pluginID := strings.TrimSpace(candidate.Manifest.ID)
	if pluginID == "" {
		return ProcessInfo{}, errors.New("plugin id is required")
	}
	supervisor.mu.Lock()
	if _, running := supervisor.processes[pluginID]; running {
		supervisor.mu.Unlock()
		return ProcessInfo{}, fmt.Errorf("plugin %q is already running", pluginID)
	}
	if _, starting := supervisor.starting[pluginID]; starting {
		supervisor.mu.Unlock()
		return ProcessInfo{}, fmt.Errorf("plugin %q is already starting", pluginID)
	}
	supervisor.starting[pluginID] = struct{}{}
	supervisor.mu.Unlock()

	process, err := startProcess(ctx, candidate, supervisor.options.HostVersion, supervisor.options.HandshakeTimeout, supervisor.command)
	supervisor.mu.Lock()
	delete(supervisor.starting, pluginID)
	if err == nil {
		supervisor.processes[pluginID] = process
	}
	supervisor.mu.Unlock()
	if err != nil {
		return ProcessInfo{}, err
	}
	go supervisor.watch(pluginID, process)
	return process.Info(), nil
}

func (supervisor *Supervisor) Call(ctx context.Context, pluginID string, method string, params any, result any) error {
	process, found := supervisor.process(pluginID)
	if !found {
		return fmt.Errorf("plugin %q is not running", strings.TrimSpace(pluginID))
	}
	return process.Call(ctx, method, params, result)
}

func (supervisor *Supervisor) Stop(ctx context.Context, pluginID string) error {
	pluginID = strings.TrimSpace(pluginID)
	supervisor.mu.Lock()
	process := supervisor.processes[pluginID]
	if process != nil {
		delete(supervisor.processes, pluginID)
	}
	supervisor.mu.Unlock()
	if process == nil {
		return nil
	}
	return process.Stop(ctx)
}

func (supervisor *Supervisor) Shutdown(ctx context.Context) error {
	supervisor.mu.Lock()
	processes := make([]*Process, 0, len(supervisor.processes))
	for _, process := range supervisor.processes {
		processes = append(processes, process)
	}
	supervisor.processes = make(map[string]*Process)
	supervisor.mu.Unlock()

	var failures []error
	for _, process := range processes {
		if err := process.Stop(ctx); err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

func (supervisor *Supervisor) Infos() []ProcessInfo {
	supervisor.mu.RLock()
	infos := make([]ProcessInfo, 0, len(supervisor.processes))
	for _, process := range supervisor.processes {
		infos = append(infos, process.Info())
	}
	supervisor.mu.RUnlock()
	sort.Slice(infos, func(i, j int) bool { return infos[i].PluginID < infos[j].PluginID })
	return infos
}

func (supervisor *Supervisor) Diagnostics(pluginID string) (string, bool) {
	process, found := supervisor.process(pluginID)
	if !found {
		return "", false
	}
	return process.Diagnostics(), true
}

func (supervisor *Supervisor) process(pluginID string) (*Process, bool) {
	supervisor.mu.RLock()
	process := supervisor.processes[strings.TrimSpace(pluginID)]
	supervisor.mu.RUnlock()
	return process, process != nil
}

func (supervisor *Supervisor) watch(pluginID string, process *Process) {
	<-process.Done()
	supervisor.mu.Lock()
	if supervisor.processes[pluginID] == process {
		delete(supervisor.processes, pluginID)
	}
	supervisor.mu.Unlock()
}
