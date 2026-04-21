# PROGRESS

Revision log for `fbe-go`. Every commit that changes behavior or shape of the
project must add an entry here and bump the version in `wails.json` and
`frontend/package.json` (keep them in sync).

---

## Rev 6 — 2026-04-21 — First editable experience: toolbar + inline marks + block styles

Version: **0.0.6**

### What changed

**Real ProseMirror commands (`frontend/src/editor/commands.ts`)**
- `toggleStrong` / `toggleEmphasis` / `toggleStrikethrough` / `toggleSub` /
  `toggleSup` / `toggleCode` — inline mark toggles wrapping
  `prosemirror-commands.toggleMark`.
- `toggleLink(href)` — link mark with href; empty href removes the mark.
- `applyStyleMark(name)` — sets the FB2 `<style name="…">` inline mark.
- `styleNormal` / `styleSubtitle` / `styleTextAuthor` — block-type commands
  via `pmSetBlockType`.
- `insertEmptyLine` — replaces selection with an `<empty-line>` node.
- `isMarkActive` / `isBlockActive` — helpers for toolbar highlighting (wired
  in a later rev).
- Structural stubs (`insertPoem`, `insertCite`, `addEpigraph`, …) kept for
  Phase 3 work with file:line references to the original FBE.

**Keyboard shortcuts in Editor.svelte**
- `Mod-B` strong, `Mod-I` emphasis, `Mod-Shift-S` strikethrough,
  `Mod-,` sub, `Mod-.` sup, `Mod-Shift-C` code. Undo/redo already wired.
- Editor.svelte exposes `exec(cmd)` and `execLink()` so the toolbar can
  dispatch commands with auto-focus. Also re-exports the command functions for
  binding by name.

**New Toolbar component (`frontend/src/editor/Toolbar.svelte`)**
- Formatting buttons wired to the exported Editor methods: undo/redo, bold,
  italic, strike, sub, sup, code, link, normal paragraph, subtitle,
  text-author, empty-line.
- Tooltips show the shortcut key. Minimal, book-friendly styling.

**App.svelte wires the toolbar above the editor**
- `bind:this={editor}` on the Editor component so the toolbar gets a
  reference to dispatch commands.
- Grid row added for the toolbar between header and main.

### Verified

- `wails build -tags xsd` → 9.4 MB `.app`, relaunches with toolbar visible.
- Clicking formatting buttons modifies the sample document and preserves
  history (undo/redo works).
- Keyboard shortcuts take effect in the editor.

### Files modified / added

- **Modified:** `frontend/src/App.svelte`, `frontend/src/editor/Editor.svelte`,
  `frontend/src/editor/commands.ts`, `PROGRESS.md`, `wails.json`,
  `frontend/package.json`.
- **Added:** `frontend/src/editor/Toolbar.svelte`.

### Versions bumped

- `wails.json`            0.0.5 → 0.0.6
- `frontend/package.json` 0.0.5 → 0.0.6

---

## Rev 5 — 2026-04-21 — Real-world corpus testing

Version: **0.0.5**

### What changed

**Corpus test harness (`internal/fb2/corpus_test.go`, build tag `corpus`)**
- Walks a directory for `.fb2` files (defaults to `~/Documents/books`,
  overridable via `FBE_CORPUS_DIR`).
- For each file: parse → write → re-parse → validate source AND output against
  the bundled FictionBook.xsd.
- Reports: parse/write/reparse/srcValid/outValid counts, plus
  **fidelityBroken** (source valid → our output invalid) and
  **fidelityPreserved** (source invalid, we emit same count of errors).
- Per-file XSD error deltas surface anywhere our writer diverges from source
  faithfulness.

### First corpus run results (3 files, 3.2 MB)

```
parse=3/3 write=3/3 reparse=3/3 srcValid=1/3 outValid=1/3 fidelityBroken=0
```

All three files parse, write, and re-parse successfully (including
`Mihalovskij_*.fb2` in `windows-1251` — encoding autodetect working).

**fidelityBroken=0** — the critical check: no valid-source file was broken
by our round-trip.

**Observation:** `Спынь Ксения - Дурные.fb2` has 6 XSD errors in source, 5 in
our output (-1). The missing error is
`Element 'empty-line': This element is not expected` — the source had an
`<empty-line>` in a position our parser didn't accept into the typed tree, so
we silently dropped it. Tracked for Phase 1:
- TODO: preserve unknown/misplaced elements via a `Raw []byte` fallback field
  on containers, so unfamiliar FB2 extensions round-trip losslessly.

### Running the corpus test

```
FBE_CORPUS_DIR=/path/to/books \
  go test -tags 'corpus xsd' -v ./internal/fb2/ -run TestCorpus
```

Default `go test ./...` does NOT run corpus (build tag gated), so CI stays hermetic.

### Files modified / added

- **Added:** `internal/fb2/corpus_test.go`.
- **Modified:** `PROGRESS.md`, `wails.json`, `frontend/package.json`.

### Versions bumped

- `wails.json`            0.0.4 → 0.0.5
- `frontend/package.json` 0.0.4 → 0.0.5

---

## Rev 4 — 2026-04-21 — Writer round-trip + polymorphic Block/Inline marshalers

Version: **0.0.4**

### What changed

**Custom XML marshalers for polymorphic types (Block, Paragraph, StyleInline, Link)**
- Removed the `xml:",any"` + `xml:",innerxml"` approach from Block and Inline
  that was losing content into the Raw field instead of populating typed
  pointers.
- Block now has `UnmarshalXML` that dispatches on the local element name
  (p / poem / subtitle / cite / empty-line / table / image → corresponding
  pointer field) and `MarshalXML` that re-emits only the populated field.
- Paragraph, StyleInline, Link now have matching custom marshalers that read
  attributes (id/style/lang, name, xlink:href/type respectively) plus mixed
  text+element content into a typed `[]Inline` children slice. Writing
  re-emits attributes and children as CharData/elements.
- Writer-side `normalize` helper deleted — no longer needed; `xml.Encoder` now
  produces clean output directly.

**Namespace handling**
- `FictionBook.XMLName` tagged with the FB2 namespace
  (`http://www.gribuser.ru/xml/fictionbook/2.0 FictionBook`) so the writer
  emits `xmlns="..."` once at the root. No more redundant xmlns on every `<p>`.

**Writer verification**
- `internal/fb2/writer/writer_test.go` — round-trip test:
  parse → write → parse → compare. Asserts the writer output contains the FB2
  xmlns at the root and does NOT re-declare it on paragraph elements.
- `internal/fb2/writer/writer_xsd_test.go` (build tag `xsd`) — validates the
  writer output against the bundled FictionBook.xsd.
- Both tests pass for `testdata/blank.fb2` and a new `testdata/rich.fb2`
  (epigraphs, cites, marks, links, nested sections, subtitles, empty-line).

**New test fixture**
- `testdata/rich.fb2` — exercises genre match, annotation, epigraph with
  text-author, strong/emphasis/code/sub/sup/links, empty-line, cite, subtitle,
  nested sections.

### Verification

```
go test ./...                                # parser (4/4) + writer (2/2 round-trip)
go test -tags xsd ./...                      # + xsd integration + writer-xsd validation
./fbe validate testdata/blank.fb2            # → VALID
./fbe validate testdata/rich.fb2             # → VALID
wails build -tags xsd                         # 9.3 MB .app rebuilt in 10s
```

### Known limitations (still Phase 1 TODO)

- Poem / Cite / Table bodies are round-tripped via the existing struct tags
  (which work for `xml:",any"` on those containers with the newly-typed Block);
  they compile and validate but haven't yet been exercised with rich content.
  This is the next round of work before Phase 3.
- Binary base64 wrapping (FBE wraps at 76 cols; we emit as a single line).
  Cosmetic, doesn't break readers.
- Whitespace inside `<p>` is not byte-exact (leading/trailing spaces may be
  preserved differently). XSD-valid either way.

### Files modified / added

- **Modified:** `internal/fb2/doc/doc.go`, `internal/fb2/writer/writer.go`,
  `internal/fb2/writer/writer_test.go`, `PROGRESS.md`, `wails.json`,
  `frontend/package.json`.
- **Added:** `testdata/rich.fb2`, `internal/fb2/writer/writer_xsd_test.go`.

### Versions bumped

- `wails.json`            0.0.3 → 0.0.4
- `frontend/package.json` 0.0.3 → 0.0.4

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
