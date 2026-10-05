//go:build !windows

package main

func listWSLSSHKeys() ([]SSHKeyInfo, error) {
	return []SSHKeyInfo{}, nil
}
