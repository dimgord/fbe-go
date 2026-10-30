// Package thumb extracts the cover image from an FB2 document.
//
// Replaces FBShell's ThumbnailHandler (Windows shell extension) for cross-platform
// thumbnail generation. On Windows, the C++ FBShell can remain as-is; on macOS/Linux
// a separate native thumbnailer can call this package via the fbe CLI.
package thumb

import (
	"fmt"

	"github.com/dimgord/fbe-go/internal/fb2/binary"
	"github.com/dimgord/fbe-go/internal/fb2/doc"
)

// Cover returns the binary entry of the first coverpage image.
func Cover(fb *doc.FictionBook) (*doc.Binary, error) {
	if fb.Description.TitleInfo == nil {
		return nil, fmt.Errorf("thumb: no title-info")
	}
	cp := fb.Description.TitleInfo.Coverpage
	if cp == nil || len(cp.Images) == 0 {
		return nil, fmt.Errorf("thumb: no coverpage")
	}
	return binary.FindByHref(fb, cp.Images[0].Href)
}

// Extract returns the raw bytes of the coverpage image and its content type.
func Extract(fb *doc.FictionBook) (data []byte, contentType string, err error) {
	bin, err := Cover(fb)
	if err != nil {
		return nil, "", err
	}
	raw, err := binary.Decode(bin)
	if err != nil {
		return nil, "", fmt.Errorf("thumb: decode: %w", err)
	}
	return raw, bin.ContentType, nil
}
