package xsd

import (
	"slices"
	"strings"
	"testing"
)

func TestFindUnknownElements_reportsEveryOccurrence(t *testing.T) {
	const src = `<?xml version="1.0" encoding="utf-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0" xmlns:l="http://www.w3.org/1999/xlink">
  <body>
    <section>
      <p>ok</p>
      <empty-lune/>
      <section>
        <p>nested</p>
      </section>
      <empty-lane/>
      <empty-lyne/>
    </section>
  </body>
</FictionBook>
`
	errs := FindUnknownElements([]byte(src))
	names := tagsFromUnknownErrs(errs)
	want := []string{"empty-lune", "empty-lane", "empty-lyne"}
	for _, w := range want {
		if !slices.Contains(names, w) {
			t.Errorf("expected %q in unknown-element report, got %v", w, names)
		}
	}
	if len(errs) < len(want) {
		t.Errorf("expected at least %d unknowns, got %d: %v", len(want), len(errs), errs)
	}
}

func TestFindUnknownElements_skipsKnownTags(t *testing.T) {
	const src = `<?xml version="1.0"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
  <description><title-info><book-title>X</book-title></title-info></description>
  <body><section><p>hi <strong>bold</strong> <a>link</a></p></section></body>
</FictionBook>
`
	if errs := FindUnknownElements([]byte(src)); len(errs) > 0 {
		t.Errorf("all-known document produced unknown-element errors: %v", errs)
	}
}

func TestFindUnknownElements_skipsCommentsAndPIs(t *testing.T) {
	const src = `<?xml version="1.0"?>
<!-- a comment -->
<?custom instruction?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0"/>
`
	if errs := FindUnknownElements([]byte(src)); len(errs) > 0 {
		t.Errorf("comments / PIs produced false-positive unknowns: %v", errs)
	}
}

func TestFindUnknownElements_linesAreOneBased(t *testing.T) {
	const src = "<FictionBook>\n  <weird/>\n</FictionBook>"
	errs := FindUnknownElements([]byte(src))
	if len(errs) != 1 {
		t.Fatalf("got %d unknowns, want 1: %v", len(errs), errs)
	}
	if errs[0].Line != 2 {
		t.Errorf("line = %d, want 2", errs[0].Line)
	}
	if errs[0].Column != 3 {
		t.Errorf("column = %d, want 3 (the `<` of `<weird/>` after 2 leading spaces)", errs[0].Column)
	}
}

func TestMergeXSDAndUnknown_dedupsOverlap(t *testing.T) {
	xsdErrs := []ValidationError{
		{
			Line:    5,
			Column:  3,
			Message: "Element '{http://www.gribuser.ru/xml/fictionbook/2.0}empty-lune': This element is not expected.",
		},
	}
	unknowns := []ValidationError{
		// Same line + tag as above → should be dropped.
		{Line: 5, Column: 3, Message: "Unknown FB2 element 'empty-lune' — not in the FictionBook 2.0 schema. Preserved verbatim on save."},
		// Different line → kept.
		{Line: 9, Column: 3, Message: "Unknown FB2 element 'empty-lyne' — not in the FictionBook 2.0 schema. Preserved verbatim on save."},
	}
	merged := MergeXSDAndUnknown(xsdErrs, unknowns)
	if len(merged) != 2 {
		t.Fatalf("expected 2 merged errors (1 xsd + 1 kept unknown), got %d: %v", len(merged), merged)
	}
	// libxml2 entry preserved.
	if !strings.Contains(merged[0].Message, "{http") {
		t.Errorf("xsd entry was dropped or mangled: %v", merged[0])
	}
	// kept unknown is the empty-lyne one.
	if !strings.Contains(merged[1].Message, "empty-lyne") {
		t.Errorf("wrong unknown kept: %v", merged[1])
	}
}

func tagsFromUnknownErrs(errs []ValidationError) []string {
	out := make([]string, 0, len(errs))
	for _, e := range errs {
		m := unknownMsgTagRE.FindStringSubmatch(e.Message)
		if len(m) >= 2 {
			out = append(out, m[1])
		}
	}
	return out
}

