// Package search implements find / find-next / replace over FB2 text content.
//
// The editor surface (ProseMirror) has its own find-in-document plugin; this
// package is for batch / CLI / script-mode operations (e.g., "replace all smart
// quotes across library of .fb2 files").
package search

import "regexp"

// Flags maps 1:1 to FBE's FRF_* constants.
type Flags struct {
	CaseSensitive bool
	WholeWord     bool
	Reverse       bool
	Regex         bool
}

// Compile converts a user pattern + flags into a *regexp.Regexp.
func Compile(pattern string, f Flags) (*regexp.Regexp, error) {
	var p string
	switch {
	case f.Regex:
		p = pattern
	case f.WholeWord:
		p = `\b` + regexp.QuoteMeta(pattern) + `\b`
	default:
		p = regexp.QuoteMeta(pattern)
	}
	if !f.CaseSensitive {
		p = "(?i)" + p
	}
	return regexp.Compile(p)
}
