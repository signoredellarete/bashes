//go:build windows

package main

import (
	"context"
	"encoding/binary"
	"os/exec"
	"strings"
	"syscall"
	"time"
	"unicode/utf16"
)

const wslCommandTimeout = 5 * time.Second

func listWSLSSHKeys() ([]SSHKeyInfo, error) {
	if _, err := exec.LookPath("wsl.exe"); err != nil {
		return []SSHKeyInfo{}, nil
	}

	distributions, err := runWSLCommand("--list", "--quiet")
	if err != nil {
		return []SSHKeyInfo{}, nil
	}

	keys := []SSHKeyInfo{}
	for _, distribution := range nonEmptyLines(distributions) {
		if strings.ContainsAny(distribution, "\\/") {
			continue
		}
		sshDirectory, err := runWSLCommand(
			"--distribution", distribution,
			"--exec", "sh", "-lc", `wslpath -w "$HOME/.ssh"`,
		)
		if err != nil {
			continue
		}
		sshDirectory = strings.TrimSpace(sshDirectory)
		if sshDirectory == "" {
			continue
		}
		distributionKeys, err := listSSHKeysInDirectory(sshDirectory, "wsl", true)
		if err != nil {
			continue
		}
		for index := range distributionKeys {
			distributionKeys[index].Distribution = distribution
		}
		keys = append(keys, distributionKeys...)
	}
	return keys, nil
}

func runWSLCommand(arguments ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), wslCommandTimeout)
	defer cancel()
	command := exec.CommandContext(ctx, "wsl.exe", arguments...)
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	output, err := command.Output()
	return decodeWSLOutput(output), err
}

func nonEmptyLines(value string) []string {
	lines := []string{}
	for _, line := range strings.Split(value, "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func decodeWSLOutput(data []byte) string {
	if len(data) < 2 || (!(data[0] == 0xff && data[1] == 0xfe) && !hasNullByte(data)) {
		return string(data)
	}
	if len(data)%2 != 0 {
		data = data[:len(data)-1]
	}
	units := make([]uint16, 0, len(data)/2)
	for index := 0; index+1 < len(data); index += 2 {
		value := binary.LittleEndian.Uint16(data[index : index+2])
		if value == 0xfeff {
			continue
		}
		units = append(units, value)
	}
	return string(utf16.Decode(units))
}

func hasNullByte(data []byte) bool {
	for _, value := range data {
		if value == 0 {
			return true
		}
	}
	return false
}
