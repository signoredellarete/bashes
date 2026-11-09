package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/signoredellarete/bashes/internal/desktop"
)

type recordedDesktopEvent struct {
	name string
	data any
}

type fakeDesktopRuntime struct {
	directoryPath string
	savePath      string
	messageChoice string
	events        []recordedDesktopEvent
	messages      []desktop.MessageDialogOptions
	urls          []string
}

func (runtime *fakeDesktopRuntime) Emit(_ context.Context, name string, data any) {
	runtime.events = append(runtime.events, recordedDesktopEvent{name: name, data: data})
}

func (runtime *fakeDesktopRuntime) OpenDirectoryDialog(context.Context, desktop.OpenDialogOptions) (string, error) {
	return runtime.directoryPath, nil
}

func (*fakeDesktopRuntime) OpenFileDialog(context.Context, desktop.OpenDialogOptions) (string, error) {
	return "", nil
}

func (runtime *fakeDesktopRuntime) SaveFileDialog(context.Context, desktop.SaveDialogOptions) (string, error) {
	return runtime.savePath, nil
}

func (runtime *fakeDesktopRuntime) MessageDialog(_ context.Context, options desktop.MessageDialogOptions) (string, error) {
	runtime.messages = append(runtime.messages, options)
	return runtime.messageChoice, nil
}

func (runtime *fakeDesktopRuntime) OpenURL(_ context.Context, url string) {
	runtime.urls = append(runtime.urls, url)
}

func TestAppDesktopRuntimeHandlesDialogsEventsAndURLsWithoutWails(t *testing.T) {
	root := t.TempDir()
	exportPath := filepath.Join(root, "export.json")
	runtime := &fakeDesktopRuntime{
		directoryPath: filepath.Join(root, "keys"),
		savePath:      exportPath,
	}
	app := NewApp(filepath.Join(root, "data", "hosts.json"))
	app.desktopRuntime = runtime
	app.startup(context.Background())

	directory, err := app.ChooseSSHKeyDirectory()
	if err != nil {
		t.Fatalf("ChooseSSHKeyDirectory() error = %v", err)
	}
	if directory != runtime.directoryPath {
		t.Fatalf("directory = %q, want %q", directory, runtime.directoryPath)
	}

	app.exportDatabaseFromMenu()
	if _, err := os.Stat(exportPath); err != nil {
		t.Fatalf("exported database missing: %v", err)
	}
	if len(runtime.messages) != 1 || runtime.messages[0].Type != desktop.MessageInfo {
		t.Fatalf("messages = %+v, want one info dialog", runtime.messages)
	}
	if len(runtime.events) != 1 || runtime.events[0].name != desktop.EventDatabaseExported {
		t.Fatalf("events = %+v, want database exported event", runtime.events)
	}

	app.openReadmeFromMenu()
	app.openReleasesFromMenu()
	if len(runtime.urls) != 2 || runtime.urls[0] != readmeURL || runtime.urls[1] != releasesURL {
		t.Fatalf("opened URLs = %v", runtime.urls)
	}
}
