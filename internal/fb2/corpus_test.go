//go:build corpus

// Run with:
//   FBE_CORPUS_DIR=~/Documents/books go test -tags 'corpus xsd' -v ./internal/fb2/ -run TestCorpus
//
// Gate behind -tags corpus so plain `go test ./...` stays hermetic.
package fb2_test

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/dimgord/fbe-go/internal/fb2/parser"
	"github.com/dimgord/fbe-go/internal/fb2/writer"
	"github.com/dimgord/fbe-go/internal/fb2/xsd"
)

type result struct {
	path        string
	size        int64
	parseErr    error
	writeErr    error
	reparseErr  error
	srcXSDCount int // errors in the source file
	outXSDCount int // errors in our writer's output
	srcXSDErr   error
	outXSDErr   error
	srcErrs     []xsd.ValidationError
	outErrs     []xsd.ValidationError
	firstOutErr string
	elapsed     time.Duration
}

func TestCorpus(t *testing.T) {
	dir := os.Getenv("FBE_CORPUS_DIR")
	if dir == "" {
		dir = os.ExpandEnv("$HOME/Documents/books")
	}
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("corpus dir %q not accessible: %v", dir, err)
	}

	files := discover(t, dir)
	t.Logf("corpus: %d .fb2 files under %s", len(files), dir)
	if len(files) == 0 {
		t.Skip("no .fb2 files found")
	}

	var results []result
	for _, f := range files {
		results = append(results, runOne(f))
	}

	summarize(t, results)
}

func discover(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		low := strings.ToLower(p)
		if strings.HasSuffix(low, ".fb2") {
			out = append(out, p)
		}
		return nil
	})
	sort.Strings(out)
	return out
}

func runOne(path string) result {
	r := result{path: path}
	start := time.Now()
	defer func() { r.elapsed = time.Since(start) }()

	st, err := os.Stat(path)
	if err != nil {
		r.parseErr = err
		return r
	}
	r.size = st.Size()

	raw, err := os.ReadFile(path)
	if err != nil {
		r.parseErr = err
		return r
	}

	// Validate source first.
	srcErrs, srcErr := xsd.Validate(bytes.NewReader(raw))
	r.srcXSDErr = srcErr
	r.srcXSDCount = len(srcErrs)
	r.srcErrs = srcErrs

	fb, err := parser.Parse(bytes.NewReader(raw))
	if err != nil {
		r.parseErr = err
		return r
	}

	var buf bytes.Buffer
	if err := writer.Write(&buf, fb); err != nil {
		r.writeErr = err
		return r
	}

	if _, err := parser.Parse(bytes.NewReader(buf.Bytes())); err != nil {
		r.reparseErr = err
		return r
	}

	outErrs, outErr := xsd.Validate(bytes.NewReader(buf.Bytes()))
	r.outXSDErr = outErr
	r.outXSDCount = len(outErrs)
	r.outErrs = outErrs
	if len(outErrs) > 0 {
		r.firstOutErr = outErrs[0].Message
	}
	return r
}

func summarize(t *testing.T, rs []result) {
	t.Helper()
	var (
		n              = len(rs)
		parseOK        int
		writeOK        int
		reparseOK      int
		srcValid       int
		outValid       int
		fidelityBroken int // source was valid but output is not
		fidelityKept   int // source was invalid, output has same number of errors
		totalBytes     int64
	)
	for _, r := range rs {
		totalBytes += r.size
		if r.parseErr == nil {
			parseOK++
		}
		if r.parseErr == nil && r.writeErr == nil {
			writeOK++
		}
		if r.parseErr == nil && r.writeErr == nil && r.reparseErr == nil {
			reparseOK++
		}
		if r.srcXSDErr == nil && r.srcXSDCount == 0 {
			srcValid++
		}
		if r.parseErr == nil && r.writeErr == nil && r.outXSDErr == nil && r.outXSDCount == 0 {
			outValid++
		}
		// Fidelity check: if source was valid, output must also be valid.
		if r.srcXSDErr == nil && r.srcXSDCount == 0 {
			if r.outXSDCount > 0 || r.outXSDErr != nil {
				fidelityBroken++
			}
		} else if r.srcXSDCount > 0 && r.outXSDCount == r.srcXSDCount {
			fidelityKept++
		}
	}

	t.Logf("── corpus summary ──")
	t.Logf("  files:              %d (%.1f MB total)", n, float64(totalBytes)/1024/1024)
	t.Logf("  parse OK:           %d / %d", parseOK, n)
	t.Logf("  write OK:           %d / %d", writeOK, n)
	t.Logf("  re-parse OK:        %d / %d", reparseOK, n)
	t.Logf("  source XSD-valid:   %d / %d", srcValid, n)
	t.Logf("  output XSD-valid:   %d / %d", outValid, n)
	t.Logf("  fidelity broken:    %d (source valid → our output invalid)", fidelityBroken)
	t.Logf("  fidelity preserved: %d (source already invalid, same # of errs)", fidelityKept)

	t.Logf("── per-file XSD error counts (src → out) ──")
	for _, r := range rs {
		if r.srcXSDCount == 0 && r.outXSDCount == 0 {
			continue
		}
		delta := ""
		if r.outXSDCount != r.srcXSDCount {
			delta = fmt.Sprintf("  Δ=%+d", r.outXSDCount-r.srcXSDCount)
		}
		t.Logf("  %-60s %d → %d%s", filepath.Base(r.path), r.srcXSDCount, r.outXSDCount, delta)
		if r.outXSDCount != r.srcXSDCount {
			t.Logf("    SRC errors:")
			for _, e := range r.srcErrs {
				t.Logf("      - %s", truncate(e.Message, 220))
			}
			t.Logf("    OUT errors:")
			for _, e := range r.outErrs {
				t.Logf("      - %s", truncate(e.Message, 220))
			}
		}
	}

	// Report up to 10 real failures — parse/write/reparse errors, or fidelity breakage.
	reported := 0
	for _, r := range rs {
		if reported >= 10 {
			break
		}
		if msg := failureMessage(r); msg != "" {
			short := filepath.Base(r.path)
			t.Errorf("❌ %s (%d bytes, %s): %s", short, r.size, r.elapsed, msg)
			reported++
		}
	}

	fmt.Printf("fbe-go corpus: parse=%d/%d write=%d/%d reparse=%d/%d srcValid=%d/%d outValid=%d/%d fidelityBroken=%d\n",
		parseOK, n, writeOK, n, reparseOK, n, srcValid, n, outValid, n, fidelityBroken)
}

func failureMessage(r result) string {
	switch {
	case r.parseErr != nil:
		return "parse: " + r.parseErr.Error()
	case r.writeErr != nil:
		return "write: " + r.writeErr.Error()
	case r.reparseErr != nil:
		return "reparse: " + r.reparseErr.Error()
	}
	// Fidelity break: source was valid, our output is not.
	if r.srcXSDErr == nil && r.srcXSDCount == 0 && (r.outXSDErr != nil || r.outXSDCount > 0) {
		if r.outXSDErr != nil {
			return "fidelity-broken: " + r.outXSDErr.Error()
		}
		return fmt.Sprintf("fidelity-broken: source valid, output has %d xsd errors; first: %s",
			r.outXSDCount, truncate(r.firstOutErr, 200))
	}
	return ""
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

// Ensure imports are kept even when only used in a branch above.
var _ = strings.TrimSpace
