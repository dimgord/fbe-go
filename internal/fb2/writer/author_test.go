package writer_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/dimgord/fbe-go/internal/fb2/parser"
	"github.com/dimgord/fbe-go/internal/fb2/writer"
)

// TestAuthorEmptyLastNamePreserved is a fidelity-broken regression guard:
// some real-world FB2 files contain an author with first-name + empty
// <last-name/> + nickname (XSD authorType sequence A — last-name is
// required, even if empty). Go's encoding/xml with `,omitempty` on
// LastName silently drops the element on round-trip, turning a valid
// document into an invalid one (sequence A's last-name is missing).
//
// Source: ~/Documents/books/rozstrilyane_pokolinnya/valerjan_pidmogyl6nyj/
//         Nevelichka drama.fb2 (third <author> under <document-info>).
func TestAuthorEmptyLastNamePreserved(t *testing.T) {
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
      <author>
        <first-name>Сонячна</first-name>
        <last-name/>
        <nickname>VV</nickname>
      </author>
      <id>x</id><version>1.0</version>
      <date value="2026-04-25">x</date>
    </document-info>
  </description>
  <body><section><p>x</p></section></body>
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

	if !strings.Contains(out, "<first-name>Сонячна</first-name>") {
		t.Errorf("first-name dropped:\n%s", out)
	}
	// The required <last-name> must survive even when empty.
	if !strings.Contains(out, "<last-name></last-name>") &&
		!strings.Contains(out, "<last-name/>") {
		t.Errorf("empty <last-name> dropped — schema sequence A requires it:\n%s", out)
	}
	if !strings.Contains(out, "<nickname>VV</nickname>") {
		t.Errorf("nickname dropped:\n%s", out)
	}
}

// TestAuthorNicknameOnlyDoesNotInjectEmptyNames guards the other branch of
// the authorType choice (sequence B — nickname only). If the fix overshoots
// and always emits empty <first-name>/<last-name>, nickname-only authors
// would round-trip as invalid sequence-A documents.
func TestAuthorNicknameOnlyDoesNotInjectEmptyNames(t *testing.T) {
	const src = `<?xml version="1.0" encoding="utf-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0" xmlns:l="http://www.w3.org/1999/xlink">
  <description>
    <title-info>
      <genre>sf</genre>
      <author><nickname>OnlyNick</nickname></author>
      <book-title>t</book-title>
      <lang>en</lang>
    </title-info>
    <document-info>
      <author><nickname>x</nickname></author>
      <id>x</id><version>1.0</version>
      <date value="2026-04-25">x</date>
    </document-info>
  </description>
  <body><section><p>x</p></section></body>
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

	if !strings.Contains(out, "<nickname>OnlyNick</nickname>") {
		t.Errorf("nickname dropped:\n%s", out)
	}
	// Title-info author was nickname-only; it must NOT gain empty name fields.
	idx := strings.Index(out, "<nickname>OnlyNick</nickname>")
	if idx < 0 {
		t.Fatal("anchor element missing")
	}
	// Inspect the enclosing <author>…</author> for unwanted siblings.
	authorOpen := strings.LastIndex(out[:idx], "<author>")
	authorClose := strings.Index(out[idx:], "</author>")
	if authorOpen < 0 || authorClose < 0 {
		t.Fatal("could not locate enclosing <author>")
	}
	block := out[authorOpen : idx+authorClose+len("</author>")]
	for _, bad := range []string{"<first-name>", "<first-name/>", "<last-name>", "<last-name/>", "<middle-name>"} {
		if strings.Contains(block, bad) {
			t.Errorf("nickname-only author leaked %q:\n%s", bad, block)
		}
	}
}
