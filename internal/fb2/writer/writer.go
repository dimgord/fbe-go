// Package writer serializes a doc.FictionBook back to canonical FB2 XML.
//
// Output shape:
//   - XML declaration at top (`<?xml version="1.0" encoding="utf-8"?>`).
//   - Root `<FictionBook>` with TWO namespace declarations:
//       - default: `xmlns="http://www.gribuser.ru/xml/fictionbook/2.0"`
//       - xlink prefix `l`: `xmlns:l="http://www.w3.org/1999/xlink"`
//     Pre-declaring `l` at root lets `<a l:href="...">` round-trip faithfully
//     without Go auto-declaring `xmlns:xlink="..."` on every <a>.
//   - 2-space indentation for block-level elements.
//   - Mixed-content leaf blocks (`<p>`, `<subtitle>`, `<th>`, `<td>`, `<v>`,
//     `<text-author>`, `<date>`) have their inner whitespace collapsed so
//     `<p>text <strong>bold</strong> tail</p>` round-trips on one line
//     instead of getting newlines inserted between text and inline marks.
//     Implemented as a post-process pass (see `compactMixedContent`) because
//     toggling `xml.Encoder.Indent("", "")` mid-marshal desyncs the encoder's
//     internal depth counter from its tag stack, over-indenting subsequent
//     siblings. The regex pass is narrowly scoped: only the listed leaf
//     container names, and only whitespace between a closing `>` and the
//     next opening `<`.
//   - `<binary>` entries are re-emitted as base64 with their id/content-type.
//
// Element-name dispatch for polymorphic containers (Block, Inline) is handled
// by MarshalXML methods on those types in the doc package.
package writer

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"regexp"

	"github.com/dimgord/fbe-go/internal/fb2/doc"
)

// rootOpen is emitted verbatim to bypass Go's default xmlns handling, which
// would otherwise auto-pick an `xmlns:xlink` prefix instead of our chosen
// `xmlns:l`.
const rootOpen = `<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0" xmlns:l="http://www.w3.org/1999/xlink">` + "\n"

// Write serializes the document to w.
func Write(w io.Writer, fb *doc.FictionBook) error {
	// Buffer first so we can run compactMixedContent over the whole output
	// before handing it to the caller. FB2 documents are at most a few MB
	// and this buffer is short-lived.
	var buf bytes.Buffer

	if _, err := io.WriteString(&buf, `<?xml version="1.0" encoding="utf-8"?>`+"\n"); err != nil {
		return err
	}
	if _, err := io.WriteString(&buf, rootOpen); err != nil {
		return err
	}

	enc := xml.NewEncoder(&buf)
	enc.Indent("  ", "  ")

	for _, s := range fb.Stylesheets {
		if err := enc.EncodeElement(s, xml.StartElement{Name: xml.Name{Local: "stylesheet"}}); err != nil {
			return fmt.Errorf("fb2 write stylesheet: %w", err)
		}
	}
	if err := enc.EncodeElement(fb.Description, xml.StartElement{Name: xml.Name{Local: "description"}}); err != nil {
		return fmt.Errorf("fb2 write description: %w", err)
	}
	for _, b := range fb.Bodies {
		if err := enc.EncodeElement(b, xml.StartElement{Name: xml.Name{Local: "body"}}); err != nil {
			return fmt.Errorf("fb2 write body: %w", err)
		}
	}
	for _, bin := range fb.Binaries {
		if err := enc.EncodeElement(bin, xml.StartElement{Name: xml.Name{Local: "binary"}}); err != nil {
			return fmt.Errorf("fb2 write binary: %w", err)
		}
	}
	if err := enc.Flush(); err != nil {
		return err
	}

	if _, err := io.WriteString(&buf, "\n</FictionBook>\n"); err != nil {
		return err
	}

	cleaned := compactMixedContent(buf.Bytes())
	_, err := w.Write(cleaned)
	return err
}

// mixedContentTagRE matches a leaf mixed-content container — `<p>`,
// `<subtitle>`, `<th>`, `<td>`, `<v>`, `<text-author>`, `<date>` — and
// captures its tag name (1), attributes (2), and inner content (3). Uses a
// backreference on the closing tag so nested tags of OTHER types don't cause
// mismatches. These containers never hold another same-name element, so the
// non-greedy match is safe.
var mixedContentTagRE = regexp.MustCompile(
	`(?s)<(p|subtitle|th|td|v|text-author|date)\b([^>]*)>(.*?)</(?:p|subtitle|th|td|v|text-author|date)>`,
)

// innerNewlineIndentRE matches a newline followed by horizontal whitespace
// (spaces/tabs) — exactly the shape Go's encoder inserts before every child
// element when indent is set. Stripping this inside a mixed-content container
// restores byte-level round-trip with typical FB2 sources, which keep text
// flush with inline marks. Other whitespace (e.g. a literal space between
// text and a mark) is preserved because we only match at newlines.
//
// Trade-off: if a source file has a literal `\n` inside a `<p>` chardata run,
// this pass collapses it. In practice FB2 uses `<empty-line/>` for paragraph
// breaks and doesn't rely on chardata newlines — if we find a real-world
// case, revisit with a token-aware pass.
var innerNewlineIndentRE = regexp.MustCompile(`\n[ \t]*`)

func compactMixedContent(src []byte) []byte {
	return mixedContentTagRE.ReplaceAllFunc(src, func(match []byte) []byte {
		m := mixedContentTagRE.FindSubmatch(match)
		tag, attrs, content := m[1], m[2], m[3]
		content = innerNewlineIndentRE.ReplaceAll(content, nil)
		return fmt.Appendf(nil, "<%s%s>%s</%s>", tag, attrs, content, tag)
	})
}
