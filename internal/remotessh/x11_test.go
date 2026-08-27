package remotessh

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
)

func TestRewriteX11SetupReplacesForwardingCookie(t *testing.T) {
	for _, test := range []struct {
		name  string
		order binary.ByteOrder
		mark  byte
	}{
		{name: "little endian", order: binary.LittleEndian, mark: 'l'},
		{name: "big endian", order: binary.BigEndian, mark: 'B'},
	} {
		t.Run(test.name, func(t *testing.T) {
			fakeCookie := bytes.Repeat([]byte{0x11}, x11CookieSize)
			localCookie := bytes.Repeat([]byte{0x22}, x11CookieSize)
			packet := x11SetupPacket(test.order, test.mark, fakeCookie)
			var output bytes.Buffer

			if err := rewriteX11Setup(bytes.NewReader(packet), &output, fakeCookie, localCookie); err != nil {
				t.Fatalf("rewriteX11Setup() error = %v", err)
			}

			got := output.Bytes()
			cookieOffset := 12 + paddedX11Length(len(x11AuthProtocol))
			if !bytes.Equal(got[cookieOffset:cookieOffset+x11CookieSize], localCookie) {
				t.Fatalf("rewritten cookie = %x, want %x", got[cookieOffset:cookieOffset+x11CookieSize], localCookie)
			}
			if !bytes.Equal(got[:cookieOffset], packet[:cookieOffset]) {
				t.Fatal("rewriteX11Setup() changed the setup packet before the cookie")
			}
		})
	}
}

func TestRewriteX11SetupRejectsUnexpectedCookie(t *testing.T) {
	fakeCookie := bytes.Repeat([]byte{0x11}, x11CookieSize)
	packet := x11SetupPacket(binary.LittleEndian, 'l', bytes.Repeat([]byte{0x33}, x11CookieSize))

	err := rewriteX11Setup(bytes.NewReader(packet), &bytes.Buffer{}, fakeCookie, bytes.Repeat([]byte{0x22}, x11CookieSize))
	if err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("rewriteX11Setup() error = %v, want cookie mismatch", err)
	}
}

func TestRewriteX11SetupRejectsUnsupportedAuthentication(t *testing.T) {
	packet := x11SetupPacketWithAuth(binary.BigEndian, 'B', "OTHER-AUTH", bytes.Repeat([]byte{0x11}, x11CookieSize))
	err := rewriteX11Setup(bytes.NewReader(packet), &bytes.Buffer{}, bytes.Repeat([]byte{0x11}, x11CookieSize), bytes.Repeat([]byte{0x22}, x11CookieSize))
	if err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("rewriteX11Setup() error = %v, want unsupported authentication", err)
	}
}

func x11SetupPacket(order binary.ByteOrder, mark byte, cookie []byte) []byte {
	return x11SetupPacketWithAuth(order, mark, x11AuthProtocol, cookie)
}

func x11SetupPacketWithAuth(order binary.ByteOrder, mark byte, protocol string, cookie []byte) []byte {
	header := make([]byte, 12)
	header[0] = mark
	order.PutUint16(header[2:4], 11)
	order.PutUint16(header[6:8], uint16(len(protocol)))
	order.PutUint16(header[8:10], uint16(len(cookie)))

	packet := append([]byte{}, header...)
	packet = append(packet, protocol...)
	packet = append(packet, make([]byte, paddedX11Length(len(protocol))-len(protocol))...)
	packet = append(packet, cookie...)
	packet = append(packet, make([]byte, paddedX11Length(len(cookie))-len(cookie))...)
	return packet
}
