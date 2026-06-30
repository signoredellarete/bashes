package remotessh

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"

	"golang.org/x/crypto/ssh"
)

type DialFunc func(network string, address string) (net.Conn, error)

func ServeSOCKS5(ctx context.Context, listener net.Listener, client *ssh.Client) error {
	if listener == nil {
		return errors.New("socks listener is required")
	}
	if client == nil {
		return errors.New("ssh client is required")
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

		go func() {
			_ = HandleSOCKS5(conn, client.Dial)
		}()
	}
}

func HandleSOCKS5(conn net.Conn, dial DialFunc) error {
	defer conn.Close()

	if dial == nil {
		return errors.New("socks dialer is required")
	}

	if err := readSOCKS5Greeting(conn); err != nil {
		return err
	}
	if _, err := conn.Write([]byte{0x05, 0x00}); err != nil {
		return err
	}

	target, err := readSOCKS5ConnectTarget(conn)
	if err != nil {
		_ = writeSOCKS5Reply(conn, 0x07)
		return err
	}

	remote, err := dial("tcp", target)
	if err != nil {
		_ = writeSOCKS5Reply(conn, 0x05)
		return err
	}
	defer remote.Close()

	if err := writeSOCKS5Reply(conn, 0x00); err != nil {
		return err
	}

	done := make(chan struct{}, 2)
	go proxyCopy(remote, conn, done)
	go proxyCopy(conn, remote, done)
	<-done
	return nil
}

func readSOCKS5Greeting(conn net.Conn) error {
	header := make([]byte, 2)
	if _, err := io.ReadFull(conn, header); err != nil {
		return err
	}
	if header[0] != 0x05 {
		return fmt.Errorf("unsupported socks version %d", header[0])
	}
	methods := make([]byte, int(header[1]))
	if _, err := io.ReadFull(conn, methods); err != nil {
		return err
	}
	for _, method := range methods {
		if method == 0x00 {
			return nil
		}
	}
	_, _ = conn.Write([]byte{0x05, 0xff})
	return errors.New("socks client does not support no-auth mode")
}

func readSOCKS5ConnectTarget(conn net.Conn) (string, error) {
	header := make([]byte, 4)
	if _, err := io.ReadFull(conn, header); err != nil {
		return "", err
	}
	if header[0] != 0x05 {
		return "", fmt.Errorf("unsupported socks version %d", header[0])
	}
	if header[1] != 0x01 {
		return "", fmt.Errorf("unsupported socks command %d", header[1])
	}

	host, err := readSOCKS5Host(conn, header[3])
	if err != nil {
		return "", err
	}

	portBytes := make([]byte, 2)
	if _, err := io.ReadFull(conn, portBytes); err != nil {
		return "", err
	}
	port := int(binary.BigEndian.Uint16(portBytes))
	if port < 1 || port > 65535 {
		return "", fmt.Errorf("invalid socks target port %d", port)
	}
	return net.JoinHostPort(host, strconv.Itoa(port)), nil
}

func readSOCKS5Host(conn net.Conn, addressType byte) (string, error) {
	switch addressType {
	case 0x01:
		ip := make([]byte, net.IPv4len)
		if _, err := io.ReadFull(conn, ip); err != nil {
			return "", err
		}
		return net.IP(ip).String(), nil
	case 0x03:
		length := make([]byte, 1)
		if _, err := io.ReadFull(conn, length); err != nil {
			return "", err
		}
		if length[0] == 0 {
			return "", errors.New("empty socks domain name")
		}
		domain := make([]byte, int(length[0]))
		if _, err := io.ReadFull(conn, domain); err != nil {
			return "", err
		}
		return string(domain), nil
	case 0x04:
		ip := make([]byte, net.IPv6len)
		if _, err := io.ReadFull(conn, ip); err != nil {
			return "", err
		}
		return net.IP(ip).String(), nil
	default:
		return "", fmt.Errorf("unsupported socks address type %d", addressType)
	}
}

func writeSOCKS5Reply(conn net.Conn, status byte) error {
	_, err := conn.Write([]byte{0x05, status, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
	return err
}

func proxyCopy(dst net.Conn, src net.Conn, done chan<- struct{}) {
	_, _ = io.Copy(dst, src)
	_ = dst.Close()
	_ = src.Close()
	done <- struct{}{}
}
