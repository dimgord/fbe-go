package binary

import (
	"testing"

	"github.com/dimgord/fbe-go/internal/fb2/doc"
)

func TestDataURL(t *testing.T) {
	got := DataURL(&doc.Binary{ContentType: "image/png", Data: "\n  QUJD\n"})
	if got != "data:image/png;base64,QUJD" {
		t.Errorf("got %q", got)
	}
	if got := DataURL(&doc.Binary{Data: "QUJD"}); got != "data:image/jpeg;base64,QUJD" {
		t.Errorf("no content type: %q", got)
	}
}
