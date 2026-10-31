package session

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/dimgord/fbe-go/internal/fb2/doc"
)

const rich = "../../../testdata/rich.fb2"

func TestNoDocument(t *testing.T) {
	var s Session
	if _, err := s.Require(); !errors.Is(err, ErrNoDocument) {
		t.Errorf("Require: %v", err)
	}
	if _, err := s.Save("x.fb2"); !errors.Is(err, ErrNoDocument) {
		t.Errorf("Save: %v", err)
	}
	if err := s.AddBinary(doc.Binary{}); !errors.Is(err, ErrNoDocument) {
		t.Errorf("AddBinary: %v", err)
	}
}

func TestReplaceIgnoresNil(t *testing.T) {
	var s Session
	fb, err := s.Open(rich)
	if err != nil {
		t.Fatal(err)
	}
	s.Replace(nil)
	if s.Document() != fb {
		t.Fatal("Replace(nil) dropped the document")
	}
}

func TestSaveDefaultsToCurrentPathAndTracksSaveAs(t *testing.T) {
	var s Session
	if _, err := s.Open(rich); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	first := filepath.Join(dir, "a.fb2")
	if got, err := s.Save(first); err != nil || got != first {
		t.Fatalf("Save = %q, %v", got, err)
	}
	if err := os.Remove(first); err != nil {
		t.Fatal(err)
	}
	if got, err := s.Save(""); err != nil || got != first {
		t.Fatalf("Save(\"\") = %q, %v", got, err)
	}
	if _, err := os.Stat(first); err != nil {
		t.Fatal(err)
	}
}

func TestFailedSaveKeepsPath(t *testing.T) {
	var s Session
	if _, err := s.Open(rich); err != nil {
		t.Fatal(err)
	}
	good := filepath.Join(t.TempDir(), "good.fb2")
	if _, err := s.Save(good); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Save(filepath.Join(good, "not-a-dir", "x.fb2")); err == nil {
		t.Fatal("want an error")
	}
	if got, err := s.Save(""); err != nil || got != good {
		t.Fatalf("path moved after a failed Save: %q, %v", got, err)
	}
}

func TestAddBinaryLeavesSnapshotsAlone(t *testing.T) {
	var s Session
	before, err := s.Open(rich)
	if err != nil {
		t.Fatal(err)
	}
	n := len(before.Binaries)
	if err := s.AddBinary(doc.Binary{ID: "new.png"}); err != nil {
		t.Fatal(err)
	}
	if len(before.Binaries) != n {
		t.Fatal("snapshot was mutated")
	}
	if got := s.Document().Binaries; len(got) != n+1 || got[n].ID != "new.png" {
		t.Fatalf("binaries: %+v", got)
	}
}

func TestReplaceDropsEmptyCoverpage(t *testing.T) {
	fb := &doc.FictionBook{}
	fb.Description.TitleInfo = &doc.TitleInfo{Coverpage: &doc.Coverpage{}}
	var s Session
	s.Replace(fb)
	if s.Document().Description.TitleInfo.Coverpage != nil {
		t.Fatal("empty coverpage kept")
	}
}

// Run with -race: this is what Wails does, one goroutine per bound call.
func TestConcurrentUse(t *testing.T) {
	var s Session
	fb, err := s.Open(rich)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s.Replace(fb)
			_ = s.AddBinary(doc.Binary{ID: "b"})
			if _, err := s.Save(filepath.Join(dir, strings.Repeat("x", i+1)+".fb2")); err != nil {
				t.Error(err)
			}
			_ = s.Document()
		}(i)
	}
	wg.Wait()
}
