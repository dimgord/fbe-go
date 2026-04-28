// Package settings persists user preferences.
//
// Original FBE stored settings in the Windows Registry (see FBE/Settings.h, CRegKey).
// This port migrates to a JSON file at:
//
//   Linux:   $XDG_CONFIG_HOME/fbe/config.json  (or ~/.config/fbe/config.json)
//   macOS:   ~/Library/Application Support/fbe/config.json
//   Windows: %APPDATA%/fbe/config.json
package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Settings captures everything originally held in FBE's Settings.h.
type Settings struct {
	InterfaceLanguage string            `json:"interfaceLanguage"` // "english" | "russian" | "ukrainian"
	Dictionaries      []string          `json:"dictionaries"`      // enabled Hunspell locales
	NBSPChar          string            `json:"nbspChar"`          // configured non-breaking space replacement
	Font              FontSettings      `json:"font"`
	Colors            ColorSettings     `json:"colors"`
	Theme             string            `json:"theme"`    // "system" (default) | "light" | "dark"
	LastView          string            `json:"lastView"` // "body" | "description"
	Window            WindowGeom        `json:"window"`
	Panes             PaneSizes         `json:"panes"`
	Hotkeys           map[string]string `json:"hotkeys"` // action -> accelerator (e.g. "InsertPoem": "Ctrl+Shift+P")
	RecentFiles       []string          `json:"recentFiles"`
	WordsList         []WordsEntry      `json:"wordsList"`
	FastMode          bool              `json:"fastMode"`
}

// WindowGeom captures the OS window's last-seen position and size so the
// app can restore it on next launch. Zero values mean "never saved" and
// the startup code should fall back to compiled-in defaults (1280x800
// at the OS-default position).
type WindowGeom struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

// PaneSizes holds user-adjusted sizes of splitter panes inside the editor.
// Kept in a dedicated struct so we can grow it without further field sprawl
// on Settings. All values are in CSS pixels; 0 means "use the CSS default".
type PaneSizes struct {
	// OutlineWidth — width of the left outline sidebar.
	OutlineWidth int `json:"outlineWidth"`
	// ValidationWidth — width of the right validation / XML-source panel
	// when it's open.
	ValidationWidth int `json:"validationWidth"`
	// ValidationErrorsHeight — pixels from the bottom of the validation
	// panel given to the errors list.
	ValidationErrorsHeight int `json:"validationErrorsHeight"`
}

// FontSettings mirrors FBE's editor-view font config.
type FontSettings struct {
	Family string `json:"family"`
	Size   int    `json:"size"`
}

// ColorSettings — editor foreground/background.
type ColorSettings struct {
	FG string `json:"fg"` // hex #RRGGBB
	BG string `json:"bg"`
}

// WordsEntry — user-maintained replacement pairs.
type WordsEntry struct {
	Word        string `json:"word"`
	Replacement string `json:"replacement"`
	Flags       int    `json:"flags"`
}

// ConfigPath returns the per-OS config file path.
func ConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "fbe", "config.json"), nil
}

// Load reads settings from disk; returns defaults if the file does not exist.
// Any missing hotkey-action keys are backfilled from DefaultHotkeys so users
// picking up a new release don't have to reset their config to see the new
// bindings.
func Load() (*Settings, error) {
	p, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return Default(), nil
	}
	if err != nil {
		return nil, err
	}
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	MergeDefaultHotkeys(&s)
	return &s, nil
}

// MergeDefaultHotkeys ensures every action in DefaultHotkeys has an entry in
// s.Hotkeys, keeping user-set overrides untouched. Called on Load so upgrades
// don't drop the user into a broken state with new commands unbound.
func MergeDefaultHotkeys(s *Settings) {
	if s.Hotkeys == nil {
		s.Hotkeys = map[string]string{}
	}
	for k, v := range DefaultHotkeys() {
		if _, ok := s.Hotkeys[k]; !ok {
			s.Hotkeys[k] = v
		}
	}
}

// Save writes settings to disk, creating the config directory as needed.
func Save(s *Settings) error {
	p, err := ConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

// Default returns a reasonable starting configuration.
func Default() *Settings {
	return &Settings{
		InterfaceLanguage: "english",
		Dictionaries:      []string{"en_US"},
		NBSPChar:          " ",
		Font:              FontSettings{Family: "Trebuchet MS", Size: 12},
		Colors:            ColorSettings{FG: "#000000", BG: "#FFFFFF"},
		Theme:             "system",
		Hotkeys:           DefaultHotkeys(),
	}
}

// DefaultHotkeys returns the full canonical map of action id → accelerator.
// Keys must stay in sync with frontend/src/settings/hotkeys.ts::HOTKEY_ACTIONS.
// Accelerator strings use the human-readable "Ctrl+Shift+X" form; the
// frontend converts to platform-specific modifiers (macOS maps Ctrl → ⌘
// via ProseMirror's `Mod-` prefix at keymap-build time).
//
// Actions listed with an empty accelerator are intentionally unbound by
// default — users can assign a key via the Settings → Shortcuts tab.
func DefaultHotkeys() map[string]string {
	return map[string]string{
		// File
		"Save":   "Ctrl+S",
		"SaveAs": "Ctrl+Shift+S",

		// Edit — search
		"Find":     "Ctrl+F",
		"Replace":  "Ctrl+H",
		"FindNext": "Ctrl+G",
		"FindPrev": "Ctrl+Shift+G",

		// Format — inline marks
		"ToggleStrong":        "Ctrl+B",
		"ToggleEmphasis":      "Ctrl+I",
		"ToggleStrikethrough": "Ctrl+Shift+D",
		"ToggleSub":           "Ctrl+,",
		"ToggleSup":           "Ctrl+.",
		"ToggleCode":          "Ctrl+Shift+C",

		// Paragraph style
		"StyleNormal":     "",
		"StyleSubtitle":   "Ctrl+Shift+U",
		"StyleTextAuthor": "",

		// Blocks
		"InsertEmptyLine":      "Ctrl+Shift+L",
		"CloneContainer":       "",
		"RemoveOuterContainer": "",
		"AddTitle":             "",
		"AddEpigraph":          "Ctrl+Shift+E",
		"AddAnnotation":        "Ctrl+Shift+A",
		"AddTextAuthor":        "",
		"InsertCite":           "Ctrl+Shift+Q",
		"InsertPoem":           "Ctrl+Shift+P",
		"InsertTable":          "Ctrl+Shift+T",
		"MergeContainers":      "Ctrl+Shift+M",

		// Dialogs
		"OpenBinaries": "",
		"OpenSettings": "",
		"OpenHelp":     "",
	}
}
