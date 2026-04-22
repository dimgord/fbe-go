//go:build xsd

// libxml2-backed validator. Requires libxml-2.0 at build time (pkg-config --libs libxml-2.0).
//
//   macOS:  available via Command Line Tools or `brew install libxml2`
//   Linux:  apt install libxml2-dev / pacman -S libxml2 / dnf install libxml2-devel
//   Windows: msys2 `pacman -S mingw-w64-x86_64-libxml2`
package xsd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
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
// success; on failure, one ValidationError per schema violation.
//
// Line/column population is best-effort. lestrrat-go/libxml2 registers a plain
// `xmlSchemaValidityErrorFunc` that only forwards the message string —
// libxml2's native `xmlErrorPtr` (with real line/int2 fields) is discarded
// before we see it. To compensate, we parse the QName out of the message
// ("Element '{ns}name': ...") and scan the input bytes for the first
// occurrence of `<name`. This works for the common "element not expected" /
// "missing element" cases; messages without a quoted element name fall back
// to Line:0, Column:0.
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
			raw := sve.Errors()
			out := make([]ValidationError, 0, len(raw))
			for _, e := range raw {
				msg := e.Error()
				line, col := locateElementInSource(data, msg)
				out = append(out, ValidationError{Line: line, Column: col, Message: msg})
			}
			return out, nil
		}
		return []ValidationError{{Message: err.Error()}}, nil
	}
	return nil, nil
}

// elementInMessageRE captures the local name of the element libxml2 mentions
// first in a schema-violation message. Typical shapes:
//
//	Element '{http://www.gribuser.ru/xml/fictionbook/2.0}book-title': This element is not expected.
//	Element '{http://www.gribuser.ru/xml/fictionbook/2.0}description': Missing child element(s). Expected is one of ( ... ).
//	Element 'book-title': ...   (namespace-unqualified variant)
var elementInMessageRE = regexp.MustCompile(`Element '(?:\{[^}]*\})?([^']+)'`)

// locateElementInSource returns a 1-based (line, column) pointing at the first
// `<name` occurrence in src, where `name` is parsed out of msg. Returns (0, 0)
// if no element name can be extracted or no match is found.
func locateElementInSource(src []byte, msg string) (int, int) {
	m := elementInMessageRE.FindStringSubmatch(msg)
	if len(m) < 2 {
		return 0, 0
	}
	name := m[1]
	// Require the character after `<name` to be a name-terminator so that
	// `<p` does not accidentally match inside `<publish-info>`.
	pat := regexp.MustCompile(`<` + regexp.QuoteMeta(name) + `[\s/>]`)
	loc := pat.FindIndex(src)
	if loc == nil {
		return 0, 0
	}
	// Convert byte offset → (line, column).
	before := src[:loc[0]]
	line := bytes.Count(before, []byte{'\n'}) + 1
	lastNL := bytes.LastIndexByte(before, '\n')
	col := loc[0] - (lastNL + 1) + 1 // 1-based, counting the `<` itself
	return line, col
}
