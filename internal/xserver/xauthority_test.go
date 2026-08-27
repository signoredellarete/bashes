package xserver

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestWriteXAuthorityCreatesMITCookieEntry(t *testing.T) {
	cookie := bytes.Repeat([]byte{0x7a}, 16)
	var output bytes.Buffer
	if err := writeXAuthority(&output, "101", cookie); err != nil {
		t.Fatalf("writeXAuthority() error = %v", err)
	}

	reader := bytes.NewReader(output.Bytes())
	var family uint16
	if err := binary.Read(reader, binary.BigEndian, &family); err != nil {
		t.Fatalf("read family: %v", err)
	}
	if family != xAuthorityFamilyWild {
		t.Fatalf("family = %d, want %d", family, xAuthorityFamilyWild)
	}
	for index, want := range [][]byte{nil, []byte("101"), []byte(xAuthorityProtocol), cookie} {
		var length uint16
		if err := binary.Read(reader, binary.BigEndian, &length); err != nil {
			t.Fatalf("read field %d length: %v", index, err)
		}
		got := make([]byte, length)
		if _, err := reader.Read(got); err != nil {
			t.Fatalf("read field %d: %v", index, err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("field %d = %q, want %q", index, got, want)
		}
	}
}
