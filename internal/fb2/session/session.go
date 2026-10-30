// Package session owns the document the app has open.
//
// Wails runs every bound method on its own goroutine, so Session guards the current document and
// path with a mutex. A stored *doc.FictionBook is never mutated: edits swap in a new value. That
// lets readers serialize a snapshot outside the lock, where a slow save cannot block the UI.
package session

import (
	"errors"
	"io"
	"sync"

	"github.com/dimgord/fbe-go/internal/fb2/doc"
	"github.com/dimgord/fbe-go/internal/fb2/fsutil"
	"github.com/dimgord/fbe-go/internal/fb2/parser"
	"github.com/dimgord/fbe-go/internal/fb2/writer"
)

// ErrNoDocument is returned when an operation needs an open document.
var ErrNoDocument = errors.New("no document open")

// Session is the open document and where it lives. The zero value has no document.
type Session struct {
	mu   sync.Mutex
	fb   *doc.FictionBook
	path string
}

// Document returns the current document, or nil.
func (s *Session) Document() *doc.FictionBook {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.fb
}

// Require returns the current document or ErrNoDocument.
func (s *Session) Require() (*doc.FictionBook, error) {
	if fb := s.Document(); fb != nil {
		return fb, nil
	}
	return nil, ErrNoDocument
}

// Open parses path and makes it the current document.
func (s *Session) Open(path string) (*doc.FictionBook, error) {
	fb, err := parser.ParseFile(path)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fb, s.path = fb, path
	return fb, nil
}

// Replace installs the frontend's edited document, cleaning it first. nil is ignored:
// no caller means "close the document", and accepting it would silently drop the book.
func (s *Session) Replace(fb *doc.FictionBook) {
	if fb == nil {
		return
	}
	fb.Normalize()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fb = fb
}

// Save writes the document to path, or to its current path when path is empty, and returns
// the path written. The session's path changes only after the write has succeeded.
func (s *Session) Save(path string) (string, error) {
	s.mu.Lock()
	fb := s.fb
	if path == "" {
		path = s.path
	}
	s.mu.Unlock()
	if fb == nil {
		return "", ErrNoDocument
	}
	if err := fsutil.WriteAtomic(path, func(w io.Writer) error { return writer.Write(w, fb) }); err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.path = path
	return path, nil
}

// AddBinary appends b to a copy of the document, so a snapshot held by a reader stays unchanged.
func (s *Session) AddBinary(b doc.Binary) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fb == nil {
		return ErrNoDocument
	}
	next := *s.fb
	next.Binaries = append(append([]doc.Binary(nil), s.fb.Binaries...), b)
	s.fb = &next
	return nil
}
