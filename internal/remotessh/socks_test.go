package remotessh

import (
	"io"
	"net"
	"testing"
)

func TestHandleSOCKS5ConnectsToDomainTarget(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()

	targetClient, targetServer := net.Pipe()
	defer targetClient.Close()

	targets := make(chan string, 1)
	errs := make(chan error, 1)
	go func() {
		errs <- HandleSOCKS5(server, func(network string, address string) (net.Conn, error) {
			targets <- network + " " + address
			return targetServer, nil
		})
	}()

	writeAll(t, client, []byte{0x05, 0x01, 0x00})
	readExact(t, client, []byte{0x05, 0x00})

	request := []byte{
		0x05, 0x01, 0x00, 0x03,
		byte(len("example.test")),
	}
	request = append(request, []byte("example.test")...)
	request = append(request, 0x01, 0xbb)
	writeAll(t, client, request)
	readExact(t, client, []byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0})

	got := <-targets
	if got != "tcp example.test:443" {
		t.Fatalf("dial target = %q, want tcp example.test:443", got)
	}

	writeAll(t, client, []byte("ping"))
	readExact(t, targetClient, []byte("ping"))

	writeAll(t, targetClient, []byte("pong"))
	readExact(t, client, []byte("pong"))

	targetClient.Close()
	client.Close()
	if err := <-errs; err != nil && err != io.ErrClosedPipe {
		t.Fatalf("HandleSOCKS5() error = %v", err)
	}
}

func TestHandleSOCKS5RejectsUnsupportedCommand(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	errs := make(chan error, 1)
	go func() {
		_, err := readSOCKS5ConnectTarget(server)
		errs <- err
	}()

	writeAll(t, client, []byte{0x05, 0x02, 0x00, 0x01})

	if err := <-errs; err == nil {
		t.Fatal("HandleSOCKS5() error = nil, want unsupported command error")
	}
}

func writeAll(t *testing.T, conn net.Conn, data []byte) {
	t.Helper()
	if _, err := conn.Write(data); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func readExact(t *testing.T, conn net.Conn, want []byte) {
	t.Helper()
	got := make([]byte, len(want))
	if _, err := io.ReadFull(conn, got); err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("read bytes = %v, want %v", got, want)
	}
}
