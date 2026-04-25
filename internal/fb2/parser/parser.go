// Package parser reads FB2 XML and produces a doc.FictionBook.
//
// Responsibilities:
//   - Handle BOM + XML encoding declaration. FB2 files in the wild use utf-8
//     (majority), utf-16, windows-1251, koi8-r, iso-8859-1.
//   - Preserve whitespace inside paragraphs (significant in poetry).
//   - Accept malformed-but-recoverable docs (legacy FBE produced non-canonical output).
package parser

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/ianaindex"
	"golang.org/x/text/encoding/unicode"

	"github.com/dimgord/fbe-go/internal/fb2/doc"
)

// Parse reads an FB2 XML stream into a doc.FictionBook. The input may be in
// any encoding supported by charsetReader (utf-8, utf-16, windows-1251, koi8-r,
// and anything IANA registers).
func Parse(r io.Reader) (*doc.FictionBook, error) {
	br := bufio.NewReader(r)

	// Detect BOM. If present, strip it and force the encoding.
	forced, err := detectBOM(br)
	if err != nil {
		return nil, err
	}

	// When a BOM forced a non-UTF-8 encoding (UTF-16 LE/BE), pre-decode
	// the byte stream into UTF-8 *before* xml.NewDecoder reads it.
	// Reason: encoding/xml reads the `<?xml encoding="..."?>` declaration
	// with the raw bytes, assuming UTF-8, and only consults CharsetReader
	// after parsing the declaration. UTF-16 LE bytes start with `<\x00?\x00…`
	// which the decoder fails on as "expected element name after <" — so
	// CharsetReader never gets a chance.
	xmlInput := io.Reader(br)
	if forced != nil && forced != unicode.UTF8 {
		xmlInput = forced.NewDecoder().Reader(br)
	}

	dec := xml.NewDecoder(xmlInput)
	dec.DefaultSpace = doc.NSFictionBook
	dec.Strict = false // FBE output isn't always canonical
	if forced != nil {
		// xmlInput is already UTF-8; if the declaration names a non-UTF-8
		// encoding, returning a passthrough avoids double-decoding.
		dec.CharsetReader = func(_ string, in io.Reader) (io.Reader, error) {
			return in, nil
		}
	} else {
		dec.CharsetReader = charsetReader
	}

	var fb doc.FictionBook
	if err := dec.Decode(&fb); err != nil {
		return nil, fmt.Errorf("fb2 parse: %w", err)
	}
	return &fb, nil
}

// detectBOM inspects the first few bytes, unreads them, and if a BOM is found
// returns the forced encoding (and the reader is positioned past the BOM).
func detectBOM(br *bufio.Reader) (encoding.Encoding, error) {
	head, err := br.Peek(4)
	if err != nil && err != io.EOF {
		return nil, err
	}
	switch {
	case len(head) >= 3 && head[0] == 0xEF && head[1] == 0xBB && head[2] == 0xBF:
		_, _ = br.Discard(3)
		return unicode.UTF8, nil
	case len(head) >= 2 && head[0] == 0xFF && head[1] == 0xFE:
		_, _ = br.Discard(2)
		return unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM), nil
	case len(head) >= 2 && head[0] == 0xFE && head[1] == 0xFF:
		_, _ = br.Discard(2)
		return unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM), nil
	}
	return nil, nil
}

// charsetReader maps an encoding name (from <?xml encoding="X"?>) to a decoder.
// Handles the common FB2 cases explicitly; falls back to IANA registry lookup.
func charsetReader(name string, in io.Reader) (io.Reader, error) {
	enc, err := resolveEncoding(name)
	if err != nil {
		return nil, err
	}
	if enc == nil {
		// No transformation needed (utf-8 is the Go XML default).
		return in, nil
	}
	return enc.NewDecoder().Reader(in), nil
}

// resolveEncoding returns nil for utf-8 (no-op), or a decoder for other encodings.
func resolveEncoding(name string) (encoding.Encoding, error) {
	n := strings.ToLower(strings.TrimSpace(name))
	switch n {
	case "", "utf-8", "utf8", "us-ascii", "ascii":
		return nil, nil
	case "utf-16", "utf16":
		return unicode.UTF16(unicode.LittleEndian, unicode.UseBOM), nil
	case "utf-16le":
		return unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM), nil
	case "utf-16be":
		return unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM), nil
	case "windows-1251", "cp1251", "cp-1251":
		return charmap.Windows1251, nil
	case "windows-1252", "cp1252":
		return charmap.Windows1252, nil
	case "koi8-r", "koi8r":
		return charmap.KOI8R, nil
	case "koi8-u", "koi8u":
		return charmap.KOI8U, nil
	case "iso-8859-1", "iso8859-1", "latin1":
		return charmap.ISO8859_1, nil
	case "iso-8859-5":
		return charmap.ISO8859_5, nil
	}
	// Fallback: IANA registry (covers ~200 encodings).
	enc, err := ianaindex.IANA.Encoding(name)
	if err != nil {
		return nil, fmt.Errorf("parser: unknown encoding %q: %w", name, err)
	}
	if enc == nil {
		return nil, fmt.Errorf("parser: unsupported encoding %q", name)
	}
	return enc, nil
}
