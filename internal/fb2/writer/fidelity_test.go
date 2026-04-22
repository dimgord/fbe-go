package writer_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/dimgord/fbe-go/internal/fb2/parser"
	"github.com/dimgord/fbe-go/internal/fb2/writer"
)

// minimal fb2 source containing the shapes we need to pin.
const fidelitySrc = `<?xml version="1.0" encoding="utf-8"?>
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
      <p>before <strong>bold</strong>, <emphasis>italic</emphasis> and <a l:href="https://example.com">a link</a> tail</p>
      <table>
        <tr><th>head <strong>bold</strong> tail</th></tr>
        <tr><td>cell <emphasis>em</emphasis> end</td></tr>
      </table>
    </section>
  </body>
</FictionBook>`

// roundTrip parses then writes, returning the writer output as a string.
func roundTrip(t *testing.T, src string) string {
	t.Helper()
	fb, err := parser.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := writer.Write(&buf, fb); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

// TestXLinkPrefixRoundTrip: the writer declares `xmlns:l` on the root and
// emits `l:href="..."` on <a>, instead of letting Go auto-pick `xmlns:xlink`
// and re-declaring it on every <a>.
func TestXLinkPrefixRoundTrip(t *testing.T) {
	out := roundTrip(t, fidelitySrc)

	// Root must declare both default FB2 namespace and the xlink prefix `l`.
	if !strings.Contains(out, `<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0" xmlns:l="http://www.w3.org/1999/xlink">`) {
		t.Errorf("root missing xmlns:l declaration:\n%s", out)
	}
	// <a> must use `l:href`, not `xlink:href`.
	if !strings.Contains(out, `l:href="https://example.com"`) {
		t.Errorf(`expected l:href="..." on <a>; got:\n%s`, out)
	}
	// No per-<a> xmlns:xlink redeclaration.
	if strings.Contains(out, `xmlns:xlink=`) {
		t.Errorf("unexpected xmlns:xlink redeclaration (should reuse root's xmlns:l):\n%s", out)
	}
	if strings.Contains(out, `xlink:href=`) {
		t.Errorf("unexpected xlink:href attribute (should be l:href):\n%s", out)
	}
}

// TestMixedContentInlineWhitespace: a paragraph with text interleaved with
// inline marks should keep everything on one line. Go's default encoder
// indent would insert `\n    ` before every child element; Paragraph.MarshalXML
// disables indent around its mixed content so bytes round-trip stably.
func TestMixedContentInlineWhitespace(t *testing.T) {
	out := roundTrip(t, fidelitySrc)

	for _, want := range []string{
		`<p>before <strong>bold</strong>, <emphasis>italic</emphasis> and <a l:href="https://example.com">a link</a> tail</p>`,
		`<th>head <strong>bold</strong> tail</th>`,
		`<td>cell <emphasis>em</emphasis> end</td>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected mixed-content fragment %q in output; got:\n%s", want, out)
		}
	}

	// Regression guards: no Go-auto-indent newline+spaces between text and
	// inline marks, and no indent before the closing tag of a mixed-content
	// paragraph.
	for _, forbid := range []string{
		"before\n",            // text-then-newline inside <p>
		"\n        <strong>",  // indented <strong> inside <p>
		"\n      </p>",        // indented </p> after inline content
	} {
		if strings.Contains(out, forbid) {
			t.Errorf("mixed-content paragraph contains forbidden whitespace %q:\n%s", forbid, out)
		}
	}
}

// TestBlockLevelIndentStillWorks: the mixed-content whitespace fix must not
// disable indent globally — block-level siblings (sections, bodies,
// descriptions, <p>s at the same level) should still be indented 2 spaces
// per depth the way the pre-fix output had them.
func TestBlockLevelIndentStillWorks(t *testing.T) {
	out := roundTrip(t, fidelitySrc)

	for _, want := range []string{
		"\n  <description>",
		"\n    <title-info>",
		"\n  <body>",
		"\n    <section>",
		"\n      <p>",
		"\n      <table>",
		"\n</FictionBook>",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("block-level indent regressed — missing %q in output:\n%s", want, out)
		}
	}
}
