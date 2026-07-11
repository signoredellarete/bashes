package remotessh

import (
	"context"
	"errors"
	"net"
	"sync"

	"golang.org/x/crypto/ssh"
)

func ServeLocalForward(ctx context.Context, listener net.Listener, client *ssh.Client, remoteAddress string) error {
	if client == nil {
		return errors.New("ssh client is required")
	}
	return serveForward(ctx, listener, func() (net.Conn, error) {
		return client.Dial("tcp", remoteAddress)
	})
}

func ServeRemoteForward(ctx context.Context, listener net.Listener, localAddress string) error {
	return serveForward(ctx, listener, func() (net.Conn, error) {
		return net.Dial("tcp", localAddress)
	})
}

func serveForward(ctx context.Context, listener net.Listener, dial func() (net.Conn, error)) error {
	if listener == nil {
		return errors.New("tunnel listener is required")
	}
	if dial == nil {
		return errors.New("tunnel dialer is required")
	}

	connections := newConnectionSet()
	go func() {
		<-ctx.Done()
		_ = listener.Close()
		connections.closeAll()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				return err
			}
		}

		connections.add(conn)
		go handleForwardConn(conn, dial, connections)
	}
}

func handleForwardConn(conn net.Conn, dial func() (net.Conn, error), connections *connectionSet) {
	defer connections.remove(conn)
	defer conn.Close()

	remote, err := dial()
	if err != nil {
		return
	}
	defer remote.Close()

	done := make(chan struct{}, 2)
	go proxyCopy(remote, conn, done)
	go proxyCopy(conn, remote, done)
	<-done
}

type connectionSet struct {
	mu    sync.Mutex
	items map[net.Conn]struct{}
}

func newConnectionSet() *connectionSet {
	return &connectionSet{items: make(map[net.Conn]struct{})}
}

func (s *connectionSet) add(conn net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[conn] = struct{}{}
}

func (s *connectionSet) remove(conn net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, conn)
}

func (s *connectionSet) closeAll() {
	s.mu.Lock()
	connections := make([]net.Conn, 0, len(s.items))
	for conn := range s.items {
		connections = append(connections, conn)
	}
	s.items = make(map[net.Conn]struct{})
	s.mu.Unlock()
	for _, conn := range connections {
		_ = conn.Close()
	}
}
