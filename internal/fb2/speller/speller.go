// Package speller provides spell checking for text nodes in a doc.FictionBook.
//
// Backend options:
//   - CGo → hunspell: most compatible with original FBE dictionaries (uk_UA, ru_RU, en_US, ...).
//   - Pure-Go (e.g., github.com/client9/misspell) — limited, English-focused.
//   - External process: exec hunspell; portable but slow.
//
// Recommendation: CGo hunspell with a build tag; fall back to no-op on builds without it.
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

// Open initializes a speller for the given language code ("uk_UA", "ru_RU", "en_US", ...).
func Open(lang, dictsDir string) (Speller, error) {
	// TODO: wire up hunspell via CGo; see FBE/Speller.cpp for reference.
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
