package desktop

import (
	"context"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type MessageType string

const (
	MessageInfo     MessageType = "info"
	MessageError    MessageType = "error"
	MessageQuestion MessageType = "question"
)

type FileFilter struct {
	DisplayName string
	Pattern     string
}

type OpenDialogOptions struct {
	Title   string
	Filters []FileFilter
}

type SaveDialogOptions struct {
	Title           string
	DefaultFilename string
	Filters         []FileFilter
}

type MessageDialogOptions struct {
	Type          MessageType
	Title         string
	Message       string
	Buttons       []string
	DefaultButton string
	CancelButton  string
}

type Runtime interface {
	Emit(context.Context, string, any)
	OpenDirectoryDialog(context.Context, OpenDialogOptions) (string, error)
	OpenFileDialog(context.Context, OpenDialogOptions) (string, error)
	SaveFileDialog(context.Context, SaveDialogOptions) (string, error)
	MessageDialog(context.Context, MessageDialogOptions) (string, error)
	OpenURL(context.Context, string)
}

type WailsRuntime struct{}

func NewWailsRuntime() WailsRuntime {
	return WailsRuntime{}
}

func (WailsRuntime) Emit(ctx context.Context, name string, data any) {
	wailsruntime.EventsEmit(ctx, name, data)
}

func (WailsRuntime) OpenDirectoryDialog(ctx context.Context, options OpenDialogOptions) (string, error) {
	return wailsruntime.OpenDirectoryDialog(ctx, wailsruntime.OpenDialogOptions{
		Title:   options.Title,
		Filters: wailsFilters(options.Filters),
	})
}

func (WailsRuntime) OpenFileDialog(ctx context.Context, options OpenDialogOptions) (string, error) {
	return wailsruntime.OpenFileDialog(ctx, wailsruntime.OpenDialogOptions{
		Title:   options.Title,
		Filters: wailsFilters(options.Filters),
	})
}

func (WailsRuntime) SaveFileDialog(ctx context.Context, options SaveDialogOptions) (string, error) {
	return wailsruntime.SaveFileDialog(ctx, wailsruntime.SaveDialogOptions{
		Title:           options.Title,
		DefaultFilename: options.DefaultFilename,
		Filters:         wailsFilters(options.Filters),
	})
}

func (WailsRuntime) MessageDialog(ctx context.Context, options MessageDialogOptions) (string, error) {
	return wailsruntime.MessageDialog(ctx, wailsruntime.MessageDialogOptions{
		Type:          wailsMessageType(options.Type),
		Title:         options.Title,
		Message:       options.Message,
		Buttons:       options.Buttons,
		DefaultButton: options.DefaultButton,
		CancelButton:  options.CancelButton,
	})
}

func (WailsRuntime) OpenURL(ctx context.Context, url string) {
	wailsruntime.BrowserOpenURL(ctx, url)
}

func wailsFilters(filters []FileFilter) []wailsruntime.FileFilter {
	result := make([]wailsruntime.FileFilter, 0, len(filters))
	for _, filter := range filters {
		result = append(result, wailsruntime.FileFilter{
			DisplayName: filter.DisplayName,
			Pattern:     filter.Pattern,
		})
	}
	return result
}

func wailsMessageType(messageType MessageType) wailsruntime.DialogType {
	switch messageType {
	case MessageError:
		return wailsruntime.ErrorDialog
	case MessageQuestion:
		return wailsruntime.QuestionDialog
	default:
		return wailsruntime.InfoDialog
	}
}
