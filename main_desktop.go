//go:build desktop

package main

import (
	"embed"
	"log"
	goruntime "runtime"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	linuxopts "github.com/wailsapp/wails/v2/pkg/options/linux"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed icons/bashes.png
var appIcon []byte

func main() {
	app := NewApp("")

	err := wails.Run(&options.App{
		Title:         "Bashes",
		Width:         1180,
		Height:        760,
		Menu:          applicationMenu(app),
		OnStartup:     app.startup,
		OnBeforeClose: app.beforeClose,
		OnShutdown:    app.shutdown,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		Linux: &linuxopts.Options{
			Icon:        appIcon,
			ProgramName: "bashes",
		},
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}

func applicationMenu(app *App) *menu.Menu {
	appMenu := menu.NewMenu()
	if goruntime.GOOS == "darwin" {
		appMenu.Append(menu.AppMenu())
		appMenu.Append(menu.EditMenu())
	}

	tools := appMenu.AddSubmenu("Tools")
	tools.AddText("Export Database...", nil, func(_ *menu.CallbackData) {
		app.exportDatabaseFromMenu()
	})
	tools.AddText("Import Database...", nil, func(_ *menu.CallbackData) {
		app.importDatabaseFromMenu()
	})
	tools.AddText("Import from hosts file", nil, func(_ *menu.CallbackData) {
		app.importHostsFileFromMenu()
	})
	tools.AddSeparator()
	tools.AddText("Check for Updates", nil, func(_ *menu.CallbackData) {
		app.checkForUpdatesFromMenu()
	})

	settings := appMenu.AddSubmenu("Settings")
	settings.AddText("Settings...", nil, func(_ *menu.CallbackData) {
		app.openSettingsFromMenu()
	})

	if goruntime.GOOS == "darwin" {
		appMenu.Append(menu.WindowMenu())
	}

	help := appMenu.AddSubmenu("Help")
	help.AddText("About Bashes", nil, func(_ *menu.CallbackData) {
		app.showAboutFromMenu()
	})
	help.AddSeparator()
	help.AddText("README", nil, func(_ *menu.CallbackData) {
		app.openReadmeFromMenu()
	})
	help.AddText("GitHub Releases", nil, func(_ *menu.CallbackData) {
		app.openReleasesFromMenu()
	})

	return appMenu
}
