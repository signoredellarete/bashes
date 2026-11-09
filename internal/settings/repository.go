package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/signoredellarete/bashes/internal/fsutil"
)

const CurrentVersion = 1

type SSHKeys struct {
	CustomDirectory string `json:"customDirectory"`
}

type WebAppBrowser struct {
	PreferredBrowserID string `json:"preferredBrowserId,omitempty"`
	CustomBrowserPath  string `json:"customBrowserPath,omitempty"`
}

// Data intentionally retains the Bashes 1.x flat JSON field names.
type Data struct {
	Version            int    `json:"version,omitempty"`
	CustomDirectory    string `json:"customDirectory,omitempty"`
	PreferredBrowserID string `json:"preferredBrowserId,omitempty"`
	CustomBrowserPath  string `json:"customBrowserPath,omitempty"`
}

type Repository struct {
	path string
	mu   sync.Mutex
}

func NewRepository(path string) *Repository {
	return &Repository{path: path}
}

func (r *Repository) Path() string {
	return r.path
}

func (r *Repository) Load() (Data, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.load()
}

func (r *Repository) Update(update func(*Data) error) (Data, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := r.load()
	if err != nil {
		return Data{}, err
	}
	if err := update(&data); err != nil {
		return Data{}, err
	}
	data = normalize(data)
	data.Version = CurrentVersion
	if err := r.save(data); err != nil {
		return Data{}, err
	}
	return data, nil
}

func (r *Repository) load() (Data, error) {
	data, err := os.ReadFile(r.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Data{Version: CurrentVersion}, nil
		}
		return Data{}, fmt.Errorf("read settings: %w", err)
	}
	var settings Data
	if err := json.Unmarshal(data, &settings); err != nil {
		return Data{}, fmt.Errorf("decode settings: %w", err)
	}
	if settings.Version < 0 || settings.Version > CurrentVersion {
		return Data{}, fmt.Errorf("unsupported settings version %d", settings.Version)
	}
	settings = normalize(settings)
	settings.Version = CurrentVersion
	return settings, nil
}

func (r *Repository) save(settings Data) error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("encode settings: %w", err)
	}
	if err := fsutil.WriteFileAtomic(r.path, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write settings: %w", err)
	}
	return nil
}

func normalize(settings Data) Data {
	settings.CustomDirectory = strings.TrimSpace(settings.CustomDirectory)
	settings.PreferredBrowserID = strings.TrimSpace(settings.PreferredBrowserID)
	settings.CustomBrowserPath = strings.TrimSpace(settings.CustomBrowserPath)
	return settings
}
