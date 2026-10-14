package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const CurrentSchemaVersion = 2

const (
	MaxTagsPerResource = 32
	MaxTagsPerStore    = 512
	MaxTagLength       = 40
)

type ResourceType string

const (
	ResourceHost   ResourceType = "host"
	ResourceLXC    ResourceType = "lxc"
	ResourceVM     ResourceType = "vm"
	ResourceDocker ResourceType = "docker"
	ResourceWebApp ResourceType = "webapp"
)

type Store struct {
	Version       int      `json:"version"`
	Tags          []string `json:"tags,omitempty"`
	Hosts         []Host   `json:"hosts"`
	WebApps       []WebApp `json:"webApps,omitempty"`
	TopLevelOrder []string `json:"topLevelOrder,omitempty"`
}

type WebAppConnectionMode string

const (
	WebAppConnectionDirect WebAppConnectionMode = "direct"
	WebAppConnectionTunnel WebAppConnectionMode = "tunnel"
)

type WebAppPortMode string

const (
	WebAppPortAutomatic WebAppPortMode = "automatic"
	WebAppPortManual    WebAppPortMode = "manual"
)

type WebAppBrowserMode string

const (
	WebAppBrowserStandard WebAppBrowserMode = "standard"
	WebAppBrowserPrivate  WebAppBrowserMode = "private"
)

type WebApp struct {
	ID                string               `json:"id"`
	Name              string               `json:"name"`
	URL               string               `json:"url"`
	Tags              []string             `json:"tags,omitempty"`
	ConnectionMode    WebAppConnectionMode `json:"connectionMode"`
	GatewayResourceID string               `json:"gatewayResourceId,omitempty"`
	PortMode          WebAppPortMode       `json:"portMode,omitempty"`
	LocalSOCKSPort    int                  `json:"localSocksPort,omitempty"`
	BrowserMode       WebAppBrowserMode    `json:"browserMode"`
	KeepCookies       bool                 `json:"keepCookies"`
	KeepLocalStorage  bool                 `json:"keepLocalStorage"`
	RestoreSession    bool                 `json:"restoreSession"`
	ClearCacheOnClose bool                 `json:"clearCacheOnClose"`
}

type Host struct {
	ID                 string     `json:"id"`
	Hostname           string     `json:"hostname"`
	IP                 string     `json:"ip"`
	Port               int        `json:"port"`
	User               string     `json:"user"`
	Tags               []string   `json:"tags,omitempty"`
	HostKeyFingerprint string     `json:"hostKeyFingerprint,omitempty"`
	Auth               *Auth      `json:"auth,omitempty"`
	Subsystems         []Endpoint `json:"subsystems"`
}

type Endpoint struct {
	ID                 string       `json:"id"`
	Type               ResourceType `json:"type"`
	Hostname           string       `json:"hostname"`
	IP                 string       `json:"ip"`
	Port               int          `json:"port"`
	User               string       `json:"user"`
	Tags               []string     `json:"tags,omitempty"`
	HostKeyFingerprint string       `json:"hostKeyFingerprint,omitempty"`
	Auth               *Auth        `json:"auth,omitempty"`
	Subsystems         []Endpoint   `json:"subsystems,omitempty"`
}

type AuthMethod string

const (
	AuthMethodPassword AuthMethod = "password"
	AuthMethodKey      AuthMethod = "key"
	AuthMethodPath     AuthMethod = "path"
	AuthMethodAgent    AuthMethod = "agent"
)

type Auth struct {
	Method         AuthMethod `json:"method"`
	KeyName        string     `json:"keyName,omitempty"`
	PrivateKeyPath string     `json:"privateKeyPath,omitempty"`
}

func NewStore() Store {
	return Store{
		Version:       CurrentSchemaVersion,
		Hosts:         []Host{},
		WebApps:       []WebApp{},
		TopLevelOrder: []string{},
	}
}

func (s Store) Validate() error {
	if s.Version != CurrentSchemaVersion {
		return fmt.Errorf("unsupported schema version %d", s.Version)
	}
	if err := ValidateTagCatalog(s.Tags); err != nil {
		return fmt.Errorf("tags: %w", err)
	}

	seen := map[string]struct{}{}
	resourceIDs := map[string]struct{}{}
	for i, host := range s.Hosts {
		if err := validateEndpointFields("host", host.ID, host.Hostname, host.IP, host.Port, host.User); err != nil {
			return fmt.Errorf("hosts[%d]: %w", i, err)
		}
		if hasControl(host.HostKeyFingerprint) {
			return fmt.Errorf("hosts[%d]: host key fingerprint contains control characters", i)
		}
		if err := ValidateTags(host.Tags); err != nil {
			return fmt.Errorf("hosts[%d].tags: %w", i, err)
		}
		if err := validateAuth(host.Auth); err != nil {
			return fmt.Errorf("hosts[%d].auth: %w", i, err)
		}
		if _, exists := seen[host.ID]; exists {
			return fmt.Errorf("hosts[%d]: duplicate id %q", i, host.ID)
		}
		seen[host.ID] = struct{}{}
		resourceIDs[host.ID] = struct{}{}

		for j := range host.Subsystems {
			if err := validateSubsystem(host.Subsystems[j], fmt.Sprintf("hosts[%d].subsystems[%d]", i, j), seen); err != nil {
				return err
			}
			collectSubsystemIDs(host.Subsystems[j], resourceIDs)
		}
	}

	topLevelIDs := make(map[string]struct{}, len(s.Hosts)+len(s.WebApps))
	for _, host := range s.Hosts {
		topLevelIDs[host.ID] = struct{}{}
	}
	for i, app := range s.WebApps {
		if err := validateWebApp(app, resourceIDs); err != nil {
			return fmt.Errorf("webApps[%d]: %w", i, err)
		}
		if _, exists := seen[app.ID]; exists {
			return fmt.Errorf("webApps[%d]: duplicate id %q", i, app.ID)
		}
		seen[app.ID] = struct{}{}
		topLevelIDs[app.ID] = struct{}{}
	}
	if len(s.TopLevelOrder) != len(topLevelIDs) {
		return fmt.Errorf("topLevelOrder has %d entries, want %d", len(s.TopLevelOrder), len(topLevelIDs))
	}
	ordered := make(map[string]struct{}, len(s.TopLevelOrder))
	for i, id := range s.TopLevelOrder {
		if _, exists := topLevelIDs[id]; !exists {
			return fmt.Errorf("topLevelOrder[%d]: unknown id %q", i, id)
		}
		if _, exists := ordered[id]; exists {
			return fmt.Errorf("topLevelOrder[%d]: duplicate id %q", i, id)
		}
		ordered[id] = struct{}{}
	}

	return nil
}

func collectSubsystemIDs(sub Endpoint, ids map[string]struct{}) {
	ids[sub.ID] = struct{}{}
	for _, child := range sub.Subsystems {
		collectSubsystemIDs(child, ids)
	}
}

func validateWebApp(app WebApp, resourceIDs map[string]struct{}) error {
	if strings.TrimSpace(app.ID) == "" {
		return errors.New("id is required")
	}
	if strings.TrimSpace(app.Name) == "" {
		return errors.New("name is required")
	}
	if hasControl(app.Name) {
		return errors.New("name contains control characters")
	}
	parsed, err := url.ParseRequestURI(strings.TrimSpace(app.URL))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return errors.New("url must be an absolute HTTP or HTTPS URL")
	}
	if parsed.User != nil {
		return errors.New("url must not contain credentials")
	}
	if hasControl(app.URL) {
		return errors.New("url contains control characters")
	}
	if err := ValidateTags(app.Tags); err != nil {
		return fmt.Errorf("tags: %w", err)
	}
	if app.BrowserMode != WebAppBrowserStandard && app.BrowserMode != WebAppBrowserPrivate {
		return fmt.Errorf("invalid browser mode %q", app.BrowserMode)
	}
	switch app.ConnectionMode {
	case WebAppConnectionDirect:
		if app.GatewayResourceID != "" || app.LocalSOCKSPort != 0 {
			return errors.New("direct connection must not define a gateway or SOCKS port")
		}
	case WebAppConnectionTunnel:
		if _, exists := resourceIDs[app.GatewayResourceID]; !exists {
			return fmt.Errorf("gateway resource %q not found", app.GatewayResourceID)
		}
		if app.PortMode != WebAppPortAutomatic && app.PortMode != WebAppPortManual {
			return fmt.Errorf("invalid port mode %q", app.PortMode)
		}
		if app.PortMode == WebAppPortManual {
			if app.LocalSOCKSPort < 1 || app.LocalSOCKSPort > 65535 {
				return errors.New("local SOCKS port must be between 1 and 65535")
			}
		} else if app.LocalSOCKSPort != 0 {
			return errors.New("automatic port mode must not define a SOCKS port")
		}
	default:
		return fmt.Errorf("invalid connection mode %q", app.ConnectionMode)
	}
	return nil
}

func validateSubsystem(sub Endpoint, path string, seen map[string]struct{}) error {
	if !ValidResourceType(sub.Type) || sub.Type == ResourceHost {
		return fmt.Errorf("%s: invalid type %q", path, sub.Type)
	}
	if err := validateEndpointFields(string(sub.Type), sub.ID, sub.Hostname, sub.IP, sub.Port, sub.User); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if hasControl(sub.HostKeyFingerprint) {
		return fmt.Errorf("%s: host key fingerprint contains control characters", path)
	}
	if err := ValidateTags(sub.Tags); err != nil {
		return fmt.Errorf("%s.tags: %w", path, err)
	}
	if err := validateAuth(sub.Auth); err != nil {
		return fmt.Errorf("%s.auth: %w", path, err)
	}
	if _, exists := seen[sub.ID]; exists {
		return fmt.Errorf("%s: duplicate id %q", path, sub.ID)
	}
	seen[sub.ID] = struct{}{}

	for i := range sub.Subsystems {
		if err := validateSubsystem(sub.Subsystems[i], fmt.Sprintf("%s.subsystems[%d]", path, i), seen); err != nil {
			return err
		}
	}
	return nil
}

func ValidAuthMethod(method AuthMethod) bool {
	switch method {
	case AuthMethodPassword, AuthMethodKey, AuthMethodPath, AuthMethodAgent:
		return true
	default:
		return false
	}
}

func NormalizeStore(store Store) Store {
	if store.Version == 0 {
		store.Version = 1
	}
	if store.Hosts == nil {
		store.Hosts = []Host{}
	}
	store.Tags = NormalizeTags(store.Tags)

	for i := range store.Hosts {
		host := &store.Hosts[i]
		host.Tags = NormalizeTags(host.Tags)
		store.Tags = MergeTags(store.Tags, host.Tags)
		if host.ID == "" {
			host.ID = StableID(ResourceHost, host.Hostname, host.IP, host.Port, host.User, i)
		}
		if host.Subsystems == nil {
			host.Subsystems = []Endpoint{}
		}
		for j := range host.Subsystems {
			normalizeSubsystem(&host.Subsystems[j], i, j)
			store.Tags = mergeSubsystemTags(store.Tags, host.Subsystems[j])
		}
	}
	if store.Version == 1 {
		store.Version = CurrentSchemaVersion
	}
	if store.WebApps == nil {
		store.WebApps = []WebApp{}
	}
	for i := range store.WebApps {
		app := &store.WebApps[i]
		app.Name = strings.TrimSpace(app.Name)
		app.URL = strings.TrimSpace(app.URL)
		app.Tags = NormalizeTags(app.Tags)
		store.Tags = MergeTags(store.Tags, app.Tags)
		if app.ID == "" {
			app.ID = StableID(ResourceWebApp, app.Name, app.URL, 0, "", i)
		}
	}
	store.TopLevelOrder = normalizeTopLevelOrder(store.TopLevelOrder, store.Hosts, store.WebApps)

	return store
}

func normalizeTopLevelOrder(order []string, hosts []Host, apps []WebApp) []string {
	valid := make(map[string]struct{}, len(hosts)+len(apps))
	for _, host := range hosts {
		valid[host.ID] = struct{}{}
	}
	for _, app := range apps {
		valid[app.ID] = struct{}{}
	}

	normalized := make([]string, 0, len(valid))
	seen := make(map[string]struct{}, len(valid))
	for _, id := range order {
		if _, exists := valid[id]; !exists {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		normalized = append(normalized, id)
	}
	for _, host := range hosts {
		if _, exists := seen[host.ID]; !exists {
			seen[host.ID] = struct{}{}
			normalized = append(normalized, host.ID)
		}
	}
	for _, app := range apps {
		if _, exists := seen[app.ID]; !exists {
			seen[app.ID] = struct{}{}
			normalized = append(normalized, app.ID)
		}
	}
	return normalized
}

func mergeSubsystemTags(catalog []string, subsystem Endpoint) []string {
	catalog = MergeTags(catalog, subsystem.Tags)
	for _, child := range subsystem.Subsystems {
		catalog = mergeSubsystemTags(catalog, child)
	}
	return catalog
}

func normalizeSubsystem(sub *Endpoint, parts ...int) {
	sub.Tags = NormalizeTags(sub.Tags)
	if sub.ID == "" {
		sub.ID = StableID(sub.Type, sub.Hostname, sub.IP, sub.Port, sub.User, parts...)
	}
	if sub.Subsystems == nil {
		sub.Subsystems = []Endpoint{}
	}
	for i := range sub.Subsystems {
		normalizeSubsystem(&sub.Subsystems[i], append(parts, i)...)
	}
}

func NormalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		key := strings.ToLower(tag)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, tag)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func MergeTags(catalog, tags []string) []string {
	return NormalizeTags(append(append([]string{}, catalog...), tags...))
}

func ValidateTags(tags []string) error {
	return validateTags(tags, MaxTagsPerResource)
}

func ValidateTagCatalog(tags []string) error {
	return validateTags(tags, MaxTagsPerStore)
}

func validateTags(tags []string, limit int) error {
	if len(tags) > limit {
		return fmt.Errorf("at most %d tags are allowed", limit)
	}
	seen := make(map[string]struct{}, len(tags))
	for i, tag := range tags {
		if strings.TrimSpace(tag) == "" {
			return fmt.Errorf("tag %d is empty", i)
		}
		if hasControl(tag) {
			return fmt.Errorf("tag %q contains control characters", tag)
		}
		if utf8.RuneCountInString(tag) > MaxTagLength {
			return fmt.Errorf("tag %q exceeds %d characters", tag, MaxTagLength)
		}
		key := strings.ToLower(tag)
		if _, exists := seen[key]; exists {
			return fmt.Errorf("duplicate tag %q", tag)
		}
		seen[key] = struct{}{}
	}
	return nil
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

func validateAuth(auth *Auth) error {
	if auth == nil {
		return nil
	}
	if !ValidAuthMethod(auth.Method) {
		return fmt.Errorf("invalid method %q", auth.Method)
	}
	if hasControl(string(auth.Method)) {
		return errors.New("method contains control characters")
	}
	if hasControl(auth.KeyName) {
		return errors.New("key name contains control characters")
	}
	if hasControl(auth.PrivateKeyPath) {
		return errors.New("private key path contains control characters")
	}
	switch auth.Method {
	case AuthMethodKey:
		if strings.TrimSpace(auth.KeyName) == "" {
			return errors.New("key name is required for key auth")
		}
	case AuthMethodPath:
		if strings.TrimSpace(auth.PrivateKeyPath) == "" {
			return errors.New("private key path is required for path auth")
		}
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
