package settings

import (
	"encoding/json"
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
