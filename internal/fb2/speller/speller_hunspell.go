//go:build speller_hunspell

// Hunspell-backed speller. Compile with `-tags speller_hunspell` after
// installing libhunspell + dictionaries:
//
//   macOS: brew install hunspell
//   Linux: apt install hunspell hunspell-en-us hunspell-ru hunspell-uk-ua
//
// TODO(phase-4): wire up a Go CGo binding (one of):
//   - github.com/akhenakh/hunspellgo
//   - github.com/trustmaster/go-hunspell
//   - a small hand-written cgo wrapper around Hunspell_create / Hunspell_spell /
//     Hunspell_suggest / Hunspell_add / Hunspell_destroy.
//
// For now this is a placeholder so the build tag compiles — it behaves the
// same as the default (noop) backend.
package speller
