# FB2 Operations Catalog

Every editing operation in the original FictionBook Editor (FBE) that needs a ProseMirror equivalent in the Go port. This is the master work-list for Phase 3 (Editor MVP) and part of Phase 4 (feature parity).

Columns:
- **FBE source** — file:line in the original C++ / JS codebase
- **Availability check** — what the original predicate was (FBE passes `fCheck=true` to probe whether a command is enabled in the UI)
- **PM command** — target function in `frontend/src/editor/commands.ts`
- **Complexity** — 🟢 easy (< 1 day), 🟡 medium (1–3 days), 🔴 hard (> 3 days)

---

## 1. Load / Save

| FBE source | What it does | Go/Wails equivalent | Complexity |
|---|---|---|---|
| `FBDoc.cpp:323 LoadFromHTML` + `main.js:508 apiLoadFB2` | Read .fb2, parse XML (MSXML), XSL-transform to editor HTML | `parser.Parse()` → `App.OpenFile()` → `fb2ToPMDoc()` | 🟡 |
| `FBDoc.cpp:894 SaveToFile` + `main.js:1525 GetDesc` + `main.js:1539 GetBinaries` | Walk MSHTML DOM, build MSXML DOM, XSD-validate, write .fb2 | `pmDocToFB2()` → `App.UpdateDocument()` → `writer.Write()` + `xsd.Validate()` | 🟡 |
| `main.js:517 xml.load` on `.fb2` encoded in win-1251/koi8-r | Autodetect encoding from `<?xml encoding=...?>` PI | `parser.Parse` must set `dec.CharsetReader` using `golang.org/x/text/encoding/charmap` | 🟡 |
| Load `.fb2.zip` | Open archive, read first .fb2 entry | `zipfb2.Unpack()` in `App.OpenFile()` | 🟢 |
| `main.js:499 recursiveChangeNbsp` | Replace U+00A0 with user-configured nbsp char in annotation/history/body | Apply in parser post-process using `settings.NBSPChar` | 🟢 |

## 2. Container operations (block-level structure)

| FBE source | Operation | PM command | Availability check | Complexity |
|---|---|---|---|---|
| `FBEview.cpp:903 InsertPoem` | Wrap selected paragraphs in `<poem><stanza>…</stanza></poem>`. Parent must be section/epigraph/annotation/history/cite. | `insertPoem` | caret inside allowed parent | 🔴 (range→stanza split) |
| `FBEview.cpp:1048 InsertCite` | Wrap selected paragraphs in `<cite>`. | `insertCite` | caret inside section/poem/annotation | 🟡 |
| `FBEview.cpp:3556 InsertTable` | Insert a `rows×cols` table with optional header row. | `insertTable(rows, cols, header)` | caret inside section | 🟡 |
| `main.js:1766 AddTitle` | Add a title block at section/body/stanza/poem start. Accepts selection text as the title. | `addTitle` | parent in {body, section, stanza, poem}, no existing title | 🟡 |
| `main.js:1894 AddBody` | Add a new `<body>` container (typically for notes). | `addBody` | — | 🟢 |
| `main.js:1940 CloneContainer` | Duplicate the surrounding section/poem/stanza/cite/epigraph. | `cloneContainer` | cp.className ∈ {section, poem, stanza, cite, epigraph} | 🟢 |
| `main.js:2030 AddImage` | Insert a block image at body/section start (after title/epigraph). | `addImage` | parent is body or section | 🟢 |
| `main.js:2050 AddEpigraph` | Insert epigraph from selection (auto-splits last paragraph into text-author if styled). | `addEpigraph` | parent in {body, section, poem} | 🟡 |
| `main.js:2142 AddAnnotation` | Prepend `<annotation>` to a section. | `addAnnotation` | parent is section, no existing annotation | 🟢 |
| `main.js:2168 AddTA` | Append a `<text-author>` trailer to enclosing poem/epigraph/cite. | `addTextAuthor` | enclosing container is poem/epigraph/cite, no existing trailer | 🟢 |
| `main.js:2216 MergeContainers` | Merge adjacent same-class section/stanza/cite. Handles nested section vs. flat-content merge rules. | `mergeContainers` | next sibling has same className | 🔴 (section merge has 6+ sub-rules) |
| `main.js:2357 RemoveOuterContainer` | Dissolve section into its child sections (move children up a level). | `removeOuterContainer` | current section contains only sections | 🟡 |
| `ID_EDIT_SPLIT` (resource.h:509) → FBEview split | Split enclosing section at caret. | `splitSection` | inside section | 🟡 |

## 3. Insert at cursor

| FBE source | Operation | PM command | Complexity |
|---|---|---|---|
| `main.js:1971 InsImage` | Insert block image at caret (inside section). | `insertImage(href)` | 🟢 |
| `main.js:2001 InsInlineImage` | Insert inline image inside p/subtitle/text-author/annotation/history. | `insertInlineImage(href)` | 🟢 |
| `ID_STYLE_LINK` (resource.h:491) | Insert a link (`<a>`). | `insertLink(href)` via link mark | 🟢 |
| `ID_STYLE_NOTE` (resource.h:492) | Insert a footnote reference (`<a type="note">`). Optionally creates target body. | `insertFootnote(id)` | 🟡 |
| `ID_EDIT_INS_SYMBOL` (resource.h:479) | Insert arbitrary Unicode symbol via dialog. | UI + `insertText` | 🟢 |

## 4. Paragraph styles (swap `<p>` class)

Reference: `main.js:1630 StyleCheck` — validates that a paragraph can accept a given style based on its parent class.

| FBE source | Style | PM command | Allowed parents |
|---|---|---|---|
| `main.js:1687 StyleNormal` | `<p>` (no class) | `styleNormal` | section, title, epigraph, stanza, cite, annotation, history |
| `main.js:1699 StyleSubtitle` | `<p class="subtitle">` | `styleSubtitle` | section, stanza, cite, annotation |
| `main.js:1693 StyleTextAuthor` | `<p class="text-author">` | `styleTextAuthor` | cite, epigraph, poem (and must be terminal sibling) |
| `main.js:1705 StyleCode` | `<span class="code">` around selection | `styleCode` | inside text-author, subtitle, p, stanza, table cells |

All go through `main.js:1672 SetStyle` which wraps the change in `window.external.BeginUndoUnit / EndUndoUnit`. ProseMirror `history` plugin handles undo automatically.

## 5. Inline marks (character formatting)

| Toolbar ID (resource.h) | FB2 element | PM mark |
|---|---|---|
| `ID_EDIT_BOLD (32779)` | `<strong>` | `strong` |
| `ID_EDIT_ITALIC (32780)` | `<emphasis>` | `emphasis` |
| `ID_EDIT_STRIK (32840)` | `<strikethrough>` | `strikethrough` |
| `ID_EDIT_SUB (32841)` | `<sub>` | `sub` |
| `ID_EDIT_SUP (32842)` | `<sup>` | `sup` |
| `ID_EDIT_CODE (32844)` | `<code>` | `code` |
| `ID_STYLE_LINK (32800)` | `<a>` | `link` |
| — | `<style name="...">` | `style` mark with `name` attribute |

All implementable via `prosemirror-commands.toggleMark`.

## 6. Binary objects (images)

| FBE source | Operation | Go/Wails equivalent |
|---|---|---|
| `main.js:45 apiAddBinary` | Add a base64 payload with auto-deduplicated id | `App.AddBinaryFromDisk(id, ct, path)` or `App.AddBinaryFromData(id, ct, data)` |
| `main.js:27 apiGetBinary` | Return base64 for `<img src="fbw-internal:#id">` | `App.GetBinaryDataURL(href)` → returns data: URL |
| `main.js:364 SaveImage` | Save a binary to disk | `App.SaveBinaryToDisk(href, path)` |
| `FBShell/ThumbnailHandler.h` | Extract coverpage for Explorer preview | `thumb.Extract()` + native thumbnailer binary per OS |
| Paste image from clipboard | Add binary + insert inline image | `App.AddBinaryFromClipboard()` + `insertInlineImage` |

## 7. Description (metadata form)

The left-hand editable view (`#fbw_desc`) corresponds to FB2 `<description>`. In the port, this is a dedicated Svelte form (not part of ProseMirror).

| Section | FB2 element | Form component | Source (fb2.xsl) |
|---|---|---|---|
| Book | `title-info` | `dialogs/TitleInfo.svelte` | line 297 |
| Source | `src-title-info` | `dialogs/SrcTitleInfo.svelte` | line 422 |
| Document | `document-info` | `dialogs/DocumentInfo.svelte` | line 530 |
| Publish | `publish-info` | `dialogs/PublishInfo.svelte` | line 589 |
| Custom | `custom-info` | `dialogs/CustomInfo.svelte` | line 623 |
| Stylesheets | `stylesheet` | `dialogs/Stylesheets.svelte` | line 82 |
| Binary manager | `binary[]` | `dialogs/BinaryManager.svelte` | line 82 (fieldset#binobj) |

Sub-components used in multiple sections:
- `author_main` / `author` (fb2.xsl:652, 684) → `dialogs/parts/AuthorField.svelte`
- `seq` (fb2.xsl:739) → recursive `dialogs/parts/SequenceField.svelte`
- `authors` list → `dialogs/parts/AuthorList.svelte`
- genre + match → `dialogs/parts/GenreField.svelte`
- coverpage → `dialogs/parts/CoverPicker.svelte`

## 8. Search / Replace

| FBE source | Feature | Go/Wails equivalent |
|---|---|---|
| `SearchReplace.h:10 FRBase` | Pattern + flags (CASE/WHOLE/REVERSE/REGEX) | `search.Flags` struct, `search.Compile` |
| `SearchReplace.h:129 IHTMLTxtRange::findText` | Find in editor text | `prosemirror-search` or custom walker over PM doc |
| Multi-file replace | Apply pattern across folder of .fb2 files | `fbe search --replace` CLI |

## 9. Speller

Reference: `Speller.h:15` (Hunspell), `Speller.h:59–72` (locale list).

| FBE source | Operation | Go/Wails equivalent |
|---|---|---|
| `Speller.cpp` | Init per-language dictionary | `speller.Open(lang, dictsDir)` |
| Red squiggles in editor | Mark misspelled words | PM decoration plugin calling `App.SpellCheck(word, lang)` |
| `CSpellDialog` (Speller.h:78) | Modal with Ignore / Change / Add | `dialogs/Spell.svelte` |
| Per-word replacements (`WordsItem` in Settings.h) | User dictionary | `settings.WordsList` |

## 10. Scripts (user automation)

FBE loads JS files from `FBE/files/Scripts/*.js` via `#userCmd` slot (main.html:9). Calls `apiRunCmd(path)` / `apiProcessCmd(path)` (main.js:612/639).

| Feature | Go/Wails equivalent |
|---|---|
| Load `.js` at runtime | Dynamic import in webview or pass via `App.RunScript(src)` |
| Script API | Expose `App.*` Wails methods to scripts; mirror of old `window.external.*` |
| Script UI | Menu populated from scripts folder; `dialogs/ScriptRunner.svelte` |

⚠️ Scripts are a substantial portion of user workflow (FBE ships hundreds of them in `FBE/files/Scripts/`). Consider keeping script compatibility as a separate project, or provide a migration tool.

## 11. Hotkeys

| FBE source | Format | Go/Wails equivalent |
|---|---|---|
| `Hotkeys.xml` + `Settings.cpp:237+` | XML-serialized bindings (CHotkey objects) | `settings.Hotkeys map[string]string` in JSON config |
| WTL command handlers | Static in C++ | `prosemirror-keymap` plugin + `dialogs/Settings/Hotkeys.svelte` editor |

## 12. Export

| FBE source | Format | Go/Wails equivalent |
|---|---|---|
| `ExportHTML/html.xsl` (493 lines) | HTML | `internal/fb2/export/html` (Go text/template) or libxslt CGo |
| — | EPUB 2/3 | Future: `internal/fb2/export/epub` |
| — | Markdown | Future: `internal/fb2/export/md` |
| — | Plain text | Future: `internal/fb2/export/txt` |

## 13. Platform integration

**Scope: macOS + Linux only.** Windows-specific FBShell components stay with the
original C++ FBE and are not ported. Native platform code may use Rust or C.

| FBE source | Feature | Go/Wails equivalent |
|---|---|---|
| macOS QuickLook | Preview in Finder | `macos/QuickLook/` — separate native target (Swift or Rust) calling `thumb.Extract` via shared library or exec |
| GNOME thumbnailer | Thumbnails in Files (Nautilus, Nemo, Thunar) | `linux/thumbnailer/` — `fbe thumb` wrapper + `.thumbnailer` spec |
| File associations | Double-click .fb2 opens editor | DMG + Info.plist UTIs (macOS); `.desktop` + mimeapps.list (Linux) |
| ~~`FBShell/ThumbnailHandler`~~ | ~~Windows Explorer cover preview~~ | Out of scope — stays in original C++ FBE |

---

## Total surface

| Category | Items | Aggregate complexity |
|---|---|---|
| Load/Save | 5 | 🟡 |
| Containers | 13 | 🟡🔴 — 2 🔴 items (InsertPoem, MergeContainers) |
| Insert at cursor | 5 | 🟢 |
| Styles | 4 | 🟢 |
| Marks | 8 | 🟢 (mostly toggleMark) |
| Binaries | 5 | 🟢 |
| Description form | 7 sections × ~3 subcomponents | 🟡 (mostly form boilerplate) |
| Search | 3 | 🟡 |
| Speller | 4 | 🟡 (CGo setup) |
| Scripts | 3 | 🔴 (API compatibility) |
| Hotkeys | 2 | 🟢 |
| Export | 1+ | 🟡 |
| Platform | 6 | 🔴 (native code per OS) |
| **Total** | **~65 discrete work items** | |

## Non-obvious gotchas (from reading the C++/JS)

1. **`IMarkupServices::BeginUndoUnit`** is called around every structural edit in FBE. ProseMirror already batches a single transaction as one undo step — do not manually wrap commands in additional history units, but DO make sure grouped commands dispatch a single `tr`.
2. **`InflateParagraphs` / `InflateIt`** (main.js:971, FBEview::inflateBlock): FBE tags some `<p>` elements with a marker so MSHTML treats them as editable blocks. ProseMirror doesn't need this; a paragraph is always a block.
3. **MSHTML paste hook (`FBEview::OnPaste`, `OnRealPaste`)** strips Word formatting, converts bitmap to base64, swaps nbsp. In ProseMirror, override `transformPasted` / `transformPastedHTML` / `transformPastedText`.
4. **Section merge** (`MergeContainers`) has SIX distinct cases depending on whether source/target contain nested sections vs. flat paragraphs — read `main.js:2216-2354` carefully before writing the PM equivalent. Budget 2–3 days for this alone.
5. **`<empty-line/>`** is a real FB2 element (not just an empty `<p>`). Handle as a dedicated schema node (`empty_line`) and preserve on save.
6. **Inline vs. block `<image>`** — same FB2 element, different DOM position. `fb2.xsl:224` uses XPath `f:p//f:image | f:subtitle//f:image | f:text-author//f:image` to detect inline; preserve this at parse time by choosing `image_inline` vs. `image_block`.
7. **Encoding round-trip**: FBE preserves the original file encoding if it was not UTF-8 (see `FBDoc.cpp:921 m_encoding`). Decide early whether the port will always write UTF-8 (breaking round-trip but simpler) or preserve encoding.
8. **`apiCleanUp(className)`** (main.js:658) unwraps temp `<div class=X>` wrappers used by scripts. If supporting legacy scripts, implement.
9. **UUID generation** for `<id>` — FBE calls `window.external.GetUUID()`. In Go: `crypto/rand` + RFC 4122 format (or pull `github.com/google/uuid`).
10. **SaveSelectedPos / GetSavedPos** (FBDoc.h:43–45) — FBE stores a marker in the document to restore caret after reload. ProseMirror uses `Selection.toJSON` / `fromJSON` for this.
