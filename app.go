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
	recordRecentFile(path)
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
	recordRecentFile(path)
	return nil
}

// recentFilesCap is the maximum length of the Most-Recently-Used list.
// Chosen to match what the original FBE used (see Settings.h::recentFiles).
const recentFilesCap = 10

// recordRecentFile prepends `path` to settings.RecentFiles, dedupes earlier
// occurrences, caps the list at recentFilesCap, and persists. Silent on
// error — recent-files tracking is a convenience, not a correctness path;
// we'd rather continue the primary flow than fail Open/Save because
// settings.json couldn't be written.
func recordRecentFile(path string) {
	if path == "" {
		return
	}
	s, err := settings.Load()
	if err != nil || s == nil {
		return
	}
	out := make([]string, 0, len(s.RecentFiles)+1)
	out = append(out, path)
	for _, p := range s.RecentFiles {
		if p == path {
			continue
		}
		out = append(out, p)
		if len(out) == recentFilesCap {
			break
		}
	}
	s.RecentFiles = out
	_ = settings.Save(s)
}

// RecentFiles returns the persisted most-recently-used file paths
// (most-recent first). On read error returns an empty list rather than
// propagating — the frontend just shows no history in that case.
func (a *App) RecentFiles() []string {
	s, err := settings.Load()
	if err != nil || s == nil {
		return []string{}
	}
	if s.RecentFiles == nil {
		return []string{}
	}
	return s.RecentFiles
}

// RemoveFromRecent drops a path from the MRU list (e.g. when it no longer
// exists on disk). No-op if absent.
func (a *App) RemoveFromRecent(path string) error {
	s, err := settings.Load()
	if err != nil || s == nil {
		return err
	}
	out := s.RecentFiles[:0]
	for _, p := range s.RecentFiles {
		if p != path {
			out = append(out, p)
		}
	}
	s.RecentFiles = out
	return settings.Save(s)
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

// ValidateCurrent validates the in-memory document against the bundled XSD
// and supplements libxml2's findings with a structure-agnostic unknown-
// element scan. Unlike Validate(path), this reflects unsaved edits.
//
// Two error sources are merged:
//
//   - libxml2 schema validation (xsd.Validate) — reports XSD content-model
//     violations with rich "Expected is one of (...)" messages. Subject to
//     libxml2's recovery quirks, which can drop later unknown-element
//     errors after the first in a content group.
//   - Our own unknown-element scanner (xsd.FindUnknownElements) — regexes
//     through the serialized source and flags any tag outside the
//     FictionBook 2.0 vocabulary. Structure-agnostic, so every occurrence
//     shows up regardless of libxml2's DFA recovery.
//
// Dedup is handled in xsd.MergeXSDAndUnknown: if both sources cover the same
// element at the same line, only libxml2's richer entry is kept.
func (a *App) ValidateCurrent() ([]xsd.ValidationError, error) {
	if a.current == nil {
		return nil, fmt.Errorf("no document open")
	}
	var buf bytes.Buffer
	if err := writer.Write(&buf, a.current); err != nil {
		return nil, err
	}
	src := buf.Bytes()
	xsdErrs, err := xsd.Validate(bytes.NewReader(src))
	if err != nil {
		return nil, err
	}
	return xsd.MergeXSDAndUnknown(xsdErrs, xsd.FindUnknownElements(src)), nil
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
