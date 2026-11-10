package connectionauth

import (
	"fmt"
	"strings"

	"github.com/signoredellarete/bashes/internal/credentials"
	"github.com/signoredellarete/bashes/internal/domain"
	"github.com/signoredellarete/bashes/internal/sshkeys"
)

type Input struct {
	Password       string
	PrivateKeyPath string
	KeyName        string
	ManagePassword bool
	SavePassword   bool
}

type Service struct {
	passwords credentials.Store
}

func NewService(passwords credentials.Store) *Service {
	return &Service{passwords: passwords}
}

func (s *Service) HasSavedPassword(resourceID string) (bool, error) {
	_, found, err := s.passwords.Password(resourceID)
	if err != nil {
		return false, fmt.Errorf("check saved SSH password: %w", err)
	}
	return found, nil
}

func (s *Service) Prepare(resource domain.Endpoint, input Input) (Input, error) {
	input = ApplyPreference(resource, input)
	if HasExplicitAuthentication(input) || resource.Auth == nil || resource.Auth.Method != domain.AuthMethodPassword {
		return input, nil
	}

	password, found, err := s.passwords.Password(resource.ID)
	if err != nil {
		return input, fmt.Errorf("read saved SSH password: %w", err)
	}
	if found {
		input.Password = password
	}
	return input, nil
}

func (s *Service) PersistPasswordChoice(resourceID string, input Input) error {
	if !input.ManagePassword {
		return nil
	}
	if input.SavePassword {
		if err := s.passwords.SavePassword(resourceID, input.Password); err != nil {
			return fmt.Errorf("save SSH password: %w", err)
		}
		return nil
	}
	return s.ForgetPassword(resourceID)
}

func (s *Service) ForgetPassword(resourceID string) error {
	if err := s.passwords.DeletePassword(resourceID); err != nil {
		return fmt.Errorf("remove saved SSH password: %w", err)
	}
	return nil
}

func ApplyPreference(resource domain.Endpoint, input Input) Input {
	if HasExplicitAuthentication(input) || resource.Auth == nil {
		return input
	}

	switch resource.Auth.Method {
	case domain.AuthMethodKey:
		input.KeyName = resource.Auth.KeyName
	case domain.AuthMethodPath:
		input.PrivateKeyPath = resource.Auth.PrivateKeyPath
	case domain.AuthMethodPassword, domain.AuthMethodAgent:
	}
	return input
}

func HasExplicitAuthentication(input Input) bool {
	return strings.TrimSpace(input.Password) != "" ||
		strings.TrimSpace(input.KeyName) != "" ||
		strings.TrimSpace(input.PrivateKeyPath) != ""
}

func Preference(input Input) *domain.Auth {
	auth := domain.Auth{}
	switch {
	case strings.TrimSpace(input.PrivateKeyPath) != "":
		auth.Method = domain.AuthMethodPath
		auth.PrivateKeyPath = strings.TrimSpace(input.PrivateKeyPath)
	case strings.TrimSpace(input.KeyName) != "":
		auth.Method = domain.AuthMethodKey
		auth.KeyName = sshkeys.SanitizeName(input.KeyName)
	case strings.TrimSpace(input.Password) != "":
		auth.Method = domain.AuthMethodPassword
	default:
		auth.Method = domain.AuthMethodAgent
	}
	return &auth
}
