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
	keys, _ := listWSLSSHKeyInventory()
	return keys, nil
}

func listWSLSSHKeyInventory() ([]SSHKeyInfo, []SSHKeySourceInfo) {
	if _, err := exec.LookPath("wsl.exe"); err != nil {
		return []SSHKeyInfo{}, []SSHKeySourceInfo{}
	}

	distributions, err := runWSLCommand("--list", "--quiet")
	if err != nil {
		return []SSHKeyInfo{}, []SSHKeySourceInfo{{
			Source: "wsl",
			Label:  "WSL distributions",
			Error:  err.Error(),
		}}
	}

	keys := []SSHKeyInfo{}
	sources := []SSHKeySourceInfo{}
	for _, distribution := range nonEmptyLines(distributions) {
		if strings.ContainsAny(distribution, "\\/") {
			continue
		}
		source := SSHKeySourceInfo{
			Source:       "wsl",
			Label:        "WSL: " + distribution,
			Distribution: distribution,
		}
		sshDirectory, err := runWSLCommand(
			"--distribution", distribution,
			"--exec", "sh", "-lc", `wslpath -w "$HOME/.ssh"`,
		)
		if err != nil {
			source.Error = err.Error()
			sources = append(sources, source)
			continue
		}
		sshDirectory = strings.TrimSpace(sshDirectory)
		source.Path = sshDirectory
		if sshDirectory == "" {
			source.Error = "could not resolve the SSH directory"
			sources = append(sources, source)
			continue
		}
		distributionKeys, err := listSSHKeysInDirectory(sshDirectory, "wsl", true)
		if err != nil {
			source.Error = err.Error()
			sources = append(sources, source)
			continue
		}
		for index := range distributionKeys {
			distributionKeys[index].Distribution = distribution
		}
		source.Count = len(distributionKeys)
		sources = append(sources, source)
		keys = append(keys, distributionKeys...)
	}
	return keys, sources
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
