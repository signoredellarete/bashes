package terminalruntime

import (
	"context"
	"errors"
	"fmt"
	"io"
	"runtime"
	"strings"
	"time"

	"github.com/signoredellarete/bashes/internal/connectionauth"
	"github.com/signoredellarete/bashes/internal/domain"
	"github.com/signoredellarete/bashes/internal/localterm"
	"github.com/signoredellarete/bashes/internal/remotessh"
	"github.com/signoredellarete/bashes/internal/sshauth"
	"github.com/signoredellarete/bashes/internal/sshconnection"
	"github.com/signoredellarete/bashes/internal/termsession"
	"golang.org/x/crypto/ssh"
)

const LocalResourceID = "__bashes_localhost__"

type ResourceStore interface {
	Endpoint(string) (domain.Endpoint, error)
	SetResourceAuth(string, domain.Auth) error
}

type KeyResolver interface {
	ManagedPrivatePath(string) string
}

type SSHInput struct {
	ResourceID           string
	Password             string
	PrivateKeyPath       string
	KeyName              string
	PrivateKeyPassphrase string
	ManagePassword       bool
	SavePassword         bool
	TrustHostKey         bool
	AcceptHostKey        bool
	ReplaceHostKey       bool
	Cols                 int
	Rows                 int
}

type LocalInput struct {
	Cols int
	Rows int
}

type EventKind string

const (
	EventOutput EventKind = "output"
	EventStatus EventKind = "status"
	EventClosed EventKind = "closed"
)

type Event struct {
	Kind      EventKind
	SessionID string
	Data      []byte
	Message   string
}

type EventSink func(Event)

type remoteClient interface {
	Close() error
}

type Service struct {
	resources   ResourceStore
	auth        *connectionauth.Service
	keys        KeyResolver
	manager     *termsession.Manager
	goos        string
	now         func() time.Time
	emit        EventSink
	dial        func(domain.Endpoint, sshconnection.Input, time.Duration) (remoteClient, error)
	startRemote func(remoteClient, remotessh.ShellOptions) (termsession.Shell, error)
	startLocal  func(localterm.ShellOptions) (termsession.Shell, error)
	keepAlive   func(context.Context, remoteClient, func(error))
}

func NewService(
	resources ResourceStore,
	auth *connectionauth.Service,
	keys KeyResolver,
	connector *sshconnection.Connector,
	manager *termsession.Manager,
	emit EventSink,
) *Service {
	return &Service{
		resources: resources,
		auth:      auth,
		keys:      keys,
		manager:   manager,
		goos:      runtime.GOOS,
		now:       time.Now,
		emit:      emit,
		dial: func(resource domain.Endpoint, input sshconnection.Input, timeout time.Duration) (remoteClient, error) {
			return connector.Dial(resource, input, timeout)
		},
		startRemote: func(client remoteClient, options remotessh.ShellOptions) (termsession.Shell, error) {
			sshClient, ok := client.(*ssh.Client)
			if !ok {
				return nil, errors.New("invalid SSH client adapter")
			}
			return remotessh.StartShell(sshClient, options)
		},
		startLocal: func(options localterm.ShellOptions) (termsession.Shell, error) {
			return localterm.StartShell(options)
		},
		keepAlive: func(ctx context.Context, client remoteClient, failed func(error)) {
			sshClient, ok := client.(*ssh.Client)
			if !ok {
				failed(errors.New("invalid SSH client adapter"))
				return
			}
			remotessh.MonitorKeepAlive(ctx, sshClient, remotessh.KeepAliveOptions{}, failed)
		},
	}
}

func (s *Service) SupportsLocalShell() bool {
	return s.goos != "windows"
}

func (s *Service) StartSSH(input SSHInput) (string, error) {
	resource, err := s.resources.Endpoint(input.ResourceID)
	if err != nil {
		return "", err
	}
	input, err = s.prepareInput(resource, input)
	if err != nil {
		return "", err
	}
	dialInput := s.resolveKeyPath(input)

	client, err := s.dial(resource, connectionInput(dialInput), remotessh.DefaultTimeout)
	if err != nil {
		return "", err
	}
	if err := s.auth.PersistPasswordChoice(resource.ID, authInput(input)); err != nil {
		_ = client.Close()
		return "", err
	}

	stdinReader, stdinWriter := io.Pipe()
	sessionID := fmt.Sprintf("ssh-%d", s.now().UnixNano())
	writer := eventWriter{sessionID: sessionID, emit: s.emitEvent}
	shell, err := s.startRemote(client, remotessh.ShellOptions{
		Size:  remotessh.TerminalSize{Cols: input.Cols, Rows: input.Rows},
		Stdin: stdinReader, Stdout: writer, Stderr: writer,
	})
	if err != nil {
		_ = stdinWriter.Close()
		_ = client.Close()
		return "", err
	}

	session, err := s.manager.Register(termsession.Options{
		ID: sessionID, ResourceID: resource.ID, Kind: termsession.KindSSH,
		Client: client, Shell: shell, Stdin: stdinWriter,
	})
	if err != nil {
		_ = stdinWriter.Close()
		_ = shell.Close()
		_ = client.Close()
		return "", err
	}
	runtimeContext := s.manager.Context(session)
	s.emitEvent(Event{Kind: EventStatus, SessionID: sessionID, Message: fmt.Sprintf(
		"Connected to %s@%s:%d", resource.User, endpointHost(resource), resource.Port,
	)})
	if err := s.resources.SetResourceAuth(resource.ID, *connectionauth.Preference(authInput(input))); err != nil {
		s.emitEvent(Event{Kind: EventStatus, SessionID: sessionID, Message: fmt.Sprintf(
			"Could not save SSH auth preference: %v", err,
		)})
	}

	go s.keepAlive(runtimeContext, client, func(error) {
		_ = shell.Close()
		_ = client.Close()
	})
	go s.wait(sessionID, session)
	return sessionID, nil
}

func (s *Service) StartLocal(input LocalInput) (string, error) {
	if !s.SupportsLocalShell() {
		return "", errors.New("local shell is not supported on Windows yet")
	}

	stdinReader, stdinWriter := io.Pipe()
	sessionID := fmt.Sprintf("local-%d", s.now().UnixNano())
	writer := eventWriter{sessionID: sessionID, emit: s.emitEvent}
	shell, err := s.startLocal(localterm.ShellOptions{
		Size:  remotessh.TerminalSize{Cols: input.Cols, Rows: input.Rows},
		Stdin: stdinReader, Stdout: writer, Stderr: writer,
	})
	if err != nil {
		_ = stdinWriter.Close()
		return "", err
	}

	session, err := s.manager.Register(termsession.Options{
		ID: sessionID, ResourceID: LocalResourceID, Kind: termsession.KindLocal,
		Shell: shell, Stdin: stdinWriter,
	})
	if err != nil {
		_ = stdinWriter.Close()
		_ = shell.Close()
		return "", err
	}
	s.emitEvent(Event{Kind: EventStatus, SessionID: sessionID, Message: "Local shell started"})
	go s.wait(sessionID, session)
	return sessionID, nil
}

func (s *Service) Write(sessionID string, data string) error {
	return s.manager.Write(sessionID, data)
}

func (s *Service) Resize(sessionID string, cols int, rows int) error {
	return s.manager.Resize(sessionID, remotessh.TerminalSize{Cols: cols, Rows: rows})
}

func (s *Service) Stop(sessionID string) bool {
	kind, stopped := s.manager.Stop(sessionID)
	if stopped {
		s.emitEvent(Event{Kind: EventClosed, SessionID: sessionID, Message: closedMessage(kind)})
	}
	return stopped
}

func (s *Service) StopForResource(resourceID string) {
	for _, sessionID := range s.manager.IDsForResource(resourceID) {
		s.Stop(sessionID)
	}
}

func (s *Service) StopAll() {
	for _, sessionID := range s.manager.IDs() {
		s.Stop(sessionID)
	}
}

func (s *Service) prepareInput(resource domain.Endpoint, input SSHInput) (SSHInput, error) {
	prepared, err := s.auth.Prepare(resource, authInput(input))
	if err != nil {
		return input, err
	}
	input.Password = prepared.Password
	input.PrivateKeyPath = prepared.PrivateKeyPath
	input.KeyName = prepared.KeyName
	return input, nil
}

func (s *Service) resolveKeyPath(input SSHInput) SSHInput {
	if strings.TrimSpace(input.PrivateKeyPath) == "" && strings.TrimSpace(input.KeyName) != "" {
		input.PrivateKeyPath = s.keys.ManagedPrivatePath(input.KeyName)
	}
	return input
}

func (s *Service) wait(sessionID string, session *termsession.Session) {
	result := s.manager.Wait(session)
	if result.Stopped || !result.Authoritative {
		return
	}
	message := closedMessage(result.Kind)
	if result.Err != nil && !errors.Is(result.Err, io.EOF) {
		message = result.Err.Error()
	}
	s.emitEvent(Event{Kind: EventClosed, SessionID: sessionID, Message: message})
}

func (s *Service) emitEvent(event Event) {
	if s.emit != nil {
		s.emit(event)
	}
}

func authInput(input SSHInput) connectionauth.Input {
	return connectionauth.Input{
		Password: input.Password, PrivateKeyPath: input.PrivateKeyPath, KeyName: input.KeyName,
		ManagePassword: input.ManagePassword, SavePassword: input.SavePassword,
	}
}

func connectionInput(input SSHInput) sshconnection.Input {
	return sshconnection.Input{
		Authentication: sshauth.Input{
			Password: input.Password, PrivateKeyPath: input.PrivateKeyPath,
			KeyName: input.KeyName, Passphrase: input.PrivateKeyPassphrase,
		},
		TrustHostKey: input.TrustHostKey, AcceptHostKey: input.AcceptHostKey,
		ReplaceHostKey: input.ReplaceHostKey,
	}
}

func endpointHost(resource domain.Endpoint) string {
	if strings.TrimSpace(resource.IP) != "" {
		return resource.IP
	}
	return resource.Hostname
}

func closedMessage(kind termsession.Kind) string {
	if kind == termsession.KindLocal {
		return "Local shell closed"
	}
	return "SSH session closed"
}

type eventWriter struct {
	sessionID string
	emit      EventSink
}

func (writer eventWriter) Write(data []byte) (int, error) {
	if len(data) > 0 && writer.emit != nil {
		copyOfData := append([]byte(nil), data...)
		writer.emit(Event{Kind: EventOutput, SessionID: writer.sessionID, Data: copyOfData})
	}
	return len(data), nil
}
