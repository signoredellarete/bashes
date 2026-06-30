package remotessh

import (
	"context"
	"errors"
	"net"

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

	go func() {
		<-ctx.Done()
		_ = listener.Close()
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

		go handleForwardConn(conn, dial)
	}
}

func handleForwardConn(conn net.Conn, dial func() (net.Conn, error)) {
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
