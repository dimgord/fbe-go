# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

`fbe-go` is a Go + Wails v2 port of the classic Windows-only FictionBook Editor (FBE), targeting **macOS + Linux only** (Windows is explicitly out of scope). It edits FB2 (FictionBook 2.x) XML documents. Status: Phase 3 editor MVP complete (all structural commands + save cycle + description form + HTML export + paste handling + native spellcheck); Phase 4 polish in progress. See `docs/PHASES.md` for the full roadmap and `PROGRESS.md` for the per-revision log.

See `docs/ARCHITECTURE.md`, `docs/OPERATIONS.md`, and `docs/PHASES.md` for deeper context before touching unfamiliar subsystems.

## Commands

```sh
# CLI — works standalone (validate, pack, thumb, info, export)
go build -o build/fbe ./cmd/fbe

# Desktop app — needs Wails CLI (go install github.com/wailsapp/wails/v2/cmd/wails@latest)
wails dev                # hot-reload dev mode
wails build -tags xsd    # production bundle with real XSD validation → build/bin/

# Go tests (hermetic)
go test ./...

# XSD validator: the Go side has two backends selected by build tag.
# Default = no-op stub. To get real validation, build/test with -tags xsd:
go build  -tags xsd ./...
go test   -tags xsd ./internal/fb2/xsd

# Corpus test — huge real-world .fb2 set, gated behind -tags corpus so it stays
# out of `go test ./...`. Requires libxml2 (-tags xsd). Point FBE_CORPUS_DIR at
# a directory of .fb2 files; default is ~/Documents/books.
FBE_CORPUS_DIR=~/Documents/books go test -tags 'corpus xsd' -v ./internal/fb2/ -run TestCorpus

# Frontend (cd frontend/)
npm run dev     # vite dev server (used by wails dev)
npm run build   # vite build → frontend/dist (embedded into Go binary via go:embed)
npm run check   # svelte-check typecheck — run this after frontend edits
npm run test    # vitest (unit tests for editor/parse, editor/serialize, tree/outline)
```

Single-test runs:

```sh
go test ./internal/fb2/xsd -run TestValidate -tags xsd
(cd frontend && npx vitest run src/editor/serialize.test.ts)
```

## Architecture

**Two-surface app sharing one Go core.** `internal/fb2/*` holds all parsing, writing, validation, zip, thumbnail, search, and export logic. Both the Wails desktop app and the `cmd/fbe` CLI consume this core — never let business logic leak into `app.go` or the frontend.

### Wails layer (`main.go`, `app.go`)

- `main.go` embeds `frontend/dist` via `go:embed` and boots Wails with the `App` struct as the single binding.
- Every public method on `App` is auto-exposed to TypeScript at `frontend/wailsjs/go/main/App`. Keep `app.go` thin — it translates frontend calls ↔ `internal/fb2` packages. No parsing, no schema logic there.
- `App` holds per-session state: `current *doc.FictionBook` and `path string`.
- **Dialogs must go through Go wrappers.** Wails v2 ships `OpenFileDialog` / `SaveFileDialog` as Go-only — they're not in the JS runtime. `app.go` exposes `PickFB2ToOpen`, `PickFB2ToSave`, `PickHTMLToSave` for the frontend.
- `OpenFile` deliberately wraps `parser.Parse` in `defer recover()` — malformed docs surface as JS errors instead of killing the webview.

### Go core (`internal/fb2/`)

Package layout — one responsibility each:

```
doc/      FictionBook struct + child types (the canonical in-memory shape)
parser/   XML → doc.FictionBook (autodetect encoding, unwrap .fb2.zip)
writer/   doc.FictionBook → canonical indented XML
xsd/      schema validation, dual-backend (see below)
zipfb2/   .fb2.zip pack/unpack
binary/   <binary> element encode/decode, FindByHref
thumb/    coverpage extractor
export/   output formats (currently export/html)
search/   text search
settings/ JSON persistence at OS-standard config dir
speller/  Hunspell (CGo, future)
```

**XSD dual-backend** (`internal/fb2/xsd/`): `xsd.go` is the shared API; `xsd_stub.go` (default) returns `ErrNotImplemented`; `xsd_libxml2.go` (behind `-tags xsd`) links against libxml2 via `github.com/lestrrat-go/libxml2`. `SchemaFiles` is an `embed.FS` of the four bundled `.xsd` files. The stub path lets `go build ./...` succeed without libxml2 on the host.

**Lossless fallback invariant:** `doc.Block` and `doc.Inline` each carry a `Raw *RawElement` field; `Block.UnmarshalXML` / `unmarshalInlineContent` route unknown elements there instead of calling `d.Skip()`. Writer round-trips `Raw` verbatim, preserving attributes and nested content. Do not reintroduce silent skips — they caused a Rev-5 regression where an `<empty-line>` in an unexpected position was dropped. When adding new typed fields, make sure the dispatcher still falls through to `Raw` for unknown elements.

### Frontend (`frontend/src/`)

- **Stack:** Svelte 4 + TypeScript + Vite + raw ProseMirror (not TipTap — see ARCHITECTURE.md §"Why ProseMirror").
- `editor/schema.ts` — the ProseMirror schema. **Key decision:** `<image>` is split into two nodes (`image_block` and `image_inline`) because FB2 allows `<image>` both as a block sibling of `<section>` and as an inline in `<p>` — a single PM node must be one or the other.
- `editor/parse.ts::fb2ToPMDoc()` — hydrates a `FictionBook` JSON from the Go side into a PM doc.
- `editor/serialize.ts::pmDocToFB2()` — serializes the PM doc back to a `FictionBook`-shaped object for `App.UpdateDocument` / `App.SaveFile`.
- `tree/outline.ts` — derives the document outline (used by `DocumentTree.svelte`) from the PM doc.
- **Description form uses a secondary ProseMirror instance.** `description/AnnotationEditor.svelte` mounts its own PM view with a derived schema (`fb2Schema.spec.nodes.update("doc", …)` restricting root content) to edit `<annotation>` rich text. Marks reuse the main schema's specs so round-trips stay clean.
- `App.svelte` holds `fb` as the canonical state. `<DescriptionPanel bind:fb />` and `Editor.currentFB()` both flow edits back into it; `Save` serializes `fb` (body edits + description edits together — no extra plumbing).
- `wailsjs/` is auto-generated by Wails from `app.go` bindings — gitignored, regenerates on `wails dev` / `wails build`. Never edit by hand.
- **Spellcheck is native webview, not Hunspell.** `Editor.svelte` sets `spellcheck="true"` + `lang={fb.TitleInfo.Lang}` on the PM view; macOS WKWebView and Linux WebKitGTK both handle dictionaries. The `internal/fb2/speller` Go package exists as a no-op stub with a roadmap for a future `-tags speller_hunspell` CGo backend — don't wire it up prematurely.

### CLI (`cmd/fbe/`)

Single `main.go` with subcommands: `validate`, `thumb`, `pack`, `unpack`, `info`, `export html`. Replaces the old FBV.exe validator and covers scripting / library-management use cases. Imports the same `internal/fb2/*` packages as the desktop app.

## Revision discipline

Every behavior- or shape-changing commit must:

1. Add an entry at the top of `PROGRESS.md` with a new Rev number, symptom/root-cause/fix (where relevant).
2. Bump the version in **both** `wails.json` (`info.productVersion`) and `frontend/package.json` (`version`) — they must stay in sync.
3. Commit message uses format: `Rev N (0.0.N): <short summary>`.

Branch workflow: `dev` is work-in-progress; `main` gets explicit merges.

## Corpus testing & fidelity

The corpus test (`-tags 'corpus xsd'`) reports two key metrics:

- `fidelityBroken` — source was XSD-valid but our round-trip output is not. **This must stay at 0.** Any change that breaks it is a regression.
- Per-file `Δ` in XSD error counts. A non-zero delta isn't automatically a bug: our writer normalizes element order (e.g., places `<empty-line>` where the schema allows instead of before `<title>`), which can legitimately change the error count on an already-invalid source. Inspect the errors before declaring a problem.

## Platform notes

- **Wails v2.9.2–v2.12.0 macOS file-dialog crash:** never pass multi-dot patterns like `*.fb2.zip` to `OpenFileDialog` — the native code feeds each token to `[UTType typeWithFilenameExtension:]`, `fb2.zip` returns nil, and `[NSArray addObject:nil]` throws `NSInvalidArgumentException` from Obj-C (unrecoverable by Go `recover()`). Use `*.fb2` only; users open archives via "All files". Re-verified on v2.12.0 (Rev 25): the filter block in `WailsContext.m` (`USE_NEW_FILTERS` path) is byte-identical to v2.9.2 — no nil-guard added upstream.
- **Linux build deps:** `libwebkit2gtk-4.1-dev`, `libgtk-3-dev`; `libxml2-dev` for `-tags xsd`.
- **Go version:** `go.mod` pins 1.25.0 — do not downgrade.

## Debugging a hung/crashing .app

Launch from the terminal so stderr is visible (Go panics print there; frontend logs go to the webview devtools, which Wails enables in dev builds):

```sh
/Users/dmitry.gordiyevsky/fbe-go/build/bin/fbe-go.app/Contents/MacOS/fbe
```

Before blaming the Go side, check: `[fbe] opening …` / `[fbe] parsed: …` / `[fbe] openFile failed: …` logs appear in the webview devtools (Rev 20 added them). A crash that never prints a Go panic usually means the bug is in native Wails code or in `fb2ToPMDoc` (schema violation) — the latter is caught by `Editor.svelte`'s try/catch and renders a placeholder instead of killing the app.
