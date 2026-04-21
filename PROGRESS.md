# PROGRESS

Revision log for `fbe-go`. Every commit that changes behavior or shape of the
project must add an entry here and bump the version in `wails.json` and
`frontend/package.json` (keep them in sync).

---

## Rev 3 — 2026-04-21 — Scope narrowed; Wails app runs; full block coverage

Version: **0.0.3**

### What changed

**Scope**
- Platforms: macOS + Linux only (Windows dropped, original C++ FBE keeps the
  Windows story). Rust / C acceptable for platform-integration pieces.
- Docs updated: README.md, docs/ARCHITECTURE.md, docs/PHASES.md, docs/OPERATIONS.md.

**Wails desktop app runs end-to-end**
- `wails doctor`: green. `go install .../wails/v2/cmd/wails@latest` → v2.12.0.
- `npm install` in `frontend/`: 90 packages, no critical issues.
- Added `vitePreprocess` to `frontend/vite.config.ts` so Svelte components
  accept TypeScript blocks.
- `wails build -tags xsd` → **9.3 MB `fbe-go.app` bundle** (macOS). Launches,
  renders the bundled sample book, Open… button wired to generated bindings.
- Wails-generated TypeScript bindings appear at `frontend/wailsjs/go/main/App.{js,d.ts}`
  with full types for OpenFile/SaveFile/Validate/… (gitignored).
- `App.svelte` now uses dynamic `import("../wailsjs/...")` so plain `vite dev`
  mode (no Wails runtime) still works by falling back to sample.

**Full FB2 block coverage in `parse.ts` / `schema.ts`**
- `parse.ts` rewritten to handle every block type the original FBE shows:
  body-level Title / Epigraph / Image, section-level Annotation,
  poem → stanza → verse, cite (with text-author trailer), block & inline
  images, tables (`<tr>`, `<th>`, `<td>` with colspan/rowspan/align).
- `sample.ts` re-authored as a Shevchenko "Заповіт" demo exercising every new
  node type: poem with two stanzas, cite, table (H/O chemistry demo),
  nested sections with annotation, subtitle, text-author, and all inline marks.
- `Editor.svelte` CSS extended with book-style rules for epigraph, cite,
  annotation, poem/stanza/verse, tables (th/td), code, links, images.

### How to try it

```
# First time:
cd /Users/dmitry.gordiyevsky/fbe-go
go install github.com/wailsapp/wails/v2/cmd/wails@latest
cd frontend && npm install && cd ..

# Build & run:
wails build -tags xsd
open build/bin/fbe-go.app
```

### Files modified / added

- **Modified:** `README.md`, `docs/ARCHITECTURE.md`, `docs/PHASES.md`,
  `docs/OPERATIONS.md`, `frontend/vite.config.ts`, `frontend/src/App.svelte`,
  `frontend/src/editor/Editor.svelte`, `frontend/src/editor/parse.ts`,
  `frontend/src/fb2/sample.ts`, `frontend/package.json`, `wails.json`,
  `PROGRESS.md`.
- **Added (gitignored, auto-generated):** `frontend/wailsjs/…`,
  `frontend/node_modules/`, `build/bin/fbe-go.app`, `frontend/dist/`.

### Versions bumped

- `wails.json`            0.0.2 → 0.0.3
- `frontend/package.json` 0.0.2 → 0.0.3

---

## Rev 2 — 2026-04-21 — Phase 0 PoC + encoding autodetect + XSD validator

Version: **0.0.2**

### What changed

**Frontend (Phase 0 PoC — renders an FB2 document end-to-end)**
- `frontend/src/fb2/types.ts` — hand-written TypeScript mirror of `internal/fb2/doc`.
  Used until `wails dev` generates its own bindings at `frontend/wailsjs/go/models.ts`.
- `frontend/src/fb2/sample.ts` — bundled sample book so the editor shows content
  in plain `vite dev` mode (no Wails runtime).
- `frontend/src/editor/parse.ts` — full implementation of `fb2ToPMDoc`: body,
  section (nested or flat), title, paragraph, empty-line, subtitle, plus the
  inline marks (strong, emphasis, strikethrough, sub, sup, code, link, style
  mark, inline image). Poem/cite/table/block-image remain TODO for Phase 3.
- `frontend/src/editor/Editor.svelte` — accepts an `fb` prop and remounts the
  ProseMirror view when it changes; adds FB2-style CSS for titles/subtitles/
  text-author/empty-line.
- `frontend/src/App.svelte` — auto-loads the sample on mount; `Open…` button
  calls `window.runtime.OpenFileDialog + window.go.main.App.OpenFile` when the
  Wails bindings are available, falls back to the sample otherwise.

**Go parser — encoding autodetect**
- `internal/fb2/parser/parser.go` rewritten to:
  - Strip UTF-8 / UTF-16 BOMs before decoding.
  - Route `<?xml encoding="X"?>` through a `CharsetReader` backed by
    `golang.org/x/text/encoding` (windows-1251, windows-1252, koi8-r, koi8-u,
    iso-8859-1/5, utf-16 LE/BE, plus IANA registry fallback).
  - Use `Decoder.Strict = false` to accept legacy non-canonical FBE output.
- `internal/fb2/parser/parser_test.go` — covers UTF-8, UTF-8+BOM, Windows-1251,
  KOI8-R (all four pass).

**Go validator — libxml2 XSD backend (build tag `xsd`)**
- `internal/fb2/xsd/FictionBook.xsd` + `FictionBookGenres.xsd` +
  `FictionBookLang.xsd` + `FictionBookLinks.xsd` — embedded via `go:embed`.
- `internal/fb2/xsd/xsd.go` — shared types (`ValidationError`,
  `ErrNotImplemented`, `SchemaFiles`, `SchemaFileNames`).
- `internal/fb2/xsd/xsd_stub.go` (`!xsd` build tag) — no-op that returns
  `ErrNotImplemented`. Keeps the default build pure-Go.
- `internal/fb2/xsd/xsd_libxml2.go` (`xsd` build tag) — `sync.Once` bootstrap
  extracts the embedded XSDs to a temp dir so `<xs:include>`s resolve, then
  parses the main schema via `lestrrat-go/libxml2`. Validation returns one
  `ValidationError` per schema violation.
- `internal/fb2/xsd/xsd_libxml2_test.go` — validates `testdata/blank.fb2`
  successfully under `-tags xsd`.

**CLI + Wails app**
- `cmd/fbe/main.go` — `fbe validate` now runs `xsd.Validate` and prints
  `VALID` / `INVALID: N error(s)` with per-error messages; exit code 1 on
  invalid.
- `app.go` — exposes `App.Validate(path)` to the frontend.

### How to build & run

```
# Pure Go (no libxml2 dep)
go build ./cmd/fbe

# With XSD validation (requires libxml-2.0 via pkg-config)
go build -tags xsd -o fbe ./cmd/fbe
./fbe validate testdata/blank.fb2            # → VALID

# Tests
go test ./...                                # parser tests
go test -tags xsd ./...                      # + xsd integration

# Desktop app (requires wails CLI)
wails dev                                    # or: wails build -tags xsd
```

### Files modified / added

- **Modified:** `app.go`, `cmd/fbe/main.go`, `frontend/src/App.svelte`,
  `frontend/src/editor/Editor.svelte`, `frontend/src/editor/parse.ts`,
  `go.mod`, `go.sum`, `internal/fb2/parser/parser.go`, `internal/fb2/xsd/xsd.go`.
- **Added:** `PROGRESS.md`, `frontend/src/fb2/types.ts`, `frontend/src/fb2/sample.ts`,
  `internal/fb2/parser/parser_test.go`, `internal/fb2/xsd/FictionBook.xsd`,
  `internal/fb2/xsd/FictionBookGenres.xsd`, `internal/fb2/xsd/FictionBookLang.xsd`,
  `internal/fb2/xsd/FictionBookLinks.xsd`, `internal/fb2/xsd/xsd_stub.go`,
  `internal/fb2/xsd/xsd_libxml2.go`, `internal/fb2/xsd/xsd_libxml2_test.go`.

### Verification

- `go build ./...` ✓
- `go build -tags xsd ./...` ✓
- `go test ./...` ✓ (parser_test: 4/4 pass)
- `go test -tags xsd ./...` ✓ (xsd_libxml2_test: 1/1 pass)
- `./fbe validate testdata/blank.fb2` → `VALID`
- `./fbe info testdata/blank.fb2` → correct metadata JSON

### Versions bumped

- `wails.json`            0.0.1 → 0.0.2
- `frontend/package.json` 0.0.1 → 0.0.2

---

## Rev 1 — 2026-04-21 — Initial skeleton

Version: **0.0.1**

Commit `d66d7df`. See message for full summary.

- Go module `github.com/dimgord/fbe-go` (go 1.24, bumped to 1.25 by `go mod tidy`).
- `internal/fb2/{doc,parser,writer,xsd,zipfb2,binary,thumb,speller,search,
   settings,export/html}` — pure-Go FB2 library with full types in `doc/`,
   stubs elsewhere.
- `cmd/fbe` — CLI with `validate | thumb | pack | unpack | info | export`
   commands (most still return `not implemented yet`).
- `frontend/` — Svelte + ProseMirror skeleton.
- `docs/` — `ARCHITECTURE.md`, `OPERATIONS.md` (65-item catalog), `PHASES.md`.
- `testdata/blank.fb2` — minimal valid document.
- Verified: `go build ./...` + `fbe info testdata/blank.fb2` produces valid JSON.

35 files, 2343 insertions.
