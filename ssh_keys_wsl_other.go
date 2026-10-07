//go:build !windows

package main

func listWSLSSHKeys() ([]SSHKeyInfo, error) {
	return []SSHKeyInfo{}, nil
}

func listWSLSSHKeyInventory() ([]SSHKeyInfo, []SSHKeySourceInfo) {
	return []SSHKeyInfo{}, []SSHKeySourceInfo{}
}
