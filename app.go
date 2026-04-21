// Wails bindings — every method on App is automatically exposed to the frontend
// as a TypeScript function under `wailsjs/go/main/App`.
//
// Keep this layer thin: it should translate between the web UI and the
// internal/fb2 packages, not contain business logic.
package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/dimgord/fbe-go/internal/fb2/binary"
	"github.com/dimgord/fbe-go/internal/fb2/doc"
	"github.com/dimgord/fbe-go/internal/fb2/parser"
	"github.com/dimgord/fbe-go/internal/fb2/settings"
	"github.com/dimgord/fbe-go/internal/fb2/thumb"
	"github.com/dimgord/fbe-go/internal/fb2/writer"
	"github.com/dimgord/fbe-go/internal/fb2/xsd"
)

// App holds per-session state.
type App struct {
	ctx     context.Context
	current *doc.FictionBook // currently-open document
	path    string           // current file path, empty if untitled
}

// NewApp constructs the app.
func NewApp() *App { return &App{} }

// OnStartup is called by Wails once the webview is ready.
func (a *App) OnStartup(ctx context.Context) {
	a.ctx = ctx
}

// --- File operations exposed to the frontend ---

// OpenFile reads an FB2 (or FB2.zip) file and returns the parsed document as JSON.
func (a *App) OpenFile(path string) (*doc.FictionBook, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fb, err := parser.Parse(f)
	if err != nil {
		return nil, err
	}
	a.current = fb
	a.path = path
	return fb, nil
}

// SaveFile serializes the current document back to disk.
func (a *App) SaveFile(path string) error {
	if a.current == nil {
		return fmt.Errorf("no document open")
	}
	if path == "" {
		path = a.path
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := writer.Write(f, a.current); err != nil {
		return err
	}
	a.path = path
	return nil
}

// UpdateDocument replaces the current document with a new version from the frontend.
// The ProseMirror editor serializes its state back to doc.FictionBook JSON via a
// TypeScript serializer (frontend/src/editor/serialize.ts).
func (a *App) UpdateDocument(fb *doc.FictionBook) {
	a.current = fb
}

// CurrentDocument returns the in-memory document (useful after Open).
func (a *App) CurrentDocument() *doc.FictionBook { return a.current }

// --- Binary / image helpers ---

// GetBinaryDataURL returns the binary payload as a data: URL for img src.
func (a *App) GetBinaryDataURL(href string) (string, error) {
	if a.current == nil {
		return "", fmt.Errorf("no document open")
	}
	bin, err := binary.FindByHref(a.current, href)
	if err != nil {
		return "", err
	}
	return "data:" + bin.ContentType + ";base64," + bin.Data, nil
}

// AddBinaryFromDisk reads a file and adds it as a <binary> entry.
func (a *App) AddBinaryFromDisk(id, contentType, path string) error {
	if a.current == nil {
		return fmt.Errorf("no document open")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	a.current.Binaries = append(a.current.Binaries, *binary.Encode(id, contentType, data))
	return nil
}

// ExtractThumbnail returns coverpage bytes as a data: URL (for recent-files UI).
func (a *App) ExtractThumbnail(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	fb, err := parser.Parse(f)
	if err != nil {
		return "", err
	}
	data, ct, err := thumb.Extract(fb)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	buf.WriteString("data:" + ct + ";base64,")
	buf.WriteString(base64.StdEncoding.EncodeToString(data))
	return buf.String(), nil
}

// Validate runs XSD validation on a file path and returns per-error messages.
// Requires the app to be built with `-tags xsd`; otherwise returns a single
// error "validator not compiled in".
func (a *App) Validate(path string) ([]xsd.ValidationError, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return xsd.Validate(f)
}

// --- Settings ---

// LoadSettings returns the persisted settings (or defaults).
func (a *App) LoadSettings() (*settings.Settings, error) {
	return settings.Load()
}

// SaveSettings writes settings to disk.
func (a *App) SaveSettings(s *settings.Settings) error {
	return settings.Save(s)
}
