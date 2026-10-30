package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLaunchArg(t *testing.T) {
	dir := t.TempDir()
	book := filepath.Join(dir, "book.fb2")
	if err := os.WriteFile(book, []byte("<FictionBook/>"), 0o644); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		args []string
		want string
	}{
		{nil, ""},
		{[]string{"-psn_0_12345"}, ""}, // old macOS Finder process serial
		{[]string{filepath.Join(dir, "missing.fb2")}, ""}, // not on disk
		{[]string{dir}, ""},                          // a directory
		{[]string{"-flag", book, "other.fb2"}, book}, // first existing file wins
	}
	for _, c := range cases {
		if got := launchArg(c.args); got != c.want {
			t.Errorf("launchArg(%q) = %q, want %q", c.args, got, c.want)
		}
	}
}

// Before the frontend is ready a request is queued, and TakeLaunchFile hands it over exactly once.
func TestRequestOpenQueuesUntilFrontendReady(t *testing.T) {
	a := NewApp()
	a.RequestOpen("/books/a.fb2")
	a.RequestOpen("") // ignored
	if got := a.TakeLaunchFile(); got != "/books/a.fb2" {
		t.Fatalf("TakeLaunchFile = %q, want the queued path", got)
	}
	if got := a.TakeLaunchFile(); got != "" {
		t.Fatalf("second TakeLaunchFile = %q, want empty", got)
	}
}
