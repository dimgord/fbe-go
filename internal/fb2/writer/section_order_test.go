package writer_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/dimgord/fbe-go/internal/fb2/parser"
	"github.com/dimgord/fbe-go/internal/fb2/writer"
)

// TestSectionBodyPreservesInterleaving: Rev 37 replaced doc.Section's
// (Sections + Blocks) pair with a single ordered Body slice so mixed content
// (e.g. `<empty-line/> <section/> <empty-line/>`) round-trips in the order
// the file actually had, not reshuffled by which-slice-you're-in. Before
// this rev, Go's default encoding/xml splits `<section>` and block siblings
// into two disjoint slices, emitting all blocks before all sections
// regardless of the source order.
func TestSectionBodyPreservesInterleaving(t *testing.T) {
	const src = `<?xml version="1.0" encoding="utf-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0" xmlns:l="http://www.w3.org/1999/xlink">
  <description>
    <title-info>
      <genre>sf</genre>
      <author><first-name>X</first-name><last-name>Y</last-name></author>
      <book-title>t</book-title>
      <lang>en</lang>
    </title-info>
    <document-info>
      <author><nickname>x</nickname></author>
      <id>x</id><version>1.0</version>
      <date value="2026-04-22">x</date>
    </document-info>
  </description>
  <body>
    <section>
      <title><p>Outer</p></title>
      <p>before-section</p>
      <section><p>nested-one</p></section>
      <p>between-sections</p>
      <section><p>nested-two</p></section>
      <p>after-sections</p>
    </section>
  </body>
</FictionBook>`

	fb, err := parser.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := writer.Write(&buf, fb); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	// Assert the six body entries appear in the source order. We pin them as
	// substrings so indent whitespace and nested-section bodies don't break
	// the assertion. `strings.Index` returns -1 for not-found.
	marks := []string{
		"<p>before-section</p>",
		"<section>",                 // opens nested-one
		"<p>nested-one</p>",
		"</section>",                // closes nested-one
		"<p>between-sections</p>",
		"<section>",                 // opens nested-two (same literal, use position after previous)
		"<p>nested-two</p>",
		"</section>",                // closes nested-two
		"<p>after-sections</p>",
	}
	cursor := 0
	for _, m := range marks {
		idx := strings.Index(out[cursor:], m)
		if idx < 0 {
			t.Fatalf("missing substring %q after position %d; got:\n%s", m, cursor, out)
		}
		cursor += idx + len(m)
	}
}
