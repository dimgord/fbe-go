package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const rich = "../../testdata/rich.fb2"

func runCLI(args ...string) (code int, out, errOut string) {
	var o, e bytes.Buffer
	code = run(args, &o, &e)
	return code, o.String(), e.String()
}

func TestUsageAndUnknownCommand(t *testing.T) {
	if code, _, e := runCLI(); code != 2 || !strings.Contains(e, "Usage:") {
		t.Errorf("no args: code %d, stderr %q", code, e)
	}
	if code, _, e := runCLI("frobnicate"); code != 2 || !strings.Contains(e, "unknown command") {
		t.Errorf("unknown: code %d, stderr %q", code, e)
	}
}

func TestValidate(t *testing.T) {
	if code, out, _ := runCLI("validate", rich); code != 0 || !strings.Contains(out, "VALID") {
		t.Fatalf("valid book: code %d, %q", code, out)
	}
	src, err := os.ReadFile(rich)
	if err != nil {
		t.Fatal(err)
	}
	bad := filepath.Join(t.TempDir(), "bad.fb2")
	broken := strings.Replace(string(src), "<book-title>", "<bogus/><book-title>", 1)
	if err := os.WriteFile(bad, []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, _ := runCLI("validate", bad)
	if code != 1 || !strings.Contains(out, "INVALID") || !strings.Contains(out, "bogus") {
		t.Fatalf("invalid book: code %d, %q", code, out)
	}
}

// pack, then read through the archive: every path-taking command accepts .fb2.zip.
func TestPackThenInfoThroughZip(t *testing.T) {
	zipped := filepath.Join(t.TempDir(), "book.fb2.zip")
	if code, _, e := runCLI("pack", rich, zipped); code != 0 {
		t.Fatalf("pack: %q", e)
	}
	code, out, e := runCLI("info", zipped)
	if code != 0 || !strings.Contains(out, "TitleInfo") {
		t.Fatalf("info: code %d, %q, %q", code, out, e)
	}
}

func TestMissingFile(t *testing.T) {
	if code, _, e := runCLI("info", "/no/such/file.fb2"); code != 1 || !strings.Contains(e, "error:") {
		t.Fatalf("code %d, stderr %q", code, e)
	}
}
