//go:build xsd

package xsd

import (
	"os"
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
			t.Logf("schema error: %s", e.Message)
		}
		t.Fatalf("blank.fb2 should be valid, got %d errors", len(errs))
	}
}

func TestLocateElementInSource(t *testing.T) {
	const src = `<?xml version="1.0"?>
<FictionBook xmlns="http://x">
  <description>
    <title-info>
      <book-title>hello</book-title>
    </title-info>
  </description>
</FictionBook>
`
	cases := []struct {
		name     string
		msg      string
		wantLine int
		wantCol  int
	}{
		{
			name:     "namespaced element",
			msg:      "Element '{http://www.gribuser.ru/xml/fictionbook/2.0}book-title': This element is not expected.",
			wantLine: 5, wantCol: 7,
		},
		{
			name:     "bare element name",
			msg:      "Element 'description': Missing child element(s).",
			wantLine: 3, wantCol: 3,
		},
		{
			name:     "unrelated message → fallback",
			msg:      "Schema validation context could not be created.",
			wantLine: 0, wantCol: 0,
		},
		{
			name:     "element not present in source → fallback",
			msg:      "Element 'nosuch': ...",
			wantLine: 0, wantCol: 0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			line, col := locateElementInSource([]byte(src), tc.msg)
			if line != tc.wantLine || col != tc.wantCol {
				t.Errorf("got (%d,%d), want (%d,%d)", line, col, tc.wantLine, tc.wantCol)
			}
		})
	}
}
