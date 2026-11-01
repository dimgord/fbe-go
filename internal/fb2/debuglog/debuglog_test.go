package debuglog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteAppendsAndCreatesDir(t *testing.T) {
	p := filepath.Join(t.TempDir(), "sub", "debug.log")
	l := New(p)
	l.Write("one")
	l.Write("two")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(b), "\n"); got != 2 || !strings.HasSuffix(string(b), " two\n") {
		t.Fatalf("log = %q", b)
	}
}

func TestRollsOverAtMaxSize(t *testing.T) {
	p := filepath.Join(t.TempDir(), "debug.log")
	if err := os.WriteFile(p, make([]byte, maxSize), 0o644); err != nil {
		t.Fatal(err)
	}
	New(p).Write("fresh")
	old, err := os.Stat(p + ".1")
	if err != nil || old.Size() != maxSize {
		t.Fatalf("rolled file: %v %v", old, err)
	}
	if info, _ := os.Stat(p); info.Size() >= maxSize {
		t.Fatalf("new file not fresh: %d", info.Size())
	}
}

func TestNilLogIsSafe(t *testing.T) {
	var l *Log
	l.Write("x")
	if l.Path() != "" {
		t.Fatal("nil log has a path")
	}
}
