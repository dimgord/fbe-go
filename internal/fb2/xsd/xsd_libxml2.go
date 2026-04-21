//go:build xsd

// libxml2-backed validator. Requires libxml-2.0 at build time (pkg-config --libs libxml-2.0).
//
//   macOS:  available via Command Line Tools or `brew install libxml2`
//   Linux:  apt install libxml2-dev / pacman -S libxml2 / dnf install libxml2-devel
//   Windows: msys2 `pacman -S mingw-w64-x86_64-libxml2`
package xsd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/lestrrat-go/libxml2"
	libxsd "github.com/lestrrat-go/libxml2/xsd"
)

var (
	schemaOnce sync.Once
	schema     *libxsd.Schema
	schemaErr  error
)

// bootstrapSchema extracts the embedded XSDs to a per-process temp directory
// (so libxml2 can resolve <xs:include>s from disk) and parses the main schema.
func bootstrapSchema() {
	dir, err := os.MkdirTemp("", "fbe-xsd-*")
	if err != nil {
		schemaErr = err
		return
	}
	for _, name := range SchemaFileNames() {
		data, err := SchemaFiles.ReadFile(name)
		if err != nil {
			schemaErr = fmt.Errorf("embedded %s: %w", name, err)
			return
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o600); err != nil {
			schemaErr = err
			return
		}
	}
	s, err := libxsd.ParseFromFile(filepath.Join(dir, "FictionBook.xsd"))
	if err != nil {
		schemaErr = fmt.Errorf("parse schema: %w", err)
		return
	}
	schema = s
}

// Validate checks r against the bundled FictionBook.xsd. Returns a nil slice on
// success; on failure, one ValidationError per schema violation. The message
// from libxml2 usually includes line/column information as plain text.
func Validate(r io.Reader) ([]ValidationError, error) {
	schemaOnce.Do(bootstrapSchema)
	if schemaErr != nil {
		return nil, schemaErr
	}

	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	docu, err := libxml2.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("parse xml: %w", err)
	}
	defer docu.Free()

	if err := schema.Validate(docu); err != nil {
		var sve libxsd.SchemaValidationError
		if errors.As(err, &sve) {
			out := make([]ValidationError, 0, len(sve.Errors()))
			for _, e := range sve.Errors() {
				out = append(out, ValidationError{Message: e.Error()})
			}
			return out, nil
		}
		return []ValidationError{{Message: err.Error()}}, nil
	}
	return nil, nil
}
