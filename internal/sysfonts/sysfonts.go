// Package sysfonts lists the font families installed on this machine, for the Settings
// dialog's font picker.
package sysfonts

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/adrg/sysfont"
	"github.com/adrg/xdg"
)

// List walks the OS font registry and returns the family names, deduped and
// sorted. Runs once per app launch — fonts rarely change
// during a session, and the dialog can happily show a stale list if the
// user installs a family mid-session (they can still type it in the
// free-text fallback).
//
// Augments `xdg.FontDirs` with NixOS-typical locations (`/run/current-system/...`,
// nix user profiles) and with every `$XDG_DATA_DIRS` entry's `/fonts`
// subdir, so the Wails flake's reduced XDG_DATA_DIRS on NixOS doesn't
// leave the finder staring at an empty /usr/share/fonts. No-op on systems
// where those paths don't exist.
//
// For fonts sysfont's filename registry doesn't recognize — common on
// NixOS because the store-path filenames carry hashes / versions not in
// the registry — falls back to a filename-to-family heuristic so the
// user still sees a reasonable label.
func List() []string {
	// On Linux, fontconfig is the source of truth. `fc-list : family` returns
	// every font the user has configured — system packages, home-manager,
	// nix profiles, user ~/.fonts, the lot — indexed in a single pass, with
	// proper family names (not filenames). Filesystem walkers like sysfont
	// can't match fontconfig's visibility on NixOS: fonts live in dozens of
	// nix-store paths joined by opaque symlink trees, and nixpkgs-packaged
	// filenames carry hashes / versions that sysfont's registry doesn't
	// recognize. Try fontconfig first; fall back to sysfont on macOS (where
	// fc-list usually isn't installed) or if the command fails.
	if runtime.GOOS == "linux" {
		if families, err := listFontsViaFontconfig(); err == nil && len(families) > 0 {
			log.Printf("[fbe] system fonts: %d families via fontconfig", len(families))
			return families
		}
	}

	extendFontDirsForNix()

	finder := sysfont.NewFinder(nil)
	all := finder.List()

	seen := make(map[string]struct{}, len(all))
	families := make([]string, 0, len(all))
	recognized := 0
	heuristic := 0
	for _, f := range all {
		if f == nil {
			continue
		}
		family := f.Family
		if family != "" {
			recognized++
		} else if f.Filename != "" {
			family = familyFromFilename(f.Filename)
			if family == "" {
				continue
			}
			heuristic++
		} else {
			continue
		}
		if _, dup := seen[family]; dup {
			continue
		}
		seen[family] = struct{}{}
		families = append(families, family)
	}
	sort.Strings(families)

	log.Printf("[fbe] system fonts: %d files scanned, %d recognized, %d via filename heuristic, %d unique families (sysfont)",
		len(all), recognized, heuristic, len(families))

	return families
}

// listFontsViaFontconfig runs `fc-list : family` and parses the output. Each
// line may be a comma-separated list of family aliases (fontconfig groups
// equivalents like localized names); we take the first as the canonical
// display name. Dedupes + sorts the result. Returns an error if the command
// isn't installed or exits non-zero.
func listFontsViaFontconfig() ([]string, error) {
	path, err := exec.LookPath("fc-list")
	if err != nil {
		return nil, err
	}
	out, err := exec.Command(path, ":", "family").Output()
	if err != nil {
		return nil, err
	}
	return parseFontconfig(string(out)), nil
}

// parseFontconfig turns `fc-list : family` output into sorted, deduped family names.
func parseFontconfig(out string) []string {
	seen := make(map[string]struct{})
	families := make([]string, 0, 128)
	for _, raw := range strings.Split(out, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		// "Noto Serif,Noto Serif Regular" → "Noto Serif"
		first := strings.TrimSpace(strings.SplitN(line, ",", 2)[0])
		if first == "" {
			continue
		}
		if _, dup := seen[first]; dup {
			continue
		}
		seen[first] = struct{}{}
		families = append(families, first)
	}
	sort.Strings(families)
	return families
}

// familyFromFilename extracts a human-readable family name from a font
// file path by stripping the extension, common weight / style suffixes,
// and replacing separators with spaces. Best-effort heuristic; caller
// dedupes against already-recognized names.
//
//	/nix/store/abc-dejavu-fonts-2.37/share/fonts/truetype/DejaVuSans-Bold.ttf
//	→ "Deja Vu Sans"
func familyFromFilename(path string) string {
	base := filepath.Base(path)
	base = strings.TrimSuffix(base, filepath.Ext(base))

	// Strip well-known style tokens. Order matters — longer first so
	// "BoldItalic" matches before "Bold".
	suffixes := []string{
		"BoldItalic", "BoldOblique", "LightItalic", "LightOblique",
		"MediumItalic", "MediumOblique", "ExtraBold", "SemiBold", "Thin",
		"Light", "Medium", "Regular", "Bold", "Italic", "Oblique",
	}
	for _, s := range suffixes {
		// Match `-Bold`, `_Bold`, or ` Bold` at the end.
		for _, sep := range []string{"-", "_", " "} {
			token := sep + s
			if strings.HasSuffix(base, token) {
				base = strings.TrimSuffix(base, token)
				break
			}
		}
	}

	// Convert CamelCase / snake / kebab into space-separated words.
	base = strings.ReplaceAll(base, "_", " ")
	base = strings.ReplaceAll(base, "-", " ")
	base = splitCamelCase(base)
	base = strings.TrimSpace(strings.Join(strings.Fields(base), " "))
	return base
}

// splitCamelCase inserts spaces before uppercase letters in camelCase
// sequences: "DejaVuSans" → "Deja Vu Sans". Keeps runs of uppercase together
// ("PTSans" → "PT Sans", not "P T Sans").
func splitCamelCase(s string) string {
	var b strings.Builder
	runes := []rune(s)
	for i, r := range runes {
		if i > 0 && r >= 'A' && r <= 'Z' {
			prev := runes[i-1]
			next := rune(0)
			if i+1 < len(runes) {
				next = runes[i+1]
			}
			prevLower := prev >= 'a' && prev <= 'z'
			nextLower := next >= 'a' && next <= 'z'
			if prevLower || (prev >= 'A' && prev <= 'Z' && nextLower) {
				b.WriteRune(' ')
			}
		}
		b.WriteRune(r)
	}
	return b.String()
}

// extendFontDirsForNix appends additional font directories to xdg.FontDirs
// that sysfont would otherwise miss on NixOS and nix-darwin setups:
//
//   - `/run/current-system/sw/share/fonts` — system-wide NixOS packages.
//   - `$HOME/.nix-profile/share/fonts`     — user-installed via nix profile.
//   - `/etc/profiles/per-user/<user>/share/fonts` — home-manager style.
//   - Each entry in `$XDG_DATA_DIRS` joined with `fonts` — this picks up
//     any activation-script-managed paths a Nix dev shell might inject.
//
// Only existing directories are appended. Idempotent per process — the
// package-level xdg.FontDirs can be mutated multiple times without harm
// beyond extra duplicate checks inside sysfont's walker.
func extendFontDirsForNix() {
	seen := make(map[string]struct{}, len(xdg.FontDirs))
	for _, p := range xdg.FontDirs {
		seen[p] = struct{}{}
	}
	add := func(p string) {
		if p == "" {
			return
		}
		if _, dup := seen[p]; dup {
			return
		}
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			seen[p] = struct{}{}
			xdg.FontDirs = append(xdg.FontDirs, p)
		}
	}
	add("/run/current-system/sw/share/fonts")
	if home, err := os.UserHomeDir(); err == nil {
		add(filepath.Join(home, ".nix-profile/share/fonts"))
	}
	if xdgDataDirs := os.Getenv("XDG_DATA_DIRS"); xdgDataDirs != "" {
		for _, d := range strings.Split(xdgDataDirs, ":") {
			add(filepath.Join(d, "fonts"))
		}
	}
	log.Printf("[fbe] font dirs: %v", xdg.FontDirs)
}
