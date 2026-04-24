# Changelog

User-facing release history for fbe-go. For the per-revision development
log (every code change, every rev, every fix) see
[`PROGRESS.md`](PROGRESS.md). Format loosely follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions use
[SemVer](https://semver.org/).

## [1.0.0-rc1] — 2026-04-24

First release candidate for 1.0. Feature set below (see `1.0.0` entry)
plus the full signed + notarized DMG pipeline live for the first time
(see Revs 78/79 in `PROGRESS.md`). Soaking for 2–3 days of manual QA
before the final 1.0 tag; please file issues against any regression or
packaging glitch you find against this tag.

## [1.0.0] — unreleased

First stable release. Feature-complete port of FictionBook Editor to
macOS and Linux, using a Go core + Wails v2 webview + ProseMirror
editor.

### Editing

- Open / edit / save / validate / export for FB2 (FictionBook 2.x)
  documents, including `.fb2.zip` archives (auto-detected on open).
- Every structural operation from the original FBE: insert / clone /
  merge sections, poems, cites, tables, epigraphs, annotations,
  empty-lines, titles, text-authors, images (block + inline).
- Inline marks: strong, emphasis, strikethrough, subscript,
  superscript, inline code, links.
- Paragraph styles: normal, subtitle, text-author.
- Paste cleanup: HTML from Word / Google Docs / other web sources is
  filtered through an allow-list; plain-text paste preserves newlines.
- Native-webview spellcheck routed by the document's `<lang>`
  — macOS WKWebView + Linux WebKitGTK both handle dictionaries.

### Description form

- Full metadata editor covering title-info, src-title-info,
  publish-info, document-info, plus version / custom-info fields.
- Rich annotation editor (ProseMirror instance mounted in the
  dialog; same marks as the body editor).
- Cover image upload + preview with dangling-reference detection.

### Binary manager

- Upload, rename, delete `<binary>` entries (embedded images /
  cover art) via a dedicated dialog.
- Rename cascades through every image `href` in the body plus the
  cover reference — one transaction, one undo step.
- COVER badge marks the binary currently referenced as
  `Description.{Src,}TitleInfo.Coverpage.Images[0]`.
- Inline image rendering in the editor body (Rev 75+).

### Search / Replace

- Inline bar opens on `Cmd-F` (find) / `Cmd-H` (replace).
- Regex, case-sensitivity, whole-word (Unicode-aware word boundaries).
- Active-match follow-scrolling so `Cmd-G` / `F3` stays visible.
- Replace-one / replace-all respect the regex and case flags.

### Keyboard shortcuts

- Settings → **Keyboard shortcuts** tab, per-action keystroke
  capture, duplicate detection (first-match-wins dispatch with a
  UI warning), reset-to-defaults.
- Bindings persist to the OS-standard config directory
  (`~/Library/Application Support/fbe/config.json` on macOS,
  `$XDG_CONFIG_HOME/fbe/config.json` on Linux).
- New actions introduced by future versions are merged into your
  existing config automatically; your custom overrides are never
  clobbered.

### XSD validation

- Bundled FictionBook 2.0 XSD schemas (embedded in the binary) via
  libxml2 (`-tags xsd` build).
- Read-only XML-source panel with per-error line highlighting;
  click a listed error → jump to the line + scroll-highlight.
- Structure-agnostic unknown-element scan supplements libxml2's
  DFA-recovery output, so every stray tag shows up even when
  libxml2 shortens its own report.

### HTML export

- Self-contained HTML output via Go's text/template
  (`internal/fb2/export/html`). Binaries inlined as data: URLs.
  Not a CSS masterpiece — faithful to FB2 structure, styled for
  readability.

### Platform polish

- macOS: code-signed + notarized universal DMG (arm64 + x86_64).
  Drag-to-Applications install from Finder works with no
  "unidentified developer" dialog. Native file association for
  `.fb2` via UTI declaration.
- Linux: x86_64 AppImage with bundled `.desktop` file, GNOME
  thumbnailer shim (calls `fbe thumb` to show covers in Nautilus
  / GNOME Files), and shared-MIME registration so other file
  managers pick up the file type.
- Auto-update notify: banner surfaces newer GitHub Releases at
  launch (non-blocking, dismissible). No auto-install — one click
  opens the Release page in the OS browser.
- Dark mode with OS-follow option; live-updates when the system
  theme changes.
- Font family picker in Settings (filesystem + fontconfig
  discovery; works on NixOS too).

### CLI

`cmd/fbe` ships alongside the desktop app and covers
scripting / library-management workflows:

- `fbe validate <file>` — XSD validation, exit code reflects result.
- `fbe thumb <file>` — extract coverpage to stdout (for thumbnailer
  integrations).
- `fbe info <file>` — print title-info fields.
- `fbe pack <in> <out.zip>` / `fbe unpack <in.zip> <out>` — FB2.zip
  wrangling.
- `fbe export html <in> <out.html>` — same renderer as the desktop
  app's Export menu.

### Lossless round-trip invariant

fbe-go's parse → PM → serialize pipeline preserves unknown FB2
elements verbatim (`doc.Block.Raw` / `doc.Inline.Raw`), so a file
with a proprietary `<custom-thing>` lives through an open-save
round-trip unchanged. Corpus-tested: `fidelityBroken` (source
XSD-valid → our output XSD-invalid) stays at **0** on every
regression run.

### Not shipping in 1.0

- **Windows** — explicitly out of scope; the original C++ FBE
  remains the Windows story.
- **Scripts compatibility** (the FBE `.js` macro surface with
  `apiRunCmd` / `apiProcessCmd`) — deferred post-1.0.
  `docs/OPERATIONS.md` §10 keeps the design sketch.
- **Hunspell CGo speller** — stubbed; native webview spellcheck
  handles dictionaries cleanly on both platforms.
- **QuickLook `.appex` preview extension** (macOS) — deferred
  pending hardware refresh.
- **Linux arm64** — deferred; GitHub's hosted runners are x86_64
  only.
- **Sparkle / AppImageUpdate auto-install** — banner-notify is the
  1.0 story; full auto-install is planned for 1.1+ once the signed
  build pipeline has run for a few releases.

## [0.2.0-beta] — 2026-04-22

Phase 5 close. Shipped the release pipeline and platform integration:

- **macOS universal DMG + Linux AppImage** on every `v*` tag via
  GitHub Actions.
- **File associations** — macOS UTI / CFBundleDocumentTypes;
  Linux `.desktop` + shared-MIME registration + GNOME thumbnailer.
- **Settings dialog** with font picker, NBSP replacement toggle,
  pane-size reset, recent-files clear.
- **Dark mode** + palette-driven theme hygiene (centralized
  palette; CI lint rejects hard-coded colors).
- **Persistence:** window geometry, last view, pane sizes.
- **Draggable** outline sidebar + validation panel + errors pane.
- **Font discovery:** real OS font enumeration via
  `sysfont` + fontconfig `fc-list` fallback on Linux (NixOS-aware).

## [0.1.0-beta] — 2026-04-01

Phase 3 MVP. First publicly-testable build:

- End-to-end editor flow: open → edit → save → validate → export.
- All structural FB2 operations wired.
- Description form with rich annotation editor.
- HTML export.
- Paste cleanup.
- Native-webview spellcheck.
- Lossless round-trip for unknown elements.
- Nix flake for reproducible builds on macOS + Linux (x86_64 +
  arm64 dev shell).

## Earlier

Pre-0.1 work covered Phase 0 (PoC) through Phase 3 MVP. The
per-revision history (Revs 1–38) lives in
[`PROGRESS.md`](PROGRESS.md).
