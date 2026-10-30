package xsd

import (
	"os"
	"strings"
	"testing"
)

func TestValidateBlank(t *testing.T) {
	f, err := os.Open("../../../testdata/blank.fb2")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	errs, err := Validate(f)
	if err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	if len(errs) > 0 {
		for _, e := range errs {
			t.Logf("schema error: %d:%d %s", e.Line, e.Column, e.Message)
		}
		t.Fatalf("blank.fb2 should be valid, got %d errors", len(errs))
	}
}

func TestValidateRich(t *testing.T) {
	f, err := os.Open("../../../testdata/rich.fb2")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	errs, err := Validate(f)
	if err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	for _, e := range errs {
		t.Logf("schema error: %d:%d %s", e.Line, e.Column, e.Message)
	}
}

// An unknown element must be reported on its own line, pointing at its tag.
func TestValidateReportsLineAndColumn(t *testing.T) {
	src, err := os.ReadFile("../../../testdata/blank.fb2")
	if err != nil {
		t.Fatal(err)
	}
	bad := strings.Replace(string(src), "<book-title>", "<bogus-tag/><book-title>", 1)
	if bad == string(src) {
		t.Fatal("blank.fb2 has no <book-title> to break")
	}
	errs, err := Validate(strings.NewReader(bad))
	if err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	if len(errs) == 0 {
		t.Fatal("expected a schema error for <bogus-tag/>")
	}
	lines := strings.Split(bad, "\n")
	e := errs[0]
	if e.Line < 1 || e.Line > len(lines) || e.Column < 1 {
		t.Fatalf("bad position %d:%d for %q", e.Line, e.Column, e.Message)
	}
	if got := lines[e.Line-1][e.Column-1:]; !strings.HasPrefix(got, "<bogus-tag") {
		t.Errorf("position %d:%d points at %q, want <bogus-tag; message %q", e.Line, e.Column, got, e.Message)
	}
}

func TestPositionOf(t *testing.T) {
	const src = "<a>\n  <poem>\n    <x:text-author>t</x:text-author>\n  </poem>\n</a>\n"
	cases := []struct {
		line     int
		elem     string
		wantLine int
		wantCol  int
	}{
		{3, "{http://ns}text-author", 3, 5}, // on the reported line
		{2, "text-author", 2, 1},            // tag not on the reported line → its start
		{2, "nosuch", 2, 1},                 // not found → start of reported line
		{0, "text-author", 0, 0},            // unknown line
		{9, "text-author", 9, 1},            // past the end
	}
	for _, c := range cases {
		l, col := positionOf([]byte(src), c.line, c.elem)
		if l != c.wantLine || col != c.wantCol {
			t.Errorf("positionOf(%d, %q) = %d:%d, want %d:%d", c.line, c.elem, l, col, c.wantLine, c.wantCol)
		}
	}
}
