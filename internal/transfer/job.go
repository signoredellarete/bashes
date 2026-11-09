package transfer

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type JobInfo struct {
	JobID            string   `json:"jobId"`
	SessionID        string   `json:"sessionId"`
	ResourceID       string   `json:"resourceId"`
	SourceIDs        []string `json:"sourceIds,omitempty"`
	SourcePaths      []string `json:"sourcePaths,omitempty"`
	TargetID         string   `json:"targetId"`
	Move             bool     `json:"move"`
	Status           string   `json:"status"`
	TotalBytes       int64    `json:"totalBytes"`
	TransferredBytes int64    `json:"transferredBytes"`
	Current          string   `json:"current,omitempty"`
	Error            string   `json:"error,omitempty"`
	Operation        string   `json:"operation,omitempty"`
	StartedAt        string   `json:"startedAt,omitempty"`
	FinishedAt       string   `json:"finishedAt,omitempty"`
}

type JobOptions struct {
	Info     JobInfo
	OpenPath string
}

type Job struct {
	mu       sync.Mutex
	info     JobInfo
	cancel   context.CancelFunc
	done     chan struct{}
	doneOnce sync.Once
	lastEmit time.Time
	openPath string
}

type Manager struct {
	mu   sync.RWMutex
	jobs map[string]*Job
}

func NewManager() *Manager {
	return &Manager{jobs: make(map[string]*Job)}
}

func (m *Manager) Start(options JobOptions) (*Job, context.Context, error) {
	options.Info.JobID = strings.TrimSpace(options.Info.JobID)
	options.Info.SessionID = strings.TrimSpace(options.Info.SessionID)
	options.Info.ResourceID = strings.TrimSpace(options.Info.ResourceID)
	if options.Info.JobID == "" || options.Info.SessionID == "" || options.Info.ResourceID == "" {
		return nil, nil, errors.New("file transfer job, session and resource IDs are required")
	}
	if options.Info.Status == "" {
		options.Info.Status = "queued"
	}

	ctx, cancel := context.WithCancel(context.Background())
	job := &Job{
		info:     options.Info,
		cancel:   cancel,
		done:     make(chan struct{}),
		openPath: options.OpenPath,
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.jobs[job.info.JobID]; exists {
		cancel()
		return nil, nil, fmt.Errorf("file transfer job %q already exists", job.info.JobID)
	}
	for _, existing := range m.jobs {
		info := existing.Snapshot()
		if info.ResourceID == job.info.ResourceID && IsActive(info.Status) {
			cancel()
			return nil, nil, fmt.Errorf("a file transfer is already running for %s", job.info.ResourceID)
		}
	}
	m.jobs[job.info.JobID] = job
	return job, ctx, nil
}

func (m *Manager) InfosForSession(sessionID string) []JobInfo {
	sessionID = strings.TrimSpace(sessionID)
	m.mu.RLock()
	infos := make([]JobInfo, 0)
	for _, job := range m.jobs {
		info := job.Snapshot()
		if info.SessionID == sessionID {
			infos = append(infos, info)
		}
	}
	m.mu.RUnlock()
	sort.Slice(infos, func(i, j int) bool { return infos[i].StartedAt < infos[j].StartedAt })
	return infos
}

func (m *Manager) Cancel(jobID string) bool {
	m.mu.RLock()
	job := m.jobs[strings.TrimSpace(jobID)]
	m.mu.RUnlock()
	if job == nil {
		return false
	}
	job.Cancel()
	return true
}

func (m *Manager) Dismiss(jobID string) error {
	jobID = strings.TrimSpace(jobID)
	m.mu.Lock()
	defer m.mu.Unlock()
	job := m.jobs[jobID]
	if job == nil {
		return nil
	}
	if IsActive(job.Snapshot().Status) {
		return errors.New("cannot dismiss an active file transfer")
	}
	delete(m.jobs, jobID)
	return nil
}

func (m *Manager) CancelForSession(sessionID string) []*Job {
	sessionID = strings.TrimSpace(sessionID)
	m.mu.RLock()
	jobs := make([]*Job, 0)
	for _, job := range m.jobs {
		if job.Snapshot().SessionID == sessionID {
			jobs = append(jobs, job)
		}
	}
	m.mu.RUnlock()
	for _, job := range jobs {
		job.Cancel()
	}
	return jobs
}

func (m *Manager) DeleteForSession(sessionID string) {
	sessionID = strings.TrimSpace(sessionID)
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, job := range m.jobs {
		if job.Snapshot().SessionID == sessionID {
			delete(m.jobs, id)
		}
	}
}

func (m *Manager) PruneFinished(resourceID string, keep int) {
	type retainedJob struct {
		id       string
		finished string
	}
	resourceID = strings.TrimSpace(resourceID)
	m.mu.Lock()
	defer m.mu.Unlock()
	finished := make([]retainedJob, 0)
	for id, job := range m.jobs {
		info := job.Snapshot()
		if info.ResourceID == resourceID && !IsActive(info.Status) {
			finished = append(finished, retainedJob{id: id, finished: info.FinishedAt})
		}
	}
	sort.Slice(finished, func(i, j int) bool { return finished[i].finished > finished[j].finished })
	if keep < 0 || len(finished) <= keep {
		return
	}
	for _, item := range finished[keep:] {
		delete(m.jobs, item.id)
	}
}

func IsActive(status string) bool {
	return status == "queued" || status == "running"
}

func (job *Job) Snapshot() JobInfo {
	job.mu.Lock()
	defer job.mu.Unlock()
	info := job.info
	info.SourceIDs = append([]string(nil), job.info.SourceIDs...)
	info.SourcePaths = append([]string(nil), job.info.SourcePaths...)
	return info
}

func (job *Job) Update(update func(*JobInfo)) {
	job.mu.Lock()
	defer job.mu.Unlock()
	update(&job.info)
}

func (job *Job) Finish(err error) {
	job.Update(func(info *JobInfo) {
		info.FinishedAt = time.Now().Format(time.RFC3339)
		switch {
		case err == nil && info.Status != "canceled":
			info.Status = "completed"
			info.Error = ""
		case errors.Is(err, context.Canceled) || info.Status == "canceled":
			info.Status = "canceled"
			if info.Error == "" {
				info.Error = "Transfer canceled."
			}
		default:
			info.Status = "failed"
			info.Error = err.Error()
		}
	})
	job.cancel()
	job.doneOnce.Do(func() { close(job.done) })
}

func (job *Job) Cancel() {
	job.cancel()
}

func (job *Job) Done() <-chan struct{} {
	return job.done
}

func (job *Job) OpenPath() string {
	return job.openPath
}

func (job *Job) ShouldEmitProgress() bool {
	job.mu.Lock()
	defer job.mu.Unlock()
	now := time.Now()
	if now.Sub(job.lastEmit) < 250*time.Millisecond {
		return false
	}
	job.lastEmit = now
	return true
}
