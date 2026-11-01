// Package debuglog keeps a small rolling record of what the editor did, for reports of the
// "Apply did nothing" kind that cannot be reproduced later. Events carry counts, never book text.
package debuglog

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// maxSize is where the log rolls over to path + ".1", so at most twice this stays on disk.
const maxSize = 1 << 20

// Log appends lines to one file. A failed write is dropped: a log must never break the editor.
type Log struct {
	path string
	mu   sync.Mutex
}

// New returns a log at path. Nothing touches the disk until the first Write.
func New(path string) *Log { return &Log{path: path} }

// UserLog returns the log next to config.json, or nil when there is no config directory.
func UserLog() *Log {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil
	}
	return New(filepath.Join(dir, "fbe", "debug.log"))
}

// Path is where the log lives; empty for a nil log.
func (l *Log) Path() string {
	if l == nil {
		return ""
	}
	return l.path
}

// Write appends one timestamped line.
func (l *Log) Write(line string) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if info, err := os.Stat(l.path); err == nil && info.Size() >= maxSize {
		_ = os.Rename(l.path, l.path+".1")
	}
	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return
	}
	f, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s %s\n", time.Now().Format("2006-01-02 15:04:05.000"), line)
}
