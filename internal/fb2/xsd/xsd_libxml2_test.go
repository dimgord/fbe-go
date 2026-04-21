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
