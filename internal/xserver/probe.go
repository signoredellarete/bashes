package xserver

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"time"
)

func xServerArguments(display int, authorityPath, logPath string) []string {
	return []string{
		fmt.Sprintf(":%d", display),
		"-noreset",
		"-multiwindow",
		"-clipboard",
		"-wgl",
		"-silent-dup-error",
		"-notrayicon",
		"-nohostintitle",
		"-listen", "tcp",
		"-auth", authorityPath,
		"-logfile", logPath,
		"-dpi", "auto",
	}
}

func probeXServer(connection net.Conn, cookie []byte) error {
	if len(cookie) != 16 {
		return errors.New("X11 cookie must be 16 bytes")
	}
	if err := connection.SetDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
		return err
	}

	header := make([]byte, 12)
	header[0] = 'l'
	binary.LittleEndian.PutUint16(header[2:4], 11)
	binary.LittleEndian.PutUint16(header[6:8], uint16(len(xAuthorityProtocol)))
	binary.LittleEndian.PutUint16(header[8:10], uint16(len(cookie)))
	packet := append(header, []byte(xAuthorityProtocol)...)
	packet = append(packet, make([]byte, (4-len(packet)%4)%4)...)
	packet = append(packet, cookie...)
	packet = append(packet, make([]byte, (4-len(packet)%4)%4)...)

	if _, err := io.Copy(connection, bytes.NewReader(packet)); err != nil {
		return fmt.Errorf("write X11 readiness request: %w", err)
	}
	var status [1]byte
	if _, err := io.ReadFull(connection, status[:]); err != nil {
		return fmt.Errorf("read X11 readiness response: %w", err)
	}
	if status[0] != 1 {
		return fmt.Errorf("X server rejected the readiness request with status %d", status[0])
	}
	return nil
}
