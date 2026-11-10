package sshtunnel

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"

	runtimestate "github.com/signoredellarete/bashes/internal/runtime"
)

type Info struct {
	TunnelID      string `json:"tunnelId"`
	ResourceID    string `json:"resourceId"`
	Type          string `json:"type"`
	LocalHost     string `json:"localHost"`
	LocalPort     int    `json:"localPort"`
	LocalAddress  string `json:"localAddress"`
	Target        string `json:"target"`
	ForwardTarget string `json:"forwardTarget"`
	StartedAt     string `json:"startedAt"`
}

type ServeFunc func(context.Context) error

type Options struct {
	Info     Info
	Client   io.Closer
	Listener io.Closer
	Serve    ServeFunc
}

type Tunnel struct {
	info      Info
	client    io.Closer
	listener  io.Closer
	serve     ServeFunc
	ctx       context.Context
	cancel    context.CancelFunc
	closeOnce sync.Once
	runMu     sync.Mutex
	running   bool
	lifecycle *runtimestate.Lifecycle
}

type RunResult struct {
	ID            string
	Err           error
	Stopped       bool
	Authoritative bool
}

type Manager struct {
	mu       sync.RWMutex
	tunnels  map[string]*Tunnel
	observer runtimestate.Observer
}

func NewManager() *Manager {
	return NewManagerWithObserver(nil)
}

func NewManagerWithObserver(observer runtimestate.Observer) *Manager {
	return &Manager{tunnels: make(map[string]*Tunnel), observer: observer}
}

func (m *Manager) Register(options Options) (*Tunnel, error) {
	options.Info.TunnelID = strings.TrimSpace(options.Info.TunnelID)
	options.Info.ResourceID = strings.TrimSpace(options.Info.ResourceID)
	options.Info.Type = strings.TrimSpace(options.Info.Type)
	if options.Info.TunnelID == "" {
		return nil, errors.New("SSH tunnel id is required")
	}
	if options.Info.ResourceID == "" {
		return nil, errors.New("SSH tunnel resource id is required")
	}
	if options.Info.Type == "" {
		return nil, errors.New("SSH tunnel type is required")
	}
	if options.Client == nil || options.Listener == nil || options.Serve == nil {
		return nil, errors.New("SSH tunnel client, listener and serve function are required")
	}

	ctx, cancel := context.WithCancel(context.Background())
	lifecycle, err := runtimestate.NewLifecycle(runtimestate.Options{
		ID: options.Info.TunnelID, ResourceID: options.Info.ResourceID,
		Kind: "ssh-tunnel:" + options.Info.Type, Observer: m.observer,
	})
	if err != nil {
		cancel()
		return nil, err
	}
	tunnel := &Tunnel{
		info:      options.Info,
		client:    options.Client,
		listener:  options.Listener,
		serve:     options.Serve,
		ctx:       ctx,
		cancel:    cancel,
		lifecycle: lifecycle,
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.tunnels[tunnel.info.TunnelID]; exists {
		cancel()
		return nil, fmt.Errorf("SSH tunnel %q already exists", tunnel.info.TunnelID)
	}
	m.tunnels[tunnel.info.TunnelID] = tunnel
	return tunnel, nil
}

func (m *Manager) Run(tunnel *Tunnel) RunResult {
	if tunnel == nil {
		return RunResult{Err: errors.New("SSH tunnel is nil")}
	}
	tunnel.runMu.Lock()
	if tunnel.running {
		tunnel.runMu.Unlock()
		return RunResult{ID: tunnel.info.TunnelID, Err: errors.New("SSH tunnel is already running")}
	}
	tunnel.running = true
	tunnel.runMu.Unlock()
	tunnel.lifecycle.MarkRunning()

	err := tunnel.serve(tunnel.ctx)
	m.mu.Lock()
	authoritative := m.tunnels[tunnel.info.TunnelID] == tunnel
	if authoritative {
		delete(m.tunnels, tunnel.info.TunnelID)
	}
	m.mu.Unlock()
	stopped := tunnel.ctx.Err() != nil
	tunnel.close()
	if stopped {
		tunnel.lifecycle.Complete(nil)
	} else {
		tunnel.lifecycle.Complete(err)
	}
	return RunResult{
		ID:            tunnel.info.TunnelID,
		Err:           err,
		Stopped:       stopped,
		Authoritative: authoritative,
	}
}

func (m *Manager) Stop(id string) bool {
	id = strings.TrimSpace(id)
	m.mu.Lock()
	tunnel := m.tunnels[id]
	delete(m.tunnels, id)
	m.mu.Unlock()
	if tunnel == nil {
		return false
	}
	tunnel.lifecycle.RequestStop()
	tunnel.close()
	tunnel.lifecycle.Complete(nil)
	return true
}

func (m *Manager) Fail(tunnel *Tunnel, cause error) bool {
	if tunnel == nil {
		return false
	}
	m.mu.Lock()
	authoritative := m.tunnels[tunnel.info.TunnelID] == tunnel
	if authoritative {
		delete(m.tunnels, tunnel.info.TunnelID)
	}
	m.mu.Unlock()
	if authoritative {
		tunnel.close()
		tunnel.lifecycle.Complete(cause)
	}
	return authoritative
}

func (m *Manager) Context(tunnel *Tunnel) context.Context {
	if tunnel == nil {
		return context.Background()
	}
	return tunnel.ctx
}

func (m *Manager) Infos() []Info {
	m.mu.RLock()
	infos := make([]Info, 0, len(m.tunnels))
	for _, tunnel := range m.tunnels {
		infos = append(infos, tunnel.info)
	}
	m.mu.RUnlock()
	sort.Slice(infos, func(i, j int) bool { return infos[i].TunnelID < infos[j].TunnelID })
	return infos
}

func (m *Manager) IDs() []string {
	return m.idsForResource("")
}

func (m *Manager) IDsForResource(resourceID string) []string {
	return m.idsForResource(strings.TrimSpace(resourceID))
}

func (m *Manager) Snapshots() []runtimestate.Snapshot {
	m.mu.RLock()
	snapshots := make([]runtimestate.Snapshot, 0, len(m.tunnels))
	for _, tunnel := range m.tunnels {
		snapshots = append(snapshots, tunnel.lifecycle.Snapshot())
	}
	m.mu.RUnlock()
	sort.Slice(snapshots, func(i, j int) bool { return snapshots[i].ID < snapshots[j].ID })
	return snapshots
}

func (m *Manager) idsForResource(resourceID string) []string {
	m.mu.RLock()
	ids := make([]string, 0, len(m.tunnels))
	for id, tunnel := range m.tunnels {
		if resourceID == "" || tunnel.info.ResourceID == resourceID {
			ids = append(ids, id)
		}
	}
	m.mu.RUnlock()
	sort.Strings(ids)
	return ids
}

func (tunnel *Tunnel) close() {
	tunnel.closeOnce.Do(func() {
		tunnel.cancel()
		_ = tunnel.listener.Close()
		_ = tunnel.client.Close()
	})
}
