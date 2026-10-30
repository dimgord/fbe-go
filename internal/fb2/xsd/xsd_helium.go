// Pure-Go validator on github.com/lestrrat-go/helium — no CGo, no libxml2 on the build host.
package xsd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sync"

	"github.com/lestrrat-go/helium"
	hxsd "github.com/lestrrat-go/helium/xsd"
)

var (
	schemaOnce sync.Once
	schema     *hxsd.Schema
	schemaErr  error
)

// bootstrapSchema compiles the embedded FictionBook.xsd; its xs:imports resolve against SchemaFiles.
func bootstrapSchema() {
	ctx := context.Background()
	src, err := SchemaFiles.ReadFile("FictionBook.xsd")
	if err != nil {
		schemaErr = fmt.Errorf("embedded FictionBook.xsd: %w", err)
		return
	}
	doc, err := helium.NewParser().Parse(ctx, src)
	if err != nil {
		schemaErr = fmt.Errorf("parse schema: %w", err)
		return
	}
	s, err := hxsd.NewCompiler().Label("FictionBook.xsd").BaseDir(".").FS(SchemaFiles).Compile(ctx, doc)
	if err != nil {
		schemaErr = fmt.Errorf("compile schema: %w", err)
		return
	}
	schema = s
}

// collector gathers every validation error helium reports (it calls Handle synchronously).
type collector struct{ errs []error }

func (c *collector) Handle(_ context.Context, err error) { c.errs = append(c.errs, err) }

// Validate checks r against the bundled FictionBook.xsd. Returns a nil slice on
// success; on failure, one ValidationError per schema violation.
func Validate(r io.Reader) ([]ValidationError, error) {
	schemaOnce.Do(bootstrapSchema)
	if schemaErr != nil {
		return nil, schemaErr
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	docu, err := helium.NewParser().Parse(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("parse xml: %w", err)
	}

	var c collector
	verr := hxsd.NewValidator(schema).Label("document").ErrorHandler(&c).Validate(ctx, docu)
	if verr == nil {
		return nil, nil
	}
	out := make([]ValidationError, 0, len(c.errs))
	for _, e := range c.errs {
		var ve *hxsd.ValidationError
		if !errors.As(e, &ve) {
			out = append(out, ValidationError{Message: e.Error()})
			continue
		}
		msg := fmt.Sprintf("Element '%s': %s", ve.Element, ve.Message)
		if ve.AttributeName != "" {
			msg = fmt.Sprintf("Element '%s', attribute '%s': %s", ve.Element, ve.AttributeName, ve.Message)
		}
		line, col := positionOf(data, ve.Line, ve.Element)
		out = append(out, ValidationError{Line: line, Column: col, Message: msg})
	}
	if len(out) == 0 {
		out = append(out, ValidationError{Message: verr.Error()})
	}
	return out, nil
}

// positionOf returns the 1-based (line, column) of the element's `<name` tag on the reported line
// (helium reports lines only). Falls back to (line, 1); (0, 0) when the line is unknown.
func positionOf(src []byte, line int, element string) (int, int) {
	if line <= 0 {
		return 0, 0
	}
	start := 0
	for l := 1; l < line; l++ {
		i := indexByteFrom(src, start, '\n')
		if i < 0 {
			return line, 1
		}
		start = i + 1
	}
	end := indexByteFrom(src, start, '\n')
	if end < 0 {
		end = len(src)
	}
	name := element
	if m := localNameRE.FindStringSubmatch(element); m != nil {
		name = m[1]
	}
	pat := regexp.MustCompile(`<(?:[\w.-]+:)?` + regexp.QuoteMeta(name) + `[\s/>]`)
	loc := pat.FindIndex(src[start:end])
	if loc == nil {
		return line, 1
	}
	return line, loc[0] + 1
}

// localNameRE strips a Clark-notation namespace: "{ns}book-title" → "book-title".
var localNameRE = regexp.MustCompile(`^(?:\{[^}]*\})?(.+)$`)

func indexByteFrom(b []byte, from int, c byte) int {
	for i := from; i < len(b); i++ {
		if b[i] == c {
			return i
		}
	}
	return -1
}
