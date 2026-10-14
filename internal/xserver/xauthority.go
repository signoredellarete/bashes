package xserver

import (
	"encoding/binary"
	"fmt"
	"io"
)

const (
	xAuthorityFamilyWild = 65535
	xAuthorityProtocol   = "MIT-MAGIC-COOKIE-1"
)

func writeXAuthority(writer io.Writer, display string, cookie []byte) error {
	fields := [][]byte{
		nil,
		[]byte(display),
		[]byte(xAuthorityProtocol),
		cookie,
	}
	if err := binary.Write(writer, binary.BigEndian, uint16(xAuthorityFamilyWild)); err != nil {
		return err
	}
	for _, field := range fields {
		if len(field) > int(^uint16(0)) {
			return fmt.Errorf("Xauthority field is too large")
		}
		if err := binary.Write(writer, binary.BigEndian, uint16(len(field))); err != nil {
			return err
		}
		if _, err := writer.Write(field); err != nil {
			return err
		}
	}
	return nil
}
