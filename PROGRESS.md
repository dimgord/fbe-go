# PROGRESS

Revision log for `fbe-go`. Every commit that changes behavior or shape of the
project must add an entry here and bump the version in `wails.json` and
`frontend/package.json` (keep them in sync).

---

## Rev 14 — 2026-04-21 — Paste handling (strip Word clutter) [dev]

Version: **0.0.14**

- `frontend/src/editor/paste.ts` — `cleanPastedHTML` strips Word
  conditional comments, `<style>` blocks, `<meta>` / `<link>` / `<xml>` /
  `<o:p>` / `<w:*>` tags, mso-* inline styles, font-family/size/color
  junk, class attributes, `<span>` wrappers; collapses multi-`<br>` into
  paragraph breaks; drops empty paragraphs; converts `&nbsp;` to regular
  space. `cleanPastedText` normalizes CRLF → LF, strips non-printable
  control chars, normalizes nbsp.
- `Editor.svelte` wires them to `transformPastedHTML` /
  `transformPastedText` on the PM view.
- 12 new vitest assertions (54/54 total).
- Matches `FBEview.cpp::OnPaste` / `OnRealPaste` spirit.

Versions bumped 0.0.13 → 0.0.14.

---

## Rev 13 — 2026-04-21 — MergeContainers — Phase 3 complete [dev branch]

Version: **0.0.13**

### What changed

Implements the last 🔴 command from FBE (`main.js:2216 MergeContainers`)
with full coverage of its four structural combinations. **Phase 3 is now
complete.**

**`mergeContainers` in `commands.ts`:**

1. Requires the cursor inside a `section` / `stanza` / `cite`.
2. Requires an immediate next sibling of the same type (refuses otherwise).
3. Picks a strategy based on the sibling pair shape:

| cp         | nx         | behavior |
|------------|------------|----------|
| section flat    | section flat     | concat block content; unwrap `nx`'s `title` → subtitles, `epigraph` / `annotation` → promote inner blocks |
| section nested  | section flat     | wrap `nx`'s flat content in a new subsection appended to `cp` |
| section flat    | section nested   | flatten `nx`'s nested sections into `cp`'s block content (recursive: nested titles → subtitles) |
| section nested  | section nested   | concat `cp`'s headers + sub-sections with `nx`'s sub-sections; drop `nx`'s headers |
| stanza          | stanza           | concat verses; drop `nx`'s title/subtitle |
| cite            | cite             | concat children; `cp`'s trailing `text_author` demotes to plain paragraphs (matches FBE's `removeAttribute("className")`) |

Helpers `isNestedSection`, `mergeSections`, `mergeStanzas`, `mergeCites`,
and `flattenSectionInto` encapsulate each case. The final replacement uses
`tr.replaceWith([cp.before, nx.after], merged)` so undo rolls it back
cleanly.

### Tests

Seven new vitest cases exercising every branch:

- flat+flat: paragraphs concat.
- flat+flat with nx's title → subtitle demotion + annotation unwrap.
- nested+flat: nx flat blocks land in a new subsection.
- flat+nested: nested sections flatten into cp (titles → subtitles).
- nested+nested: concat sub-sections, drop nx's headers.
- stanza+stanza: verses concat.
- cite+cite: children concat + cp's text-author demotes to paragraph.
- Refuses when no same-type sibling follows.

Also rewrote the `cursorInFirstSection` test helper so the cursor lands
inside the *top-level* first section (prefers `<title>`'s paragraph,
falls back to a flat block, then descends) — earlier attempts were
landing inside nested children or the next section.

**42/42 vitest pass** (14 serialize + 5 outline + 23 commands).

### Toolbar

New `⟛ Merge` button after the Table one.

### Phase 3 status

All structural commands implemented:

- ✅ cloneContainer, removeOuterContainer, addTitle, addEpigraph,
  addAnnotation, addTextAuthor (Rev 10)
- ✅ insertCite, insertPoem (Rev 11)
- ✅ insertTable (Rev 12)
- ✅ mergeContainers (this rev)

Next natural step: **Speller** (Hunspell CGo + PM decoration plugin) or
**HTML export** (Go templates from `internal/fb2/export/html`) or **rich
annotation editor** in the description form.

### Files modified

- `frontend/src/editor/commands.ts`, `commands.test.ts`, `Editor.svelte`,
  `Toolbar.svelte`, `PROGRESS.md`, `wails.json`, `frontend/package.json`.

### Versions bumped

- `wails.json`            0.0.12 → 0.0.13
- `frontend/package.json` 0.0.12 → 0.0.13

---

## Rev 12 — 2026-04-21 — InsertTable with dialog [dev branch]

Version: **0.0.12**

### What changed

Adds `insertTable` alongside a small modal dialog for entering dimensions.

**`insertTableCmd(rows, cols, header)` in `commands.ts`:**
- Parent must be `section` / `epigraph` / `annotation` / `history` / `cite`.
  (Body-level placement is rejected — matches FB2 schema.)
- Builds `table > table_row+ > table_cell` with `header=true` on the first
  row when the header flag is set.
- Inserts at `range.end` for empty selections (doesn't split the caret
  paragraph); replaces the range for non-empty selections.
- Also exports `insertTable` — a zero-arg 3×3-with-header convenience for
  menus that can't pass parameters.

**`TableDialog.svelte`** — 20 rem-wide modal, centered on a semi-transparent
backdrop:
- Number inputs for rows (1–50) and cols (1–20), plus a header checkbox.
- Rows input auto-focuses on open; `Enter` submits, `Esc` cancels.
- Click-outside closes. Emits `insert` with `{rows, cols, header}` payload.

**`Editor.svelte`** wires it:
- Exposes `openTableDialog()` for the toolbar.
- Handles `insert` event from the dialog, dispatching `insertTableCmd(...)`.

**`Toolbar.svelte`** gets a `▦ Table…` button after the Cite/Poem pair.

### Tests

Three new vitest cases (`commands.test.ts`):
- Inserts a 3×3 header table; verifies header flag on row 0, not on rows 1–2.
- Refuses to insert from inside a `<title>` (no valid container ancestor).
- Rejects `rows < 1` or `cols < 1` dimensions.

34/34 vitest pass (14 serialize + 5 outline + 15 commands).

### Still stubbed

- 🔴 `mergeContainers` — FBE `main.js:2216`, 6 sub-cases. Last major
  Phase 3 structural piece.

### Files modified / added

- **Modified:** `frontend/src/editor/commands.ts`, `commands.test.ts`,
  `Editor.svelte`, `Toolbar.svelte`, `PROGRESS.md`, `wails.json`,
  `frontend/package.json`.
- **Added:** `frontend/src/editor/TableDialog.svelte`.

### Versions bumped

- `wails.json`            0.0.11 → 0.0.12
- `frontend/package.json` 0.0.11 → 0.0.12

---

## Rev 11 — 2026-04-21 — InsertPoem + InsertCite [dev branch]

Version: **0.0.11**

### What changed

Closes the 🔴 hard-half of Phase 3's container commands. `InsertPoem` and
`InsertCite` wrap a block range in the corresponding FB2 container, replacing
the stubs that were in `commands.ts` since Rev 6.

**`insertCite`** (FBEview.cpp:1048 equivalent)
- Requires the cursor to be inside `section` / `poem` / `annotation` / `history`.
- Uses `$from.blockRange($to)` to locate the covered blocks.
- Collects paragraph / empty-line / subtitle children from that range into the
  new `<cite>`; skips incompatible blocks (nested poems, tables, images) so
  the cite doesn't violate its FB2 schema.
- Replaces the range with the cite via `tr.replaceRangeWith`.

**`insertPoem`** (FBEview.cpp:903 equivalent)
- Requires cursor inside `section` / `epigraph` / `annotation` / `history` /
  `cite` (same parents FBE allows).
- Each paragraph in the range becomes a `<v>` verse.
- `<empty-line>` blocks **split stanzas**: two paragraphs, blank line, two
  more paragraphs → two `<stanza>`s of two verses each (matches FBE's
  stanza-splitting heuristic).
- Empty ranges produce one stanza with one empty verse, keeping the poem
  editable.

### Tests

- 3 new vitest cases in `commands.test.ts`:
  - `insertCite wraps the selected paragraphs in a <cite>` — 3 paragraphs,
    selection over last two → section becomes title/paragraph/cite(2 paras).
  - `insertPoem converts selected paragraphs to a stanza of verses` — 3
    paragraphs, full selection → poem with one stanza of three verses.
  - `insertPoem splits stanzas at empty-line blocks` — 4 paragraphs with an
    empty-line in the middle → poem with two stanzas of two verses each.
- Total: **31/31** vitest pass (14 serialize + 5 outline + 12 commands).

### Toolbar

Two new buttons after the structural group: `❝ Cite`, `♪ Poem`.
Tooltips explain the block-range + empty-line semantics.

### Still stubbed

- 🔴 `mergeContainers` — FBE's `main.js:2216` has 6 sub-cases with subtle
  invariants; needs a focused rev of its own.
- 🔴 `insertTable` — rows × cols × header toggle; probably a modal dialog.

### Files modified

- `frontend/src/editor/commands.ts`, `commands.test.ts`, `Editor.svelte`,
  `Toolbar.svelte`, `PROGRESS.md`, `wails.json`, `frontend/package.json`.

### Versions bumped

- `wails.json`            0.0.10 → 0.0.11
- `frontend/package.json` 0.0.10 → 0.0.11

---

## Rev 10 — 2026-04-21 — Phase 3 structural commands [dev branch]

Version: **0.0.10**

### What changed

Implements six of the structural commands from `docs/OPERATIONS.md` as real
ProseMirror commands with selection-constraint checking, keyboard/toolbar
hookup, and vitest coverage. These close the easy half of Phase 3; the 🔴
hard ones (InsertPoem / InsertCite / MergeContainers / InsertTable) stay
stubbed.

**Implemented commands (`frontend/src/editor/commands.ts`):**

- **`cloneContainer`** — duplicates the surrounding section / poem / stanza /
  cite / epigraph. Deep-copies via `nodeFromJSON` so marks and nested
  structure survive. Matches `main.js:1940 CloneContainer`.
- **`removeOuterContainer`** — dissolves a section that contains only other
  sections (matches FBE's `IsCtSection` check), promoting the children up a
  level. Safe: returns false on sections with flat block content to avoid
  data loss. Matches `main.js:2357 RemoveOuterContainer`.
- **`addTitle`** — inserts an empty `<title>` at the start of the enclosing
  section / body / poem / stanza when none exists. Simplified from
  `main.js:1766 AddTitle` (doesn't consume selection text yet).
- **`addEpigraph`** — inserts an empty `<epigraph>` in the enclosing body /
  section / poem, positioned after any existing `<title>` to maintain
  canonical element order. Matches `main.js:2050 AddEpigraph`.
- **`addAnnotation`** — inserts `<annotation>` in the enclosing section (if
  absent), positioned after title/epigraph/image. Matches
  `main.js:2142 AddAnnotation`.
- **`addTextAuthor`** — appends a `<text-author>` trailer to the enclosing
  poem / epigraph / cite. Matches `main.js:2168 AddTA`.

Helper `findAncestor` / `findAncestorAny` walks the `ResolvedPos` chain to
locate the nearest container of a given type, plus
`firstInsertionPointAfterHeader` keeps epigraph/annotation placement
schema-legal.

**Toolbar** (`Toolbar.svelte`): new row of structure buttons after the
style/empty-line group: `Clone`, `Unwrap`, `+ Title`, `+ Epigraph`,
`+ Annot.`, `+ T-A`. Each shows a tooltip describing when it's applicable.

**Editor.svelte** re-exports the new commands so App.svelte / Toolbar can
reference them by name.

### Tests

- `commands.test.ts` — 9 new assertions covering both positive and negative
  paths: cloneContainer duplicates a section; addTitle no-ops on a titled
  section and adds one on an untitled section; addEpigraph / addAnnotation
  place the new container after `<title>`; addAnnotation no-ops on a
  pre-annotated section; addTextAuthor appends to a poem; removeOuterContainer
  refuses flat sections and correctly promotes nested ones.
- Helper `buildStateWithCursor(fb, predicate)` walks the PM doc and places
  the cursor at the first paragraph/verse whose ancestor chain satisfies the
  caller's predicate — makes the command tests read naturally regardless of
  doc layout.

### Verified

- `npm test` → **28/28** (14 serialize + 5 outline + 9 commands).
- `wails build -tags xsd` → 9.4 MB `.app`, ~10 s.

### Files modified / added

- **Modified:** `frontend/src/editor/commands.ts`,
  `frontend/src/editor/Editor.svelte`,
  `frontend/src/editor/Toolbar.svelte`, `PROGRESS.md`, `wails.json`,
  `frontend/package.json`.
- **Added:** `frontend/src/editor/commands.test.ts`.

### Versions bumped

- `wails.json`            0.0.9 → 0.0.10
- `frontend/package.json` 0.0.9 → 0.0.10

---

## Rev 9 — 2026-04-21 — Description form (all 5 metadata sections) [dev branch]

Version: **0.0.9**

### What changed

Added a full `<description>` editor. The body/description split mirrors the
original FBE's `apiShowDesc(state)` toggle: a `[Body] [Description]` segmented
button in the header swaps between the ProseMirror editor and a tabbed form.

**`DescriptionPanel.svelte`** — top-level container with 5 tabs:

- **Title info** — `TitleInfoForm.svelte`, fully wired
- **Source title** — same form component, shown only when `<src-title-info>`
  is present; offers "Add source title info" when missing
- **Document** — `DocumentInfoForm.svelte` (authors, id with New-UUID button,
  version, program-used, date, src-ocr, src-url[])
- **Publish** — `PublishInfoForm.svelte` (book-name, publisher, city, year,
  isbn, sequence)
- **Custom** — `CustomInfoForm.svelte` (repeatable type/value pairs)

**Reusable field components:**

- `AuthorField.svelte` — first/middle/last name on one row; disclosure reveals
  nickname, id, email[], home-page[]. Variants: `primary` (always expanded)
  and `compact` (collapsed). Remove + clone buttons.
- `GenreField.svelte` — genre string + match percentage, remove + clone.
- `DateField.svelte` — human-readable text + ISO value side by side.
- `SequenceField.svelte` — recursive via `<svelte:self>` so nested series
  work (FB2 allows `<sequence>` inside `<sequence>`).
- `CoverpageField.svelte` — dropdown of available binary IDs (from
  `fb.Binaries`) + custom-href fallback.

**Two-way binding through App.svelte:**

`<DescriptionPanel bind:fb />` propagates every field edit back to the parent
`fb` state, which flows through `Editor.currentFB()` on Save. This means
edits to metadata are saved alongside body edits without extra plumbing.

**Gotchas fixed during implementation:**

- Svelte's template parser does not accept TypeScript non-null assertions
  inside `{expr}` attribute bindings. Replaced `author.Email![i]` etc. with
  reactive guards (`$: if (!author.Email) author.Email = []`) + plain
  `author.Email[i]`, and wrapped nullable parents in `{#if date}` / `{#if cover}`.
- `pattern="\d{{4}}..."` inside an `<input>` triggered Svelte's mustache
  parser; removed the `pattern` attribute (HTML5 validation can come back
  later with a different escape).

### Verified

- `npm test` → 19/19 still passing (serialize + outline).
- `wails build -tags xsd` → 9.4 MB `.app`, 8.7 s; launches with working
  `[Body] [Description]` toggle and all 5 tabs functional.
- Editing a field in the form mutates `fb`; switching back to Body and
  Saving writes the updated description to disk.

### Files modified / added

- **Modified:** `frontend/src/App.svelte`, `PROGRESS.md`, `wails.json`,
  `frontend/package.json`.
- **Added:** `frontend/src/description/AuthorField.svelte`,
  `GenreField.svelte`, `DateField.svelte`, `SequenceField.svelte`,
  `CoverpageField.svelte`, `TitleInfoForm.svelte`,
  `DocumentInfoForm.svelte`, `PublishInfoForm.svelte`,
  `CustomInfoForm.svelte`, `DescriptionPanel.svelte`.

### Branch

Committed on `dev` (per new workflow: dev is work-in-progress, main gets
explicit merges).

### Versions bumped

- `wails.json`            0.0.8 → 0.0.9
- `frontend/package.json` 0.0.8 → 0.0.9

---

## Rev 8 — 2026-04-21 — Frontend round-trip tests + DocumentTree outline

Version: **0.0.8**

### What changed

**Part 1 — vitest round-trip tests for `serialize.ts`**

- Added `vitest` to devDeps; `npm test` / `npm run test:watch` scripts.
- `frontend/src/editor/serialize.test.ts` — 14 assertions running
  `fb2ToPMDoc → pmDocToFB2` on `SAMPLE_BOOK` and verifying every node kind
  survives: bodies, sections (nested), titles, epigraphs with text-author,
  poems with stanzas & text-author, all inline marks (strong/emphasis/
  strikethrough/sub/sup/code/link/style), empty-line, cite with text-author,
  subtitle, tables (th/td + colspan/rowspan/align with sub mark inside
  cells), nested sections with annotation, book-title and description.
- **Caught a real schema bug:** `schema.ts` content expressions referenced a
  nonexistent `image` node in `body` and `section` content rules. Fixed to
  `image_block` — schema now initializes cleanly.

**Part 2 — DocumentTree outline + click-to-scroll**

- `frontend/src/tree/outline.ts` — `buildOutline(fb)` walks a FictionBook and
  returns `OutlineNode[]` with `{label, kind: "body"|"section", path, children}`.
  `label` comes from the section's `<title>` (inline-flattened); falls back
  to a placeholder when untitled. `path` is an index array ([body, section, …])
  used for navigation.
- `frontend/src/tree/outline.test.ts` — 5 assertions on `SAMPLE_BOOK`:
  body count, top-level section labels, nested section labels, unique paths,
  empty-input handling.
- `DocumentTree.svelte` rewritten to accept `fb: FictionBook | null` prop;
  renders an `<ul>` of `OutlineItem.svelte` components. Recursion uses
  `<svelte:self>` for nested sections.
- `OutlineItem.svelte` — one clickable button per node. Emits `navigate`
  event with `path` on click. Styled with kind-based classes (body is
  blue/bold, section is default).
- `Editor.svelte` gains `scrollToPath(path)`: walks the ProseMirror doc by
  outline path to find the target node's position, uses `coordsAtPos` to
  scroll the editor into view, flashes `.outline-flash` on the section for
  700 ms.
- `App.svelte` wires `<DocumentTree {fb} on:navigate>` to
  `editor?.scrollToPath(e.detail.path)`.

### Verified

- `npm test` → 19/19 pass (14 serialize + 5 outline).
- `wails build -tags xsd` — 9.4 MB `.app`, relaunches with the outline in
  the left pane. Clicking an item scrolls the editor and flashes the target.

### Files modified / added

- **Modified:** `frontend/package.json`, `frontend/src/editor/schema.ts`,
  `frontend/src/editor/Editor.svelte`, `frontend/src/App.svelte`,
  `frontend/src/tree/DocumentTree.svelte`,
  `PROGRESS.md`, `wails.json`.
- **Added:** `frontend/src/editor/serialize.test.ts`,
  `frontend/src/tree/outline.ts`, `frontend/src/tree/outline.test.ts`,
  `frontend/src/tree/OutlineItem.svelte`.

### Versions bumped

- `wails.json`            0.0.7 → 0.0.8
- `frontend/package.json` 0.0.7 → 0.0.8

---

## Rev 7 — 2026-04-21 — Save cycle + Raw fallback for unknown elements

Version: **0.0.7**

### What changed

**Part 1 — Save cycle (edit → disk round-trip)**

- `frontend/src/editor/serialize.ts` fully implemented: walks the ProseMirror
  doc and builds a FictionBook shape that mirrors Go's `doc.FictionBook`.
  Covers body/section (nested or flat), title, epigraph + text-author,
  annotation, paragraph, subtitle, empty-line, poem → stanza → verse,
  cite + text-author, table (th/td with colspan/rowspan/align/valign), block
  and inline images, plus all inline marks (strong/emphasis/strikethrough/
  sub/sup/code/style/link) with stable nesting order. Description + binaries
  are preserved from the originally-loaded FictionBook.
- `Editor.svelte` exposes `currentFB()` so App.svelte can grab the current doc
  state.
- `App.svelte` adds Save / Save As… / Validate buttons:
  - **Save** — reuses `currentPath` or falls back to Save-As dialog if none.
  - **Save As…** — Wails `SaveFileDialog` with `.fb2` filter.
  - **Validate** — calls `App.Validate(path)` and shows result in status bar.
  - Keyboard: `⌘S` / `⌘⇧S` for Save / Save As.
- Status bar feedback: green "Saved X" on success (auto-clears after 3 s),
  "XSD valid ✓" or error summary on Validate.

**Part 2 — Lossless round-trip for unknown elements**

- New `doc.RawElement` type that captures arbitrary XML elements verbatim:
  name, attributes, recursive child tokens (text + nested elements).
  Custom `UnmarshalXML` / `MarshalXML` preserve nesting and attributes.
- `Block` gains a `Raw *RawElement` field for unknown block-level elements
  (FB2 extensions, future-version tags). `Block.UnmarshalXML` now captures
  unknown elements into Raw instead of silently skipping via `d.Skip()`.
- `Inline` gains the same `Raw *RawElement`. Mixed-content reader in
  `unmarshalInlineContent` routes unknown inline elements to Raw.
- `marshalInlineContent` emits Raw elements back verbatim.

**Verification**

- `go test ./...` — all pre-existing tests still pass.
- `go test -v ./internal/fb2/writer/ -run TestRaw` — two new tests:
  - `TestRawFallbackPreservesUnknownBlock`: `<custom-extension
    data-source="Flibusta" count="42">…<b>…</b>…</custom-extension>`
    survives round-trip with all attributes and nested elements intact.
  - `TestRawFallbackPreservesUnknownInline`: `<ruby rb="漢" rt="kan">漢</ruby>`
    inside a `<p>` round-trips verbatim.
- Corpus run (`go test -tags 'corpus xsd' ...`) unchanged:
  `parse=3/3 write=3/3 reparse=3/3 srcValid=1/3 outValid=1/3 fidelityBroken=0`.
  The −1 XSD-error delta on one file remains — caused by our writer
  normalizing element order (valid `<empty-line>` placed where schema allows
  instead of before `<title>`). Not a lost-content bug.

### Files modified / added

- **Modified:** `internal/fb2/doc/doc.go`, `frontend/src/App.svelte`,
  `frontend/src/editor/Editor.svelte`, `frontend/src/editor/serialize.ts`,
  `PROGRESS.md`, `wails.json`, `frontend/package.json`.
- **Added:** `internal/fb2/writer/raw_test.go`.

### Versions bumped

- `wails.json`            0.0.6 → 0.0.7
- `frontend/package.json` 0.0.6 → 0.0.7

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
