package application

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/signoredellarete/bashes/internal/domain"
)

type Store interface {
	Load() (domain.Store, error)
	Save(domain.Store) error
}

type Service struct {
	store Store
	mu    sync.RWMutex
}

type EndpointInput struct {
	Hostname string              `json:"hostname"`
	IP       string              `json:"ip"`
	Port     int                 `json:"port"`
	User     string              `json:"user"`
	Tags     []string            `json:"tags,omitempty"`
	Type     domain.ResourceType `json:"type,omitempty"`
}

type WebAppInput struct {
	Name              string                      `json:"name"`
	URL               string                      `json:"url"`
	Tags              []string                    `json:"tags,omitempty"`
	ConnectionMode    domain.WebAppConnectionMode `json:"connectionMode"`
	GatewayResourceID string                      `json:"gatewayResourceId,omitempty"`
	PortMode          domain.WebAppPortMode       `json:"portMode,omitempty"`
	LocalSOCKSPort    int                         `json:"localSocksPort,omitempty"`
	BrowserMode       domain.WebAppBrowserMode    `json:"browserMode"`
	KeepCookies       bool                        `json:"keepCookies"`
	KeepLocalStorage  bool                        `json:"keepLocalStorage"`
	RestoreSession    bool                        `json:"restoreSession"`
	ClearCacheOnClose bool                        `json:"clearCacheOnClose"`
}

type Workspace struct {
	Hosts         []domain.Host   `json:"hosts"`
	WebApps       []domain.WebApp `json:"webApps"`
	TopLevelOrder []string        `json:"topLevelOrder"`
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) ListHosts() ([]domain.Host, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := s.store.Load()
	if err != nil {
		return nil, err
	}
	return data.Hosts, nil
}

func (s *Service) ListWorkspace() (Workspace, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := s.store.Load()
	if err != nil {
		return Workspace{}, err
	}
	return Workspace{
		Hosts:         data.Hosts,
		WebApps:       data.WebApps,
		TopLevelOrder: data.TopLevelOrder,
	}, nil
}

func (s *Service) ListTags() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := s.store.Load()
	if err != nil {
		return nil, err
	}
	return append([]string(nil), data.Tags...), nil
}

func (s *Service) CreateTag(name string) (string, error) {
	name, err := normalizeTagName(name)
	if err != nil {
		return "", err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.store.Load()
	if err != nil {
		return "", err
	}
	if tagIndex(data.Tags, name) >= 0 {
		return "", fmt.Errorf("tag %q already exists", name)
	}
	data.Tags = append(data.Tags, name)
	if err := s.store.Save(data); err != nil {
		return "", err
	}
	return name, nil
}

func (s *Service) RenameTag(currentName, newName string) error {
	currentName = strings.TrimSpace(currentName)
	if currentName == "" {
		return errors.New("current tag name is required")
	}
	newName, err := normalizeTagName(newName)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.store.Load()
	if err != nil {
		return err
	}
	currentIndex := tagIndex(data.Tags, currentName)
	if currentIndex < 0 {
		return fmt.Errorf("tag %q not found", currentName)
	}
	if duplicateIndex := tagIndex(data.Tags, newName); duplicateIndex >= 0 && duplicateIndex != currentIndex {
		return fmt.Errorf("tag %q already exists", newName)
	}

	oldName := data.Tags[currentIndex]
	data.Tags[currentIndex] = newName
	for i := range data.Hosts {
		renameResourceTag(&data.Hosts[i].Tags, oldName, newName)
		renameSubsystemTags(data.Hosts[i].Subsystems, oldName, newName)
	}
	for i := range data.WebApps {
		renameResourceTag(&data.WebApps[i].Tags, oldName, newName)
	}
	return s.store.Save(data)
}

func (s *Service) DeleteTag(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("tag name is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.store.Load()
	if err != nil {
		return err
	}
	index := tagIndex(data.Tags, name)
	if index < 0 {
		return fmt.Errorf("tag %q not found", name)
	}
	canonicalName := data.Tags[index]
	data.Tags = append(data.Tags[:index], data.Tags[index+1:]...)
	for i := range data.Hosts {
		removeResourceTag(&data.Hosts[i].Tags, canonicalName)
		removeSubsystemTags(data.Hosts[i].Subsystems, canonicalName)
	}
	for i := range data.WebApps {
		removeResourceTag(&data.WebApps[i].Tags, canonicalName)
	}
	return s.store.Save(data)
}

func (s *Service) StoreSnapshot() (domain.Store, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.store.Load()
}

func (s *Service) ReplaceStore(data domain.Store) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.store.Save(data)
}

func (s *Service) UpdateStore(update func(domain.Store) (domain.Store, bool, error)) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.store.Load()
	if err != nil {
		return err
	}
	data, changed, err := update(data)
	if err != nil {
		return err
	}
	if !changed {
		return nil
	}
	return s.store.Save(data)
}

func (s *Service) AddHost(input EndpointInput) (domain.Host, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.store.Load()
	if err != nil {
		return domain.Host{}, err
	}

	host := domain.Host{
		Hostname:   strings.TrimSpace(input.Hostname),
		IP:         strings.TrimSpace(input.IP),
		Port:       input.Port,
		User:       strings.TrimSpace(input.User),
		Tags:       domain.NormalizeTags(input.Tags),
		Subsystems: []domain.Endpoint{},
	}
	host.ID = domain.StableID(domain.ResourceHost, host.Hostname, host.IP, host.Port, host.User, len(data.Hosts))

	data.Hosts = append(data.Hosts, host)
	data.TopLevelOrder = append(data.TopLevelOrder, host.ID)
	data.Tags = domain.MergeTags(data.Tags, host.Tags)
	if err := s.store.Save(data); err != nil {
		return domain.Host{}, err
	}

	return host, nil
}

func (s *Service) AddWebApp(input WebAppInput) (domain.WebApp, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.store.Load()
	if err != nil {
		return domain.WebApp{}, err
	}
	app := webAppFromInput(input)
	app.ID = domain.StableID(domain.ResourceWebApp, app.Name, app.URL, 0, "", len(data.WebApps))
	data.WebApps = append(data.WebApps, app)
	data.TopLevelOrder = append(data.TopLevelOrder, app.ID)
	data.Tags = domain.MergeTags(data.Tags, app.Tags)
	if err := s.store.Save(data); err != nil {
		return domain.WebApp{}, err
	}
	return app, nil
}

func (s *Service) UpdateWebApp(id string, input WebAppInput) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("web app id is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.store.Load()
	if err != nil {
		return err
	}
	for i := range data.WebApps {
		if data.WebApps[i].ID != id {
			continue
		}
		updated := webAppFromInput(input)
		updated.ID = id
		data.WebApps[i] = updated
		data.Tags = domain.MergeTags(data.Tags, updated.Tags)
		return s.store.Save(data)
	}
	return fmt.Errorf("web app %q not found", id)
}

func (s *Service) DeleteWebApp(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("web app id is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.store.Load()
	if err != nil {
		return err
	}
	for i := range data.WebApps {
		if data.WebApps[i].ID == id {
			data.WebApps = append(data.WebApps[:i], data.WebApps[i+1:]...)
			data.TopLevelOrder = removeID(data.TopLevelOrder, id)
			return s.store.Save(data)
		}
	}
	return fmt.Errorf("web app %q not found", id)
}

func webAppFromInput(input WebAppInput) domain.WebApp {
	app := domain.WebApp{
		Name:              strings.TrimSpace(input.Name),
		URL:               strings.TrimSpace(input.URL),
		Tags:              domain.NormalizeTags(input.Tags),
		ConnectionMode:    input.ConnectionMode,
		GatewayResourceID: strings.TrimSpace(input.GatewayResourceID),
		PortMode:          input.PortMode,
		LocalSOCKSPort:    input.LocalSOCKSPort,
		BrowserMode:       input.BrowserMode,
		KeepCookies:       input.KeepCookies,
		KeepLocalStorage:  input.KeepLocalStorage,
		RestoreSession:    input.RestoreSession,
		ClearCacheOnClose: input.ClearCacheOnClose,
	}
	if app.ConnectionMode == domain.WebAppConnectionDirect {
		app.GatewayResourceID = ""
		app.PortMode = ""
		app.LocalSOCKSPort = 0
	} else if app.PortMode == domain.WebAppPortAutomatic {
		app.LocalSOCKSPort = 0
	}
	return app
}

func (s *Service) AddSubsystem(hostID string, input EndpointInput) (domain.Endpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.store.Load()
	if err != nil {
		return domain.Endpoint{}, err
	}

	parentID := strings.TrimSpace(hostID)

	if !domain.ValidResourceType(input.Type) || input.Type == domain.ResourceHost {
		return domain.Endpoint{}, fmt.Errorf("invalid subsystem type %q", input.Type)
	}

	parent, parts := findSubsystemParent(data.Hosts, parentID)
	if parent == nil {
		return domain.Endpoint{}, fmt.Errorf("resource %q not found", parentID)
	}

	subsystem := domain.Endpoint{
		Type:       input.Type,
		Hostname:   strings.TrimSpace(input.Hostname),
		IP:         strings.TrimSpace(input.IP),
		Port:       input.Port,
		User:       strings.TrimSpace(input.User),
		Tags:       domain.NormalizeTags(input.Tags),
		Subsystems: []domain.Endpoint{},
	}
	parts = append(parts, len(*parent))
	subsystem.ID = domain.StableID(
		subsystem.Type,
		subsystem.Hostname,
		subsystem.IP,
		subsystem.Port,
		subsystem.User,
		parts...,
	)

	*parent = append(*parent, subsystem)
	data.Tags = domain.MergeTags(data.Tags, subsystem.Tags)
	if err := s.store.Save(data); err != nil {
		return domain.Endpoint{}, err
	}

	return subsystem, nil
}

func (s *Service) UpdateResource(id string, input EndpointInput) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("resource id is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.store.Load()
	if err != nil {
		return err
	}

	for i := range data.Hosts {
		if data.Hosts[i].ID == id {
			if endpointChanged(data.Hosts[i].Hostname, data.Hosts[i].IP, data.Hosts[i].Port, input) {
				data.Hosts[i].HostKeyFingerprint = ""
			}
			data.Hosts[i].Hostname = strings.TrimSpace(input.Hostname)
			data.Hosts[i].IP = strings.TrimSpace(input.IP)
			data.Hosts[i].Port = input.Port
			data.Hosts[i].User = strings.TrimSpace(input.User)
			if input.Tags != nil {
				data.Hosts[i].Tags = domain.NormalizeTags(input.Tags)
				data.Tags = domain.MergeTags(data.Tags, data.Hosts[i].Tags)
			}
			return s.store.Save(data)
		}
	}

	if subsystem := findSubsystemByID(data.Hosts, id); subsystem != nil {
		if !domain.ValidResourceType(input.Type) || input.Type == domain.ResourceHost {
			return fmt.Errorf("invalid subsystem type %q", input.Type)
		}
		if endpointChanged(subsystem.Hostname, subsystem.IP, subsystem.Port, input) {
			subsystem.HostKeyFingerprint = ""
		}
		subsystem.Type = input.Type
		subsystem.Hostname = strings.TrimSpace(input.Hostname)
		subsystem.IP = strings.TrimSpace(input.IP)
		subsystem.Port = input.Port
		subsystem.User = strings.TrimSpace(input.User)
		if input.Tags != nil {
			subsystem.Tags = domain.NormalizeTags(input.Tags)
			data.Tags = domain.MergeTags(data.Tags, subsystem.Tags)
		}
		return s.store.Save(data)
	}

	return fmt.Errorf("resource %q not found", id)
}

func (s *Service) SetResourceTags(id string, tags []string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("resource id is required")
	}
	tags = domain.NormalizeTags(tags)
	if err := domain.ValidateTags(tags); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.store.Load()
	if err != nil {
		return err
	}
	for i := range data.Hosts {
		if data.Hosts[i].ID == id {
			data.Hosts[i].Tags = tags
			data.Tags = domain.MergeTags(data.Tags, tags)
			return s.store.Save(data)
		}
	}
	if subsystem := findSubsystemByID(data.Hosts, id); subsystem != nil {
		subsystem.Tags = tags
		data.Tags = domain.MergeTags(data.Tags, tags)
		return s.store.Save(data)
	}
	for i := range data.WebApps {
		if data.WebApps[i].ID == id {
			data.WebApps[i].Tags = tags
			data.Tags = domain.MergeTags(data.Tags, tags)
			return s.store.Save(data)
		}
	}
	return fmt.Errorf("resource %q not found", id)
}

func endpointChanged(hostname, ip string, port int, input EndpointInput) bool {
	return strings.TrimSpace(hostname) != strings.TrimSpace(input.Hostname) ||
		strings.TrimSpace(ip) != strings.TrimSpace(input.IP) ||
		port != input.Port
}

func (s *Service) DeleteResource(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("resource id is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.store.Load()
	if err != nil {
		return err
	}

	for i := range data.Hosts {
		if data.Hosts[i].ID == id {
			ids := map[string]struct{}{data.Hosts[i].ID: {}}
			collectEndpointIDs(data.Hosts[i].Subsystems, ids)
			if app := webAppUsingGateway(data.WebApps, ids); app != nil {
				return fmt.Errorf("resource is used as gateway by web app %q", app.Name)
			}
			data.Hosts = append(data.Hosts[:i], data.Hosts[i+1:]...)
			data.TopLevelOrder = removeID(data.TopLevelOrder, id)
			return s.store.Save(data)
		}
	}

	for i := range data.Hosts {
		ids := map[string]struct{}{}
		if collectEndpointSubtreeIDs(data.Hosts[i].Subsystems, id, ids) {
			if app := webAppUsingGateway(data.WebApps, ids); app != nil {
				return fmt.Errorf("resource is used as gateway by web app %q", app.Name)
			}
			deleteSubsystemByID(&data.Hosts[i].Subsystems, id)
			return s.store.Save(data)
		}
	}

	return fmt.Errorf("resource %q not found", id)
}

func (s *Service) ReorderTopLevel(order []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.store.Load()
	if err != nil {
		return err
	}
	wanted := make(map[string]struct{}, len(data.Hosts)+len(data.WebApps))
	for _, host := range data.Hosts {
		wanted[host.ID] = struct{}{}
	}
	for _, app := range data.WebApps {
		wanted[app.ID] = struct{}{}
	}
	if len(order) != len(wanted) {
		return fmt.Errorf("top-level order length = %d, want %d", len(order), len(wanted))
	}
	seen := make(map[string]struct{}, len(order))
	for _, rawID := range order {
		id := strings.TrimSpace(rawID)
		if _, exists := wanted[id]; !exists {
			return fmt.Errorf("top-level item %q not found", id)
		}
		if _, exists := seen[id]; exists {
			return fmt.Errorf("duplicate top-level item %q", id)
		}
		seen[id] = struct{}{}
	}
	data.TopLevelOrder = append([]string(nil), order...)
	return s.store.Save(data)
}

func (s *Service) ReorderHosts(order []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.store.Load()
	if err != nil {
		return err
	}
	if len(order) != len(data.Hosts) {
		return fmt.Errorf("host order length = %d, want %d", len(order), len(data.Hosts))
	}

	byID := make(map[string]domain.Host, len(data.Hosts))
	for _, host := range data.Hosts {
		byID[host.ID] = host
	}

	reordered := make([]domain.Host, 0, len(data.Hosts))
	seen := make(map[string]struct{}, len(order))
	for _, id := range order {
		id = strings.TrimSpace(id)
		host, ok := byID[id]
		if !ok {
			return fmt.Errorf("host %q not found", id)
		}
		if _, exists := seen[id]; exists {
			return fmt.Errorf("duplicate host %q in order", id)
		}
		seen[id] = struct{}{}
		reordered = append(reordered, host)
	}

	data.Hosts = reordered
	orderedHosts := make([]string, 0, len(order))
	orderedHosts = append(orderedHosts, order...)
	nextHost := 0
	for i, id := range data.TopLevelOrder {
		if _, isHost := byID[id]; isHost {
			data.TopLevelOrder[i] = orderedHosts[nextHost]
			nextHost++
		}
	}
	return s.store.Save(data)
}

func collectEndpointIDs(endpoints []domain.Endpoint, ids map[string]struct{}) {
	for _, endpoint := range endpoints {
		ids[endpoint.ID] = struct{}{}
		collectEndpointIDs(endpoint.Subsystems, ids)
	}
}

func collectEndpointSubtreeIDs(endpoints []domain.Endpoint, id string, ids map[string]struct{}) bool {
	for _, endpoint := range endpoints {
		if endpoint.ID == id {
			ids[endpoint.ID] = struct{}{}
			collectEndpointIDs(endpoint.Subsystems, ids)
			return true
		}
		if collectEndpointSubtreeIDs(endpoint.Subsystems, id, ids) {
			return true
		}
	}
	return false
}

func webAppUsingGateway(apps []domain.WebApp, ids map[string]struct{}) *domain.WebApp {
	for i := range apps {
		if _, exists := ids[apps[i].GatewayResourceID]; exists {
			return &apps[i]
		}
	}
	return nil
}

func removeID(ids []string, id string) []string {
	filtered := ids[:0]
	for _, current := range ids {
		if current != id {
			filtered = append(filtered, current)
		}
	}
	return filtered
}

func (s *Service) SetResourceAuth(id string, auth domain.Auth) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("resource id is required")
	}

	normalized, err := normalizeAuth(auth)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.store.Load()
	if err != nil {
		return err
	}

	for i := range data.Hosts {
		if data.Hosts[i].ID == id {
			data.Hosts[i].Auth = &normalized
			return s.store.Save(data)
		}
	}

	if subsystem := findSubsystemByID(data.Hosts, id); subsystem != nil {
		subsystem.Auth = &normalized
		return s.store.Save(data)
	}

	return fmt.Errorf("resource %q not found", id)
}

func (s *Service) SetResourceHostKeyFingerprint(id string, fingerprint string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("resource id is required")
	}
	fingerprint = strings.TrimSpace(fingerprint)
	if fingerprint == "" {
		return errors.New("host key fingerprint is required")
	}
	if hasControl(fingerprint) {
		return errors.New("host key fingerprint contains control characters")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.store.Load()
	if err != nil {
		return err
	}

	for i := range data.Hosts {
		if data.Hosts[i].ID == id {
			data.Hosts[i].HostKeyFingerprint = fingerprint
			return s.store.Save(data)
		}
	}

	if subsystem := findSubsystemByID(data.Hosts, id); subsystem != nil {
		subsystem.HostKeyFingerprint = fingerprint
		return s.store.Save(data)
	}

	return fmt.Errorf("resource %q not found", id)
}

func findSubsystemParent(hosts []domain.Host, parentID string) (*[]domain.Endpoint, []int) {
	for i := range hosts {
		if hosts[i].ID == parentID {
			return &hosts[i].Subsystems, []int{i}
		}
		if parent, parts := findSubsystemParentInEndpoints(hosts[i].Subsystems, parentID, []int{i}); parent != nil {
			return parent, parts
		}
	}
	return nil, nil
}

func findSubsystemParentInEndpoints(subsystems []domain.Endpoint, parentID string, parts []int) (*[]domain.Endpoint, []int) {
	for i := range subsystems {
		currentParts := append(append([]int{}, parts...), i)
		if subsystems[i].ID == parentID {
			return &subsystems[i].Subsystems, currentParts
		}
		if parent, childParts := findSubsystemParentInEndpoints(subsystems[i].Subsystems, parentID, currentParts); parent != nil {
			return parent, childParts
		}
	}
	return nil, nil
}

func findSubsystemByID(hosts []domain.Host, id string) *domain.Endpoint {
	for i := range hosts {
		if subsystem := findSubsystemInEndpoints(hosts[i].Subsystems, id); subsystem != nil {
			return subsystem
		}
	}
	return nil
}

func findSubsystemInEndpoints(subsystems []domain.Endpoint, id string) *domain.Endpoint {
	for i := range subsystems {
		if subsystems[i].ID == id {
			return &subsystems[i]
		}
		if subsystem := findSubsystemInEndpoints(subsystems[i].Subsystems, id); subsystem != nil {
			return subsystem
		}
	}
	return nil
}

func deleteSubsystemByID(subsystems *[]domain.Endpoint, id string) bool {
	for i := range *subsystems {
		if (*subsystems)[i].ID == id {
			*subsystems = append((*subsystems)[:i], (*subsystems)[i+1:]...)
			return true
		}
		if deleteSubsystemByID(&(*subsystems)[i].Subsystems, id) {
			return true
		}
	}
	return false
}

func normalizeAuth(auth domain.Auth) (domain.Auth, error) {
	auth.Method = domain.AuthMethod(strings.TrimSpace(string(auth.Method)))
	auth.KeyName = strings.TrimSpace(auth.KeyName)
	auth.PrivateKeyPath = strings.TrimSpace(auth.PrivateKeyPath)
	if !domain.ValidAuthMethod(auth.Method) {
		return domain.Auth{}, fmt.Errorf("invalid auth method %q", auth.Method)
	}
	switch auth.Method {
	case domain.AuthMethodKey:
		if auth.KeyName == "" {
			return domain.Auth{}, errors.New("key name is required for key auth")
		}
		auth.PrivateKeyPath = ""
	case domain.AuthMethodPath:
		if auth.PrivateKeyPath == "" {
			return domain.Auth{}, errors.New("private key path is required for path auth")
		}
		auth.KeyName = ""
	case domain.AuthMethodPassword, domain.AuthMethodAgent:
		auth.KeyName = ""
		auth.PrivateKeyPath = ""
	}
	return auth, nil
}

func hasControl(value string) bool {
	return strings.ContainsFunc(value, func(r rune) bool {
		return r < 0x20 || r == 0x7f
	})
}

func normalizeTagName(name string) (string, error) {
	tags := domain.NormalizeTags([]string{name})
	if len(tags) != 1 {
		return "", errors.New("tag name is required")
	}
	if err := domain.ValidateTags(tags); err != nil {
		return "", err
	}
	return tags[0], nil
}

func tagIndex(tags []string, name string) int {
	for i, tag := range tags {
		if strings.EqualFold(tag, strings.TrimSpace(name)) {
			return i
		}
	}
	return -1
}

func renameResourceTag(tags *[]string, currentName, newName string) {
	for i, tag := range *tags {
		if strings.EqualFold(tag, currentName) {
			(*tags)[i] = newName
		}
	}
	*tags = domain.NormalizeTags(*tags)
}

func renameSubsystemTags(subsystems []domain.Endpoint, currentName, newName string) {
	for i := range subsystems {
		renameResourceTag(&subsystems[i].Tags, currentName, newName)
		renameSubsystemTags(subsystems[i].Subsystems, currentName, newName)
	}
}

func removeResourceTag(tags *[]string, name string) {
	filtered := (*tags)[:0]
	for _, tag := range *tags {
		if !strings.EqualFold(tag, name) {
			filtered = append(filtered, tag)
		}
	}
	*tags = filtered
}

func removeSubsystemTags(subsystems []domain.Endpoint, name string) {
	for i := range subsystems {
		removeResourceTag(&subsystems[i].Tags, name)
		removeSubsystemTags(subsystems[i].Subsystems, name)
	}
}
