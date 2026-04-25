package parser

import (
	"bytes"
	"strings"
	"testing"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"
)

const utf8Doc = `<?xml version="1.0" encoding="utf-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0" xmlns:l="http://www.w3.org/1999/xlink">
  <description>
    <title-info>
      <genre>sf</genre>
      <author><first-name>Тарас</first-name><last-name>Шевченко</last-name></author>
      <book-title>Кобзар</book-title>
      <lang>uk</lang>
    </title-info>
    <document-info>
      <author><nickname>test</nickname></author>
      <id>x</id><version>1.0</version>
      <date value="2026-04-21">21 April 2026</date>
    </document-info>
  </description>
  <body><section><p>Hello</p></section></body>
</FictionBook>`

func TestParseUTF8(t *testing.T) {
	fb, err := Parse(strings.NewReader(utf8Doc))
	if err != nil {
		t.Fatal(err)
	}
	if got := fb.Description.TitleInfo.BookTitle; got != "Кобзар" {
		t.Errorf("BookTitle = %q, want Кобзар", got)
	}
}

func TestParseUTF8BOM(t *testing.T) {
	buf := append([]byte{0xEF, 0xBB, 0xBF}, utf8Doc...)
	fb, err := Parse(bytes.NewReader(buf))
	if err != nil {
		t.Fatal(err)
	}
	if fb.Description.TitleInfo.BookTitle != "Кобзар" {
		t.Errorf("BookTitle after BOM strip = %q", fb.Description.TitleInfo.BookTitle)
	}
}

func TestParseWindows1251(t *testing.T) {
	const body = `<?xml version="1.0" encoding="windows-1251"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0" xmlns:l="http://www.w3.org/1999/xlink">
  <description>
    <title-info>
      <genre>sf</genre>
      <author><first-name>Лев</first-name><last-name>Толстой</last-name></author>
      <book-title>Война и мир</book-title>
      <lang>ru</lang>
    </title-info>
    <document-info><author><nickname>x</nickname></author><id>x</id><version>1.0</version><date value="2026-04-21">x</date></document-info>
  </description>
  <body><section><p>Привет</p></section></body>
</FictionBook>`
	encoded, err := charmap.Windows1251.NewEncoder().Bytes([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	fb, err := Parse(bytes.NewReader(encoded))
	if err != nil {
		t.Fatal(err)
	}
	if got := fb.Description.TitleInfo.BookTitle; got != "Война и мир" {
		t.Errorf("BookTitle = %q, want Война и мир", got)
	}
}

// TestParseUTF16LEWithBOM guards a regression first surfaced by the corpus
// run on `~/Documents/books/The Long Watch.fb2` (UTF-16 LE with BOM).
//
// detectBOM correctly identifies the BOM and strips it, but Go's
// encoding/xml tries to read the XML declaration as UTF-8 *before*
// CharsetReader is consulted — so it sees `<\x00?\x00x\x00m\x00l...` and
// fails with "expected element name after <". The fix is to pre-decode the
// stream into UTF-8 before constructing the xml.Decoder when a BOM forces
// a non-UTF-8 encoding.
func TestParseUTF16LEWithBOM(t *testing.T) {
	const body = `<?xml version="1.0" encoding="utf-16"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0" xmlns:l="http://www.w3.org/1999/xlink">
  <description>
    <title-info>
      <genre>sf</genre>
      <author><first-name>Robert</first-name><last-name>Heinlein</last-name></author>
      <book-title>The Long Watch</book-title>
      <lang>en</lang>
    </title-info>
    <document-info><author><nickname>x</nickname></author><id>x</id><version>1.0</version><date value="2026-04-25">x</date></document-info>
  </description>
  <body><section><p>Hello</p></section></body>
</FictionBook>`
	encoded, err := unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewEncoder().Bytes([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded) < 2 || encoded[0] != 0xFF || encoded[1] != 0xFE {
		t.Fatalf("test fixture missing UTF-16 LE BOM: % x", encoded[:8])
	}
	fb, err := Parse(bytes.NewReader(encoded))
	if err != nil {
		t.Fatal(err)
	}
	if got := fb.Description.TitleInfo.BookTitle; got != "The Long Watch" {
		t.Errorf("BookTitle = %q, want %q", got, "The Long Watch")
	}
}

// TestParseUTF16BEWithBOM mirrors TestParseUTF16LEWithBOM for the
// big-endian byte order. Less common in real corpora but the BOM branch
// in detectBOM exists for it, so it should round-trip too.
func TestParseUTF16BEWithBOM(t *testing.T) {
	const body = `<?xml version="1.0" encoding="utf-16"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0" xmlns:l="http://www.w3.org/1999/xlink">
  <description>
    <title-info>
      <genre>sf</genre>
      <author><first-name>X</first-name><last-name>Y</last-name></author>
      <book-title>BE-Title</book-title>
      <lang>en</lang>
    </title-info>
    <document-info><author><nickname>x</nickname></author><id>x</id><version>1.0</version><date value="2026-04-25">x</date></document-info>
  </description>
  <body><section><p>Hello</p></section></body>
</FictionBook>`
	encoded, err := unicode.UTF16(unicode.BigEndian, unicode.UseBOM).NewEncoder().Bytes([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded) < 2 || encoded[0] != 0xFE || encoded[1] != 0xFF {
		t.Fatalf("test fixture missing UTF-16 BE BOM: % x", encoded[:8])
	}
	fb, err := Parse(bytes.NewReader(encoded))
	if err != nil {
		t.Fatal(err)
	}
	if got := fb.Description.TitleInfo.BookTitle; got != "BE-Title" {
		t.Errorf("BookTitle = %q, want BE-Title", got)
	}
}

func TestParseKOI8R(t *testing.T) {
	const body = `<?xml version="1.0" encoding="koi8-r"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0" xmlns:l="http://www.w3.org/1999/xlink">
  <description>
    <title-info>
      <genre>sf</genre>
      <author><first-name>Иван</first-name><last-name>Тургенев</last-name></author>
      <book-title>Отцы и дети</book-title>
      <lang>ru</lang>
    </title-info>
    <document-info><author><nickname>x</nickname></author><id>x</id><version>1.0</version><date value="2026-04-21">x</date></document-info>
  </description>
  <body><section><p>Привет</p></section></body>
</FictionBook>`
	encoded, err := charmap.KOI8R.NewEncoder().Bytes([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	fb, err := Parse(bytes.NewReader(encoded))
	if err != nil {
		t.Fatal(err)
	}
	if got := fb.Description.TitleInfo.BookTitle; got != "Отцы и дети" {
		t.Errorf("BookTitle = %q, want Отцы и дети", got)
	}
}
