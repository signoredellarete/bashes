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

	hostIndex := -1
	for i := range data.Hosts {
		if data.Hosts[i].ID == hostID {
			hostIndex = i
			break
		}
	}
	if hostIndex < 0 {
		return domain.Endpoint{}, fmt.Errorf("host %q not found", hostID)
	}

	if !domain.ValidResourceType(input.Type) || input.Type == domain.ResourceHost {
		return domain.Endpoint{}, fmt.Errorf("invalid subsystem type %q", input.Type)
	}

	subsystem := domain.Endpoint{
		Type:     input.Type,
		Hostname: strings.TrimSpace(input.Hostname),
		IP:       strings.TrimSpace(input.IP),
		Port:     input.Port,
		User:     strings.TrimSpace(input.User),
	}
	subsystem.ID = domain.StableID(
		subsystem.Type,
		subsystem.Hostname,
		subsystem.IP,
		subsystem.Port,
		subsystem.User,
		hostIndex,
		len(data.Hosts[hostIndex].Subsystems),
	)

	data.Hosts[hostIndex].Subsystems = append(data.Hosts[hostIndex].Subsystems, subsystem)
	if err := s.store.Save(data); err != nil {
		return domain.Endpoint{}, err
	}

	return subsystem, nil
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
		subsystems := data.Hosts[i].Subsystems
		for j := range subsystems {
			if subsystems[j].ID == id {
				data.Hosts[i].Subsystems = append(subsystems[:j], subsystems[j+1:]...)
				return s.store.Save(data)
			}
		}
	}

	return fmt.Errorf("resource %q not found", id)
}
