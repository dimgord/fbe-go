# fbe-go

A Go + Wails port of the classic [FictionBook Editor (FBE)](https://github.com/evpobr/fictionbookeditor) — a desktop editor for FB2 (FictionBook 2.x) documents.

Original FBE is Windows-only (C++/WTL + embedded MSHTML + MSXML). This project re-implements the core in pure Go and moves the editor surface from MSHTML `contentEditable` to a web-based editor (ProseMirror) hosted in a system webview via Wails v2.

**Target platforms: macOS + Linux.** Windows is out of scope — the original C++ FBE remains the Windows story. Platform-native components (thumbnailer, QuickLook) may use Rust or C where Go is awkward.

## Project status

**Phase 3 editor MVP complete, Phase 4 polish in progress.** End-to-end flow works: open → edit → save → validate → export. All structural commands (clone / merge / insert cite / poem / table / …), inline marks, save cycle, description form with rich annotation editor, HTML export, paste cleanup, and native-webview spellcheck are wired. See `PROGRESS.md` for the per-revision log, `docs/PHASES.md` for the roadmap, and `docs/OPERATIONS.md` for the full list of FB2 operations.

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

TBD — original FBE was released open-source in 2010.
