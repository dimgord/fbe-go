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
	Hotkeys           map[string]string `json:"hotkeys"` // action -> accelerator (e.g. "InsertPoem": "Ctrl+Shift+P")
	RecentFiles       []string          `json:"recentFiles"`
	WordsList         []WordsEntry      `json:"wordsList"`
	FastMode          bool              `json:"fastMode"`
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
