package writer_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/dimgord/fbe-go/internal/fb2/parser"
	"github.com/dimgord/fbe-go/internal/fb2/writer"
)

// TestTableRoundTripPreservesThTdTags verifies that <th>/<td> tags survive
// marshal/unmarshal intact, without the `<Children><Text>…</Text></Children>`
// wrapping that Go's default encoder would produce for a struct whose
// `[]Inline` field lacks a custom MarshalXML.
func TestTableRoundTripPreservesThTdTags(t *testing.T) {
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
      <date value="2026-04-21">x</date>
    </document-info>
  </description>
  <body>
    <section>
      <table>
        <tr><th colspan="2">Header with <strong>bold</strong></th></tr>
        <tr><td>cell one</td><td>cell two</td></tr>
      </table>
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

	// Check structural pieces. We don't pin exact formatting because the
	// encoder inserts indent/newlines around nested inline elements.
	for _, want := range []string{
		`<th colspan="2">`,
		`</th>`,
		`<td>`,
		`cell one`,
		`cell two`,
		`</td>`,
		`<strong>bold</strong>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("writer output missing %q:\n%s", want, out)
		}
	}

	// No duplicate xmlns on every cell — parent namespace must apply.
	if strings.Contains(out, `<th xmlns=`) || strings.Contains(out, `<td xmlns=`) {
		t.Errorf("cells re-declared xmlns; should inherit parent default namespace:\n%s", out)
	}

	// Regression guard: the pre-fix bug produced Go field-name tags.
	for _, forbid := range []string{
		"<Children>",
		"</Children>",
		"<Text>",
		"</Text>",
	} {
		if strings.Contains(out, forbid) {
			t.Errorf("writer output contains forbidden Go-field-name tag %q:\n%s", forbid, out)
		}
	}
}
