package main

import (
	"github.com/signoredellarete/bashes/internal/connectionauth"
	"github.com/signoredellarete/bashes/internal/domain"
)

func (a *App) HasSavedPassword(resourceID string) (bool, error) {
	resource, err := a.resourceByID(resourceID)
	if err != nil {
		return false, err
	}
	return a.authPreferences.HasSavedPassword(resource.ID)
}

func (a *App) prepareSessionInput(resource domain.Endpoint, input SSHSessionInput) (SSHSessionInput, error) {
	prepared, err := a.authPreferences.Prepare(resource, connectionAuthInput(input))
	if err != nil {
		return input, err
	}
	return sshSessionInput(prepared, input), nil
}

func (a *App) persistPasswordChoice(resourceID string, input SSHSessionInput) error {
	return a.authPreferences.PersistPasswordChoice(resourceID, connectionAuthInput(input))
}

func connectionAuthInput(input SSHSessionInput) connectionauth.Input {
	return connectionauth.Input{
		Password:       input.Password,
		PrivateKeyPath: input.PrivateKeyPath,
		KeyName:        input.KeyName,
		ManagePassword: input.ManagePassword,
		SavePassword:   input.SavePassword,
	}
}

func sshSessionInput(auth connectionauth.Input, input SSHSessionInput) SSHSessionInput {
	input.Password = auth.Password
	input.PrivateKeyPath = auth.PrivateKeyPath
	input.KeyName = auth.KeyName
	input.ManagePassword = auth.ManagePassword
	input.SavePassword = auth.SavePassword
	return input
}
