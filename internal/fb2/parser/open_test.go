package parser

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func writeZip(t *testing.T, path string, data []byte) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	w, err := zw.Create("book.fb2")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
}

// Content decides, not the name: a renamed archive still opens.
func TestParseFilePlainAndZipped(t *testing.T) {
	data, err := os.ReadFile("../../../testdata/rich.fb2")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	plain := filepath.Join(dir, "plain.fb2")
	if err := os.WriteFile(plain, data, 0o644); err != nil {
		t.Fatal(err)
	}
	zipped := filepath.Join(dir, "book.fb2.zip")
	renamed := filepath.Join(dir, "renamed.fb2")
	writeZip(t, zipped, data)
	writeZip(t, renamed, data)
	for _, p := range []string{plain, zipped, renamed} {
		fb, err := ParseFile(p)
		if err != nil {
			t.Fatalf("%s: %v", filepath.Base(p), err)
		}
		if len(fb.Bodies) == 0 {
			t.Errorf("%s: no bodies", filepath.Base(p))
		}
	}
}

func TestParseFileMissing(t *testing.T) {
	if _, err := ParseFile(filepath.Join(t.TempDir(), "nope.fb2")); err == nil {
		t.Fatal("want an error")
	}
}
