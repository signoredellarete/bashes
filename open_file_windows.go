//go:build windows

package main

import "golang.org/x/sys/windows"

func openFileWithDefaultApplication(filePath string) error {
	return windows.ShellExecute(0, nil, windows.StringToUTF16Ptr(filePath), nil, nil, windows.SW_SHOWNORMAL)
}
