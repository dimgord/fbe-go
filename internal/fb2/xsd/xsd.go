// Package xsd validates an FB2 document against the bundled FictionBook.xsd.
//
// Two backends, selected via build tag:
//
//   default:      no-op validator that returns ErrNotImplemented
//   -tags xsd:    CGo-based libxml2 validator (requires libxml-2.0 on the build host)
//
// Build with `go build -tags xsd ./...` to get real validation. The CLI and
// Wails app can both be built either way.
package xsd

import (
	"embed"
	"errors"
	"io/fs"
)

// ValidationError describes a single schema violation.
type ValidationError struct {
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Message string `json:"message"`
}

// Error implements the error interface for convenience.
func (e ValidationError) Error() string { return e.Message }

// ErrNotImplemented is returned by the default (no-op) backend.
var ErrNotImplemented = errors.New("xsd: validator not compiled in — rebuild with -tags xsd")

// SchemaFiles holds the bundled FictionBook.xsd and its imports.
// Exposed so alternate backends (pure-Go, future) can access the schema payload.
//
//go:embed FictionBook.xsd FictionBookGenres.xsd FictionBookLang.xsd FictionBookLinks.xsd
var SchemaFiles embed.FS

// SchemaFileNames returns the list of bundled XSD filenames.
func SchemaFileNames() []string {
	entries, err := fs.ReadDir(SchemaFiles, ".")
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names
}
