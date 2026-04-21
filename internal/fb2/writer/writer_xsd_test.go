//go:build xsd

package writer_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/dimgord/fbe-go/internal/fb2/parser"
	"github.com/dimgord/fbe-go/internal/fb2/writer"
	"github.com/dimgord/fbe-go/internal/fb2/xsd"
)

// TestWriterOutputIsSchemaValid runs the writer output through the XSD
// validator. This catches structural regressions (missing required children,
// wrong element order) that pure round-trip comparison would miss.
func TestWriterOutputIsSchemaValid(t *testing.T) {
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

			fb, err := parser.Parse(src)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}

			var buf bytes.Buffer
			if err := writer.Write(&buf, fb); err != nil {
				t.Fatalf("write: %v", err)
			}

			errs, err := xsd.Validate(bytes.NewReader(buf.Bytes()))
			if err != nil {
				t.Fatalf("xsd.Validate: %v", err)
			}
			if len(errs) > 0 {
				t.Errorf("writer output failed XSD validation (%d errors):\n%s", len(errs), buf.String())
				for _, e := range errs {
					t.Errorf("  %s", e.Message)
				}
			}
		})
	}
}
