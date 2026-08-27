package remotessh

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"

	"golang.org/x/crypto/ssh"
)

const (
	x11AuthProtocol = "MIT-MAGIC-COOKIE-1"
	x11CookieSize   = 16
	x11MaxAuthSize  = 4096
)

type X11Options struct {
	LocalAddress     string
	LocalCookie      []byte
	Screen           uint32
	SingleConnection bool
}

type x11Request struct {
	SingleConnection       bool
	AuthenticationProtocol string
	AuthenticationCookie   string
	Screen                 uint32
}

type X11Forwarder struct {
	cancel      context.CancelFunc
	connections *closerSet
	closeOnce   sync.Once
}

func StartX11Forwarding(client *ssh.Client, session *ssh.Session, options X11Options) (*X11Forwarder, error) {
	if client == nil || session == nil {
		return nil, errors.New("ssh client and session are required for X11 forwarding")
	}
	if _, _, err := net.SplitHostPort(options.LocalAddress); err != nil {
		return nil, fmt.Errorf("invalid local X server address: %w", err)
	}
	if len(options.LocalCookie) != x11CookieSize {
		return nil, fmt.Errorf("local X11 cookie must be %d bytes", x11CookieSize)
	}

	fakeCookie := make([]byte, x11CookieSize)
	if _, err := rand.Read(fakeCookie); err != nil {
		return nil, fmt.Errorf("generate X11 forwarding cookie: %w", err)
	}

	channels := client.HandleChannelOpen("x11")
	accepted, err := session.SendRequest("x11-req", true, ssh.Marshal(x11Request{
		SingleConnection:       options.SingleConnection,
		AuthenticationProtocol: x11AuthProtocol,
		AuthenticationCookie:   hex.EncodeToString(fakeCookie),
		Screen:                 options.Screen,
	}))
	if err != nil {
		return nil, fmt.Errorf("request SSH X11 forwarding: %w", err)
	}
	if !accepted {
		return nil, errors.New("SSH server rejected X11 forwarding; check sshd X11Forwarding and remote xauth")
	}

	ctx, cancel := context.WithCancel(context.Background())
	forwarder := &X11Forwarder{
		cancel:      cancel,
		connections: newCloserSet(),
	}
	go forwarder.serve(ctx, channels, options.LocalAddress, fakeCookie, options.LocalCookie)
	return forwarder, nil
}

func (f *X11Forwarder) Close() error {
	if f == nil {
		return nil
	}
	f.closeOnce.Do(func() {
		f.cancel()
		f.connections.closeAll()
	})
	return nil
}

func (f *X11Forwarder) serve(ctx context.Context, channels <-chan ssh.NewChannel, localAddress string, fakeCookie, localCookie []byte) {
	for {
		select {
		case <-ctx.Done():
			return
		case newChannel, ok := <-channels:
			if !ok {
				return
			}
			go f.handleChannel(ctx, newChannel, localAddress, fakeCookie, localCookie)
		}
	}
}

func (f *X11Forwarder) handleChannel(ctx context.Context, newChannel ssh.NewChannel, localAddress string, fakeCookie, localCookie []byte) {
	channel, requests, err := newChannel.Accept()
	if err != nil {
		return
	}
	f.connections.add(channel)
	defer func() {
		f.connections.remove(channel)
		_ = channel.Close()
	}()
	go ssh.DiscardRequests(requests)

	local, err := (&net.Dialer{}).DialContext(ctx, "tcp", localAddress)
	if err != nil {
		return
	}
	f.connections.add(local)
	defer func() {
		f.connections.remove(local)
		_ = local.Close()
	}()

	if err := proxyX11Connection(channel, local, fakeCookie, localCookie); err != nil {
		return
	}
}

func proxyX11Connection(remote io.ReadWriter, local io.ReadWriter, fakeCookie, localCookie []byte) error {
	if err := rewriteX11Setup(remote, local, fakeCookie, localCookie); err != nil {
		return err
	}

	done := make(chan struct{}, 2)
	go func() {
		_, _ = io.Copy(local, remote)
		done <- struct{}{}
	}()
	go func() {
		_, _ = io.Copy(remote, local)
		done <- struct{}{}
	}()
	<-done
	return nil
}

func rewriteX11Setup(reader io.Reader, writer io.Writer, fakeCookie, localCookie []byte) error {
	if len(fakeCookie) == 0 || len(fakeCookie) != len(localCookie) {
		return errors.New("X11 cookies must have the same non-zero length")
	}

	header := make([]byte, 12)
	if _, err := io.ReadFull(reader, header); err != nil {
		return fmt.Errorf("read X11 setup header: %w", err)
	}

	var order binary.ByteOrder
	switch header[0] {
	case 'B':
		order = binary.BigEndian
	case 'l':
		order = binary.LittleEndian
	default:
		return fmt.Errorf("unsupported X11 byte order %q", header[0])
	}

	authNameLength := int(order.Uint16(header[6:8]))
	authDataLength := int(order.Uint16(header[8:10]))
	if authNameLength > x11MaxAuthSize || authDataLength > x11MaxAuthSize {
		return errors.New("X11 authentication payload is too large")
	}

	paddedNameLength := paddedX11Length(authNameLength)
	paddedDataLength := paddedX11Length(authDataLength)
	payload := make([]byte, paddedNameLength+paddedDataLength)
	if _, err := io.ReadFull(reader, payload); err != nil {
		return fmt.Errorf("read X11 setup authentication: %w", err)
	}

	authName := payload[:authNameLength]
	authData := payload[paddedNameLength : paddedNameLength+authDataLength]
	if string(authName) != x11AuthProtocol {
		return fmt.Errorf("unsupported X11 authentication protocol %q", authName)
	}
	if len(authData) != len(fakeCookie) || subtle.ConstantTimeCompare(authData, fakeCookie) != 1 {
		return errors.New("X11 authentication cookie does not match the SSH forwarding cookie")
	}
	copy(authData, localCookie)

	if err := writeX11Packet(writer, header); err != nil {
		return fmt.Errorf("write X11 setup header: %w", err)
	}
	if err := writeX11Packet(writer, payload); err != nil {
		return fmt.Errorf("write X11 setup authentication: %w", err)
	}
	return nil
}

func paddedX11Length(length int) int {
	return (length + 3) &^ 3
}

func writeX11Packet(writer io.Writer, data []byte) error {
	for len(data) > 0 {
		written, err := writer.Write(data)
		if err != nil {
			return err
		}
		if written == 0 {
			return io.ErrShortWrite
		}
		data = data[written:]
	}
	return nil
}

type closerSet struct {
	mu    sync.Mutex
	items map[io.Closer]struct{}
}

func newCloserSet() *closerSet {
	return &closerSet{items: make(map[io.Closer]struct{})}
}

func (s *closerSet) add(item io.Closer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[item] = struct{}{}
}

func (s *closerSet) remove(item io.Closer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, item)
}

func (s *closerSet) closeAll() {
	s.mu.Lock()
	items := make([]io.Closer, 0, len(s.items))
	for item := range s.items {
		items = append(items, item)
	}
	s.items = make(map[io.Closer]struct{})
	s.mu.Unlock()
	for _, item := range items {
		_ = item.Close()
	}
}
