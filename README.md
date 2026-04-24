# fbe-go

A Go + Wails port of the classic [FictionBook Editor (FBE)](https://github.com/evpobr/fictionbookeditor) — a desktop editor for FB2 (FictionBook 2.x) documents.

Original FBE is Windows-only (C++/WTL + embedded MSHTML + MSXML). This project re-implements the core in pure Go and moves the editor surface from MSHTML `contentEditable` to a web-based editor (ProseMirror) hosted in a system webview via Wails v2.

**Target platforms: macOS + Linux.** Windows is out of scope — the original C++ FBE remains the Windows story. Platform-native components (thumbnailer, QuickLook) may use Rust or C where Go is awkward.

## Project status

**Feature-complete for 1.0.** Phases 0–5 of the roadmap are closed:

- **Editing:** every FB2 structural operation (clone / merge / insert
  cite / poem / table / section / epigraph / annotation / empty-line …),
  inline marks, paragraph styles, and save/validate cycle — worked
  through `docs/OPERATIONS.md` row-by-row.
- **Round-trip fidelity:** unknown FB2 elements survive the parse →
  PM → serialize loop unchanged (see the "Lossless fallback invariant"
  section of `CLAUDE.md`). Corpus-tested on real-world files with
  `fidelityBroken == 0` as the gating invariant.
- **Description form:** full metadata editor — title-info, src-title-info,
  publish-info, document-info, plus a ProseMirror-in-a-dialog rich
  annotation editor with the same marks as the body editor.
- **Binary manager:** upload / rename / delete / cover-badge, with
  inline `<image>` rendering in the editor body.
- **Search / Replace:** `Cmd-F` / `Cmd-H` inline bar with regex,
  case-sensitivity, whole-word (Unicode-aware), and follow-active-match
  scrolling.
- **Configurable hotkeys:** Settings → Shortcuts tab, per-action
  keystroke capture, conflict detection, reset-to-defaults. Bindings
  are stored in the standard OS config file and migrate forward
  automatically when new actions are added.
- **Platform polish:** code-signed + notarized macOS universal DMG
  (arm64 + x86_64); Linux x86_64 AppImage with `.desktop`, GNOME
  thumbnailer, and shared-MIME registration; native-webview spellcheck
  per document `<lang>`.
- **Auto-update notify:** in-app banner surfaces newer GitHub
  Releases; one-click opens the Release page in the OS browser.
- **HTML export:** Go text/template renderer (`internal/fb2/export/html`).
- **XSD validation:** read-only XML-source panel with clickable
  line-highlighted errors; libxml2 via `-tags xsd`.

Not shipping in 1.0 — each documented with rationale in
`docs/PHASES.md`:

- **Windows** — explicitly out of scope. The C++ FBE remains the
  Windows story.
- **Scripts compatibility** (FBE's `.js` macro surface) — deferred
  post-1.0. Hundreds of user-authored macros make this a
  separate-project-scale effort; revisit on concrete user demand.
- **Hunspell CGo speller** — native webview spellcheck handles
  dictionaries on both platforms; CGo path is stubbed behind
  `-tags speller_hunspell` for a future opt-in build.
- **QuickLook `.appex` preview extension** — deferred pending
  hardware refresh.
- **Linux arm64** — deferred; GitHub's hosted runners are x86_64-only.

See `PROGRESS.md` for the per-revision development log, `CHANGELOG.md`
for the user-facing release history, `docs/PHASES.md` for the roadmap,
and `docs/OPERATIONS.md` for the full list of FB2 operations and their
ProseMirror equivalents.

## Prerequisites

- Go 1.25+
- Node 20+ (for the frontend)
- [Wails v2 CLI](https://wails.io/docs/gettingstarted/installation): `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- **macOS:** Xcode Command Line Tools (`xcode-select --install`)
- **Linux:** `libwebkit2gtk-4.1-dev`, `libgtk-3-dev`
- For XSD validation (`-tags xsd`): `libxml2` (macOS: bundled in CLT; Linux: `libxml2-dev`)
- For spellcheck: `hunspell` + dictionaries (CGo path, future)

### Nix / NixOS

A `flake.nix` provides a cross-platform dev shell (Linux + macOS) with Go 1.25, Node 22, and all native deps wired up. Wails CLI is auto-installed into `$GOPATH/bin` on first entry:

```sh
nix develop                          # enter shell
wails build -tags 'xsd webkit2_41'   # or: wails dev -tags webkit2_41
```

`webkit2_41` selects the `webkit2gtk-4.1` ABI (default is still `4.0`, not in modern nixpkgs). The tag is a no-op on macOS.

Works on `x86_64-linux`, `aarch64-linux`, `x86_64-darwin`, `aarch64-darwin`.

## Layout

```
cmd/fbe/            — CLI (replaces FBV validator and covers batch ops)
internal/fb2/       — core library (parse/write/validate/zip/binary/thumb/search)
frontend/           — TypeScript + Svelte + ProseMirror editor surface
docs/               — architecture, operations catalog, roadmap
testdata/           — sample .fb2 files
build/              — Wails build artifacts (gitignored)
```

## Build

```sh
# CLI (works standalone — validate, repack, extract thumbnail)
go build -o build/fbe ./cmd/fbe

# Desktop app (requires Wails CLI)
wails dev      # hot-reload dev mode
wails build    # production bundle
```

## Docs

- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) — module map, how pieces fit together
- [`docs/OPERATIONS.md`](docs/OPERATIONS.md) — every FB2 editing operation from original FBE + its ProseMirror equivalent
- [`docs/PHASES.md`](docs/PHASES.md) — implementation phases and estimates

## License

Released under the **MIT License** — see [LICENSE](LICENSE) for the full
text. Third-party components bundled or depended on are listed with their
own licenses and attribution notices in [NOTICE.md](NOTICE.md).

## Legacy & acknowledgements

fbe-go is an **independent rewrite**, not a code-level port. The editor
surface moved from MSHTML `contentEditable` to a ProseMirror view, the
XML layer from MSXML to Go's `encoding/xml` + libxml2 for validation,
and the host from C++/WTL to Go + Wails. No source from the original
project was reused.

Thanks to:

- **Dmitry Gribov** — the [FictionBook 2.0 specification and XSD
  schemas](internal/fb2/xsd/) (2004, BSD). Those schemas ship inside
  every fbe-go binary and are the ground truth the validator checks
  against.
- **[evpobr](https://github.com/evpobr/fictionbookeditor)** and the
  classic FBE maintainers — their Windows-only FBE defined the
  operations catalog (clone / merge / insert cite / poem / table / …)
  that `docs/OPERATIONS.md` cross-references. Their UX is why fbe-go
  has the shape it has.
- **[Wails v2](https://github.com/wailsapp/wails)** (Lea Anthony et al.)
  — Go desktop framework, why this app can ship macOS+Linux from one
  codebase.
- **[ProseMirror](https://prosemirror.net/)** (Marijn Haverbeke et al.)
  — the editor framework; FB2's mixed-content model fits its schema
  system almost perfectly.
- **[libxml2](https://gitlab.gnome.org/GNOME/libxml2)** (Daniel Veillard)
  and **[lestrrat-go/libxml2](https://github.com/lestrrat-go/libxml2)**
  (Daisuke Maki) — XSD validation we rely on in `-tags xsd` builds.
- Every upstream listed in [NOTICE.md](NOTICE.md).
