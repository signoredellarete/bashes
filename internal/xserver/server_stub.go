//go:build !windows

package xserver

func platformXServerSupported() bool {
	return false
}

func startPlatformXServer(string) (*serverProcess, error) {
	return nil, ErrUnsupported
}
