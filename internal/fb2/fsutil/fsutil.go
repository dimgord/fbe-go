// Package fsutil holds the one file-replacing primitive every writer in fbe-go goes through.
package fsutil

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// defaultPerm applies to new files; an existing target keeps its own mode.
const defaultPerm = 0o644

// WriteAtomic writes path through fn so a failure at any point leaves the previous
// file intact: fn writes a temp file beside the target, which is synced and renamed
// over it. A symlinked target stays a link, and the file it points at is replaced.
func WriteAtomic(path string, fn func(io.Writer) error) (err error) {
	target := path
	if resolved, rerr := filepath.EvalSymlinks(path); rerr == nil {
		target = resolved
	} else if !errors.Is(rerr, fs.ErrNotExist) {
		return rerr
	}
	perm := fs.FileMode(defaultPerm)
	if st, serr := os.Stat(target); serr == nil {
		perm = st.Mode().Perm()
	}

	tmp, err := os.CreateTemp(filepath.Dir(target), "."+filepath.Base(target)+".tmp-*")
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
		}
	}()
	if err = fn(tmp); err != nil {
		return err
	}
	if err = tmp.Chmod(perm); err != nil {
		return err
	}
	if err = tmp.Sync(); err != nil {
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), target)
}

// WriteFile is WriteAtomic for data already in memory.
func WriteFile(path string, data []byte) error {
	return WriteAtomic(path, func(w io.Writer) error {
		_, err := w.Write(data)
		return err
	})
}
