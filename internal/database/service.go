package database

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/signoredellarete/bashes/internal/domain"
	"github.com/signoredellarete/bashes/internal/fsutil"
	"github.com/signoredellarete/bashes/internal/importer"
	"github.com/signoredellarete/bashes/internal/store"
)

type Workspace interface {
	StoreSnapshot() (domain.Store, error)
	UpdateStore(func(domain.Store) (domain.Store, bool, error)) error
	ReplaceStore(domain.Store) error
}

type Service struct {
	workspace Workspace
}

func NewService(workspace Workspace) *Service {
	return &Service{workspace: workspace}
}

func (s *Service) Export(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("export path is required")
	}

	data, err := s.workspace.StoreSnapshot()
	if err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(domain.NormalizeStore(data), "", "  ")
	if err != nil {
		return fmt.Errorf("encode export json: %w", err)
	}
	return fsutil.WriteFileAtomic(path, append(encoded, '\n'), 0o600)
}

func (s *Service) Import(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("import path is required")
	}

	data, err := store.NewRepository(path).Load()
	if err != nil {
		return err
	}
	return s.workspace.ReplaceStore(data)
}

func (s *Service) ImportHostsFile(path string, user string, sshConfigPath string) (importer.HostsFileResult, error) {
	var result importer.HostsFileResult
	err := s.workspace.UpdateStore(func(data domain.Store) (domain.Store, bool, error) {
		nextResult, nextData, err := PrepareHostsFile(path, user, sshConfigPath, data)
		result = nextResult
		if err != nil {
			return data, false, err
		}
		return nextData, result.Imported > 0, nil
	})
	return result, err
}

func (s *Service) PreviewHostsFile(path string, user string, sshConfigPath string) (importer.HostsFileResult, domain.Store, error) {
	data, err := s.workspace.StoreSnapshot()
	if err != nil {
		return importer.HostsFileResult{Path: path, User: user}, domain.Store{}, err
	}
	return PrepareHostsFile(path, user, sshConfigPath, data)
}

func PrepareHostsFile(path string, user string, sshConfigPath string, data domain.Store) (importer.HostsFileResult, domain.Store, error) {
	return importer.PrepareHostsFile(path, user, sshConfigPath, data)
}
