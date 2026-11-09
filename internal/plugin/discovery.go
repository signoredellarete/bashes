package plugin

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type Candidate struct {
	Directory  string
	Manifest   Manifest
	Executable string
	Err        error
}

func (c Candidate) Available() bool {
	return c.Err == nil
}

// Discover inspects one plugin root without executing plugin code. Invalid or
// incompatible plugin directories are returned as candidates with an error so
// the UI can explain why they are unavailable.
func Discover(root, goos, goarch string, hostAPIVersion int) ([]Candidate, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Candidate{}, nil
		}
		return nil, fmt.Errorf("read plugin directory: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	result := make([]Candidate, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		directory := filepath.Join(root, entry.Name())
		candidate := Candidate{Directory: directory}
		manifest, err := LoadManifest(directory)
		if err != nil {
			candidate.Err = err
			result = append(result, candidate)
			continue
		}
		candidate.Manifest = manifest
		if !manifest.CompatibleWith(hostAPIVersion) {
			candidate.Err = fmt.Errorf("plugin API version %d is incompatible with host API version %d", manifest.APIVersion, hostAPIVersion)
			result = append(result, candidate)
			continue
		}
		candidate.Executable, candidate.Err = manifest.Executable(directory, goos, goarch)
		result = append(result, candidate)
	}
	return result, nil
}
