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

	"github.com/dimgord/fbe-go/internal/fb2/settings"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

// Startup defaults when the user hasn't sized the window yet (first run or
// settings.json absent). Chosen to match the pre-persistence default.
const (
	defaultWidth  = 1280
	defaultHeight = 800
)

func main() {
	app := NewApp()

	// Read persisted window geometry. Ignored on any read error — the app
	// still launches at the default size. Initial X/Y can't be set via
	// options.App in Wails v2, so positioning is restored in app.OnStartup
	// via runtime.WindowSetPosition.
	w, h := defaultWidth, defaultHeight
	if s, err := settings.Load(); err == nil && s != nil && s.Window.W > 0 && s.Window.H > 0 {
		w, h = s.Window.W, s.Window.H
	}

	err := wails.Run(&options.App{
		Title:  "FictionBook Editor",
		Width:  w,
		Height: h,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:     app.OnStartup,
		OnShutdown:    app.OnShutdown,
		OnBeforeClose: app.OnBeforeClose,
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		panic(err)
	}
}
