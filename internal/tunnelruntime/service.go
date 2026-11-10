package tunnelruntime

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/signoredellarete/bashes/internal/apperrors"
	"github.com/signoredellarete/bashes/internal/connectionauth"
	"github.com/signoredellarete/bashes/internal/domain"
	"github.com/signoredellarete/bashes/internal/remotessh"
	"github.com/signoredellarete/bashes/internal/sshauth"
	"github.com/signoredellarete/bashes/internal/sshconnection"
	"github.com/signoredellarete/bashes/internal/sshtunnel"
	"golang.org/x/crypto/ssh"
)

type ResourceStore interface {
	Endpoint(string) (domain.Endpoint, error)
	SetResourceAuth(string, domain.Auth) error
}

type KeyResolver interface {
	ManagedPrivatePath(string) string
}

type Input struct {
	ResourceID           string
	Type                 string
	LocalHost            string
	LocalPort            int
	RemoteHost           string
	RemotePort           int
	Password             string
	PrivateKeyPath       string
	KeyName              string
	PrivateKeyPassphrase string
	ManagePassword       bool
	SavePassword         bool
	TrustHostKey         bool
	AcceptHostKey        bool
	ReplaceHostKey       bool
	AllowPublicBind      bool
}

type Event struct {
	TunnelID string
	Message  string
	Closed   bool
}

type EventSink func(Event)

type client interface {
	Close() error
	Listen(string, string) (net.Listener, error)
}

type Service struct {
	resources ResourceStore
	auth      *connectionauth.Service
	keys      KeyResolver
	manager   *sshtunnel.Manager
	now       func() time.Time
	emit      EventSink
	dial      func(domain.Endpoint, sshconnection.Input, time.Duration) (client, error)
	listen    func(string, string) (net.Listener, error)
	serve     func(context.Context, string, net.Listener, client, string) error
	keepAlive func(context.Context, client, func(error))
}

func NewService(
	resources ResourceStore,
	auth *connectionauth.Service,
	keys KeyResolver,
	connector *sshconnection.Connector,
	manager *sshtunnel.Manager,
	now func() time.Time,
	emit EventSink,
) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{
		resources: resources, auth: auth, keys: keys, manager: manager, now: now, emit: emit,
		dial: func(resource domain.Endpoint, input sshconnection.Input, timeout time.Duration) (client, error) {
			return connector.Dial(resource, input, timeout)
		},
		listen: net.Listen,
		serve: func(ctx context.Context, kind string, listener net.Listener, connection client, target string) error {
			sshClient, ok := connection.(*ssh.Client)
			if !ok {
				return errors.New("invalid SSH client adapter")
			}
			switch kind {
			case "socks":
				return remotessh.ServeSOCKS5(ctx, listener, sshClient)
			case "local":
				return remotessh.ServeLocalForward(ctx, listener, sshClient, target)
			case "remote":
				return remotessh.ServeRemoteForward(ctx, listener, target)
			default:
				return fmt.Errorf("unsupported tunnel type %q", kind)
			}
		},
		keepAlive: func(ctx context.Context, connection client, failed func(error)) {
			sshClient, ok := connection.(*ssh.Client)
			if !ok {
				failed(errors.New("invalid SSH client adapter"))
				return
			}
			remotessh.MonitorKeepAlive(ctx, sshClient, remotessh.KeepAliveOptions{}, failed)
		},
	}
}

func (service *Service) Start(input Input) (sshtunnel.Info, error) {
	resource, err := service.resources.Endpoint(input.ResourceID)
	if err != nil {
		return sshtunnel.Info{}, err
	}
	if err := NormalizeInput(&input); err != nil {
		return sshtunnel.Info{}, err
	}
	prepared, err := service.auth.Prepare(resource, authInput(input))
	if err != nil {
		return sshtunnel.Info{}, err
	}
	applyAuthInput(&input, prepared)
	dialInput := service.resolveKeyPath(input)
	connection, err := service.dial(resource, connectionInput(dialInput), remotessh.DefaultTimeout)
	if err != nil {
		return sshtunnel.Info{}, err
	}
	if err := service.auth.PersistPasswordChoice(resource.ID, authInput(input)); err != nil {
		_ = connection.Close()
		return sshtunnel.Info{}, err
	}

	listener, forwardAddress, err := service.startListener(connection, input)
	if err != nil {
		_ = connection.Close()
		return sshtunnel.Info{}, err
	}
	now := service.now()
	tunnelID := fmt.Sprintf("tunnel-%d", now.UnixNano())
	info := sshtunnel.Info{
		TunnelID: tunnelID, ResourceID: resource.ID, Type: input.Type,
		LocalHost: input.LocalHost, LocalPort: input.LocalPort,
		LocalAddress:  listener.Addr().String(),
		Target:        fmt.Sprintf("%s@%s:%d", resource.User, sshconnection.Host(resource), resource.Port),
		ForwardTarget: ForwardTarget(input, forwardAddress), StartedAt: now.Format(time.RFC3339),
	}
	tunnel, err := service.manager.Register(sshtunnel.Options{
		Info: info, Client: connection, Listener: listener,
		Serve: func(ctx context.Context) error {
			return service.serve(ctx, info.Type, listener, connection, forwardAddress)
		},
	})
	if err != nil {
		_ = listener.Close()
		_ = connection.Close()
		return sshtunnel.Info{}, err
	}
	runtimeContext := service.manager.Context(tunnel)
	if err := service.resources.SetResourceAuth(resource.ID, *connectionauth.Preference(authInput(input))); err != nil {
		service.emitEvent(Event{TunnelID: tunnelID, Message: fmt.Sprintf("Could not save SSH auth preference: %v", err)})
	}

	go service.keepAlive(runtimeContext, connection, func(failure error) {
		if service.manager.Fail(tunnel, failure) {
			service.emitEvent(Event{
				TunnelID: tunnelID, Closed: true,
				Message: fmt.Sprintf("Tunnel closed: SSH keepalive failed: %v", failure),
			})
		}
	})
	go service.wait(tunnel)
	return info, nil
}

func (service *Service) Infos() []sshtunnel.Info {
	return service.manager.Infos()
}

func (service *Service) Stop(tunnelID string) {
	service.manager.Stop(tunnelID)
}

func (service *Service) StopForResource(resourceID string) {
	for _, tunnelID := range service.manager.IDsForResource(resourceID) {
		service.Stop(tunnelID)
	}
}

func (service *Service) StopAll() {
	for _, tunnelID := range service.manager.IDs() {
		service.Stop(tunnelID)
	}
}

func (service *Service) wait(tunnel *sshtunnel.Tunnel) {
	result := service.manager.Run(tunnel)
	if result.Stopped || !result.Authoritative || result.Err == nil {
		return
	}
	service.emitEvent(Event{
		TunnelID: result.ID, Closed: true, Message: fmt.Sprintf("Tunnel closed: %v", result.Err),
	})
}

func (service *Service) startListener(connection client, input Input) (net.Listener, string, error) {
	bindAddress := net.JoinHostPort(input.LocalHost, fmt.Sprint(input.LocalPort))
	switch input.Type {
	case "socks", "local":
		listener, err := service.listen("tcp", bindAddress)
		if err != nil {
			label := input.Type
			if input.Type == "socks" {
				label = "SOCKS"
			}
			return nil, "", fmt.Errorf("start %s tunnel listener: %w", label, err)
		}
		if input.Type == "socks" {
			return listener, "", nil
		}
		return listener, net.JoinHostPort(input.RemoteHost, fmt.Sprint(input.RemotePort)), nil
	case "remote":
		listener, err := connection.Listen("tcp", bindAddress)
		if err != nil {
			return nil, "", fmt.Errorf("start remote tunnel listener: %w", err)
		}
		return listener, net.JoinHostPort(input.RemoteHost, fmt.Sprint(input.RemotePort)), nil
	default:
		return nil, "", fmt.Errorf("unsupported tunnel type %q", input.Type)
	}
}

func (service *Service) resolveKeyPath(input Input) Input {
	if strings.TrimSpace(input.PrivateKeyPath) == "" && strings.TrimSpace(input.KeyName) != "" {
		input.PrivateKeyPath = service.keys.ManagedPrivatePath(input.KeyName)
	}
	return input
}

func (service *Service) emitEvent(event Event) {
	if service.emit != nil {
		service.emit(event)
	}
}

func NormalizeInput(input *Input) error {
	input.Type = strings.ToLower(strings.TrimSpace(input.Type))
	if input.Type == "" {
		input.Type = "socks"
	}
	if input.Type != "socks" && input.Type != "local" && input.Type != "remote" {
		return fmt.Errorf("unsupported tunnel type %q", input.Type)
	}
	input.LocalHost = strings.TrimSpace(input.LocalHost)
	if input.LocalHost == "" {
		input.LocalHost = "127.0.0.1"
	}
	if invalidHost(input.LocalHost) {
		return errors.New("local bind address contains invalid characters")
	}
	if !input.AllowPublicBind && !isLoopback(input.LocalHost) {
		return apperrors.New("public_tunnel_bind", "public tunnel bind requires confirmation", map[string]string{
			"host": input.LocalHost, "type": input.Type,
		})
	}
	if input.LocalPort < 1 || input.LocalPort > 65535 {
		return errors.New("local port must be between 1 and 65535")
	}
	if input.Type == "socks" {
		return nil
	}
	input.RemoteHost = strings.TrimSpace(input.RemoteHost)
	if input.RemoteHost == "" {
		input.RemoteHost = "127.0.0.1"
	}
	if invalidHost(input.RemoteHost) {
		return errors.New("forward target address contains invalid characters")
	}
	if input.RemotePort < 1 || input.RemotePort > 65535 {
		return errors.New("forward target port must be between 1 and 65535")
	}
	return nil
}

func ForwardTarget(input Input, forwardAddress string) string {
	if input.Type == "socks" {
		return "dynamic"
	}
	if input.Type == "local" || input.Type == "remote" {
		return forwardAddress
	}
	return ""
}

func isLoopback(host string) bool {
	host = strings.TrimSpace(strings.Trim(host, "[]"))
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func invalidHost(host string) bool {
	return strings.ContainsAny(host, "\r\n\t ")
}

func authInput(input Input) connectionauth.Input {
	return connectionauth.Input{
		Password: input.Password, PrivateKeyPath: input.PrivateKeyPath, KeyName: input.KeyName,
		ManagePassword: input.ManagePassword, SavePassword: input.SavePassword,
	}
}

func applyAuthInput(input *Input, auth connectionauth.Input) {
	input.Password = auth.Password
	input.PrivateKeyPath = auth.PrivateKeyPath
	input.KeyName = auth.KeyName
}

func connectionInput(input Input) sshconnection.Input {
	return sshconnection.Input{
		Authentication: sshauth.Input{
			Password: input.Password, PrivateKeyPath: input.PrivateKeyPath,
			KeyName: input.KeyName, Passphrase: input.PrivateKeyPassphrase,
		},
		TrustHostKey: input.TrustHostKey, AcceptHostKey: input.AcceptHostKey,
		ReplaceHostKey: input.ReplaceHostKey,
	}
}
