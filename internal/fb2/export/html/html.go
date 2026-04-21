// Package html exports an FB2 document as a standalone HTML file.
//
// Original FBE used an XSLT stylesheet (FBE/ExportHTML/html.xsl, 493 lines).
// Options for the Go port:
//
//  1. Keep html.xsl, run via libxslt (CGo) — minimal rewrite, largest dep.
//  2. Rewrite as Go text/html templates — pure Go, more code.
//  3. Serialize ProseMirror doc directly from the frontend — possible but ties
//     export to the editor being loaded.
//
// Recommendation for Phase 4: option (2) — transparent, pure-Go, and serves as a
// reference implementation for other export formats (EPUB, Markdown, etc.).
package html

import (
	"io"

	"github.com/dimgord/fbe-go/internal/fb2/doc"
)

// Export writes fb as HTML to w.
func Export(w io.Writer, fb *doc.FictionBook) error {
	// TODO: implement walker + html/template.
	_ = fb
	_, err := io.WriteString(w, "<!DOCTYPE html><html><body><p>TODO</p></body></html>\n")
	return err
}
