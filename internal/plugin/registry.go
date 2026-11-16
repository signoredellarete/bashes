package plugin

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	appsettings "github.com/signoredellarete/bashes/internal/settings"
)

type PluginInfo struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Version      string   `json:"version,omitempty"`
	Description  string   `json:"description,omitempty"`
	Directory    string   `json:"directory"`
	Capabilities []string `json:"capabilities,omitempty"`
	Permissions  []string `json:"permissions,omitempty"`
	Available    bool     `json:"available"`
	Enabled      bool     `json:"enabled"`
	Error        string   `json:"error,omitempty"`
}

type Inventory struct {
	Directory string       `json:"directory"`
	Plugins   []PluginInfo `json:"plugins"`
}

type Registry struct {
	mu             sync.Mutex
	root           string
	goos           string
	goarch         string
	hostAPIVersion int
	settings       *appsettings.Repository
}

func NewRegistry(root, goos, goarch string, settings *appsettings.Repository) *Registry {
	return &Registry{
		root: root, goos: goos, goarch: goarch,
		hostAPIVersion: HostAPIVersion,
		settings:       settings,
	}
}

func (r *Registry) Inventory() (Inventory, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.inventory()
}

func (r *Registry) SetEnabled(id string, enabled bool) (PluginInfo, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	id = strings.TrimSpace(id)
	if id == "" {
		return PluginInfo{}, errors.New("plugin id is required")
	}
	if enabled {
		candidate, err := r.availableCandidate(id)
		if err != nil {
			return PluginInfo{}, err
		}
		if !candidate.Available() {
			return PluginInfo{}, candidate.Err
		}
	}

	_, err := r.settings.Update(func(data *appsettings.Data) error {
		data.EnabledPlugins = setPluginEnabled(data.EnabledPlugins, id, enabled)
		return nil
	})
	if err != nil {
		return PluginInfo{}, fmt.Errorf("save plugin settings: %w", err)
	}

	inventory, err := r.inventory()
	if err != nil {
		return PluginInfo{}, err
	}
	for _, info := range inventory.Plugins {
		if info.ID == id {
			return info, nil
		}
	}
	return PluginInfo{ID: id, Name: id, Enabled: enabled, Available: false, Error: "Plugin files are missing."}, nil
}

func (r *Registry) inventory() (Inventory, error) {
	settings, err := r.settings.Load()
	if err != nil {
		return Inventory{}, fmt.Errorf("load plugin settings: %w", err)
	}
	candidates, err := Discover(r.root, r.goos, r.goarch, r.hostAPIVersion)
	if err != nil {
		return Inventory{}, err
	}
	enabled := stringSet(settings.EnabledPlugins)
	seen := make(map[string]struct{}, len(candidates))
	plugins := make([]PluginInfo, 0, len(candidates)+len(enabled))
	for _, candidate := range candidates {
		info := candidateInfo(candidate, enabled)
		plugins = append(plugins, info)
		if info.ID != "" {
			seen[info.ID] = struct{}{}
		}
	}
	for id := range enabled {
		if _, found := seen[id]; found {
			continue
		}
		plugins = append(plugins, PluginInfo{
			ID: id, Name: id, Enabled: true, Available: false,
			Error: "Plugin files are missing.",
		})
	}
	sort.SliceStable(plugins, func(i, j int) bool {
		left := strings.ToLower(plugins[i].Name + "\x00" + plugins[i].Directory)
		right := strings.ToLower(plugins[j].Name + "\x00" + plugins[j].Directory)
		return left < right
	})
	return Inventory{Directory: r.root, Plugins: plugins}, nil
}

func (r *Registry) availableCandidate(id string) (Candidate, error) {
	candidates, err := Discover(r.root, r.goos, r.goarch, r.hostAPIVersion)
	if err != nil {
		return Candidate{}, err
	}
	for _, candidate := range candidates {
		if candidate.Manifest.ID == id {
			if candidate.Err != nil {
				return Candidate{}, candidate.Err
			}
			return candidate, nil
		}
	}
	return Candidate{}, fmt.Errorf("plugin %q was not found", id)
}

func candidateInfo(candidate Candidate, enabled map[string]struct{}) PluginInfo {
	manifest := candidate.Manifest
	name := manifest.Name
	if strings.TrimSpace(name) == "" {
		name = filepath.Base(candidate.Directory)
	}
	info := PluginInfo{
		ID: manifest.ID, Name: name, Version: manifest.Version,
		Description: manifest.Description, Directory: candidate.Directory,
		Capabilities: capabilityStrings(manifest.Capabilities),
		Permissions:  permissionStrings(manifest.Permissions),
		Available:    candidate.Available(),
	}
	_, info.Enabled = enabled[manifest.ID]
	if candidate.Err != nil {
		info.Error = candidate.Err.Error()
	}
	return info
}

func setPluginEnabled(values []string, id string, enabled bool) []string {
	set := stringSet(values)
	if enabled {
		set[id] = struct{}{}
	} else {
		delete(set, id)
	}
	result := make([]string, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func stringSet(values []string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			result[value] = struct{}{}
		}
	}
	return result
}
