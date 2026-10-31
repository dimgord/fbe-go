package fsutil

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func read(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func onlyEntry(t *testing.T, dir string, want int) {
	t.Helper()
	es, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(es) != want {
		t.Fatalf("%d entries in dir, want %d (temp file left behind?)", len(es), want)
	}
}

func TestFailureKeepsOriginal(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "book.fb2")
	if err := os.WriteFile(p, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	boom := errors.New("boom")
	err := WriteAtomic(p, func(w io.Writer) error {
		io.WriteString(w, "half a bo")
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
	if got := read(t, p); got != "original" {
		t.Fatalf("file = %q", got)
	}
	onlyEntry(t, dir, 1)
}

func TestReplacesAndKeepsMode(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "book.fb2")
	if err := os.WriteFile(p, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(p, []byte("new")); err != nil {
		t.Fatal(err)
	}
	st, _ := os.Stat(p)
	if got := read(t, p); got != "new" || st.Mode().Perm() != 0o600 {
		t.Fatalf("file %q mode %v", got, st.Mode().Perm())
	}
	onlyEntry(t, dir, 1)
}

func TestNewFileGetsDefaultMode(t *testing.T) {
	p := filepath.Join(t.TempDir(), "new.fb2")
	if err := WriteFile(p, []byte("x")); err != nil {
		t.Fatal(err)
	}
	if st, _ := os.Stat(p); st.Mode().Perm() != defaultPerm {
		t.Fatalf("mode %v", st.Mode().Perm())
	}
}

func TestSymlinkStaysALink(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "real.fb2")
	link := filepath.Join(dir, "link.fb2")
	if err := os.WriteFile(real, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, link); err != nil {
		t.Skip(err)
	}
	if err := WriteFile(link, []byte("new")); err != nil {
		t.Fatal(err)
	}
	if st, _ := os.Lstat(link); st.Mode()&os.ModeSymlink == 0 {
		t.Fatal("link was replaced by a regular file")
	}
	if got := read(t, real); got != "new" {
		t.Fatalf("target = %q", got)
	}
}
