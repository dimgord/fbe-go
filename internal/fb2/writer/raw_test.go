package writer_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/dimgord/fbe-go/internal/fb2/parser"
	"github.com/dimgord/fbe-go/internal/fb2/writer"
)

// TestRawFallbackPreservesUnknownBlock verifies that a FB2 extension element
// (one we don't have a typed representation for) survives round-trip verbatim.
func TestRawFallbackPreservesUnknownBlock(t *testing.T) {
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
      <p>before</p>
      <custom-extension data-source="Flibusta" count="42">extension <b>content</b></custom-extension>
      <p>after</p>
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
	if !strings.Contains(out, "<custom-extension") {
		t.Errorf("writer output dropped custom-extension element:\n%s", out)
	}
	if !strings.Contains(out, `data-source="Flibusta"`) {
		t.Errorf("writer output dropped attribute data-source:\n%s", out)
	}
	if !strings.Contains(out, `count="42"`) {
		t.Errorf("writer output dropped attribute count:\n%s", out)
	}
	if !strings.Contains(out, "extension ") {
		t.Errorf("writer output dropped text content of custom-extension:\n%s", out)
	}
	if !strings.Contains(out, "<b>content</b>") {
		t.Errorf("writer output dropped nested <b> inside custom-extension:\n%s", out)
	}
	t.Logf("output:\n%s", out)
}

// TestRawFallbackPreservesUnknownInline: unknown inline element inside a <p>.
func TestRawFallbackPreservesUnknownInline(t *testing.T) {
	const src = `<?xml version="1.0" encoding="utf-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0" xmlns:l="http://www.w3.org/1999/xlink">
  <description>
    <title-info>
      <genre>sf</genre><author><first-name>X</first-name><last-name>Y</last-name></author>
      <book-title>t</book-title><lang>en</lang>
    </title-info>
    <document-info>
      <author><nickname>x</nickname></author><id>x</id><version>1.0</version>
      <date value="2026-04-21">x</date>
    </document-info>
  </description>
  <body>
    <section>
      <p>before <ruby rb="漢" rt="kan">漢</ruby> after</p>
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
	if !strings.Contains(out, `<ruby`) {
		t.Errorf("unknown inline <ruby> was dropped:\n%s", out)
	}
	if !strings.Contains(out, `rb="漢"`) {
		t.Errorf("ruby attribute rb was dropped:\n%s", out)
	}
}
