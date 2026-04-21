package html

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/dimgord/fbe-go/internal/fb2/parser"
)

func TestExportBlank(t *testing.T) {
	in, err := os.Open("../../../../testdata/blank.fb2")
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	fb, err := parser.Parse(in)
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := Export(&buf, fb); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	for _, want := range []string{
		"<!DOCTYPE html>",
		"<title>Blank</title>",
		"<h1 class=\"book-title\">Blank</h1>",
		"Unknown Author",
		"Start writing here",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q", want)
		}
	}
}

func TestExportRich(t *testing.T) {
	in, err := os.Open("../../../../testdata/rich.fb2")
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	fb, err := parser.Parse(in)
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := Export(&buf, fb); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	for _, want := range []string{
		// Description pieces.
		"<title>Кобзар (тест)</title>",
		"Тарас Шевченко",
		"class=\"annotation\"",
		// Body content.
		"<blockquote class=\"epigraph\">",
		"<strong>жирним</strong>",
		"<em>курсивом</em>",
		"<code>моноширинним</code>",
		"<sub>2</sub>",
		"<sup>3</sup>",
		`<a href="https://example.com"`,
		`class="empty-line"`,
		"<blockquote class=\"cite\">",
		"Підзаголовок",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("rich export missing %q\nfull:\n%s", want, out)
			return
		}
	}
}
