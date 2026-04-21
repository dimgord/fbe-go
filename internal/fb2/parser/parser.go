// Package parser reads FB2 XML and produces a doc.FictionBook.
//
// Responsibilities:
//   - Handle BOM + XML encoding declaration (FB2 files may be win-1251, koi8-r, utf-8, ...).
//   - Preserve whitespace inside paragraphs (significant in poetry).
//   - Accept malformed-but-recoverable docs (legacy FBE produced non-canonical output).
//
// NOTE: Skeleton only. The .xsd-faithful Decode in Parse() is a starting point;
// production-grade parsing needs encoding autodetection and whitespace rules.
package parser

import (
	"encoding/xml"
	"fmt"
	"io"

	"github.com/dimgord/fbe-go/internal/fb2/doc"
)

// Parse reads an FB2 XML stream into a doc.FictionBook.
func Parse(r io.Reader) (*doc.FictionBook, error) {
	dec := xml.NewDecoder(r)
	dec.DefaultSpace = doc.NSFictionBook
	// TODO: set CharsetReader to decode win-1251/koi8-r/etc. via golang.org/x/text/encoding.

	var fb doc.FictionBook
	if err := dec.Decode(&fb); err != nil {
		return nil, fmt.Errorf("fb2 parse: %w", err)
	}
	return &fb, nil
}
