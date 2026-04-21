// Wails v2 desktop app entry point.
//
// Build: `wails build`
// Dev:   `wails dev`
//
// The Go side exposes the App struct methods (see app.go) to the frontend via
// Wails's auto-generated TypeScript bindings at frontend/wailsjs/go/main/App.
package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:  "FictionBook Editor",
		Width:  1280,
		Height: 800,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: app.OnStartup,
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		panic(err)
	}
}
