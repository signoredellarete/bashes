//go:build linux

package main

import "os/exec"

func openFileWithDefaultApplication(filePath string) error {
	return exec.Command("xdg-open", filePath).Run()
}
