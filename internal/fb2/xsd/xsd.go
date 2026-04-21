// Package xsd validates an FB2 document against FictionBook.xsd.
//
// Strategy options (pick one before implementation):
//
//  1. CGo → libxml2: most accurate, battle-tested; platform build pain (ship libxml2
//     per OS). Use github.com/lestrrat-go/libxml2 or raw CGo bindings.
//  2. Pure-Go XSD: no mature library exists; would require writing a subset validator.
//     Viable because FictionBook.xsd is relatively simple (no complex type derivation).
//  3. JS-side validation in the webview: acceptable as a stopgap but not usable for the CLI.
//
// Current recommendation: (1) for Phase 2, gated behind a build tag so the pure-Go CLI
// builds without libxml2.
package xsd

import (
	"io"
)

// ValidationError describes a single schema violation.
type ValidationError struct {
	Line    int
	Column  int
	Message string
}

// Validate checks r against the bundled FictionBook.xsd.
// Returns nil (not an empty slice) on success.
func Validate(r io.Reader) ([]ValidationError, error) {
	// TODO: wire up libxml2 (CGo) or pure-Go XSD validator.
	return nil, ErrNotImplemented
}

// ErrNotImplemented is returned until a validator backend is wired up.
var ErrNotImplemented = &NotImplementedError{}

type NotImplementedError struct{}

func (*NotImplementedError) Error() string { return "xsd: validator not implemented yet" }
