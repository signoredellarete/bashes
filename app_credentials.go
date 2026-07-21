package main

import (
	"fmt"

	"github.com/signoredellarete/bashes/internal/domain"
)

func (a *App) HasSavedPassword(resourceID string) (bool, error) {
	resource, err := a.resourceByID(resourceID)
	if err != nil {
		return false, err
	}
	_, found, err := a.passwords.Password(resource.ID)
	if err != nil {
		return false, fmt.Errorf("check saved SSH password: %w", err)
	}
	return found, nil
}

func (a *App) prepareSessionInput(resource domain.Endpoint, input SSHSessionInput) (SSHSessionInput, error) {
	input = applyAuthPreference(resource, input)
	if hasExplicitAuth(input) || resource.Auth == nil || resource.Auth.Method != domain.AuthMethodPassword {
		return input, nil
	}

	password, found, err := a.passwords.Password(resource.ID)
	if err != nil {
		return input, fmt.Errorf("read saved SSH password: %w", err)
	}
	if found {
		input.Password = password
	}
	return input, nil
}

func (a *App) persistPasswordChoice(resourceID string, input SSHSessionInput) error {
	if !input.ManagePassword {
		return nil
	}
	if input.SavePassword {
		if err := a.passwords.SavePassword(resourceID, input.Password); err != nil {
			return fmt.Errorf("save SSH password: %w", err)
		}
		return nil
	}
	if err := a.passwords.DeletePassword(resourceID); err != nil {
		return fmt.Errorf("remove saved SSH password: %w", err)
	}
	return nil
}
