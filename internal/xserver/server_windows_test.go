package xserver

import (
	"bytes"
	"io"
	"net"
	"slices"
	"testing"
)

func TestXServerArgumentsDisableReset(t *testing.T) {
	arguments := xServerArguments(100, `C:\runtime\Xauthority`, `C:\runtime\vcxsrv.log`)
	if !slices.Contains(arguments, "-noreset") {
		t.Fatalf("xServerArguments() = %q, want -noreset", arguments)
	}
}

func TestProbeXServerUsesConfiguredCookie(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	cookie := bytes.Repeat([]byte{0x42}, 16)
	serverError := make(chan error, 1)
	go func() {
		defer server.Close()
		packet := make([]byte, 48)
		if _, err := io.ReadFull(server, packet); err != nil {
			serverError <- err
			return
		}
		if !bytes.Equal(packet[32:48], cookie) {
			serverError <- io.ErrUnexpectedEOF
			return
		}
		_, err := server.Write([]byte{1})
		serverError <- err
	}()

	if err := probeXServer(client, cookie); err != nil {
		t.Fatalf("probeXServer() error = %v", err)
	}
	if err := <-serverError; err != nil {
		t.Fatalf("fake X server error = %v", err)
	}
}
