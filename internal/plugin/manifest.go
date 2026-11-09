package plugin

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

const (
	ManifestFilename     = "manifest.json"
	ManifestSchema       = 1
	HostAPIVersion       = 1
	maxManifestSizeBytes = 1 << 20
)

type Capability string

const (
	CapabilityDiagnostic Capability = "diagnostic"
	CapabilityX11Server  Capability = "x11-server"
	CapabilityRDP        Capability = "rdp"
	CapabilityVNC        Capability = "vnc"
)

type Permission string

const (
	PermissionCredentialRead Permission = "credential.read"
	PermissionNetworkConnect Permission = "network.connect"
	PermissionNetworkListen  Permission = "network.listen"
	PermissionProcessLaunch  Permission = "process.launch"
	PermissionRuntimeData    Permission = "runtime-data.write"
)

var (
	pluginIDPattern  = regexp.MustCompile(`^[a-z0-9]+(?:[.-][a-z0-9]+)*$`)
	versionPattern   = regexp.MustCompile(`^(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)
	platformPattern  = regexp.MustCompile(`^[a-z0-9]+/[a-z0-9_]+$`)
	knownPermissions = map[Permission]struct{}{
		PermissionCredentialRead: {},
		PermissionNetworkConnect: {},
		PermissionNetworkListen:  {},
		PermissionProcessLaunch:  {},
		PermissionRuntimeData:    {},
	}
)

type Manifest struct {
	SchemaVersion int               `json:"schemaVersion"`
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Version       string            `json:"version"`
	APIVersion    int               `json:"apiVersion"`
	Description   string            `json:"description,omitempty"`
	Capabilities  []Capability      `json:"capabilities"`
	Permissions   []Permission      `json:"permissions,omitempty"`
	Executables   map[string]string `json:"executables"`
}

func LoadManifest(pluginDir string) (Manifest, error) {
	path := filepath.Join(pluginDir, ManifestFilename)
	file, err := os.Open(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("open plugin manifest: %w", err)
	}
	defer file.Close()

	limited := io.LimitReader(file, maxManifestSizeBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return Manifest{}, fmt.Errorf("read plugin manifest: %w", err)
	}
	if len(data) > maxManifestSizeBytes {
		return Manifest{}, errors.New("plugin manifest exceeds 1 MiB")
	}

	var manifest Manifest
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, fmt.Errorf("decode plugin manifest: %w", err)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return Manifest{}, err
	}
	if err := manifest.Validate(); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func (m Manifest) Validate() error {
	if m.SchemaVersion != ManifestSchema {
		return fmt.Errorf("unsupported plugin manifest schema %d", m.SchemaVersion)
	}
	if !pluginIDPattern.MatchString(m.ID) {
		return fmt.Errorf("invalid plugin id %q", m.ID)
	}
	if strings.TrimSpace(m.Name) == "" || hasControl(m.Name) {
		return errors.New("plugin name is required and must not contain control characters")
	}
	if !versionPattern.MatchString(m.Version) {
		return fmt.Errorf("invalid plugin version %q", m.Version)
	}
	if m.APIVersion <= 0 {
		return errors.New("plugin API version must be positive")
	}
	if hasControl(m.Description) {
		return errors.New("plugin description must not contain control characters")
	}
	if len(m.Capabilities) == 0 {
		return errors.New("plugin must declare at least one capability")
	}
	if err := validateUniqueStrings("capability", capabilityStrings(m.Capabilities)); err != nil {
		return err
	}
	for _, capability := range m.Capabilities {
		if !pluginIDPattern.MatchString(string(capability)) {
			return fmt.Errorf("invalid plugin capability %q", capability)
		}
	}
	if err := validateUniqueStrings("permission", permissionStrings(m.Permissions)); err != nil {
		return err
	}
	for _, permission := range m.Permissions {
		if _, known := knownPermissions[permission]; !known {
			return fmt.Errorf("unknown plugin permission %q", permission)
		}
	}
	if len(m.Executables) == 0 {
		return errors.New("plugin must declare at least one executable")
	}
	for platform, executable := range m.Executables {
		if !platformPattern.MatchString(platform) {
			return fmt.Errorf("invalid plugin platform %q", platform)
		}
		if err := validateRelativeExecutable(executable); err != nil {
			return fmt.Errorf("executable for %s: %w", platform, err)
		}
	}
	return nil
}

func (m Manifest) CompatibleWith(hostAPIVersion int) bool {
	return m.APIVersion == hostAPIVersion
}

func (m Manifest) Supports(capability Capability) bool {
	for _, current := range m.Capabilities {
		if current == capability {
			return true
		}
	}
	return false
}

func (m Manifest) Executable(pluginDir, goos, goarch string) (string, error) {
	platform := strings.ToLower(strings.TrimSpace(goos)) + "/" + strings.ToLower(strings.TrimSpace(goarch))
	relativePath, ok := m.Executables[platform]
	if !ok {
		return "", fmt.Errorf("plugin %q does not support %s", m.ID, platform)
	}
	if err := validateRelativeExecutable(relativePath); err != nil {
		return "", err
	}

	root, err := filepath.EvalSymlinks(pluginDir)
	if err != nil {
		return "", fmt.Errorf("resolve plugin directory: %w", err)
	}
	path, err := filepath.EvalSymlinks(filepath.Join(root, filepath.FromSlash(relativePath)))
	if err != nil {
		return "", fmt.Errorf("resolve plugin executable: %w", err)
	}
	inside, err := pathWithin(root, path)
	if err != nil {
		return "", err
	}
	if !inside {
		return "", errors.New("plugin executable resolves outside the plugin directory")
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("stat plugin executable: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("plugin executable is not a regular file")
	}
	if goos != "windows" && info.Mode().Perm()&0o111 == 0 {
		return "", errors.New("plugin executable is not executable")
	}
	return path, nil
}

func SortedCapabilities(capabilities []Capability) []Capability {
	result := append([]Capability(nil), capabilities...)
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); errors.Is(err, io.EOF) {
		return nil
	} else if err != nil {
		return fmt.Errorf("decode plugin manifest trailing data: %w", err)
	}
	return errors.New("plugin manifest contains multiple JSON values")
}

func validateRelativeExecutable(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("path is required")
	}
	converted := filepath.FromSlash(path)
	if filepath.IsAbs(converted) {
		return errors.New("path must be relative to the plugin directory")
	}
	clean := filepath.Clean(converted)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return errors.New("path must stay inside the plugin directory")
	}
	return nil
}

func pathWithin(root, path string) (bool, error) {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return false, fmt.Errorf("compare plugin executable path: %w", err)
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)), nil
}

func validateUniqueStrings(kind string, values []string) error {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			return fmt.Errorf("duplicate plugin %s %q", kind, value)
		}
		seen[value] = struct{}{}
	}
	return nil
}

func capabilityStrings(values []Capability) []string {
	result := make([]string, len(values))
	for i, value := range values {
		result[i] = string(value)
	}
	return result
}

func permissionStrings(values []Permission) []string {
	result := make([]string, len(values))
	for i, value := range values {
		result[i] = string(value)
	}
	return result
}

func hasControl(value string) bool {
	return strings.IndexFunc(value, unicode.IsControl) >= 0
}
