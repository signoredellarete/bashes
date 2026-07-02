package application

import (
	"errors"
	"fmt"
	"strings"

	"github.com/signoredellarete/bashes/internal/domain"
)

type Store interface {
	Load() (domain.Store, error)
	Save(domain.Store) error
}

type Service struct {
	store Store
}

type EndpointInput struct {
	Hostname string              `json:"hostname"`
	IP       string              `json:"ip"`
	Port     int                 `json:"port"`
	User     string              `json:"user"`
	Type     domain.ResourceType `json:"type,omitempty"`
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) ListHosts() ([]domain.Host, error) {
	data, err := s.store.Load()
	if err != nil {
		return nil, err
	}
	return data.Hosts, nil
}

func (s *Service) AddHost(input EndpointInput) (domain.Host, error) {
	data, err := s.store.Load()
	if err != nil {
		return domain.Host{}, err
	}

	host := domain.Host{
		Hostname:   strings.TrimSpace(input.Hostname),
		IP:         strings.TrimSpace(input.IP),
		Port:       input.Port,
		User:       strings.TrimSpace(input.User),
		Subsystems: []domain.Endpoint{},
	}
	host.ID = domain.StableID(domain.ResourceHost, host.Hostname, host.IP, host.Port, host.User, len(data.Hosts))

	data.Hosts = append(data.Hosts, host)
	if err := s.store.Save(data); err != nil {
		return domain.Host{}, err
	}

	return host, nil
}

func (s *Service) AddSubsystem(hostID string, input EndpointInput) (domain.Endpoint, error) {
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

	data, err := s.store.Load()
	if err != nil {
		return err
	}

	for i := range data.Hosts {
		if data.Hosts[i].ID == id {
			data.Hosts[i].Hostname = strings.TrimSpace(input.Hostname)
			data.Hosts[i].IP = strings.TrimSpace(input.IP)
			data.Hosts[i].Port = input.Port
			data.Hosts[i].User = strings.TrimSpace(input.User)
			return s.store.Save(data)
		}
	}

	if subsystem := findSubsystemByID(data.Hosts, id); subsystem != nil {
		if !domain.ValidResourceType(input.Type) || input.Type == domain.ResourceHost {
			return fmt.Errorf("invalid subsystem type %q", input.Type)
		}
		subsystem.Type = input.Type
		subsystem.Hostname = strings.TrimSpace(input.Hostname)
		subsystem.IP = strings.TrimSpace(input.IP)
		subsystem.Port = input.Port
		subsystem.User = strings.TrimSpace(input.User)
		return s.store.Save(data)
	}

	return fmt.Errorf("resource %q not found", id)
}

func (s *Service) DeleteResource(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("resource id is required")
	}

	data, err := s.store.Load()
	if err != nil {
		return err
	}

	for i := range data.Hosts {
		if data.Hosts[i].ID == id {
			data.Hosts = append(data.Hosts[:i], data.Hosts[i+1:]...)
			return s.store.Save(data)
		}
	}

	for i := range data.Hosts {
		if deleteSubsystemByID(&data.Hosts[i].Subsystems, id) {
			return s.store.Save(data)
		}
	}

	return fmt.Errorf("resource %q not found", id)
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
