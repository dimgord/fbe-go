// Package zipfb2 packs and unpacks .fb2.zip files — a common distribution format
// where a single .fb2 file is zipped.
package zipfb2

import (
	"archive/zip"
	"fmt"
	"io"
	"strings"
)

// Unpack returns a reader for the single .fb2 entry inside a .fb2.zip.
// If the archive contains multiple .fb2 entries, the first one is returned.
// The caller must close the returned reader.
func Unpack(zr *zip.Reader) (io.ReadCloser, error) {
	for _, f := range zr.File {
		if strings.HasSuffix(strings.ToLower(f.Name), ".fb2") {
			return f.Open()
		}
	}
	return nil, fmt.Errorf("zipfb2: no .fb2 entry in archive")
}

// Pack writes fb2Data as a single entry named entryName inside a new zip stream on w.
func Pack(w io.Writer, entryName string, fb2Data io.Reader) error {
	zw := zip.NewWriter(w)
	defer zw.Close()

	fw, err := zw.Create(entryName)
	if err != nil {
		return err
	}
	if _, err := io.Copy(fw, fb2Data); err != nil {
		return err
	}
	return nil
}
