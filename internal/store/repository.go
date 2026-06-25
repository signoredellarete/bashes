package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/signoredellarete/bashes/internal/domain"
)

type Repository struct {
	path string
}

func NewRepository(path string) *Repository {
	return &Repository{path: path}
}

func (r *Repository) Load() (domain.Store, error) {
	data, err := os.ReadFile(r.path)
	if errors.Is(err, os.ErrNotExist) {
		return domain.NewStore(), nil
	}
	if err != nil {
		return domain.Store{}, fmt.Errorf("read store: %w", err)
	}

	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return domain.NewStore(), nil
	}

	var loaded domain.Store
	if data[0] == '[' {
		legacy, err := decodeLegacy(data)
		if err != nil {
			return domain.Store{}, err
		}
		loaded = legacy
	} else {
		if err := json.Unmarshal(data, &loaded); err != nil {
			return domain.Store{}, fmt.Errorf("decode store json: %w", err)
		}
	}

	loaded = domain.NormalizeStore(loaded)
	if err := loaded.Validate(); err != nil {
		return domain.Store{}, fmt.Errorf("validate store: %w", err)
	}

	return loaded, nil
}

func (r *Repository) Save(store domain.Store) error {
	store = domain.NormalizeStore(store)
	if err := store.Validate(); err != nil {
		return fmt.Errorf("validate store: %w", err)
	}

	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("encode store json: %w", err)
	}
	data = append(data, '\n')

	dir := filepath.Dir(r.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create store directory: %w", err)
	}

	if err := backupExisting(r.path); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, "."+filepath.Base(r.path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary store: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write temporary store: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync temporary store: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary store: %w", err)
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		return fmt.Errorf("chmod temporary store: %w", err)
	}
	if err := os.Rename(tmpName, r.path); err != nil {
		return fmt.Errorf("replace store: %w", err)
	}

	return nil
}

func backupExisting(path string) error {
	source, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("open current store for backup: %w", err)
	}
	defer source.Close()

	info, err := source.Stat()
	if err != nil {
		return fmt.Errorf("stat current store for backup: %w", err)
	}
	if info.Size() == 0 {
		return nil
	}

	backupPath := path + ".bak"
	target, err := os.OpenFile(backupPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("create store backup: %w", err)
	}
	defer target.Close()

	if _, err := io.Copy(target, source); err != nil {
		return fmt.Errorf("write store backup: %w", err)
	}
	if err := target.Sync(); err != nil {
		return fmt.Errorf("sync store backup: %w", err)
	}

	return nil
}

type legacyEndpoint struct {
	Hostname string     `json:"hostname"`
	IP       string     `json:"ip"`
	Port     legacyPort `json:"port"`
	User     string     `json:"user"`
}

type legacyHost struct {
	legacyEndpoint
	LXC    []legacyEndpoint `json:"lxc"`
	VM     []legacyEndpoint `json:"vm"`
	Docker []legacyEndpoint `json:"docker"`
}

func decodeLegacy(data []byte) (domain.Store, error) {
	var legacyHosts []legacyHost
	if err := json.Unmarshal(data, &legacyHosts); err != nil {
		return domain.Store{}, fmt.Errorf("decode legacy hosts json: %w", err)
	}

	result := domain.NewStore()
	for i, oldHost := range legacyHosts {
		host := domain.Host{
			Hostname:   oldHost.Hostname,
			IP:         oldHost.IP,
			Port:       int(oldHost.Port),
			User:       oldHost.User,
			Subsystems: []domain.Endpoint{},
		}
		host.ID = domain.StableID(domain.ResourceHost, host.Hostname, host.IP, host.Port, host.User, i)

		host.Subsystems = append(host.Subsystems, migrateLegacyEndpoints(domain.ResourceLXC, oldHost.LXC, i)...)
		host.Subsystems = append(host.Subsystems, migrateLegacyEndpoints(domain.ResourceVM, oldHost.VM, i)...)
		host.Subsystems = append(host.Subsystems, migrateLegacyEndpoints(domain.ResourceDocker, oldHost.Docker, i)...)

		result.Hosts = append(result.Hosts, host)
	}

	return result, nil
}

func migrateLegacyEndpoints(kind domain.ResourceType, endpoints []legacyEndpoint, hostIndex int) []domain.Endpoint {
	result := make([]domain.Endpoint, 0, len(endpoints))
	for i, old := range endpoints {
		endpoint := domain.Endpoint{
			Type:     kind,
			Hostname: old.Hostname,
			IP:       old.IP,
			Port:     int(old.Port),
			User:     old.User,
		}
		endpoint.ID = domain.StableID(kind, endpoint.Hostname, endpoint.IP, endpoint.Port, endpoint.User, hostIndex, i)
		result = append(result, endpoint)
	}
	return result
}

type legacyPort int

func (p *legacyPort) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		*p = 0
		return nil
	}

	var number int
	if err := json.Unmarshal(data, &number); err == nil {
		*p = legacyPort(number)
		return nil
	}

	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return fmt.Errorf("decode port: %w", err)
	}

	text = strings.TrimSpace(text)
	parsed, err := strconv.Atoi(text)
	if err != nil {
		return fmt.Errorf("decode port %q: %w", text, err)
	}
	*p = legacyPort(parsed)
	return nil
}
