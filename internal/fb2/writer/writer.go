// Package writer serializes a doc.FictionBook back to canonical FB2 XML.
//
// Output shape:
//   - XML declaration at top (`<?xml version="1.0" encoding="utf-8"?>`).
//   - Root `<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">`
//     with the FB2 namespace declared once.
//   - 2-space indentation.
//   - `<binary>` entries are re-emitted as base64 with their id/content-type.
//
// Element-name dispatch for polymorphic containers (Block, Inline) is handled
// by MarshalXML methods on those types in the doc package.
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
	if err := enc.Flush(); err != nil {
		return err
	}
	_, err := io.WriteString(w, "\n")
	return err
}
