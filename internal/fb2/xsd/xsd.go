// Package xsd validates an FB2 document against the bundled FictionBook.xsd,
// using the pure-Go github.com/lestrrat-go/helium (no CGo, no build tag).
package xsd

import (
	"embed"
	"io/fs"
)

// ValidationError describes a single schema violation.
type ValidationError struct {
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Element string `json:"element"` // local name, "" when the error names no element
	Message string `json:"message"`
}

// Error implements the error interface for convenience.
func (e ValidationError) Error() string { return e.Message }

// SchemaFiles holds the bundled FictionBook.xsd and its imports.
// The validator resolves the main schema's xs:imports against it.
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
