// Package binary helps work with FB2 <binary> entries — base64-encoded blobs
// (usually images) referenced by xlink:href="#id".
package binary

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/dimgord/fbe-go/internal/fb2/doc"
)

// Decode returns the raw bytes of a binary entry.
func Decode(b *doc.Binary) ([]byte, error) {
	return base64.StdEncoding.DecodeString(strings.TrimSpace(b.Data))
}

// Encode creates a new Binary entry from raw bytes.
func Encode(id, contentType string, data []byte) *doc.Binary {
	return &doc.Binary{
		ID:          id,
		ContentType: contentType,
		Data:        base64.StdEncoding.EncodeToString(data),
	}
}

// FindByHref looks up a binary by href like "#cover.jpg" or "cover.jpg".
func FindByHref(fb *doc.FictionBook, href string) (*doc.Binary, error) {
	id := strings.TrimPrefix(href, "#")
	for i := range fb.Binaries {
		if fb.Binaries[i].ID == id {
			return &fb.Binaries[i], nil
		}
	}
	return nil, fmt.Errorf("binary %q not found", id)
}
