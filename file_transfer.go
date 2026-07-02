package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
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

func (a *App) StartFileTransfer(input SSHSessionInput) (FileTransferSessionInfo, error) {
	resource, err := a.resourceByID(input.ResourceID)
	if err != nil {
		return FileTransferSessionInfo{}, err
	}

	input = applyAuthPreference(resource, input)
	dialInput := a.resolveSessionKeyPath(input)
	client, err := dialResource(resource, dialInput, remotessh.DefaultTimeout)
	if err != nil {
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

	return session.info(), nil
}

func (a *App) CloseFileTransfer(sessionID string) error {
	a.mu.Lock()
	session := a.transfers[strings.TrimSpace(sessionID)]
	delete(a.transfers, strings.TrimSpace(sessionID))
	a.mu.Unlock()

	if session == nil {
		return nil
	}
	session.sftp.Close()
	session.client.Close()
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

func (a *App) fileTransfer(sessionID string) (*fileTransferSession, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	session := a.transfers[strings.TrimSpace(sessionID)]
	if session == nil {
		return nil, fmt.Errorf("file transfer session %q not found", sessionID)
	}
	return session, nil
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

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.copyUnlocked(sourceScope, sourceRel, targetScope, destinationRel); err != nil {
		return err
	}
	if move {
		return s.removeUnlocked(sourceScope, sourceRel)
	}
	return nil
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
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(absRoot, absTarget)
	if err != nil {
		return "", err
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return "", errors.New("local path escapes transfer root")
	}
	return absTarget, nil
}

func (s *fileTransferSession) remotePath(rel string) (string, error) {
	cleaned := cleanRelative(rel)
	target := path.Clean(path.Join(s.remoteRoot, cleaned))
	if s.remoteRoot != "." && target != s.remoteRoot && !strings.HasPrefix(target, strings.TrimRight(s.remoteRoot, "/")+"/") {
		return "", errors.New("remote path escapes transfer root")
	}
	return target, nil
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
