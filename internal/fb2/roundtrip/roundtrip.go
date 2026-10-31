// Package roundtrip guards a parse against silent data loss.
//
// parser.Parse is lenient on purpose: the Raw fallback keeps unknown elements so that real-world
// FB2s round-trip. But a typo in hand-edited XML (`<autho>`) leaves encoding/xml's Skip hunting for
// a close tag that never comes, and it eats the rest of the document, body included, with no error.
// ParseVerified parses, writes the result back and compares structural tag counts with the input.
package roundtrip

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/dimgord/fbe-go/internal/fb2/doc"
	"github.com/dimgord/fbe-go/internal/fb2/parser"
	"github.com/dimgord/fbe-go/internal/fb2/writer"
	"github.com/dimgord/fbe-go/internal/fb2/xsd"
)

// ParseVerified parses src and fails, naming the first problem, if the result lost structure.
// A panic on a broken document comes back as an error.
func ParseVerified(src string) (fb *doc.FictionBook, err error) {
	defer func() {
		if r := recover(); r != nil {
			fb = nil
			err = fmt.Errorf("parse panic: %v", r)
		}
	}()
	fb, err = parser.Parse(strings.NewReader(src))
	if err != nil {
		return nil, err
	}
	var rt bytes.Buffer
	if err := writer.Write(&rt, fb); err != nil {
		return nil, fmt.Errorf("round-trip serialize: %w", err)
	}
	if msg := dataLossMessage(src, rt.String()); msg != "" {
		return nil, fmt.Errorf("%s", msg)
	}
	return fb, nil
}

// dataLossMessage compares structurally significant FB2 element
// openings between the user-provided `in` and the writer's round-trip
// output `rt`, and returns ONE focused, actionable error string — or
// "" if no loss.
//
// Strategy:
//
//  1. Scan `in` for unknown FB2 element names via xsd.FindUnknownElements.
//     If any unknown tag appears (e.g. `<autho>` from a typo'd `<author>`),
//     return a message that names the first one and its line number.
//     This is the root cause in the vast majority of editor typos — every
//     "missing element" downstream is just a cascading consequence.
//
//  2. Only when no unknown tag is found, fall back to lost-element count.
//     Then report just the FIRST (and topmost-in-hierarchy) element that
//     dropped, not the full cascade. Mentioning every consequence makes
//     the message a wall of text that obscures the actual fix site.
//
// The caller (frontend Apply handler) keeps the user's CM6 text intact
// on error, so a focused single-issue message lets them jump to the
// problem and re-Apply.
func dataLossMessage(in, rt string) string {
	if unks := xsd.FindUnknownElements([]byte(in)); len(unks) > 0 {
		u := unks[0]
		return fmt.Sprintf(
			"Unknown element <%s> at line %d — likely a typo. Check that tag and try Apply again.",
			u.Element, u.Line,
		)
	}
	// Topmost-first ordering — a missing <body> is the root cause when
	// everything below it also goes missing.
	hierarchical := []string{
		"FictionBook", "description", "title-info", "src-title-info",
		"document-info", "publish-info", "body",
		"author", "coverpage", "annotation",
		"section", "epigraph", "cite", "poem", "stanza",
		"table", "tr", "td", "th",
		"p", "v", "image", "binary",
	}
	for _, name := range hierarchical {
		inC := countOpen(in, name)
		rtC := countOpen(rt, name)
		if rtC < inC {
			return fmt.Sprintf(
				"after parse, <%s> count dropped from %d to %d. The most likely cause is a structural error (mismatched tags, missing close). Fix and try Apply again",
				name, inC, rtC,
			)
		}
	}
	return ""
}

// countOpen counts `<NAME` occurrences where the byte right after the
// name is a non-name char (space, `>`, `/`, newline, tab). Misses
// nothing valid; over-counts only if the same prefix appears verbatim
// inside a CDATA / comment — acceptable for a typo-detection heuristic.
func countOpen(s, name string) int {
	needle := "<" + name
	count := 0
	pos := 0
	for {
		i := strings.Index(s[pos:], needle)
		if i < 0 {
			return count
		}
		end := pos + i + len(needle)
		if end >= len(s) {
			return count
		}
		c := s[end]
		// Name boundary: tag can be followed by space, `>`, `/`, or
		// whitespace; not by another name char. Avoids `<p` matching
		// `<publish-info` etc.
		if c == ' ' || c == '>' || c == '/' || c == '\t' || c == '\n' || c == '\r' {
			count++
		}
		pos = end
	}
}
