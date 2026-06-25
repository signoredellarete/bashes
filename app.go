package main

import (
	"path/filepath"

	"github.com/signoredellarete/bashes/internal/application"
	"github.com/signoredellarete/bashes/internal/domain"
	"github.com/signoredellarete/bashes/internal/store"
)

type App struct {
	service *application.Service
}

func NewApp(dataPath string) *App {
	if dataPath == "" {
		dataPath = filepath.Join("data", "hosts.json")
	}

	return &App{
		service: application.NewService(store.NewRepository(dataPath)),
	}
}

func (a *App) ListHosts() ([]domain.Host, error) {
	return a.service.ListHosts()
}

func (a *App) AddHost(input application.EndpointInput) (domain.Host, error) {
	return a.service.AddHost(input)
}

func (a *App) AddSubsystem(hostID string, input application.EndpointInput) (domain.Endpoint, error) {
	return a.service.AddSubsystem(hostID, input)
}

func (a *App) DeleteResource(id string) error {
	return a.service.DeleteResource(id)
}
