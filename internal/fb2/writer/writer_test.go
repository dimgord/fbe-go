package writer_test

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/dimgord/fbe-go/internal/fb2/doc"
	"github.com/dimgord/fbe-go/internal/fb2/parser"
	"github.com/dimgord/fbe-go/internal/fb2/writer"
)

// TestRoundTrip parses a file, writes it back, re-parses the output, and checks
// that the two FictionBook structures are equivalent on the fields we currently
// round-trip. Byte-exact preservation is NOT a goal — whitespace and attribute
// order differ. What we verify is that no content is lost.
func TestRoundTrip(t *testing.T) {
	cases := []string{
		"../../../testdata/blank.fb2",
		"../../../testdata/rich.fb2",
	}
	for _, path := range cases {
		t.Run(path, func(t *testing.T) {
			src, err := os.Open(path)
			if err != nil {
				t.Fatal(err)
			}
			defer src.Close()

			fb1, err := parser.Parse(src)
			if err != nil {
				t.Fatalf("first parse: %v", err)
			}

			var buf bytes.Buffer
			if err := writer.Write(&buf, fb1); err != nil {
				t.Fatalf("write: %v", err)
			}

			fb2, err := parser.Parse(bytes.NewReader(buf.Bytes()))
			if err != nil {
				t.Fatalf("second parse of writer output:\n%s\nerror: %v", buf.String(), err)
			}

			check(t, fb1, fb2)

			// Sanity: writer output should declare the FB2 namespace at the root.
			if !strings.Contains(buf.String(), `xmlns="http://www.gribuser.ru/xml/fictionbook/2.0"`) {
				t.Errorf("writer output missing FB2 xmlns declaration:\n%s", buf.String())
			}
			// Sanity: writer output should NOT redeclare xmlns on every paragraph.
			if strings.Contains(buf.String(), `<p xmlns=`) {
				t.Errorf("writer output re-declares xmlns on <p> elements:\n%s", buf.String())
			}
		})
	}
}

func check(t *testing.T, a, b *doc.FictionBook) {
	t.Helper()
	if got, want := b.Description.TitleInfo.BookTitle, a.Description.TitleInfo.BookTitle; got != want {
		t.Errorf("BookTitle: %q → %q", want, got)
	}
	if got, want := b.Description.TitleInfo.Lang, a.Description.TitleInfo.Lang; got != want {
		t.Errorf("Lang: %q → %q", want, got)
	}
	if got, want := len(b.Bodies), len(a.Bodies); got != want {
		t.Fatalf("Bodies count: %d → %d", want, got)
	}
	for i := range a.Bodies {
		if got, want := len(b.Bodies[i].Sections), len(a.Bodies[i].Sections); got != want {
			t.Errorf("body[%d].Sections count: %d → %d", i, want, got)
		}
	}
	// Quick paragraph-content sanity on the first body's first section.
	if len(a.Bodies) > 0 && len(a.Bodies[0].Sections) > 0 {
		aBlocks := a.Bodies[0].Sections[0].Blocks
		bBlocks := b.Bodies[0].Sections[0].Blocks
		if got, want := len(bBlocks), len(aBlocks); got != want {
			t.Errorf("section[0].Blocks count: %d → %d", want, got)
		}
	}
}
