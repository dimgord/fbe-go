// Package thumb extracts the cover image from an FB2 document.
//
// Replaces FBShell's ThumbnailHandler (Windows shell extension) for cross-platform
// thumbnail generation. On Windows, the C++ FBShell can remain as-is; on macOS/Linux
// a separate native thumbnailer can call this package via the fbe CLI.
package thumb

import (
	"bytes"
	"fmt"

	"github.com/dimgord/fbe-go/internal/fb2/binary"
	"github.com/dimgord/fbe-go/internal/fb2/doc"
)

// Extract returns the raw bytes of the coverpage image, its content type, or an
// error if no cover is present / referenced binary is missing.
func Extract(fb *doc.FictionBook) (data []byte, contentType string, err error) {
	cp := fb.Description.TitleInfo.Coverpage
	if cp == nil || len(cp.Images) == 0 {
		return nil, "", fmt.Errorf("thumb: no coverpage")
	}
	bin, err := binary.FindByHref(fb, cp.Images[0].Href)
	if err != nil {
		return nil, "", err
	}
	raw, err := binary.Decode(bin)
	if err != nil {
		return nil, "", fmt.Errorf("thumb: decode: %w", err)
	}
	return raw, bin.ContentType, nil
}

// ExtractToReader is a convenience for callers that want an io.Reader.
func ExtractToReader(fb *doc.FictionBook) (*bytes.Reader, string, error) {
	raw, ct, err := Extract(fb)
	if err != nil {
		return nil, "", err
	}
	return bytes.NewReader(raw), ct, nil
}
