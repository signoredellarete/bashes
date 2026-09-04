//go:build darwin

package main

import "os/exec"

func openFileWithDefaultApplication(filePath string) error {
	return exec.Command("open", filePath).Run()
}
