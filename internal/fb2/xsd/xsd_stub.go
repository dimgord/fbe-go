//go:build !xsd

package xsd

import "io"

// Validate is a no-op in the default build. Use `-tags xsd` to compile in the
// libxml2-backed validator.
func Validate(r io.Reader) ([]ValidationError, error) {
	_ = r
	return nil, ErrNotImplemented
}
