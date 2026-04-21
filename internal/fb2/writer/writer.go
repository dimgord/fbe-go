// Package writer serializes a doc.FictionBook back to canonical FB2 XML.
//
// Responsibilities:
//   - Emit the correct XML declaration (encoding="utf-8" by default).
//   - Indent to match FBE's historical output style (2-space indent, single blank line between bodies).
//   - Preserve insignificant whitespace where meaningful (verses).
//   - Re-emit xlink:href attributes with the expected prefix.
//
// NOTE: Skeleton only. Replicates the recursive algorithm from FBDoc.cpp::SaveToFile
// (see docs/OPERATIONS.md).
package writer

import (
	"encoding/xml"
	"fmt"
	"io"

	"github.com/dimgord/fbe-go/internal/fb2/doc"
)

// Write serializes the document to w.
func Write(w io.Writer, fb *doc.FictionBook) error {
	if _, err := io.WriteString(w, `<?xml version="1.0" encoding="utf-8"?>`+"\n"); err != nil {
		return err
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(fb); err != nil {
		return fmt.Errorf("fb2 write: %w", err)
	}
	return enc.Flush()
}
