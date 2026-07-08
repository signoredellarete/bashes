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
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp("")

	err := wails.Run(&options.App{
		Title:     "Bashes",
		Width:     1180,
		Height:    760,
		Menu:      applicationMenu(app),
		OnStartup: app.startup,
		AssetServer: &assetserver.Options{
			Assets: assets,
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
