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
	"github.com/dimgord/fbe-go/internal/fb2/export/html"
	"github.com/dimgord/fbe-go/internal/fb2/parser"
	"github.com/dimgord/fbe-go/internal/fb2/settings"
	"github.com/dimgord/fbe-go/internal/fb2/thumb"
	"github.com/dimgord/fbe-go/internal/fb2/writer"
	"github.com/dimgord/fbe-go/internal/fb2/xsd"
	wailsrt "github.com/wailsapp/wails/v2/pkg/runtime"
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

// --- Native dialogs (Wails' runtime.OpenFileDialog / SaveFileDialog are Go-only;
// expose wrappers so the frontend can open dialogs without a direct window.runtime
// dependency, which in Wails v2 doesn't ship dialog helpers to JS). ---

// PickFB2ToOpen shows a native "open file" dialog filtered for .fb2 files.
// Returns an empty string if the user cancels.
//
// Note: Wails v2.9.2 on macOS crashes with `NSInvalidArgumentException` when a
// filter pattern contains multi-dot extensions like `*.fb2.zip`, because its
// native code feeds each split token to `[UTType typeWithFilenameExtension:]`
// without a nil check — multi-dot extensions return nil and then
// `[NSArray addObject:nil]` throws. We stick to the single `*.fb2` extension
// and let users pick `.fb2.zip` archives via "All files" instead.
func (a *App) PickFB2ToOpen() (string, error) {
	return wailsrt.OpenFileDialog(a.ctx, wailsrt.OpenDialogOptions{
		Title: "Open FB2 file",
		Filters: []wailsrt.FileFilter{
			{DisplayName: "FictionBook (*.fb2)", Pattern: "*.fb2"},
		},
	})
}

// PickFB2ToSave shows a native "save file" dialog defaulted to a .fb2 extension.
func (a *App) PickFB2ToSave(suggested string) (string, error) {
	if suggested == "" {
		suggested = "untitled.fb2"
	}
	return wailsrt.SaveFileDialog(a.ctx, wailsrt.SaveDialogOptions{
		Title:           "Save FB2",
		DefaultFilename: suggested,
		Filters: []wailsrt.FileFilter{
			{DisplayName: "FictionBook (*.fb2)", Pattern: "*.fb2"},
		},
	})
}

// PickHTMLToSave shows a native "save file" dialog for the HTML exporter.
func (a *App) PickHTMLToSave(suggested string) (string, error) {
	if suggested == "" {
		suggested = "untitled.html"
	}
	return wailsrt.SaveFileDialog(a.ctx, wailsrt.SaveDialogOptions{
		Title:           "Export HTML",
		DefaultFilename: suggested,
		Filters: []wailsrt.FileFilter{
			{DisplayName: "HTML (*.html)", Pattern: "*.html"},
		},
	})
}

// --- File operations exposed to the frontend ---

// OpenFile reads an FB2 (or FB2.zip) file and returns the parsed document as JSON.
// Panics are recovered so a bad document surfaces as a normal JS-side error
// instead of killing the webview.
func (a *App) OpenFile(path string) (fb *doc.FictionBook, err error) {
	defer func() {
		if r := recover(); r != nil {
			fb = nil
			err = fmt.Errorf("OpenFile panic: %v", r)
		}
	}()
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fb, err = parser.Parse(f)
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

// ExportHTML writes the currently-open document as a self-contained HTML file.
func (a *App) ExportHTML(path string) error {
	if a.current == nil {
		return fmt.Errorf("no document open")
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return html.Export(f, a.current)
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

// SerializeCurrent returns the canonical FB2 XML of the in-memory document as
// a string. Frontend uses this for the read-only XML source panel — the
// output reflects any unsaved edits that were pushed via UpdateDocument.
func (a *App) SerializeCurrent() (string, error) {
	if a.current == nil {
		return "", fmt.Errorf("no document open")
	}
	var buf bytes.Buffer
	if err := writer.Write(&buf, a.current); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// ValidateCurrent validates the in-memory document against the bundled XSD.
// Unlike Validate(path), this reflects unsaved edits — it serializes first
// and validates the result, so line numbers in the returned errors map
// directly to the output of SerializeCurrent.
func (a *App) ValidateCurrent() ([]xsd.ValidationError, error) {
	if a.current == nil {
		return nil, fmt.Errorf("no document open")
	}
	var buf bytes.Buffer
	if err := writer.Write(&buf, a.current); err != nil {
		return nil, err
	}
	return xsd.Validate(&buf)
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
