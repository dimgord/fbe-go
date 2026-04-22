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
// Kept in a dedicated struct so we can grow it (outline width, split
// columns, etc.) without further field sprawl on Settings.
type PaneSizes struct {
	// ValidationErrorsHeight — pixels from the bottom of the validation
	// panel given to the errors list. 0 = "use CSS default (45%)".
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
	return &s, nil
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
		Hotkeys:           defaultHotkeys(),
	}
}

// defaultHotkeys mirrors FBE/Hotkeys.xml defaults (see FBE/Settings.cpp).
func defaultHotkeys() map[string]string {
	return map[string]string{
		"InsertPoem":       "Ctrl+Shift+P",
		"InsertCite":       "Ctrl+Shift+Q",
		"InsertTable":      "Ctrl+Shift+T",
		"AddSectionImage":  "Ctrl+I",
		"AddEpigraph":      "Ctrl+Shift+E",
		"AddAnnotation":    "Ctrl+Shift+A",
		"AddTextAuthor":    "Ctrl+Shift+U",
		"StyleSubtitle":    "Ctrl+Shift+S",
		"StyleTextAuthor":  "Ctrl+Shift+X",
		"StyleCode":        "Ctrl+Shift+K",
	}
}
