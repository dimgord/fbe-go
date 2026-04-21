// Package speller provides spell checking for text nodes in a doc.FictionBook.
//
// Current status:
//   - The built-in noop speller (this file) always reports words as valid.
//     The Wails app relies on the webview's native OS spellchecker
//     (enabled via `spellcheck="true"` on the PM editor), which is sufficient
//     for macOS + Linux deployments.
//   - A full Hunspell CGo backend lives under a `speller_hunspell` build tag
//     (stubbed in speller_hunspell.go). It needs `libhunspell` + dictionaries
//     to compile, which is why it's not the default.
//
// Dictionary file locations (for the future Hunspell backend):
//   macOS (Homebrew): /usr/local/share/hunspell/ or /opt/homebrew/share/hunspell/
//   Linux:            /usr/share/hunspell/
//
// FB2 locales commonly used (from FBE/Speller.h:59–72): en_US, ru_RU, de_DE,
// fr_FR, es_ES, uk_UA, cs_CZ, be_BY, bg_BG, pl_PL, it_IT.
package speller

import (
	"errors"
	"strings"
)

// Speller checks and suggests words in a specified language.
type Speller interface {
	Check(word string) bool
	Suggest(word string) []string
	AddToSession(word string) error
	Close() error
}

// Open initializes a speller for the given language code (e.g., "uk_UA").
// The default backend returns a no-op speller; use `-tags speller_hunspell`
// for the real CGo-hunspell implementation.
func Open(lang, dictsDir string) (Speller, error) {
	return &noop{lang: strings.ToLower(lang)}, nil
}

// noop is used when no backend is compiled in. Every word is considered valid.
type noop struct{ lang string }

func (*noop) Check(string) bool         { return true }
func (*noop) Suggest(string) []string   { return nil }
func (*noop) AddToSession(string) error { return nil }
func (*noop) Close() error              { return nil }

// ErrNoDict is returned when a dictionary file cannot be located.
var ErrNoDict = errors.New("speller: dictionary not found")
