package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/pkg/sftp"
	"github.com/signoredellarete/bashes/internal/remotessh"
	"golang.org/x/crypto/ssh"
)

const (
	transferLocalRootID  = "/local"
	transferRemoteRootID = "/remote"
)

type FileTransferSessionInfo struct {
	SessionID  string `json:"sessionId"`
	ResourceID string `json:"resourceId"`
	Target     string `json:"target"`
	LocalRoot  string `json:"localRoot"`
	RemoteRoot string `json:"remoteRoot"`
}

type FileTransferEntry struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Size int64  `json:"size,omitempty"`
	Date string `json:"date,omitempty"`
	Lazy bool   `json:"lazy,omitempty"`
}

type FileTransferListInput struct {
	SessionID string `json:"sessionId"`
	ID        string `json:"id"`
}

type FileTransferCreateInput struct {
	SessionID string `json:"sessionId"`
	ParentID  string `json:"parentId"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Data      string `json:"data,omitempty"`
}

type FileTransferRenameInput struct {
	SessionID string `json:"sessionId"`
	ID        string `json:"id"`
	Name      string `json:"name"`
}

type FileTransferDeleteInput struct {
	SessionID string   `json:"sessionId"`
	IDs       []string `json:"ids"`
}

type FileTransferCopyInput struct {
	SessionID string   `json:"sessionId"`
	IDs       []string `json:"ids"`
	TargetID  string   `json:"targetId"`
	Move      bool     `json:"move"`
}

type FileTransferUploadInput struct {
	SessionID string   `json:"sessionId"`
	Paths     []string `json:"paths"`
	TargetID  string   `json:"targetId"`
	Move      bool     `json:"move"`
}

type FileTransferJobInfo struct {
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
	StartedAt        string   `json:"startedAt,omitempty"`
	FinishedAt       string   `json:"finishedAt,omitempty"`
}

type FileTransferCancelJobInput struct {
	JobID string `json:"jobId"`
}

type FileTransferDismissJobInput struct {
	JobID string `json:"jobId"`
}

type fileTransferSession struct {
	id         string
	resourceID string
	target     string
	localRoot  string
	remoteRoot string
	client     *ssh.Client
	sftp       *sftp.Client
	mu         sync.Mutex
}

type fileTransferJob struct {
	mu       sync.Mutex
	info     FileTransferJobInfo
	cancel   context.CancelFunc
	done     chan struct{}
	lastEmit time.Time
}

func (a *App) StartFileTransfer(input SSHSessionInput) (FileTransferSessionInfo, error) {
	resource, err := a.resourceByID(input.ResourceID)
	if err != nil {
		return FileTransferSessionInfo{}, err
	}

	input, err = a.prepareSessionInput(resource, input)
	if err != nil {
		return FileTransferSessionInfo{}, err
	}
	dialInput := a.resolveSessionKeyPath(input)
	client, err := a.dialResource(resource, dialInput, remotessh.DefaultTimeout)
	if err != nil {
		return FileTransferSessionInfo{}, err
	}
	if err := a.persistPasswordChoice(resource.ID, input); err != nil {
		client.Close()
		return FileTransferSessionInfo{}, err
	}

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		client.Close()
		return FileTransferSessionInfo{}, fmt.Errorf("start sftp client: %w", err)
	}

	localRoot, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(localRoot) == "" {
		sftpClient.Close()
		client.Close()
		return FileTransferSessionInfo{}, errors.New("resolve local home directory")
	}
	localRoot, _ = filepath.Abs(localRoot)

	remoteRoot, err := sftpClient.Getwd()
	if err != nil || strings.TrimSpace(remoteRoot) == "" {
		remoteRoot = "."
	}
	remoteRoot = path.Clean(remoteRoot)
	if resolved, resolveErr := sftpClient.RealPath(remoteRoot); resolveErr == nil {
		remoteRoot = path.Clean(resolved)
	}

	sessionID := fmt.Sprintf("files-%d", time.Now().UnixNano())
	session := &fileTransferSession{
		id:         sessionID,
		resourceID: resource.ID,
		target:     fmt.Sprintf("%s@%s:%d", resource.User, sshHost(resource), resource.Port),
		localRoot:  localRoot,
		remoteRoot: remoteRoot,
		client:     client,
		sftp:       sftpClient,
	}

	a.mu.Lock()
	a.transfers[sessionID] = session
	a.mu.Unlock()

	if auth := authPreferenceFromSessionInput(input); auth != nil && hasExplicitAuth(input) {
		_ = a.service.SetResourceAuth(resource.ID, *auth)
	}

	return session.info(), nil
}

func (a *App) CloseFileTransfer(sessionID string) error {
	sessionID = strings.TrimSpace(sessionID)
	jobs := a.cancelFileTransferJobsForSession(sessionID)
	for _, job := range jobs {
		select {
		case <-job.done:
		case <-time.After(3 * time.Second):
		}
	}

	a.mu.Lock()
	session := a.transfers[strings.TrimSpace(sessionID)]
	delete(a.transfers, strings.TrimSpace(sessionID))
	a.mu.Unlock()

	if session == nil {
		a.deleteFileTransferJobsForSession(sessionID)
		return nil
	}
	session.sftp.Close()
	session.client.Close()
	a.deleteFileTransferJobsForSession(sessionID)
	return nil
}

func (a *App) ListFileTransferFiles(input FileTransferListInput) ([]FileTransferEntry, error) {
	session, err := a.fileTransfer(input.SessionID)
	if err != nil {
		return nil, err
	}
	return session.list(input.ID)
}

func (a *App) CreateFileTransferItem(input FileTransferCreateInput) (FileTransferEntry, error) {
	session, err := a.fileTransfer(input.SessionID)
	if err != nil {
		return FileTransferEntry{}, err
	}
	return session.create(input)
}

func (a *App) RenameFileTransferItem(input FileTransferRenameInput) (FileTransferEntry, error) {
	session, err := a.fileTransfer(input.SessionID)
	if err != nil {
		return FileTransferEntry{}, err
	}
	return session.rename(input)
}

func (a *App) DeleteFileTransferItems(input FileTransferDeleteInput) error {
	session, err := a.fileTransfer(input.SessionID)
	if err != nil {
		return err
	}
	for _, id := range input.IDs {
		if err := session.remove(id); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) CopyFileTransferItems(input FileTransferCopyInput) error {
	session, err := a.fileTransfer(input.SessionID)
	if err != nil {
		return err
	}
	for _, id := range input.IDs {
		if err := session.copy(id, input.TargetID, input.Move); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) StartFileTransferCopyJob(input FileTransferCopyInput) (FileTransferJobInfo, error) {
	session, err := a.fileTransfer(input.SessionID)
	if err != nil {
		return FileTransferJobInfo{}, err
	}
	if len(input.IDs) == 0 {
		return FileTransferJobInfo{}, errors.New("no files selected for transfer")
	}
	if strings.TrimSpace(input.TargetID) == "" {
		return FileTransferJobInfo{}, errors.New("transfer target is required")
	}
	ctx, cancel := context.WithCancel(context.Background())
	job := &fileTransferJob{
		cancel: cancel,
		done:   make(chan struct{}),
		info: FileTransferJobInfo{
			JobID:      fmt.Sprintf("file-job-%d", time.Now().UnixNano()),
			SessionID:  session.id,
			ResourceID: session.resourceID,
			SourceIDs:  append([]string(nil), input.IDs...),
			TargetID:   input.TargetID,
			Move:       input.Move,
			Status:     "queued",
			StartedAt:  time.Now().Format(time.RFC3339),
		},
	}

	if err := a.reserveFileTransferJob(job); err != nil {
		cancel()
		return FileTransferJobInfo{}, err
	}
	a.emitFileTransferJob(job, true)

	go a.runFileTransferCopyJob(ctx, session, job, input)
	return job.snapshot(), nil
}

func (a *App) StartFileTransferUploadJob(input FileTransferUploadInput) (FileTransferJobInfo, error) {
	session, err := a.fileTransfer(input.SessionID)
	if err != nil {
		return FileTransferJobInfo{}, err
	}
	if len(input.Paths) == 0 {
		return FileTransferJobInfo{}, errors.New("no local files selected for upload")
	}
	if strings.TrimSpace(input.TargetID) == "" {
		return FileTransferJobInfo{}, errors.New("upload target is required")
	}
	if input.Move {
		return FileTransferJobInfo{}, errors.New("moving files dropped from outside Bashes is not supported")
	}

	paths := make([]string, 0, len(input.Paths))
	for _, value := range input.Paths {
		cleaned := strings.TrimSpace(value)
		if cleaned == "" {
			continue
		}
		abs, err := filepath.Abs(cleaned)
		if err != nil {
			return FileTransferJobInfo{}, err
		}
		paths = append(paths, abs)
	}
	if len(paths) == 0 {
		return FileTransferJobInfo{}, errors.New("no valid local files selected for upload")
	}

	ctx, cancel := context.WithCancel(context.Background())
	job := &fileTransferJob{
		cancel: cancel,
		done:   make(chan struct{}),
		info: FileTransferJobInfo{
			JobID:       fmt.Sprintf("file-job-%d", time.Now().UnixNano()),
			SessionID:   session.id,
			ResourceID:  session.resourceID,
			SourcePaths: paths,
			TargetID:    input.TargetID,
			Move:        input.Move,
			Status:      "queued",
			StartedAt:   time.Now().Format(time.RFC3339),
		},
	}

	if err := a.reserveFileTransferJob(job); err != nil {
		cancel()
		return FileTransferJobInfo{}, err
	}
	a.emitFileTransferJob(job, true)

	go a.runFileTransferUploadJob(ctx, session, job, paths, input)
	return job.snapshot(), nil
}

func (a *App) ListFileTransferJobs(sessionID string) ([]FileTransferJobInfo, error) {
	sessionID = strings.TrimSpace(sessionID)
	out := []FileTransferJobInfo{}
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, job := range a.transferJobs {
		info := job.snapshot()
		if info.SessionID == sessionID {
			out = append(out, info)
		}
	}
	return out, nil
}

func (a *App) CancelFileTransferJob(input FileTransferCancelJobInput) error {
	a.mu.Lock()
	job := a.transferJobs[strings.TrimSpace(input.JobID)]
	a.mu.Unlock()
	if job == nil {
		return nil
	}
	job.cancel()
	return nil
}

func (a *App) DismissFileTransferJob(input FileTransferDismissJobInput) error {
	jobID := strings.TrimSpace(input.JobID)
	a.mu.Lock()
	defer a.mu.Unlock()
	job := a.transferJobs[jobID]
	if job == nil {
		return nil
	}
	if isActiveFileTransferJob(job.snapshot().Status) {
		return errors.New("cannot dismiss an active file transfer")
	}
	delete(a.transferJobs, jobID)
	return nil
}

func (a *App) fileTransfer(sessionID string) (*fileTransferSession, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	session := a.transfers[strings.TrimSpace(sessionID)]
	if session == nil {
		return nil, fmt.Errorf("file transfer session %q not found", sessionID)
	}
	return session, nil
}

func (a *App) reserveFileTransferJob(job *fileTransferJob) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, existing := range a.transferJobs {
		info := existing.snapshot()
		if info.ResourceID == job.info.ResourceID && isActiveFileTransferJob(info.Status) {
			return fmt.Errorf("a file transfer is already running for %s", job.info.ResourceID)
		}
	}
	a.transferJobs[job.info.JobID] = job
	return nil
}

func (a *App) cancelFileTransferJobsForSession(sessionID string) []*fileTransferJob {
	a.mu.Lock()
	jobs := make([]*fileTransferJob, 0)
	for _, job := range a.transferJobs {
		if job.snapshot().SessionID == sessionID {
			jobs = append(jobs, job)
		}
	}
	a.mu.Unlock()

	for _, job := range jobs {
		job.cancel()
	}
	return jobs
}

func (a *App) deleteFileTransferJobsForSession(sessionID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for id, job := range a.transferJobs {
		if job.snapshot().SessionID == sessionID {
			delete(a.transferJobs, id)
		}
	}
}

func (a *App) runFileTransferCopyJob(ctx context.Context, session *fileTransferSession, job *fileTransferJob, input FileTransferCopyInput) {
	job.update(func(info *FileTransferJobInfo) {
		info.Status = "running"
	})
	a.emitFileTransferJob(job, true)

	err := session.copyJob(ctx, job, input, func(force bool) {
		a.emitFileTransferJob(job, force)
	})
	a.finishFileTransferJob(job, err)
}

func (a *App) runFileTransferUploadJob(ctx context.Context, session *fileTransferSession, job *fileTransferJob, sourcePaths []string, input FileTransferUploadInput) {
	job.update(func(info *FileTransferJobInfo) {
		info.Status = "running"
	})
	a.emitFileTransferJob(job, true)

	err := session.uploadJob(ctx, job, sourcePaths, input, func(force bool) {
		a.emitFileTransferJob(job, force)
	})
	a.finishFileTransferJob(job, err)
}

func (a *App) finishFileTransferJob(job *fileTransferJob, err error) {
	defer close(job.done)
	job.update(func(info *FileTransferJobInfo) {
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
	a.emitFileTransferJob(job, true)
	a.pruneFinishedFileTransferJobs(job.snapshot().ResourceID, 16)
}

func (a *App) pruneFinishedFileTransferJobs(resourceID string, keep int) {
	type retainedJob struct {
		id       string
		finished string
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	finished := make([]retainedJob, 0)
	for id, job := range a.transferJobs {
		info := job.snapshot()
		if info.ResourceID == resourceID && !isActiveFileTransferJob(info.Status) {
			finished = append(finished, retainedJob{id: id, finished: info.FinishedAt})
		}
	}
	sort.Slice(finished, func(i, j int) bool {
		return finished[i].finished > finished[j].finished
	})
	if keep < 0 || len(finished) <= keep {
		return
	}
	for _, item := range finished[keep:] {
		delete(a.transferJobs, item.id)
	}
}

func (a *App) emitFileTransferJob(job *fileTransferJob, force bool) {
	if !force && !job.shouldEmitProgress() {
		return
	}
	a.emitData("file-transfer:job", job.snapshot())
}

func isActiveFileTransferJob(status string) bool {
	return status == "queued" || status == "running"
}

func (j *fileTransferJob) snapshot() FileTransferJobInfo {
	j.mu.Lock()
	defer j.mu.Unlock()
	info := j.info
	info.SourceIDs = append([]string(nil), j.info.SourceIDs...)
	info.SourcePaths = append([]string(nil), j.info.SourcePaths...)
	return info
}

func (j *fileTransferJob) update(fn func(*FileTransferJobInfo)) {
	j.mu.Lock()
	defer j.mu.Unlock()
	fn(&j.info)
}

func (j *fileTransferJob) shouldEmitProgress() bool {
	j.mu.Lock()
	defer j.mu.Unlock()
	now := time.Now()
	if now.Sub(j.lastEmit) < 250*time.Millisecond {
		return false
	}
	j.lastEmit = now
	return true
}

func (s *fileTransferSession) info() FileTransferSessionInfo {
	return FileTransferSessionInfo{
		SessionID:  s.id,
		ResourceID: s.resourceID,
		Target:     s.target,
		LocalRoot:  s.localRoot,
		RemoteRoot: s.remoteRoot,
	}
}

func (s *fileTransferSession) list(id string) ([]FileTransferEntry, error) {
	scope, rel, err := parseTransferID(id)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	switch scope {
	case "local":
		dir, err := s.localPath(rel)
		if err != nil {
			return nil, err
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		out := make([]FileTransferEntry, 0, len(entries))
		for _, entry := range entries {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			out = append(out, entryFromInfo(scope, path.Join(rel, entry.Name()), info))
		}
		return out, nil
	case "remote":
		dir, err := s.remotePath(rel)
		if err != nil {
			return nil, err
		}
		entries, err := s.sftp.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		out := make([]FileTransferEntry, 0, len(entries))
		for _, info := range entries {
			out = append(out, entryFromInfo(scope, path.Join(rel, info.Name()), info))
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported file transfer scope %q", scope)
	}
}

func (s *fileTransferSession) create(input FileTransferCreateInput) (FileTransferEntry, error) {
	parentScope, parentRel, err := parseTransferID(input.ParentID)
	if err != nil {
		return FileTransferEntry{}, err
	}
	name, err := cleanFileName(input.Name)
	if err != nil {
		return FileTransferEntry{}, err
	}
	itemType := strings.TrimSpace(input.Type)
	if itemType == "" {
		itemType = "file"
	}
	rel := path.Join(parentRel, name)

	s.mu.Lock()
	defer s.mu.Unlock()

	switch parentScope {
	case "local":
		target, err := s.localPath(rel)
		if err != nil {
			return FileTransferEntry{}, err
		}
		if itemType == "folder" {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return FileTransferEntry{}, err
			}
		} else {
			if err := writeLocalFile(target, input.Data); err != nil {
				return FileTransferEntry{}, err
			}
		}
		info, err := os.Stat(target)
		if err != nil {
			return FileTransferEntry{}, err
		}
		return entryFromInfo(parentScope, rel, info), nil
	case "remote":
		target, err := s.remotePath(rel)
		if err != nil {
			return FileTransferEntry{}, err
		}
		if itemType == "folder" {
			if err := s.sftp.MkdirAll(target); err != nil {
				return FileTransferEntry{}, err
			}
		} else {
			if err := s.writeRemoteFile(target, input.Data); err != nil {
				return FileTransferEntry{}, err
			}
		}
		info, err := s.sftp.Stat(target)
		if err != nil {
			return FileTransferEntry{}, err
		}
		return entryFromInfo(parentScope, rel, info), nil
	default:
		return FileTransferEntry{}, fmt.Errorf("unsupported file transfer scope %q", parentScope)
	}
}

func (s *fileTransferSession) rename(input FileTransferRenameInput) (FileTransferEntry, error) {
	scope, rel, err := parseTransferID(input.ID)
	if err != nil {
		return FileTransferEntry{}, err
	}
	name, err := cleanFileName(input.Name)
	if err != nil {
		return FileTransferEntry{}, err
	}
	parent := path.Dir(rel)
	if parent == "." {
		parent = ""
	}
	newRel := path.Join(parent, name)

	s.mu.Lock()
	defer s.mu.Unlock()

	switch scope {
	case "local":
		oldPath, err := s.localPath(rel)
		if err != nil {
			return FileTransferEntry{}, err
		}
		newPath, err := s.localPath(newRel)
		if err != nil {
			return FileTransferEntry{}, err
		}
		if err := os.Rename(oldPath, newPath); err != nil {
			return FileTransferEntry{}, err
		}
		info, err := os.Stat(newPath)
		if err != nil {
			return FileTransferEntry{}, err
		}
		return entryFromInfo(scope, newRel, info), nil
	case "remote":
		oldPath, err := s.remotePath(rel)
		if err != nil {
			return FileTransferEntry{}, err
		}
		newPath, err := s.remotePath(newRel)
		if err != nil {
			return FileTransferEntry{}, err
		}
		if err := s.sftp.Rename(oldPath, newPath); err != nil {
			return FileTransferEntry{}, err
		}
		info, err := s.sftp.Stat(newPath)
		if err != nil {
			return FileTransferEntry{}, err
		}
		return entryFromInfo(scope, newRel, info), nil
	default:
		return FileTransferEntry{}, fmt.Errorf("unsupported file transfer scope %q", scope)
	}
}

func (s *fileTransferSession) remove(id string) error {
	scope, rel, err := parseTransferID(id)
	if err != nil {
		return err
	}
	if rel == "" {
		return errors.New("cannot delete file transfer root")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	switch scope {
	case "local":
		target, err := s.localPath(rel)
		if err != nil {
			return err
		}
		return os.RemoveAll(target)
	case "remote":
		target, err := s.remotePath(rel)
		if err != nil {
			return err
		}
		return s.removeRemote(target)
	default:
		return fmt.Errorf("unsupported file transfer scope %q", scope)
	}
}

func (s *fileTransferSession) copy(sourceID string, targetID string, move bool) error {
	sourceScope, sourceRel, err := parseTransferID(sourceID)
	if err != nil {
		return err
	}
	targetScope, targetRel, err := parseTransferID(targetID)
	if err != nil {
		return err
	}
	if sourceRel == "" {
		return errors.New("cannot transfer file transfer root")
	}

	name := path.Base(sourceRel)
	destinationRel := path.Join(targetRel, name)
	if err := validateCopyDestination(sourceScope, sourceRel, targetScope, destinationRel); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.copyItemAtomic(context.Background(), sourceScope, sourceRel, targetScope, destinationRel); err != nil {
		return err
	}
	if move {
		return s.removeUnlocked(sourceScope, sourceRel)
	}
	return nil
}

func (s *fileTransferSession) copyJob(ctx context.Context, job *fileTransferJob, input FileTransferCopyInput, emit func(bool)) error {
	targetScope, targetRel, err := parseTransferID(input.TargetID)
	if err != nil {
		return err
	}

	type plannedCopy struct {
		sourceID    string
		sourceScope string
		sourceRel   string
		targetScope string
		targetRel   string
	}

	plans := make([]plannedCopy, 0, len(input.IDs))
	for _, sourceID := range input.IDs {
		sourceScope, sourceRel, err := parseTransferID(sourceID)
		if err != nil {
			return err
		}
		if sourceRel == "" {
			return errors.New("cannot transfer file transfer root")
		}
		plans = append(plans, plannedCopy{
			sourceID:    sourceID,
			sourceScope: sourceScope,
			sourceRel:   sourceRel,
			targetScope: targetScope,
			targetRel:   path.Join(targetRel, path.Base(sourceRel)),
		})
	}
	for _, plan := range plans {
		if err := validateCopyDestination(plan.sourceScope, plan.sourceRel, plan.targetScope, plan.targetRel); err != nil {
			return err
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	var total int64
	for _, plan := range plans {
		size, err := s.transferSizeUnlocked(plan.sourceScope, plan.sourceRel)
		if err != nil {
			return err
		}
		total += size
	}
	job.update(func(info *FileTransferJobInfo) {
		info.TotalBytes = total
	})
	emit(true)

	for _, plan := range plans {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := s.copyItemAtomicWithProgress(ctx, job, plan.sourceScope, plan.sourceRel, plan.targetScope, plan.targetRel, emit); err != nil {
			return err
		}
		if input.Move {
			if err := s.removeUnlocked(plan.sourceScope, plan.sourceRel); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *fileTransferSession) uploadJob(ctx context.Context, job *fileTransferJob, sourcePaths []string, input FileTransferUploadInput, emit func(bool)) error {
	targetScope, targetRel, err := parseTransferID(input.TargetID)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	var total int64
	for _, sourcePath := range sourcePaths {
		size, err := localAbsoluteTransferSize(sourcePath)
		if err != nil {
			return err
		}
		total += size
	}
	job.update(func(info *FileTransferJobInfo) {
		info.TotalBytes = total
	})
	emit(true)

	for _, sourcePath := range sourcePaths {
		if err := ctx.Err(); err != nil {
			return err
		}
		name := filepath.Base(sourcePath)
		if err := s.copyAbsoluteLocalAtomicWithProgress(ctx, job, sourcePath, targetScope, path.Join(targetRel, name), emit); err != nil {
			return err
		}
	}
	return nil
}

func (s *fileTransferSession) copyItemAtomic(ctx context.Context, sourceScope, sourceRel, targetScope, targetRel string) error {
	return s.withAtomicDestination(targetScope, targetRel, func(tempRel string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		return s.copyUnlocked(sourceScope, sourceRel, targetScope, tempRel)
	})
}

func (s *fileTransferSession) copyItemAtomicWithProgress(ctx context.Context, job *fileTransferJob, sourceScope, sourceRel, targetScope, targetRel string, emit func(bool)) error {
	return s.withAtomicDestination(targetScope, targetRel, func(tempRel string) error {
		return s.copyUnlockedWithProgress(ctx, job, sourceScope, sourceRel, targetScope, tempRel, emit)
	})
}

func (s *fileTransferSession) copyAbsoluteLocalAtomicWithProgress(ctx context.Context, job *fileTransferJob, sourcePath, targetScope, targetRel string, emit func(bool)) error {
	return s.withAtomicDestination(targetScope, targetRel, func(tempRel string) error {
		return s.copyAbsoluteLocalWithProgress(ctx, job, sourcePath, targetScope, tempRel, emit)
	})
}

func (s *fileTransferSession) withAtomicDestination(scope, targetRel string, write func(tempRel string) error) error {
	exists, err := s.pathExists(scope, targetRel)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("destination already exists: %s", transferID(scope, targetRel))
	}

	tempRel := path.Join(
		path.Dir(targetRel),
		fmt.Sprintf(".%s.bashes-partial-%d", path.Base(targetRel), time.Now().UnixNano()),
	)
	committed := false
	defer func() {
		if !committed {
			_ = s.removeUnlocked(scope, tempRel)
		}
	}()

	if err := write(tempRel); err != nil {
		return err
	}
	exists, err = s.pathExists(scope, targetRel)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("destination appeared during transfer: %s", transferID(scope, targetRel))
	}
	if err := s.renamePath(scope, tempRel, targetRel); err != nil {
		return err
	}
	committed = true
	return nil
}

func (s *fileTransferSession) pathExists(scope, rel string) (bool, error) {
	_, err := s.stat(scope, rel)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, fs.ErrNotExist) || os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (s *fileTransferSession) renamePath(scope, oldRel, newRel string) error {
	switch scope {
	case "local":
		oldPath, err := s.localPath(oldRel)
		if err != nil {
			return err
		}
		newPath, err := s.localPath(newRel)
		if err != nil {
			return err
		}
		return os.Rename(oldPath, newPath)
	case "remote":
		oldPath, err := s.remotePath(oldRel)
		if err != nil {
			return err
		}
		newPath, err := s.remotePath(newRel)
		if err != nil {
			return err
		}
		return s.sftp.Rename(oldPath, newPath)
	default:
		return fmt.Errorf("unsupported file transfer scope %q", scope)
	}
}

func (s *fileTransferSession) copyUnlocked(sourceScope, sourceRel, targetScope, targetRel string) error {
	sourceIsDir, err := s.isDir(sourceScope, sourceRel)
	if err != nil {
		return err
	}
	if sourceIsDir {
		if err := s.mkdir(targetScope, targetRel); err != nil {
			return err
		}
		children, err := s.listUnlocked(sourceScope, sourceRel)
		if err != nil {
			return err
		}
		for _, child := range children {
			_, childRel, err := parseTransferID(child.ID)
			if err != nil {
				return err
			}
			if err := s.copyUnlocked(sourceScope, childRel, targetScope, path.Join(targetRel, path.Base(childRel))); err != nil {
				return err
			}
		}
		return nil
	}

	reader, err := s.openReader(sourceScope, sourceRel)
	if err != nil {
		return err
	}
	defer reader.Close()
	writer, err := s.openWriter(targetScope, targetRel)
	if err != nil {
		return err
	}
	defer writer.Close()
	_, err = io.Copy(writer, reader)
	return err
}

func (s *fileTransferSession) copyUnlockedWithProgress(ctx context.Context, job *fileTransferJob, sourceScope, sourceRel, targetScope, targetRel string, emit func(bool)) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	sourceIsDir, err := s.isDir(sourceScope, sourceRel)
	if err != nil {
		return err
	}
	job.update(func(info *FileTransferJobInfo) {
		info.Current = transferID(sourceScope, sourceRel)
	})
	emit(false)

	if sourceIsDir {
		if err := s.mkdir(targetScope, targetRel); err != nil {
			return err
		}
		children, err := s.listUnlocked(sourceScope, sourceRel)
		if err != nil {
			return err
		}
		for _, child := range children {
			_, childRel, err := parseTransferID(child.ID)
			if err != nil {
				return err
			}
			if err := s.copyUnlockedWithProgress(ctx, job, sourceScope, childRel, targetScope, path.Join(targetRel, path.Base(childRel)), emit); err != nil {
				return err
			}
		}
		return nil
	}

	reader, err := s.openReader(sourceScope, sourceRel)
	if err != nil {
		return err
	}
	defer reader.Close()
	writer, err := s.openWriter(targetScope, targetRel)
	if err != nil {
		return err
	}
	defer writer.Close()
	return copyWithProgress(ctx, writer, reader, func(written int64) {
		job.update(func(info *FileTransferJobInfo) {
			info.TransferredBytes += written
		})
		emit(false)
	})
}

func validateCopyDestination(sourceScope, sourceRel, targetScope, targetRel string) error {
	if sourceScope != targetScope {
		return nil
	}
	sourceRel = cleanRelative(sourceRel)
	targetRel = cleanRelative(targetRel)
	if sourceRel == "" {
		return errors.New("cannot transfer file transfer root")
	}
	if targetRel == sourceRel || strings.HasPrefix(targetRel, strings.TrimRight(sourceRel, "/")+"/") {
		return errors.New("cannot copy an item into itself")
	}
	return nil
}

func (s *fileTransferSession) copyAbsoluteLocalWithProgress(ctx context.Context, job *fileTransferJob, sourcePath, targetScope, targetRel string, emit func(bool)) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	info, err := os.Stat(sourcePath)
	if err != nil {
		return err
	}
	job.update(func(jobInfo *FileTransferJobInfo) {
		jobInfo.Current = sourcePath
	})
	emit(false)

	if info.IsDir() {
		if err := s.mkdir(targetScope, targetRel); err != nil {
			return err
		}
		entries, err := os.ReadDir(sourcePath)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if err := s.copyAbsoluteLocalWithProgress(ctx, job, filepath.Join(sourcePath, entry.Name()), targetScope, path.Join(targetRel, entry.Name()), emit); err != nil {
				return err
			}
		}
		return nil
	}

	reader, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer reader.Close()
	writer, err := s.openWriter(targetScope, targetRel)
	if err != nil {
		return err
	}
	defer writer.Close()
	return copyWithProgress(ctx, writer, reader, func(written int64) {
		job.update(func(info *FileTransferJobInfo) {
			info.TransferredBytes += written
		})
		emit(false)
	})
}

func (s *fileTransferSession) transferSizeUnlocked(scope, rel string) (int64, error) {
	sourceIsDir, err := s.isDir(scope, rel)
	if err != nil {
		return 0, err
	}
	if !sourceIsDir {
		info, err := s.stat(scope, rel)
		if err != nil {
			return 0, err
		}
		return info.Size(), nil
	}
	children, err := s.listUnlocked(scope, rel)
	if err != nil {
		return 0, err
	}
	var total int64
	for _, child := range children {
		_, childRel, err := parseTransferID(child.ID)
		if err != nil {
			return 0, err
		}
		size, err := s.transferSizeUnlocked(scope, childRel)
		if err != nil {
			return 0, err
		}
		total += size
	}
	return total, nil
}

func (s *fileTransferSession) listUnlocked(scope string, rel string) ([]FileTransferEntry, error) {
	switch scope {
	case "local":
		dir, err := s.localPath(rel)
		if err != nil {
			return nil, err
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		out := make([]FileTransferEntry, 0, len(entries))
		for _, entry := range entries {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			out = append(out, entryFromInfo(scope, path.Join(rel, entry.Name()), info))
		}
		return out, nil
	case "remote":
		dir, err := s.remotePath(rel)
		if err != nil {
			return nil, err
		}
		entries, err := s.sftp.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		out := make([]FileTransferEntry, 0, len(entries))
		for _, info := range entries {
			out = append(out, entryFromInfo(scope, path.Join(rel, info.Name()), info))
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported file transfer scope %q", scope)
	}
}

func (s *fileTransferSession) removeUnlocked(scope, rel string) error {
	switch scope {
	case "local":
		target, err := s.localPath(rel)
		if err != nil {
			return err
		}
		return os.RemoveAll(target)
	case "remote":
		target, err := s.remotePath(rel)
		if err != nil {
			return err
		}
		return s.removeRemote(target)
	default:
		return fmt.Errorf("unsupported file transfer scope %q", scope)
	}
}

func (s *fileTransferSession) isDir(scope, rel string) (bool, error) {
	switch scope {
	case "local":
		target, err := s.localPath(rel)
		if err != nil {
			return false, err
		}
		info, err := os.Stat(target)
		if err != nil {
			return false, err
		}
		return info.IsDir(), nil
	case "remote":
		target, err := s.remotePath(rel)
		if err != nil {
			return false, err
		}
		info, err := s.sftp.Stat(target)
		if err != nil {
			return false, err
		}
		return info.IsDir(), nil
	default:
		return false, fmt.Errorf("unsupported file transfer scope %q", scope)
	}
}

func (s *fileTransferSession) stat(scope, rel string) (fs.FileInfo, error) {
	switch scope {
	case "local":
		target, err := s.localPath(rel)
		if err != nil {
			return nil, err
		}
		return os.Stat(target)
	case "remote":
		target, err := s.remotePath(rel)
		if err != nil {
			return nil, err
		}
		return s.sftp.Stat(target)
	default:
		return nil, fmt.Errorf("unsupported file transfer scope %q", scope)
	}
}

func (s *fileTransferSession) mkdir(scope, rel string) error {
	switch scope {
	case "local":
		target, err := s.localPath(rel)
		if err != nil {
			return err
		}
		return os.MkdirAll(target, 0o755)
	case "remote":
		target, err := s.remotePath(rel)
		if err != nil {
			return err
		}
		return s.sftp.MkdirAll(target)
	default:
		return fmt.Errorf("unsupported file transfer scope %q", scope)
	}
}

func (s *fileTransferSession) openReader(scope, rel string) (io.ReadCloser, error) {
	switch scope {
	case "local":
		target, err := s.localPath(rel)
		if err != nil {
			return nil, err
		}
		return os.Open(target)
	case "remote":
		target, err := s.remotePath(rel)
		if err != nil {
			return nil, err
		}
		return s.sftp.Open(target)
	default:
		return nil, fmt.Errorf("unsupported file transfer scope %q", scope)
	}
}

func (s *fileTransferSession) openWriter(scope, rel string) (io.WriteCloser, error) {
	switch scope {
	case "local":
		target, err := s.localPath(rel)
		if err != nil {
			return nil, err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return nil, err
		}
		return os.Create(target)
	case "remote":
		target, err := s.remotePath(rel)
		if err != nil {
			return nil, err
		}
		if err := s.sftp.MkdirAll(path.Dir(target)); err != nil {
			return nil, err
		}
		return s.sftp.Create(target)
	default:
		return nil, fmt.Errorf("unsupported file transfer scope %q", scope)
	}
}

func (s *fileTransferSession) removeRemote(target string) error {
	info, err := s.sftp.Stat(target)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return s.sftp.Remove(target)
	}
	entries, err := s.sftp.ReadDir(target)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := s.removeRemote(path.Join(target, entry.Name())); err != nil {
			return err
		}
	}
	return s.sftp.RemoveDirectory(target)
}

func (s *fileTransferSession) localPath(rel string) (string, error) {
	cleaned := cleanRelative(rel)
	target := filepath.Join(s.localRoot, filepath.FromSlash(cleaned))
	absRoot, err := filepath.Abs(s.localRoot)
	if err != nil {
		return "", err
	}
	resolvedRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return "", err
	}
	resolvedTarget, err := resolveLocalPath(target)
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(resolvedRoot, resolvedTarget)
	if err != nil {
		return "", err
	}
	if pathEscapesRoot(relative) {
		return "", errors.New("local path escapes transfer root")
	}
	return resolvedTarget, nil
}

func (s *fileTransferSession) remotePath(rel string) (string, error) {
	cleaned := cleanRelative(rel)
	target := path.Clean(path.Join(s.remoteRoot, cleaned))
	if target != s.remoteRoot && !strings.HasPrefix(target, strings.TrimRight(s.remoteRoot, "/")+"/") {
		return "", errors.New("remote path escapes transfer root")
	}
	resolvedTarget, err := s.resolveRemotePath(target)
	if err != nil {
		return "", err
	}
	if resolvedTarget != s.remoteRoot && !strings.HasPrefix(resolvedTarget, strings.TrimRight(s.remoteRoot, "/")+"/") {
		return "", errors.New("remote path escapes transfer root through symlink")
	}
	return resolvedTarget, nil
}

func resolveLocalPath(target string) (string, error) {
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	current := absTarget
	var missing []string
	for {
		resolved, err := filepath.EvalSymlinks(current)
		if err == nil {
			for index := len(missing) - 1; index >= 0; index-- {
				resolved = filepath.Join(resolved, missing[index])
			}
			return filepath.Clean(resolved), nil
		}
		if !errors.Is(err, fs.ErrNotExist) && !os.IsNotExist(err) {
			return "", err
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", err
		}
		missing = append(missing, filepath.Base(current))
		current = parent
	}
}

func pathEscapesRoot(relative string) bool {
	return relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator))
}

func (s *fileTransferSession) resolveRemotePath(target string) (string, error) {
	current := target
	var missing []string
	for {
		resolved, err := s.sftp.RealPath(current)
		if err == nil {
			for index := len(missing) - 1; index >= 0; index-- {
				resolved = path.Join(resolved, missing[index])
			}
			return path.Clean(resolved), nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		parent := path.Dir(current)
		if parent == current {
			return "", err
		}
		missing = append(missing, path.Base(current))
		current = parent
	}
}

func parseTransferID(id string) (string, string, error) {
	id = path.Clean("/" + strings.TrimSpace(id))
	switch {
	case id == transferLocalRootID:
		return "local", "", nil
	case id == transferRemoteRootID:
		return "remote", "", nil
	case strings.HasPrefix(id, transferLocalRootID+"/"):
		return "local", strings.TrimPrefix(id, transferLocalRootID+"/"), nil
	case strings.HasPrefix(id, transferRemoteRootID+"/"):
		return "remote", strings.TrimPrefix(id, transferRemoteRootID+"/"), nil
	default:
		return "", "", fmt.Errorf("invalid file transfer id %q", id)
	}
}

func transferRootID(scope string) string {
	if scope == "remote" {
		return transferRemoteRootID
	}
	return transferLocalRootID
}

func transferID(scope, rel string) string {
	rel = cleanRelative(rel)
	if rel == "" {
		return transferRootID(scope)
	}
	return transferRootID(scope) + "/" + rel
}

func cleanRelative(rel string) string {
	rel = strings.TrimSpace(filepath.ToSlash(rel))
	rel = strings.TrimPrefix(path.Clean("/"+rel), "/")
	if rel == "." {
		return ""
	}
	return rel
}

func cleanFileName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\`) {
		return "", errors.New("invalid file name")
	}
	return name, nil
}

func entryFromInfo(scope, rel string, info fs.FileInfo) FileTransferEntry {
	entry := FileTransferEntry{
		ID:   transferID(scope, rel),
		Type: "file",
		Size: info.Size(),
		Date: info.ModTime().Format(time.RFC3339),
	}
	if info.IsDir() {
		entry.Type = "folder"
		entry.Lazy = true
		entry.Size = 0
	}
	return entry
}

func copyWithProgress(ctx context.Context, writer io.Writer, reader io.Reader, progress func(int64)) error {
	buffer := make([]byte, 1024*1024)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		n, readErr := reader.Read(buffer)
		if n > 0 {
			written, writeErr := writer.Write(buffer[:n])
			if written > 0 {
				progress(int64(written))
			}
			if writeErr != nil {
				return writeErr
			}
			if written != n {
				return io.ErrShortWrite
			}
		}
		if readErr == io.EOF {
			return nil
		}
		if readErr != nil {
			return readErr
		}
	}
}

func localAbsoluteTransferSize(sourcePath string) (int64, error) {
	info, err := os.Stat(sourcePath)
	if err != nil {
		return 0, err
	}
	if !info.IsDir() {
		return info.Size(), nil
	}
	var total int64
	err = filepath.WalkDir(sourcePath, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		total += info.Size()
		return nil
	})
	return total, err
}

func writeLocalFile(target string, data string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	decoded, err := decodeOptionalBase64(data)
	if err != nil {
		return err
	}
	return os.WriteFile(target, decoded, 0o644)
}

func (s *fileTransferSession) writeRemoteFile(target string, data string) error {
	if err := s.sftp.MkdirAll(path.Dir(target)); err != nil {
		return err
	}
	file, err := s.sftp.Create(target)
	if err != nil {
		return err
	}
	defer file.Close()
	decoded, err := decodeOptionalBase64(data)
	if err != nil {
		return err
	}
	_, err = file.Write(decoded)
	return err
}

func decodeOptionalBase64(data string) ([]byte, error) {
	if strings.TrimSpace(data) == "" {
		return []byte{}, nil
	}
	return base64.StdEncoding.DecodeString(data)
}
