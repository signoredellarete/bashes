package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

const CurrentSchemaVersion = 1

type ResourceType string

const (
	ResourceHost   ResourceType = "host"
	ResourceLXC    ResourceType = "lxc"
	ResourceVM     ResourceType = "vm"
	ResourceDocker ResourceType = "docker"
)

type Store struct {
	Version int    `json:"version"`
	Hosts   []Host `json:"hosts"`
}

type Host struct {
	ID         string     `json:"id"`
	Hostname   string     `json:"hostname"`
	IP         string     `json:"ip"`
	Port       int        `json:"port"`
	User       string     `json:"user"`
	Subsystems []Endpoint `json:"subsystems"`
}

type Endpoint struct {
	ID       string       `json:"id"`
	Type     ResourceType `json:"type"`
	Hostname string       `json:"hostname"`
	IP       string       `json:"ip"`
	Port     int          `json:"port"`
	User     string       `json:"user"`
}

func NewStore() Store {
	return Store{Version: CurrentSchemaVersion, Hosts: []Host{}}
}

func (s Store) Validate() error {
	if s.Version != CurrentSchemaVersion {
		return fmt.Errorf("unsupported schema version %d", s.Version)
	}

	seen := map[string]struct{}{}
	for i, host := range s.Hosts {
		if err := validateEndpointFields("host", host.ID, host.Hostname, host.IP, host.Port, host.User); err != nil {
			return fmt.Errorf("hosts[%d]: %w", i, err)
		}
		if _, exists := seen[host.ID]; exists {
			return fmt.Errorf("hosts[%d]: duplicate id %q", i, host.ID)
		}
		seen[host.ID] = struct{}{}

		for j, sub := range host.Subsystems {
			if !ValidResourceType(sub.Type) || sub.Type == ResourceHost {
				return fmt.Errorf("hosts[%d].subsystems[%d]: invalid type %q", i, j, sub.Type)
			}
			if err := validateEndpointFields(string(sub.Type), sub.ID, sub.Hostname, sub.IP, sub.Port, sub.User); err != nil {
				return fmt.Errorf("hosts[%d].subsystems[%d]: %w", i, j, err)
			}
			if _, exists := seen[sub.ID]; exists {
				return fmt.Errorf("hosts[%d].subsystems[%d]: duplicate id %q", i, j, sub.ID)
			}
			seen[sub.ID] = struct{}{}
		}
	}

	return nil
}

func NormalizeStore(store Store) Store {
	if store.Version == 0 {
		store.Version = CurrentSchemaVersion
	}
	if store.Hosts == nil {
		store.Hosts = []Host{}
	}

	for i := range store.Hosts {
		host := &store.Hosts[i]
		if host.ID == "" {
			host.ID = StableID(ResourceHost, host.Hostname, host.IP, host.Port, host.User, i)
		}
		if host.Subsystems == nil {
			host.Subsystems = []Endpoint{}
		}
		for j := range host.Subsystems {
			sub := &host.Subsystems[j]
			if sub.ID == "" {
				sub.ID = StableID(sub.Type, sub.Hostname, sub.IP, sub.Port, sub.User, i, j)
			}
		}
	}

	return store
}

func ValidResourceType(kind ResourceType) bool {
	switch kind {
	case ResourceHost, ResourceLXC, ResourceVM, ResourceDocker:
		return true
	default:
		return false
	}
}

func StableID(kind ResourceType, hostname, ip string, port int, user string, parts ...int) string {
	label := slug(hostname)
	if label == "" {
		label = string(kind)
	}

	var suffixParts []string
	for _, part := range parts {
		suffixParts = append(suffixParts, strconv.Itoa(part))
	}

	sum := sha256.Sum256([]byte(strings.Join([]string{
		string(kind),
		hostname,
		ip,
		strconv.Itoa(port),
		user,
		strings.Join(suffixParts, "."),
	}, "|")))

	return fmt.Sprintf("%s-%s-%s", kind, label, hex.EncodeToString(sum[:])[:10])
}

func validateEndpointFields(kind, id, hostname, ip string, port int, user string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%s id is required", kind)
	}
	if strings.TrimSpace(hostname) == "" {
		return fmt.Errorf("%s hostname is required", kind)
	}
	if hasControl(hostname) {
		return fmt.Errorf("%s hostname contains control characters", kind)
	}
	if strings.TrimSpace(ip) == "" {
		return fmt.Errorf("%s ip is required", kind)
	}
	if hasControl(ip) {
		return fmt.Errorf("%s ip contains control characters", kind)
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("%s port must be between 1 and 65535", kind)
	}
	if strings.TrimSpace(user) == "" {
		return fmt.Errorf("%s user is required", kind)
	}
	if hasControl(user) || strings.ContainsFunc(user, unicode.IsSpace) {
		return fmt.Errorf("%s user contains invalid characters", kind)
	}
	return nil
}

func hasControl(value string) bool {
	return strings.ContainsFunc(value, unicode.IsControl)
}

var slugPattern = regexp.MustCompile(`[^a-z0-9]+`)

func slug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = slugPattern.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	if len(value) > 40 {
		value = strings.Trim(value[:40], "-")
	}
	return value
}
