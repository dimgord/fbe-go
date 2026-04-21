package parser

import (
	"bytes"
	"strings"
	"testing"

	"golang.org/x/text/encoding/charmap"
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
