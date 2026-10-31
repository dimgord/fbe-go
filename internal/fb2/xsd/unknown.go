package xsd

import (
	"bytes"
	"regexp"
)

// knownFB2Elements is the set of element local-names in FictionBook.xsd.
// Tags outside this set are flagged by FindUnknownElements. Maintained by
// hand rather than derived from the XSD at runtime — the list changes only
// when the bundled schema changes, and hand-maintenance is cheaper than
// shipping an XSD-introspection pass that would otherwise be called on
// every Validate.
//
// Keep this list in sync with `SchemaFiles` (FictionBook.xsd + friends).
var knownFB2Elements = map[string]bool{
	// Root + top-level
	"FictionBook": true, "stylesheet": true, "description": true,
	"body": true, "binary": true,

	// Description sub-tree
	"title-info": true, "src-title-info": true, "document-info": true,
	"publish-info": true, "custom-info": true, "output": true,
	"output-document-class": true, "part": true,
	"genre": true, "author": true, "book-title": true, "annotation": true,
	"keywords": true, "date": true, "coverpage": true, "lang": true,
	"src-lang": true, "translator": true, "sequence": true,
	"first-name": true, "middle-name": true, "last-name": true,
	"nickname": true, "home-page": true, "email": true, "id": true,
	"program-used": true, "version": true, "history": true, "publisher": true,
	"book-name": true, "city": true, "year": true, "isbn": true,
	"src-ocr": true, "src-url": true,

	// Body
	"section": true, "title": true, "epigraph": true, "image": true,

	// Block-level
	"p": true, "poem": true, "subtitle": true, "cite": true,
	"empty-line": true, "table": true, "text-author": true,
	"stanza": true, "v": true,

	// Table
	"tr": true, "th": true, "td": true,

	// Inline marks
	"strong": true, "emphasis": true, "style": true, "a": true,
	"strikethrough": true, "sub": true, "sup": true, "code": true,
}

// openTagRE matches an opening tag's local name: `<name ...>`, `<name/>`,
// `<name>`. Skips closing tags (`</name>` starts with `/`), processing
// instructions (`<?xml`), and comments (`<!--`) via the alphabetic
// first-char requirement.
var openTagRE = regexp.MustCompile(`<([a-zA-Z][\w-]*)`)

// FindUnknownElements scans src for element names outside the bundled
// FictionBook 2.0 vocabulary and returns one ValidationError per
// occurrence. Supplements Validate: the schema validator's content-model
// recovery sometimes suppresses later unknown-element errors once the
// first violation in a content group trips up the DFA — this scan is
// structure-agnostic, so every unknown element shows up regardless.
//
// Line/col are 1-based, pointing at the `<` of the opening tag.
func FindUnknownElements(src []byte) []ValidationError {
	var out []ValidationError
	for _, m := range openTagRE.FindAllSubmatchIndex(src, -1) {
		tag := string(src[m[2]:m[3]])
		if knownFB2Elements[tag] {
			continue
		}
		line, col := byteOffsetToLineCol(src, m[0])
		out = append(out, ValidationError{
			Line:    line,
			Column:  col,
			Element: tag,
			Message: "Unknown FB2 element '" + tag + "' — not in the FictionBook 2.0 schema. Preserved verbatim on save.",
		})
	}
	return out
}

// MergeXSDAndUnknown returns a combined error list: all xsdErrs (from
// Validate), plus unknowns that aren't already reported by Validate on the
// same line with the same element name. Keeps the validator's richer messages
// when both cover the same issue.
func MergeXSDAndUnknown(xsdErrs, unknowns []ValidationError) []ValidationError {
	type key struct {
		line int
		tag  string
	}
	seen := make(map[key]bool, len(xsdErrs))
	for _, e := range xsdErrs {
		if e.Element != "" {
			seen[key{e.Line, e.Element}] = true
		}
	}
	out := make([]ValidationError, 0, len(xsdErrs)+len(unknowns))
	out = append(out, xsdErrs...)
	for _, u := range unknowns {
		if !seen[key{u.Line, u.Element}] {
			out = append(out, u)
		}
	}
	return out
}

// 1-based (line, column). Newlines terminate lines.
func byteOffsetToLineCol(src []byte, offset int) (int, int) {
	if offset < 0 || offset > len(src) {
		return 0, 0
	}
	before := src[:offset]
	line := bytes.Count(before, []byte{'\n'}) + 1
	lastNL := bytes.LastIndexByte(before, '\n')
	col := offset - (lastNL + 1) + 1
	return line, col
}
