package settings

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
)

// MergeDefaultHotkeys must backfill missing actions without clobbering the
// user's overrides — the upgrade path depends on this invariant.
func TestMergeDefaultHotkeys(t *testing.T) {
	s := &Settings{
		Hotkeys: map[string]string{
			// User has remapped Bold and explicitly cleared Italic (empty string).
			"ToggleStrong":   "Ctrl+Alt+B",
			"ToggleEmphasis": "",
			// Unknown-action entry from a future version — must be preserved.
			"FutureAction": "Ctrl+Shift+F12",
		},
	}
	MergeDefaultHotkeys(s)

	if got := s.Hotkeys["ToggleStrong"]; got != "Ctrl+Alt+B" {
		t.Errorf("user override lost: ToggleStrong = %q, want %q", got, "Ctrl+Alt+B")
	}
	if got, ok := s.Hotkeys["ToggleEmphasis"]; !ok || got != "" {
		t.Errorf("explicit empty override lost: ToggleEmphasis = %q, ok=%v", got, ok)
	}
	if got := s.Hotkeys["FutureAction"]; got != "Ctrl+Shift+F12" {
		t.Errorf("unknown action dropped: FutureAction = %q", got)
	}
	// Missing action backfilled from defaults.
	if got := s.Hotkeys["Save"]; got != "Ctrl+S" {
		t.Errorf("default not backfilled: Save = %q, want Ctrl+S", got)
	}
	// Nil-map case.
	s2 := &Settings{}
	MergeDefaultHotkeys(s2)
	if s2.Hotkeys == nil {
		t.Fatalf("nil Hotkeys not initialized")
	}
	if got := s2.Hotkeys["Save"]; got != "Ctrl+S" {
		t.Errorf("nil-map backfill failed: Save = %q", got)
	}
}

func TestDefaultHotkeysJSONRoundTrip(t *testing.T) {
	s := Default()
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var s2 Settings
	if err := json.Unmarshal(data, &s2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(s.Hotkeys) != len(s2.Hotkeys) {
		t.Fatalf("hotkey count mismatch: %d → %d", len(s.Hotkeys), len(s2.Hotkeys))
	}
	for k, v := range s.Hotkeys {
		if s2.Hotkeys[k] != v {
			t.Errorf("hotkey %q: %q → %q", k, v, s2.Hotkeys[k])
		}
	}
}

func TestStoreUpdateAndMRU(t *testing.T) {
	st := NewStore(filepath.Join(t.TempDir(), "fbe", "config.json"))
	for i := 0; i < recentFilesCap+3; i++ {
		if err := st.Update(func(s *Settings) { s.AddRecent(fmt.Sprintf("/b%d.fb2", i)) }); err != nil {
			t.Fatal(err)
		}
	}
	if err := st.Update(func(s *Settings) { s.AddRecent("/b5.fb2") }); err != nil {
		t.Fatal(err)
	}
	s, err := st.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(s.RecentFiles) != recentFilesCap || s.RecentFiles[0] != "/b5.fb2" {
		t.Fatalf("recents: %v", s.RecentFiles)
	}
	s.RemoveRecent("/b5.fb2")
	if len(s.RecentFiles) != recentFilesCap-1 {
		t.Fatalf("recents after remove: %v", s.RecentFiles)
	}
}

// The frontend's settings draft must not roll back what Go wrote since it was loaded.
func TestReplaceKeepsGoOwnedFields(t *testing.T) {
	st := NewStore(filepath.Join(t.TempDir(), "config.json"))
	draft, err := st.Load()
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Update(func(s *Settings) { s.AddRecent("/new.fb2"); s.Window.W = 900 }); err != nil {
		t.Fatal(err)
	}
	draft.Theme = "dark"
	if err := st.Replace(draft); err != nil {
		t.Fatal(err)
	}
	got, err := st.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.Theme != "dark" || len(got.RecentFiles) != 1 || got.Window.W != 900 {
		t.Fatalf("got %+v", got)
	}
}

// Run with -race: recents and window geometry update from different goroutines.
func TestConcurrentUpdatesAllLand(t *testing.T) {
	st := NewStore(filepath.Join(t.TempDir(), "config.json"))
	var wg sync.WaitGroup
	for i := 0; i < recentFilesCap; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if err := st.Update(func(s *Settings) { s.AddRecent(fmt.Sprintf("/b%d.fb2", i)) }); err != nil {
				t.Error(err)
			}
		}(i)
	}
	wg.Wait()
	s, err := st.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(s.RecentFiles) != recentFilesCap {
		t.Fatalf("lost updates: %v", s.RecentFiles)
	}
}
