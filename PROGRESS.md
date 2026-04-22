# PROGRESS

Revision log for `fbe-go`. Every commit that changes behavior or shape of the
project must add an entry here and bump the version in `wails.json` and
`frontend/package.json` (keep them in sync).

---

## Rev 50 — 2026-04-22 — Dark mode: `background: white` named-color sweep [dev]

Version: **0.1.11**

### Symptom (beta feedback, Dmitry screenshot)

Annotation textarea inside the Description form stayed white in dark
mode, and several small `.aux` "×" delete buttons next to
author/genre/sequence/custom rows rendered as white squares.

### Root cause

Rev 47's sweep used a sed pattern matching `#[0-9a-fA-F]{3,8}` — it
caught 56 hex-color literals but silently skipped **named colors**.
Seven files had `background: white;` as a literal keyword:

- `AnnotationEditor.svelte` — the nested-ProseMirror container.
- `AuthorField.svelte`, `CoverpageField.svelte`, `CustomInfoForm.svelte`,
  `DocumentInfoForm.svelte`, `SequenceField.svelte`,
  `GenreField.svelte` — all on `.aux` (the per-row × remove button).

None of those were flagged because my regex only looked for hex.

### Fix

One sed pass in `frontend/src/description/`:

```
sed -i '' 's/background: white;/background: var(--bg-surface);/g' *.svelte
```

Grep after: `background: white` (and `background: #fff(?!\w)`) empty
across `frontend/src/`.

Not caught by this pass but worth noting — `background: transparent`,
`background: none`, and explicit rgba() literals all remain. Those
are intentionally neutral and should still look right in both modes.

### Verification

- `npm run check` 0/0, `npm run test` 58/58.
- Dmitry to re-check dark-mode rendering after pull: annotation
  editor should be dark-card; ✕ buttons should read as dark chips
  with the ✕ glyph visible.

### Files modified

- `frontend/src/description/AnnotationEditor.svelte`
- `frontend/src/description/AuthorField.svelte`
- `frontend/src/description/CoverpageField.svelte`
- `frontend/src/description/CustomInfoForm.svelte`
- `frontend/src/description/DocumentInfoForm.svelte`
- `frontend/src/description/GenreField.svelte`
- `frontend/src/description/SequenceField.svelte`
- `PROGRESS.md`, `wails.json`, `frontend/package.json`,
  `frontend/package-lock.json`.

### Versions bumped

- `wails.json`                  0.1.10 → 0.1.11
- `frontend/package.json`       0.1.10 → 0.1.11
- `frontend/package-lock.json`  0.1.10 → 0.1.11

---

## Rev 49 — 2026-04-22 — External `<a href>` clicks route through BrowserOpenURL [dev]

Version: **0.1.10**

### Symptom (long-standing)

Clicking a link in the editor (an FB2 `<a l:href="https://…">`)
navigated the webview away from the editor. No back button on
desktop — the app was essentially dead until restart. The Help
dialog was handled with per-link `on:click={openExternal}`
wrappers (Rev 40), but editor content didn't have that.

### Fix

New `frontend/src/runtime/externalLink.ts`:

- `isExternalUrl(href)` — true for `http(s)`, `ftp`, `mailto`,
  `file:`, and protocol-relative `//…`. Fragment-only (`#note`),
  relative (`../foo`), and `javascript:` hrefs pass through to
  default behavior — we don't want to hijack internal navigation
  (future citation scroll, etc.) or execute JS URLs externally.
- `openExternalUrl(url)` — routes via Wails
  `runtime.BrowserOpenURL`; falls back to `window.open` outside
  Wails (plain vite dev / dev-server tab).
- `installExternalLinkHandler()` — document-level capture-phase
  click listener. One install at app bootstrap catches every
  external `<a>` click anywhere: editor content, Help modal,
  future UI. Returns a disposer.

`App.svelte::onMount` installs the handler; cleanup in the
returned destructor. HelpDialog's local `openExternal` + per-link
`on:click` wrappers removed — global handler covers them.

Capture phase chosen so we run before component-level handlers;
only `preventDefault()` (not `stopPropagation()`) so ProseMirror
can still do its cursor-placement thing when the click lands
inside an editor link.

### Verification

- `npm run check` 0/0, `npm run test` 58/58.
- UI not clicked-through from dev env; Dmitry to verify:
  (a) editor link → opens in system browser, editor stays put.
  (b) Help dialog links still open.
  (c) Hypothetical `href="#foo"` still behaves as fragment nav
  (no interception).

### Files added / modified

- `frontend/src/runtime/externalLink.ts` (new)
- `frontend/src/App.svelte` — import + onMount install + cleanup.
- `frontend/src/help/HelpDialog.svelte` — removed local
  `openExternal` + per-link wrappers.
- `PROGRESS.md`, `wails.json`, `frontend/package.json`,
  `frontend/package-lock.json`.

### Versions bumped

- `wails.json`                  0.1.9 → 0.1.10
- `frontend/package.json`       0.1.9 → 0.1.10
- `frontend/package-lock.json`  0.1.9 → 0.1.10

---

## Rev 48 — 2026-04-22 — Dark mode: final hex sweep (outline items) [dev]

Version: **0.1.9**

### What

After Rev 46 + 47, only 4 hardcoded hex colors remained outside
the palette: all in `tree/OutlineItem.svelte`. Converted:

- `#333`    → `var(--fg)`
- `#e5e5da` → `var(--bg-hover)`
- `#1a5490` → `var(--fg-link)`
- `#444`    → `var(--fg-secondary)`

After this rev: every hex in the frontend lives inside the
palette block in `App.svelte`. Components reference only
`var(--xxx)`. Adding / tuning a color now means editing one place.

### Verification

- `grep -rE '#[0-9a-fA-F]{3,8}' src/ | grep -v App.svelte | grep -v '{#each…'` → empty.
- `npm run check` 0/0, `npm run test` 58/58.

### Files modified

- `frontend/src/tree/OutlineItem.svelte`
- `PROGRESS.md`, `wails.json`, `frontend/package.json`,
  `frontend/package-lock.json`.

### Versions bumped

- `wails.json`                  0.1.8 → 0.1.9
- `frontend/package.json`       0.1.8 → 0.1.9
- `frontend/package-lock.json`  0.1.8 → 0.1.9

---

## Rev 47 — 2026-04-22 — Dark mode sweep: description-form sub-components [dev]

Version: **0.1.8**

### What

Rev 46 left 10 description-form sub-components (AuthorField,
CoverpageField, CustomInfoForm, DateField, DocumentInfoForm,
GenreField, PublishInfoForm, SequenceField, TitleInfoForm,
AnnotationEditor) still hard-coded with hex colors. On dark
theme they'd show as light islands inside an otherwise-dark
DescriptionPanel.

Batch-replaced 11 unique hex colors with var(--xxx) across all
10 files via a single sed pass. Mappings:

  #ccc     → --border-input
  #bbb     → --border-button
  #666     → --fg-secondary
  #888     → --fg-muted
  #aaa     → --fg-muted-soft
  #1a5490  → --fg-link
  #fff8e5  → --bg-hover
  #e5e5da  → --border
  #dcdcd0  → --border
  #d5d5cb  → --border
  #fcfbf6  → --bg-card

Grep after: zero hex colors in description/ directory.

### Verification

- `npm run check` 0/0.
- `npm run test` 58/58.
- Visually not tested; Dmitry to check all 10 description tabs in
  both light and dark.

### Files modified

- `frontend/src/description/*.svelte` (10 files)
- `PROGRESS.md`, `wails.json`, `frontend/package.json`,
  `frontend/package-lock.json`.

### Versions bumped

- `wails.json`                  0.1.7 → 0.1.8
- `frontend/package.json`       0.1.7 → 0.1.8
- `frontend/package-lock.json`  0.1.7 → 0.1.8

---

## Rev 46 — 2026-04-22 — Dark mode (Phase 2 A2) [dev]

Version: **0.1.7**

### What

Toolbar gains a theme cycle button (◐/☀/☾) right after Help.
Clicking cycles system → light → dark → system. Choice persists in
`settings.Theme` on the Go side.

When theme is `"system"`, the app live-follows the OS
`prefers-color-scheme` media query — flipping the OS from light to
dark re-themes the editor immediately without restart.

### CSS architecture

Added ~30 semantic CSS custom properties at `:root` (light defaults)
and `[data-theme="dark"]` (dark overrides). Set on
`document.documentElement` via a reactive `$:` block in `App.svelte`
that listens to the computed `effectiveTheme`.

Palette covers: surface/chrome/sidebar/card backgrounds, hover and
active button states, errors pane + errors-title, validation OK
banner, text colors (strong/default/secondary/muted/link), borders
(default/strong/input/button), warn family (raw-block dashed yellow),
highlight (flash-on-jump), drop shadow opacity.

`color-scheme: light | dark` is also declared so native widgets
(scrollbars, form controls, focus rings in WebKitGTK) adapt.

### Refactor sweep

Replaced 56 unique hex colors across 7 Svelte files with the new
var(--xxx) references — each hex mapped to the semantically nearest
variable:

- `App.svelte` — layout chrome, recent-files menu, view-toggle,
  status/err spans.
- `editor/Editor.svelte` — ProseMirror chrome, epigraph/cite/
  annotation colors, table borders, code inline, raw-block hatched
  placeholders.
- `editor/Toolbar.svelte` — the inline-mark toolbar chrome.
- `editor/TableDialog.svelte` — modal.
- `validation/ValidationPanel.svelte` — panel, resizer, errors list,
  XML source line gutter + highlight.
- `help/HelpDialog.svelte` — modal, kbd chips, copy-url buttons,
  links.
- `description/DescriptionPanel.svelte` — tabs, prompt button.
- `tree/DocumentTree.svelte` — empty-state text.

### Settings wiring

- `settings.Settings` gains `Theme string json:"theme"`; `Default()`
  sets `"system"`.
- `App.LoadSettings()` / `App.SaveSettings()` are already exposed —
  no new Go bindings needed.
- `App.svelte::cycleTheme()` writes the new theme into settings
  immediately (no explicit Save step on the user side).
- Wails regen: TS `Settings` type now has `theme: string`.

### Known rough edges

- Dark palette is a first pass; some saturations might feel off on
  OLED. Real-world beta feedback welcome.
- Didn't adjust Description-form sub-components (TitleInfoForm,
  DocumentInfoForm, AnnotationEditor) — they're read-heavy on
  native inputs which inherit `color-scheme: dark` automatically,
  but custom wrappers may need follow-up.
- `color-scheme` media query detection is build-time; no dedicated
  "auto-switch at time of day" — follows OS as-is.

### Verification

- `go build -tags xsd ./...` clean.
- `wails build -tags xsd` — regen picked up `theme: string` on
  `Settings` (used in `LoadSettings()` / `SaveSettings()`).
- `npm run check` 0/0.
- `npm run test` 58/58.
- UI flow not clicked-through — Dmitry to verify theme cycle +
  persistence + OS live-follow on NixOS.

### Files modified

- `internal/fb2/settings/settings.go` — Theme field + Default().
- `frontend/src/App.svelte` — palette, state, toggle button, refactor.
- `frontend/src/editor/Editor.svelte`
- `frontend/src/editor/Toolbar.svelte`
- `frontend/src/editor/TableDialog.svelte`
- `frontend/src/validation/ValidationPanel.svelte`
- `frontend/src/help/HelpDialog.svelte`
- `frontend/src/description/DescriptionPanel.svelte`
- `frontend/src/tree/DocumentTree.svelte`
- `PROGRESS.md`, `wails.json`, `frontend/package.json`,
  `frontend/package-lock.json`.

### Versions bumped

- `wails.json`                  0.1.6 → 0.1.7
- `frontend/package.json`       0.1.6 → 0.1.7
- `frontend/package-lock.json`  0.1.6 → 0.1.7

---

## Rev 45 — 2026-04-22 — Validation errors pane: larger default [dev]

Version: **0.1.6**

### Symptom (beta feedback, Dmitry)

With two XSD errors, only the first was visible in the validation
panel at default layout — the second hid behind a scrollbar and
users had to drag the resizer up to see it. Screenshot confirmed
the errors pane at 35% of panel was ~180px, and two multi-line
libxml2 messages (each wraps to 3+ lines once the namespace URI
is inlined in the string) exceed that.

### Fix

Bumped `.errors` default height in `ValidationPanel.svelte` from
35% to 45% of panel height. Leaves `min-height: 60px` unchanged so
the drag resizer's `panelBounds.min` (60) isn't fought by the CSS
clamp when the user wants to shrink the pane manually.

### Not done here

- Didn't switch to `height: auto; max-height: 45%;` even though it
  would give better UX for single-error cases (pane hugs content,
  no wasted space). Problem: grid-template-rows `auto` + content
  max-height doesn't cap the grid track itself — the row is sized
  by the content's max-content, and max-height only clips the
  element's visible box inside. Plus the drag path sets inline
  `height: Npx` which would have to also disable `max-height` via
  JS. Not worth the complexity for the marginal gain.

### Verification

- `npm run check` 0/0, `npm run test` 58/58.
- Manual eyeball: two-error case now shows both rows without
  scrolling on a typical 1080p window; third error would still
  scroll.

### Files modified

- `frontend/src/validation/ValidationPanel.svelte`
- `PROGRESS.md`, `wails.json`, `frontend/package.json`,
  `frontend/package-lock.json`.

### Versions bumped

- `wails.json`                  0.1.5 → 0.1.6
- `frontend/package.json`       0.1.5 → 0.1.6
- `frontend/package-lock.json`  0.1.5 → 0.1.6

---

## Rev 44 — 2026-04-22 — Recent files (Phase 2 gap) [dev]

Version: **0.1.5**

### What

Toolbar's "Open…" button grows a split-button dropdown: click the main
button for the native file picker, click the `▾` caret for the last 10
opened/saved files. Click an item → opens directly, no picker round-trip.

### Go side

`app.go` gains three things:

- `recordRecentFile(path)` — prepends `path` to `settings.RecentFiles`,
  dedupes earlier occurrences, caps at 10 (const `recentFilesCap`,
  matches FBE's `Settings.h`). Silent on error — recent-list is a
  convenience, not a correctness path, so a settings-write failure
  doesn't block `OpenFile` / `SaveFile`.
- `App.RecentFiles()` — Wails binding returning the MRU list for the
  frontend.
- `App.RemoveFromRecent(path)` — frontend calls this when a recent-menu
  click fails (file moved or deleted) so the menu doesn't keep
  offering a dead entry.

Both `App.OpenFile` and `App.SaveFile` call `recordRecentFile` after
their primary success path.

### Frontend

- `App.svelte`: `recentFiles: string[]` + `recentMenuOpen: boolean`.
  `refreshRecent()` fetches the list; called on mount and after every
  successful Open/Save.
- `openFile()` now accepts an optional `preset?: string` — when set,
  skips `PickFB2ToOpen()` and opens that path directly. On failure
  with a preset, purges the dead entry via `RemoveFromRecent`.
- Split-button UI: main "Open…" + `▾` caret sharing a border. Caret
  is disabled when the list is empty. Clicking the caret toggles a
  positioned dropdown; a transparent full-viewport backdrop closes
  it on outside-click.
- Menu items show basename (bold) + directory (dim, small) so the
  user sees both without hovering for the tooltip.

### What's deferred

- **Thumbnails** next to each item — needs `GetBinaryDataURL` per file
  (a re-parse of every recent .fb2 on menu open). Worth doing but
  wants caching first; out of scope for this rev.
- **"Clear recent" menu item** — simple, skipped for now. Add if beta
  users ask.
- **Keyboard navigation of the dropdown** (arrow keys, Enter) — nice
  a11y polish, deferred.

### Verification

- `go build -tags xsd ./...` — clean.
- `wails build -tags xsd` — regen pulled `RecentFiles` and
  `RemoveFromRecent` into `frontend/wailsjs/go/main/App.d.ts`
  automatically.
- `npm run check` 0/0, `npm run test` 58/58.

### Files added / modified

- `app.go` — three new methods + `recordRecentFile` helper + integrate
  into OpenFile/SaveFile success paths.
- `frontend/src/App.svelte` — state, refresh wiring, split-button UI,
  dropdown menu, styles.
- `PROGRESS.md`, `wails.json`, `frontend/package.json`,
  `frontend/package-lock.json`.

### Versions bumped

- `wails.json`                  0.1.4 → 0.1.5
- `frontend/package.json`       0.1.4 → 0.1.5
- `frontend/package-lock.json`  0.1.4 → 0.1.5

---

## Rev 43 — 2026-04-22 — New app icon (blue squircle + book + code brackets) [dev]

Version: **0.1.4**

### What

`build/appicon.png` replaced with a new 1024×1024 RGBA master: a
dark-blue squircle holding an open book with inline `<>` code
brackets on the right page. The glyph says "book editor with
structured/XML underpinnings" without the "AI-assistant" or
"generic notes" ambiguity the two alternatives carried.

### Pipeline

The source PNG from the image generator came at 1254×1254 **without
an alpha channel** — corners were filled with srgb(232,232,231), an
off-white that would show as a visible square on dark-mode docks.
ImageMagick pass (via `nix-shell -p imagemagick`) floodfills from
(0,0) with 12% fuzz to match the near-white corners, replaces them
with transparency, then downscales to 1024×1024:

```
magick input.png \
  -alpha set \
  -fuzz 12% -fill none -floodfill +0+0 "srgb(232,232,231)" \
  -resize 1024x1024 \
  build/appicon.png
```

Result is RGBA with proper transparent corners around the squircle
silhouette, ready for both macOS (bundle generates `.icns` from it
during `wails build`) and Linux (GTK launcher picks up the PNG
directly).

### Verification

- `file build/appicon.png` → `PNG image data, 1024 x 1024, 8-bit/color RGBA`.
- `sips` reports `hasAlpha: yes`.
- `wails build -tags xsd` regenerated
  `build/bin/fbe-go.app/Contents/Resources/iconfile.icns` (987 KB,
  timestamp post-build). Bundle launches.
- UI un-touched; `go test` and `npm test` not re-run — purely an
  asset swap.

### Files added / modified

- `build/appicon.png` — the 1024×1024 master (binary, tracked).
- `PROGRESS.md`, `wails.json`, `frontend/package.json`, `frontend/package-lock.json`

### Versions bumped

- `wails.json`                  0.1.3 → 0.1.4
- `frontend/package.json`       0.1.3 → 0.1.4
- `frontend/package-lock.json`  0.1.3 → 0.1.4

---

## Rev 42 — 2026-04-22 — MIT LICENSE + NOTICE.md + credits [dev]

Version: **0.1.3**

### What

Closes the licensing story the beta release left at "TBD".

- `LICENSE` at repo root — full MIT License text, © 2026 Dmitry
  Gordiyevsky.
- `NOTICE.md` — exhaustive third-party attribution: bundled FB2 XSD
  schemas (© 2004 Dmitry Gribov, 2-clause BSD, full text reproduced
  inline to satisfy the "binary redistribution must reproduce notice"
  clause), Go deps (Wails v2, lestrrat-go/libxml2, golang.org/x/*),
  native C libs (libxml2, GTK 3, WebKitGTK, Cocoa), frontend deps
  (Svelte, Vite, ProseMirror, Vitest, svelte-check, TypeScript), Nix
  flake dependencies, and an inspiration-not-code-reuse note for the
  classic FBE.
- `README.md` — replaced the "TBD" license placeholder with a real
  License section + a "Legacy & acknowledgements" section that
  thanks Gribov, evpobr + FBE team, Wails (Lea Anthony), ProseMirror
  (Marijn Haverbeke), libxml2 (Daniel Veillard), and
  lestrrat-go/libxml2 (Daisuke Maki). Points at NOTICE for the
  formal list.
- `frontend/package.json` — `"license": "MIT"` field added.
- `HelpDialog.svelte` — header line extended to
  `Version X.Y.Z-beta · MIT-licensed · LICENSE · NOTICE` with the
  two links opening via the existing `openExternal(url)` helper
  (points at the main branch on GitHub). Added a small credits
  footer in the About section.

### No code changes

Pure docs / metadata. No behavior changes. Version bump is the
standard rev-cadence discipline per CLAUDE.md.

### Verification

- `npm run check` 0/0, `npm run test` 58/58.
- LICENSE + NOTICE render correctly on github.com once pushed.

### Files added / modified

- `LICENSE` (new), `NOTICE.md` (new)
- `README.md` — License + Legacy & acknowledgements sections
- `frontend/src/help/HelpDialog.svelte` — license line + credits
- `frontend/package.json` — `license` field
- `PROGRESS.md`, `wails.json`, `frontend/package-lock.json`

### Versions bumped

- `wails.json`                  0.1.2 → 0.1.3
- `frontend/package.json`       0.1.2 → 0.1.3
- `frontend/package-lock.json`  0.1.2 → 0.1.3

---

## Rev 41 — 2026-04-22 — Explicit copy-URL buttons in Help dialog [dev]

Version: **0.1.2**

### Symptom

After Rev 40 the Help links OPEN externally (BrowserOpenURL works),
but users can't COPY a link URL. Right-click → "Copy Link Address"
is unreliable in Wails webviews: WKWebView's context menu is
suppressed in release bundles, and WebKitGTK's default menu on
NixOS doesn't always include the link-copy entry.

### Fix

Each Resources link in HelpDialog.svelte now has an inline
`[ copy ]` button to its right that writes the URL to the clipboard
via `navigator.clipboard.writeText()`, with a
`document.execCommand("copy")` textarea fallback for older webviews
that lack the async Clipboard API. Success flashes the button to
`✓ copied` for 1.5s.

Resources list refactored into a Svelte `{#each}` over a
`[{label, url}, …]` array so the markup is DRY; 3-column flex row
keeps the `copy` button aligned right even when the label wraps
on a narrow dialog.

Left the inline "Wails v2" link in the prose untouched — prose
links don't warrant the copy-button chrome, and their URL is short
enough to paste from the rendered href anyway.

### Verification

- `npm run check` 0/0.
- `npm run test` 58/58.
- UI: Dmitry to verify on NixOS that clicking `copy` copies the URL
  into the system clipboard (paste-test in another app).

### Files modified

- `frontend/src/help/HelpDialog.svelte`
- `PROGRESS.md`, `wails.json`, `frontend/package.json`, `frontend/package-lock.json`

### Versions bumped

- `wails.json`                  0.1.1 → 0.1.2
- `frontend/package.json`       0.1.1 → 0.1.2
- `frontend/package-lock.json`  0.1.1 → 0.1.2

---

## Rev 40 — 2026-04-22 — Help-dialog links open externally, text is selectable [dev]

Version: **0.1.1**

### Symptoms (beta feedback, Dmitry)

1. Links in the Help modal didn't do anything on click — no new tab,
   no Go-side action.
2. Text inside the modal (version string, link text, kbd labels)
   couldn't be selected or copied.

### Root causes

1. Wails' WKWebView / WebKitGTK doesn't route `<a href="http…">`
   clicks to the system browser. That's deliberate — if it did,
   random links in user content would open browser windows. The
   contract is: the frontend intercepts the click and calls
   `runtime.BrowserOpenURL(url)`, which on macOS fires `open`, on
   Linux fires `xdg-open`.
2. Chromed-up dialogs inherited `cursor: auto` weirdly + `user-select`
   wasn't explicitly set on `.dialog`. WebKit on macOS sometimes treats
   descendants of `role="button"` elements (the backdrop) as
   non-selectable by default, and our backdrop is `role="button"`.

### Fixes

- `openExternal(e, url)` in HelpDialog.svelte: `preventDefault`, then
  dynamic-import `wailsjs/runtime/runtime` and call `BrowserOpenURL`.
  If Wails isn't running (plain vite dev), falls back to
  `window.open(url, "_blank", "noopener,noreferrer")`. Every `<a>`
  in the dialog wires an `on:click={...openExternal(…)}` handler.
- `.dialog` CSS now declares `user-select: text` and
  `-webkit-user-select: text` explicitly, plus `cursor: auto` to
  reset from the inherited `cursor: pointer` the `role="button"`
  backdrop was bleeding in.
- All link `rel` attributes bumped to `noreferrer noopener` (browser
  fallback hardening).

### Verification

- `npm run check` — 0/0.
- `npm run test` — 58/58 (no test for Wails-side link flow; manual
  on Dmitry's NixOS box needed).

### Files modified

- `frontend/src/help/HelpDialog.svelte`
- `PROGRESS.md`, `wails.json`, `frontend/package.json`, `frontend/package-lock.json`

### Versions bumped

- `wails.json`                  0.1.0 → 0.1.1
- `frontend/package.json`       0.1.0 → 0.1.1
- `frontend/package-lock.json`  0.1.0 → 0.1.1

---

## Rev 39 — 2026-04-22 — Help dialog + v0.1.0-beta milestone [dev → main]

Version: **0.1.0** (product); git tag: **v0.1.0-beta**

### What

Toolbar gains a `Help` button right after `Export HTML…`. Clicking it
opens a modal (`frontend/src/help/HelpDialog.svelte`) with:

- App name + version (read from `frontend/package.json` via JSON
  import so the value stays in sync with the semver-bumped file).
  Displayed as "Version 0.1.0-beta" with the `-beta` suffix hardcoded
  in the template until we cut `0.1.0` final.
- A short "what this is" paragraph.
- A keyboard-shortcuts table (Save / Save As / Undo / Redo / Bold /
  Italic / Strikethrough / Sub / Sup). Modifier key resolves to ⌘ on
  macOS, Ctrl elsewhere. **Table is hand-maintained** — if
  `Editor.svelte`'s keymap or `App.svelte`'s Cmd-S handler change,
  the table must change too.
- Resource links (repo, FB2 spec, original FBE).

Modal pattern copied from `TableDialog.svelte` for consistency:
backdrop click / Escape / × button all close. Scoped keydown with
`if (!open)` so Escape doesn't steal focus globally.

### Milestone — v0.1.0-beta

First release cut. Version bumped 0.0.38 → 0.1.0 in `wails.json`,
`frontend/package.json`, `frontend/package-lock.json`. Git tag
`v0.1.0-beta` annotates the main-branch merge commit (the `-beta`
prerelease marker lives only in the tag, not in the version files,
so npm and wails both stay semver-happy).

Release scope — everything landed by Rev 38 plus this Help dialog:
full FB2 round-trip (including Raw fallback, mixed section content,
exact block/section interleaving), writer fidelity (xmlns:l prefix,
mixed-content whitespace), XSD validation with clickable errors and
XML source panel, supplementary unknown-element scanner, Nix flake
for macOS + Linux (NixOS-ready shell), description form with rich
annotation editor, HTML export, paste cleanup, native-webview
spellcheck.

Status line in `README.md` and `CLAUDE.md` updated from "Phase 3 MVP
+ Phase 4 polish in progress" to "v0.1.0-beta shipped". See
`docs/PHASES.md` for what's deferred to 0.2.0 (structured libxml2
errors, Section.Children order-preserving parent refactor, editable
XML view, Hunspell wiring).

### Verification

- `npm run check` — 0 errors, 0 warnings.
- `npm run test` — 58/58.
- `wails build -tags xsd` — clean production bundle.
- UI flow not clicked-through from dev env; Dmitry to verify Help
  opens, shortcut table renders with correct modifier per OS,
  Escape/backdrop/× all close the modal.

### Files added / modified

- `frontend/src/help/HelpDialog.svelte` (new)
- `frontend/src/App.svelte` — import + toolbar button + state + mount
- `README.md`, `CLAUDE.md` — status line updates
- `PROGRESS.md`, `wails.json`, `frontend/package.json`, `frontend/package-lock.json`

### Versions bumped

- `wails.json`                  0.0.38 → 0.1.0
- `frontend/package.json`       0.0.38 → 0.1.0
- `frontend/package-lock.json`  0.0.38 → 0.1.0

---

## Rev 38 — 2026-04-22 — Supplementary unknown-element scanner [dev]

Version: **0.0.38**

### Why

After Rev 37 the UI correctly showed 3 XSD errors for Dmitry's
`book-broken.fb2` — matching the CLI's output on disk bytes. But one of
the three misspelled empty-lines (`<empty-lyne/>`, inside the second
section right after a subsection) was silently absent from the errors
list despite being present in the XML source pane. libxml2's
content-model recovery: after the first violation in a content group
it enters "don't cascade" mode and stops reporting later unknowns in
that group. Not our bug — but the user experience is "the editor lost
one of my errors".

### Fix

Added a supplementary scanner (`xsd.FindUnknownElements`) that regexes
the serialized source for opening tags and flags any name outside the
bundled FictionBook 2.0 vocabulary. Structure-agnostic, so libxml2's
DFA recovery can't hide any unknown — every occurrence shows up.

Combined with libxml2's output via `xsd.MergeXSDAndUnknown`, which
dedupes by `(line, tag-name)`: if libxml2 already reported an element
at a given line, our scanner's entry is dropped (libxml2's message is
richer, including the full `Expected is one of (…)` list).

### Implementation

- `internal/fb2/xsd/unknown.go` (new, no build tag so it works in stub
  builds too):
  - `knownFB2Elements` — hand-maintained set of ~55 valid FB2 tags.
  - `FindUnknownElements(src []byte) []ValidationError` — regex
    `<([a-zA-Z][\w-]*)` scans src, filters by the vocab map, emits one
    entry per occurrence. Skips closing tags / PIs / comments via the
    alphabetic-first-char requirement.
  - `MergeXSDAndUnknown(xsdErrs, unknowns)` — builds a
    `(line, tag)` seen-set from libxml2 messages (tag via the regex
    `Element '(?:\{[^}]*\})?([^']+)'`), then filters unknowns against it.
  - `byteOffsetToLineCol` — small helper for 1-based positions.

- `app.go::ValidateCurrent` — after `xsd.Validate`, calls
  `xsd.FindUnknownElements(src)` and merges via `MergeXSDAndUnknown`.

### Tests

`internal/fb2/xsd/unknown_test.go` — 5 cases (build-tag-agnostic, so
they run in both plain and `-tags xsd` modes):

1. Reports every occurrence of three distinct misspellings (the exact
   scenario Dmitry hit on `book-broken.fb2`).
2. Skips known tags — a legit document produces zero entries.
3. Skips comments and processing instructions — no false positives.
4. Line/column are 1-based, pointing at the `<` of the tag.
5. `MergeXSDAndUnknown` correctly dedupes the libxml2/scanner overlap:
   libxml2 entry preserved, same-line same-tag scanner entry dropped,
   different-line scanner entry kept.

### Vocabulary maintenance

`knownFB2Elements` is hand-maintained. Keep it in sync with
`SchemaFiles` (`FictionBook.xsd` + friends) — if a new element is
legitimized in a future FB2 revision, add it to the map or it'll get
flagged as unknown. An XSD-introspection pass was considered and
rejected: the schema is ~60 elements total, hand-maintenance is
cheaper than shipping runtime XSD walking on every Validate.

### Out of scope

- **Unknown attribute scan.** Same idea but for attribute names
  instead of elements. FB2 files rarely have unknown attributes in
  practice; revisit if a real case surfaces.
- **Deeper error categorization.** We currently expose a flat
  `[]ValidationError`. A future UI pass could render libxml2 entries
  (red) distinctly from scanner entries (orange) so users can tell
  "this element fits somewhere but not here" from "this element
  doesn't exist at all".

### Verification

- `go build ./...` and `go build -tags xsd ./...` clean.
- `go test ./...` / `go test -tags xsd ./...` — all packages green;
  5 new tests in the xsd package pass.
- `npm run check` 0/0, `npm run test` 58/58.
- UI flow not clicked-through from dev env. Dmitry to re-open
  `book-broken.fb2` and confirm: error list now contains an entry for
  `empty-lyne` (in addition to the libxml2 entries for title-info,
  empty-lune, empty-lane) — total 4 items, matching the three
  misspellings plus the missing title-info.

### Files modified

- `internal/fb2/xsd/unknown.go` (new) — scanner + merger
- `internal/fb2/xsd/unknown_test.go` (new) — 5 unit tests
- `app.go` — ValidateCurrent call-chain expanded
- `PROGRESS.md`, `wails.json`, `frontend/package.json`, `frontend/package-lock.json`

### Versions bumped

- `wails.json`                  0.0.37 → 0.0.38
- `frontend/package.json`       0.0.37 → 0.0.38
- `frontend/package-lock.json`  0.0.37 → 0.0.38

---

## Rev 37 — 2026-04-22 — doc.Section.Body — ordered children (sections + blocks) [dev]

Version: **0.0.37**

### Why

Rev 34 let PM carry mixed `section + block` content past the XSD's
strict `(section+ | block+)` choice. But round-trip still lost the
exact *order* inside such a section: Go's `encoding/xml` splits matches
into the two disjoint slices `Sections []Section` and `Blocks []Block`,
and the writer emits all Sections before all Blocks (field-declaration
order). A source with `<empty-lane/> <section/> <empty-lyne/>` came
back as `<section/> <empty-lane/> <empty-lyne/>`. Content preserved,
position canonicalized.

### What

Collapsed `doc.Section.Sections + Blocks` into a single ordered slice
`Body []Block`. Subsections live inside `Body` as a `Block` whose new
`Section *Section` field is non-nil — the same variant-discrimination
pattern `Block` already used for Paragraph, Poem, Cite, Raw, etc.

### Go changes

`internal/fb2/doc/doc.go`:
- `Section`: `Sections` + `Blocks` removed; added `Body []Block` with
  `xml:"-"` and custom `UnmarshalXML` / `MarshalXML`.
- `UnmarshalXML`: reads header (id attr, title?, epigraph*, image?,
  annotation?) then collects everything else — including `<section>` —
  into `Body` via `Block.UnmarshalXML`, in source order.
- `MarshalXML`: emits header in XSD-required order, then iterates
  `Body` and calls `Block.MarshalXML` directly (EncodeElement with
  an empty StartElement errors "missing name" because Block has no
  XMLName field; direct call bypasses that).
- `Block`: new `Section *Section` variant.
- `Block.UnmarshalXML`: new `"section"` case.
- `Block.MarshalXML`: new case that emits `<section>` for the variant.

### Go consumers

- `internal/fb2/export/html/html.go::writeSection`: replaced the
  if/else (nested sections → recurse; else → writeBlock) with a single
  walk of `s.Body` that dispatches on `b.Section != nil`. Ordered
  output matches source regardless of mixing.
- `internal/fb2/writer/writer_test.go::check`: `Sections[0].Blocks`
  → `Sections[0].Body` in the body-count assertion.

### Frontend changes

Wails regenerates TS models from the Go struct, so `Section.Body: Block[]`
propagates automatically. Hand-written types + code that used the old
names had to follow:

- `frontend/src/fb2/types.ts`: `Section` — `Sections` / `Blocks`
  removed, `Body?: Block[] | null` added. `Block` — new
  `Section?: Section | null` field.
- `frontend/src/fb2/sample.ts`: each section's `Blocks:` / `Sections:`
  lists reshaped into a single `Body:` with `{ Section: { ... } }`
  wrappers for the subsection entries.
- `frontend/src/editor/parse.ts::buildSection`: single loop over
  `s.Body`; relies on `buildBlock` to dispatch each item.
  `buildBlock` gains `if (b.Section) return buildSection(b.Section);`.
- `frontend/src/editor/serialize.ts::buildSection`: emits `{ Body: [...] }`
  in PM-child order. Section-type children become `{ Section: ... }`
  entries. `buildBlock` gains `case "section"`.
- `frontend/src/tree/outline.ts::buildSection`: filters `s.Body` by
  `b.Section` to enumerate subsections for the outline tree; path
  indices still count only subsection children (matches
  `Editor.svelte::findNodePos`'s "i-th section child" semantics).
- `frontend/src/editor/commands.test.ts` (23 tests) — bulk
  `Blocks:` → `Body:` rename, plus four manual nested-sections
  unwrapping to `Body: [{ Section: {...} }]`.
- `frontend/src/editor/serialize.test.ts` — assertions that used
  `section.Blocks.find(...)` updated to `section.Body.find(...)`;
  the "preserves nested section count" and "preserves nested section
  with annotation" tests now filter `Body` by Section variant.
- `frontend/src/editor/raw.test.ts` — shared `minimalBook` helper
  + the mixed-content regression test reshaped to the new Body
  structure.

### Tests

New `internal/fb2/writer/section_order_test.go::TestSectionBodyPreservesInterleaving`:
parses a section with `[p, section, p, section, p]` alternating, writes
it, and asserts the nine substring markers appear in source order in
the output. Before Rev 37 this test would have seen the two sections
bunched at the top of the section's body.

### Doc note

CLAUDE.md Architecture section gains a "Section order invariant" entry
next to the existing "Lossless fallback invariant" so future code
changes don't accidentally revert the pair back to Sections+Blocks.

### Verification

- `go build -tags xsd ./...` clean.
- `go test -tags xsd ./...` clean — new interleaving test passes.
- `wails build -tags xsd` — bindings regenerated, Section now carries
  `Body: Block[]` in `frontend/wailsjs/go/models.ts`.
- `npm run check` 0/0.
- `npm run test` 58/58 (54 existing + 3 raw + 1 mixed — unchanged count
  since the raw mixed-section test simply changed shape of its input).
- UI flow not clicked-through from dev env. Dmitry to re-open
  `book-broken.fb2` and verify: XML source pane shows all three
  misspelled elements now IN THEIR ORIGINAL POSITIONS
  (empty-lane before section, empty-lyne after section — not bunched
  at the top).

### Files modified

- `internal/fb2/doc/doc.go` — Section refactor + Block.Section variant
- `internal/fb2/export/html/html.go` — writeSection Body walk
- `internal/fb2/writer/writer_test.go` — field rename
- `internal/fb2/writer/section_order_test.go` (new) — interleaving regression
- `frontend/src/fb2/types.ts`, `frontend/src/fb2/sample.ts`
- `frontend/src/editor/parse.ts`, `frontend/src/editor/serialize.ts`
- `frontend/src/editor/commands.test.ts`, `frontend/src/editor/serialize.test.ts`, `frontend/src/editor/raw.test.ts`
- `frontend/src/tree/outline.ts`
- `CLAUDE.md`
- `PROGRESS.md`, `wails.json`, `frontend/package.json`, `frontend/package-lock.json`

### Versions bumped

- `wails.json`                  0.0.36 → 0.0.37
- `frontend/package.json`       0.0.36 → 0.0.37
- `frontend/package-lock.json`  0.0.36 → 0.0.37

---

## Rev 36 — 2026-04-22 — Cleanup: compactMixedContent tag assembly via fmt.Appendf [dev]

Version: **0.0.36**

Cosmetic: Rev 35's `compactMixedContent` assembled each rewritten tag
as a 9-line sequence of `out = append(out, …)` calls — readable but
C-ish. Replaced with a single `fmt.Appendf(nil, "<%s%s>%s</%s>", …)`
call. `fmt.Appendf` (Go 1.19+) appends formatted output directly to a
nil byte slice, returning the grown slice — one allocation, zero
intermediate strings. Idiomatic Go, same output, easier to review.

`text/template` considered and rejected for this use — it's the right
tool when there's user-facing template text, loops, or conditionals,
not for four positional interpolations inside one function. Would
have added an import, a package-level `*template.Template`, and two
lookups per call without saving any real lines.

No behaviour change. Tests unchanged and still green.

### Versions bumped

- `wails.json`                  0.0.35 → 0.0.36
- `frontend/package.json`       0.0.35 → 0.0.36
- `frontend/package-lock.json`  0.0.35 → 0.0.36

---

## Rev 35 — 2026-04-22 — Writer fidelity: xmlns:l prefix + mixed-content whitespace [dev]

Version: **0.0.35**

### Why

Diff between Dmitry's on-disk `book-broken.fb2` and the XML-source pane
(which reflects `writer.Write(a.current)`) showed two byte-level drifts
that survived even faithful content round-trip:

1. Source used `xmlns:l="http://www.w3.org/1999/xlink"` at the root and
   `<a l:href="...">` in content. Writer output dropped the `l` prefix
   declaration, then re-emitted `xmlns:xlink="http://www.w3.org/1999/xlink" xlink:href="..."`
   on every single `<a>`. Functionally equivalent XML, but different
   bytes per save and uglier on inspection.
2. `<p>before <strong>bold</strong>, <emphasis>italic</emphasis> tail</p>`
   in the source became
   ```
   <p>before 
     <strong>bold</strong>, 
     <emphasis>italic</emphasis> tail
   </p>
   ```
   in writer output. Go's `xml.Encoder.Indent` inserts `\n` + indent
   before every child element, regardless of whether the surrounding
   context is block or inline. Browsers collapse the whitespace so
   visual rendering is unchanged, but file bytes change on every save.

### Fix 1 — xmlns:l prefix preserved

- `doc.Link.Href` struct tag changed from
  `xml:"http://www.w3.org/1999/xlink href,attr"` to `xml:"-"`. The
  Go-namespace-aware tag was what triggered `xmlns:xlink` auto-decl
  on each `<a>`.
- `doc.Link.MarshalXML` now emits `xml.Attr{Name: xml.Name{Local: "l:href"}, Value: l.Href}`
  — a literal attribute name with the `l:` prefix baked into the local
  name, bypassing Go's namespace machinery entirely. This is correct
  only because the FictionBook root declares `xmlns:l` up-front.
- `writer.Write` bypasses Go's default root-element emission (which
  would insist on auto-picking `xmlns:xlink`). It emits the root tag
  literally:
  `<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0" xmlns:l="http://www.w3.org/1999/xlink">\n`,
  then uses `EncodeElement` on each top-level child (Stylesheets,
  Description, Bodies, Binaries) with a Local-only StartElement so Go
  doesn't redeclare the default xmlns on each child either. Closes
  with a manual `\n</FictionBook>\n`.
- Link unmarshal still accepts any of `l:href`, `xlink:href`, bare
  `href`, or a namespace-resolved variant with `a.Name.Space == NSXLink`.
  So any real-world file parses fine regardless of prefix choice.

### Fix 2 — mixed-content whitespace collapsed

Tried the "clean" approach first — toggle `e.Indent("", "")` around
inline children inside `Paragraph.MarshalXML` / `Cell.MarshalXML`, then
restore. Discovered a `encoding/xml` quirk: `writeIndent` short-circuits
when both prefix and indent are empty, and that short-circuit **skips
the depth-- bookkeeping** on the closing tag. Toggling mid-marshal thus
desyncs the encoder's internal `p.depth` counter from its tag stack,
and every subsequent block sibling renders one indent level too deep
per toggle. Since `p.depth` isn't exposed, there's no clean reset.

Reverted the toggle and went with a narrowly-scoped post-process
regex pass in `writer.Write` instead:

- `mixedContentTagRE` matches a leaf mixed-content container
  (`<p>`, `<subtitle>`, `<th>`, `<td>`, `<v>`, `<text-author>`, `<date>`)
  including its attributes and inner content, using a non-greedy match
  plus an end-tag backreference pattern. These containers never nest
  another of the same type, so non-greedy is safe.
- `innerNewlineIndentRE` inside the match strips every `\n[ \t]*`
  occurrence. That's exactly the shape Go's encoder indent inserts
  before each child. Other whitespace (e.g. a literal space between
  text and `<strong>`) isn't matched because the pattern requires a
  newline; single-line spaces are preserved.

Trade-off: a literal `\n` inside `<p>` chardata (rare — FB2 uses
`<empty-line/>` for paragraph breaks) would also be collapsed. If we
ever find real-world files that rely on that, revisit with a
token-aware pass.

### Why not do it as a custom MarshalXML

Documented in a comment on `Paragraph.MarshalXML`: the toggle approach
is appealing but `xml.Encoder` doesn't support it without reflection
into private state. The post-process pass is localized (one function,
two regexes) and runs once on the finished buffer — easy to audit,
easy to test.

### New tests

`internal/fb2/writer/fidelity_test.go`:

- `TestXLinkPrefixRoundTrip` — asserts root declares `xmlns:l`,
  `<a l:href="...">` uses `l:` prefix, and no per-element
  `xmlns:xlink=` redecl nor `xlink:href=` attribute.
- `TestMixedContentInlineWhitespace` — asserts three mixed-content
  fragments (`<p>...`, `<th>...`, `<td>...`) appear with text and
  inline marks all on the same line, plus regression guards against
  the old `before\n` / `\n        <strong>` / `\n      </p>` shapes.
- `TestBlockLevelIndentStillWorks` — sanity that the post-process
  pass doesn't swallow block-level indent. Pins each known nesting
  level (`\n  <description>`, `\n    <title-info>`, `\n  <body>`,
  etc.) so a future regex tweak can't accidentally flatten the
  whole doc.

### Out of scope

- **Exact interleaving preservation of `section` / `block` siblings
  within a section** — still requires the `doc.Section.Children
  []SectionChild` refactor tracked in Rev 34 notes. Unrelated to
  writer fidelity.

### Verification

- `go build -tags xsd ./...` clean.
- `go test -tags xsd ./...` — all packages green; three new
  `TestXLinkPrefix…` / `TestMixedContent…` / `TestBlockLevelIndent…`
  tests pass.
- `npm run check` 0/0, `npm run test` 58/58.
- Manual sanity dump of a small parse→write round-trip confirms the
  expected shape (xmlns:l declared once at root; `<a l:href=...>` on
  its own; mixed-content paragraphs on one line; block-level nesting
  preserved).

### Files modified

- `internal/fb2/doc/doc.go` — Link struct tag + MarshalXML change; reverted
  the Paragraph/Cell indent-toggle experiment with a comment explaining
  why we didn't take that path
- `internal/fb2/writer/writer.go` — manual root emission, buffer +
  post-process pass, inline helpers + comments
- `internal/fb2/writer/fidelity_test.go` (new) — three fidelity tests
- `PROGRESS.md`, `wails.json`, `frontend/package.json`, `frontend/package-lock.json`

### Versions bumped

- `wails.json`                  0.0.34 → 0.0.35
- `frontend/package.json`       0.0.34 → 0.0.35
- `frontend/package-lock.json`  0.0.34 → 0.0.35

---

## Rev 34 — 2026-04-22 — Allow mixed section content (section + block siblings) [dev]

Version: **0.0.34**

### Symptom

After Rev 33, `book-broken.fb2` (three misspelled `<empty-line/>`s as
`<empty-lune/>`, `<empty-lane/>`, `<empty-lyne/>`) still produced only
2 XSD errors in the app (vs. 3 from the CLI on raw bytes). XML-source
pane confirmed it: the first raw block (inside a section with flat
blocks only) survived, but the other two were silently missing — they
sat inside a section that also had a nested `<section>`, and the
round-trip through PM dropped them.

### Root cause — two layers

1. **PM schema:** `section` content was
   `(title | epigraph | image_block | annotation)* (section+ | block+)`
   — a strict XSD-aligned choice: either only subsections or only flat
   blocks, not mixed. PM dropped the block-level children of a section
   that also had a nested subsection.

2. **`parse.ts::buildSection`:** mirrored the strict choice in an
   explicit `if (s.Sections.length > 0) { …emit only Sections… } else
   { …emit only Blocks… }`. Even if the PM schema were relaxed, this
   code would still silently lose Blocks on sections that had
   subsections.

### Fix

- Schema: `(title | epigraph | image_block | annotation)* (section | block)+`.
  The inline comment explains we're deliberately wider than the FB2 XSD
  so real-world files with technically-invalid-but-present mixed content
  survive a round-trip; Validate still flags the XSD breach.
- `parse.ts::buildSection`: emit Sections first, then Blocks,
  unconditionally. Order matches Go's `encoding/xml` field-declaration
  order in `doc.Section` (`Sections` field declared before `Blocks`),
  so save-and-reopen is idempotent.
- `serialize.ts::buildSection` already routed PM children into the
  right Go-side slice (Sections vs Blocks) per node type — no change
  needed.

### Note on ordering

Go's `doc.Section` stores Blocks and nested Sections in separate
slices. Original inter-leaving (e.g., `block, section, block`) is lost
at the struct level — we only know "this section had these blocks
and these subsections". On re-emit we emit all subsections, then all
blocks. A source file whose section was `[empty-lane, section, empty-lyne]`
round-trips as `[section, empty-lane, empty-lyne]`. Content is
preserved; position relative to each other is canonicalized. Fixing
this would require changing `doc.Section` to carry a single ordered
`Children` slice — a larger refactor, tracked as potential future work.

### Test

New `raw.test.ts` case: "preserves raw blocks flanking a nested section".
Feeds a Go-shaped section with both `Sections: [nestedSection]` and
`Blocks: [Raw(empty-lane)]`, round-trips through `fb2ToPMDoc` →
`pmDocToFB2`, and asserts:

- Outer section still has at least one entry in both `Blocks` and `Sections`.
- The raw block's `localName` ("empty-lane") is preserved.

Pre-fix: the raw block was silently dropped by `buildSection`'s
if/else; test failed with `expected > 0, got undefined`. Post-fix: passes.

### Out of scope (deferred)

- **Exact interleaving preservation.** Needs a `doc.Section.Children []SectionChild`
  refactor, which cascades into parser / writer / Wails bindings.
  Separate rev.
- **Other serialization drift** Dmitry spotted in the same XML pane —
  `xmlns:l → xmlns:xlink` per-`<a>`, and whitespace-around-inline inside
  `<p>`. Both harmless for display but change file bytes on save.
  Dedicated rev (writer-indent refactor).

### Verification

- `npm run check` — 0 errors, 0 warnings.
- `npm run test` — 58/58 (57 old + 1 new in raw.test.ts).
- `go test -tags xsd ./...` — unchanged, green.
- UI round-trip not clicked-through from dev env; Dmitry to re-open
  `book-broken.fb2` and confirm the XML pane now shows all three of
  `<empty-lune/>`, `<empty-lane/>`, `<empty-lyne/>` and that the errors
  list has 3 (plus the title-info one = 4 total).

### Files modified

- `frontend/src/editor/schema.ts` — relaxed section content model
- `frontend/src/editor/parse.ts` — always emit Sections + Blocks, not one-or-the-other
- `frontend/src/editor/raw.test.ts` — new mixed-content regression case
- `PROGRESS.md`, `wails.json`, `frontend/package.json`, `frontend/package-lock.json`

### Versions bumped

- `wails.json`                  0.0.33 → 0.0.34
- `frontend/package.json`       0.0.33 → 0.0.34
- `frontend/package-lock.json`  0.0.33 → 0.0.34

---

## Rev 33 — 2026-04-22 — Lossless fallback in PM (raw_block / raw_inline) [dev]

Version: **0.0.33**

### Why

Rev 31 validation debugging on `book-broken.fb2` (renamed every
`<empty-line/>` to misspellings `<empty-lune/>`, `<empty-lane/>`,
`<empty-lyne/>`) hit a silent-drop bug in the desktop round-trip:

- `./build/fbe validate book-broken.fb2` (CLI, raw bytes) reported the
  three unknown elements as expected.
- Opening the same file in the app and clicking Validate showed **none
  of them** — they weren't in the errors, not in the XML pane either.

Root cause: the CLAUDE.md "Lossless fallback invariant" held on the Go
side (`Block.UnmarshalXML` / `unmarshalInlineContent` routed unknown
elements into `Block.Raw` / `Inline.Raw`; `writer.Write` re-emitted
them), but `frontend/src/editor/parse.ts::buildBlock` and `pushInline`
never read the `.Raw` field. They returned `null` for blocks without a
typed match and skipped `Inline` entries without a recognized child —
so Raw got dropped the moment the doc went through the PM editor.
When `validate()` in `App.svelte` pushed the PM-round-tripped doc back
to Go via `UpdateDocument`, Raw was already gone; `writer.Write(a.current)`
produced a clean FB2 with no ghost elements; the validator saw no
errors about them.

### Fix

Two new PM schema nodes — `raw_block` and `raw_inline` — that stash the
full `RawElement` as a JSON-stringified attribute:

```
raw_block: atom, group: "block",   attrs: { raw, localName }
raw_inline: atom, group: "inline", attrs: { raw, localName }, inline: true
```

They render as a hatched-yellow placeholder with the element's local
name (`<empty-lune/>`) and a tooltip explaining the element is unknown
and preserved verbatim for save. Non-editable (`contenteditable="false"`,
`atom: true`) but selectable — user can delete them if they really want
to strip the unknown content.

### Wiring

`parse.ts`:

- `buildBlock` — new trailing case `if (b.Raw) return buildRawBlock(b)`.
- `buildBlockList` (`titleOnly` path) — also handles Raw so title-level
  extensions survive.
- `pushInline` — new trailing case handling `i.Raw`.
- Helper `buildRawBlock` returns an `N.raw_block.create({ raw, localName })`
  node with `JSON.stringify(b.Raw!)` in the attr.

`serialize.ts`:

- `buildBlock` — new case `"raw_block"` calls `decodeRaw(node.attrs.raw, "Block")`.
- `buildInlines` — handles `raw_inline` the same way.
- New helper `decodeRaw` — JSON.parses the attr with defensive guards;
  returns null if the blob is missing / malformed (block silently
  dropped rather than corrupting the document — but practically never
  happens since `parse.ts` always stringifies a valid shape).

`schema.ts`:

- New `raw_block` / `raw_inline` nodes.
- Extended content models to allow `raw_block` in `title`, `epigraph`,
  `cite`, `annotation` — matching every container that holds `Block[]`
  on the Go side. `section` already allows it via `block+` (raw_block is
  in the "block" group). Inline containers (`paragraph`, `subtitle`,
  `verse`, `text_author`, `date`, `table_cell`) already use `inline*`,
  and `raw_inline` is in the "inline" group, so those auto-include it.

`types.ts`:

- New `RawElement` interface (XMLName, Attrs, Items) mirroring the Go
  struct so Wails unmarshals cleanly.
- `Block.Raw?` and `Inline.Raw?` fields added.

`Editor.svelte`:

- New `.raw-block` / `.raw-inline` styles — hatched yellow background,
  dashed ocher border, monospace font. Also a selected-node outline
  variant for the PM `ProseMirror-selectednode` class.

### Test

New `frontend/src/editor/raw.test.ts` (3 cases):

1. A block with `Raw` survives PM round-trip and keeps its local name
   between two Paragraphs.
2. A complex Raw block preserves attributes (`data-source="Flibusta"`)
   and nested Elem items (`<b>content</b>`) exactly.
3. An inline Raw (`<ruby rb="漢" rt="kan">漢</ruby>`) inside a paragraph
   survives with both attrs and inner text, and the surrounding text
   segments (`"before "`, `" after"`) still flank it.

### Out of scope

- No XSD-valid editing of raw blocks from inside PM. They're a
  preservation mechanism, not an editing one. Editing requires the
  (future) raw-XML editing pane.
- No UI affordance to promote a raw_block into a typed node. If a
  misspelled `<empty-lune/>` is fixed, it happens externally (text
  editor or future XML view).
- Raw-block positions that would violate the FB2 XSD (e.g., at body
  level) still violate. We only promise *loss-less round-trip of what
  was in the source file*, not *schema-validity of arbitrary content*.

### Verification

- Go tests 56/56 (unchanged, no Go code touched this rev).
- `npm run check` — 0 errors, 0 warnings.
- `npm run test` — 57/57 (54 old + 3 new in raw.test.ts).
- UI placeholder rendering not clicked-through from dev env; Dmitry to
  re-open `book-broken.fb2` and confirm: (a) XML source pane now shows
  `<empty-lune/>`, `<empty-lane/>`, `<empty-lyne/>` in the output (not
  silently stripped); (b) errors list includes them with proper line
  numbers (via Rev 31 heuristic); (c) the misspelled elements appear in
  the editor as hatched-yellow `<empty-lune/>` placeholders instead of
  vanishing.

### Files added / modified

- `frontend/src/fb2/types.ts` — RawElement interface + Raw on Block/Inline
- `frontend/src/editor/schema.ts` — raw_block / raw_inline nodes + content-model
- `frontend/src/editor/parse.ts` — buildRawBlock + inline Raw handling
- `frontend/src/editor/serialize.ts` — decodeRaw + raw_block / raw_inline cases
- `frontend/src/editor/Editor.svelte` — `.raw-block` / `.raw-inline` CSS
- `frontend/src/editor/raw.test.ts` (new) — 3 round-trip tests
- `CLAUDE.md` — frontend side of the Lossless invariant
- `PROGRESS.md`, `wails.json`, `frontend/package.json`, `frontend/package-lock.json`

### Versions bumped

- `wails.json`                  0.0.32 → 0.0.33
- `frontend/package.json`       0.0.32 → 0.0.33
- `frontend/package-lock.json`  0.0.32 → 0.0.33

---

## Rev 32 — 2026-04-22 — Table cells: fix `<Children><Text>` ghost tags [dev]

Version: **0.0.32**

### Symptom

Running Validate on the bundled SAMPLE_BOOK produced six XSD errors, all
pointing at a `<Children>` element inside `<th>` / `<td>`:

    L67:13  Element '{…}Children': This element is not expected.
    Expected is one of ( strong, emphasis, style, a, strikethrough,
    sub, sup, code, image ).

XML source pane showed:

```xml
<th>
  <Children>
    <Text>Елемент</Text>
  </Children>
</th>
```

Capital-C `<Children>` and capital-T `<Text>` — Go struct field names
leaking into the XML, not valid FB2.

### Root cause

`doc.Cell` had `Children []Inline xml:",any"` and no custom
`MarshalXML` / `UnmarshalXML`. Go's default encoder for a `,any`-tagged
slice of structs **uses the field name as the element tag** (there's no
XMLName on the nested `Inline`), and then for each `Inline` value it
emits every non-zero field as a nested element using the Go field name
— so `Text string` became `<Text>`, `Strong *Paragraph` would have
become `<Strong>`, etc.

The existing inline containers (`Paragraph`, `StyleInline`, `Link`)
sidestep this by carrying `Children []Inline xml:"-"` and providing a
pair of `(Un)MarshalXML` methods that route through
`marshalInlineContent` / `unmarshalInlineContent`. `Cell` was added
later and skipped that pattern.

### Fix

Applied the same pattern to `Cell`:

- `Children []Inline xml:",any"` → `Children []Inline xml:"-"`.
- New `(*Cell).UnmarshalXML` — captures `th`/`td` from `start.Name`,
  reads six attributes explicitly, delegates mixed content to
  `unmarshalInlineContent`.
- New `(Cell).MarshalXML` — emits only the local name (`xml.Name{Local:
  "th"}` or `{Local: "td"}`) so the parent's default namespace applies
  and we don't re-declare `xmlns=".../fictionbook/2.0"` on every cell.
  Clears `start.Attr` before re-adding attrs so nothing inherited from
  the caller leaks through. Uses `marshalInlineContent` for children.

### Test

New `TestTableRoundTripPreservesThTdTags` parses a minimal doc with one
header row (`<th colspan="2">` with nested `<strong>`) and two data
cells, round-trips through parser→writer, and asserts:

- `<th colspan="2">` / `</th>` present
- `<td>`, `cell one`, `cell two` present
- `<strong>bold</strong>` preserved inside the header
- **No `<Children>` / `</Children>` / `<Text>` / `</Text>` in the output**
  (direct regression guard for the old bug)
- **No `<th xmlns=…>` / `<td xmlns=…>`** (parent namespace must apply;
  catches the secondary issue I hit mid-fix when I initially copied the
  full `xml.Name` including `Space`)

### Why existing writer tests missed this

The pre-existing writer-level tests (`TestRoundTrip`,
`TestRawFallback*`, `TestWriterOutputIsSchemaValid`) exercise sections,
paragraphs, and raw-element fallback but none of them touch `<table>`.
The schema-validity test could have caught it if its fixtures included
tables — worth adding a table in that corpus as a follow-up.

### Why it was only visible in the SAMPLE_BOOK flow

On the Wails side, `App.OpenFile` parses a real `.fb2` from disk using
Go's `parser.Parse`, which calls `Cell.UnmarshalXML` — BUT pre-fix that
path *also* relied on `xml:",any"`, and `,any`-based unmarshal happens
to work reasonably for Inlines because Go matches sub-elements by their
struct field tags. So reading a real `.fb2` round-trips correctly in
memory. The bug manifested only on the *marshal* side.

The frontend-SAMPLE path stresses only the marshal side:
`Editor.currentFB()` → `App.UpdateDocument(fb)` → `writer.Write(fb)` for
the XML preview. No parse-from-XML step in between, so marshal ran on
a doc where every Cell was a fresh Go struct built by
`editor/serialize.ts::buildTable` — no XMLName namespace baggage, just
the plain `{XMLName: {Local: "th"}, Children: [...]}` JSON. That fed
directly into the buggy marshal path and produced the `<Children>`
output Dmitry screenshotted.

### Files modified

- `internal/fb2/doc/doc.go` — Cell (Un)MarshalXML + rationale comments
- `internal/fb2/writer/table_test.go` (new) — regression guard
- `PROGRESS.md`, `wails.json`, `frontend/package.json`, `frontend/package-lock.json`

### Versions bumped

- `wails.json`                  0.0.31 → 0.0.32
- `frontend/package.json`       0.0.31 → 0.0.32
- `frontend/package-lock.json`  0.0.31 → 0.0.32

---

## Rev 31 — 2026-04-22 — Validation line numbers + title-info faithful round-trip [dev]

Version: **0.0.31**

Bundles two related fixes spotted during Rev 30 testing on NixOS: Dmitry
opened a deliberately-broken `.fb2` (with `<title-info>` removed) to
test validation and saw:

1. Error at `L0:0` — line numbers weren't being extracted.
2. A ghost `<title-info>` in the XML pane that wasn't in his source
   file, and the error message talked about `book-title` instead of
   the expected `genre` / `title-info`.

### Part A — Populate Line/Column in ValidationError

Root cause: `lestrrat-go/libxml2` registers a plain
`xmlSchemaValidityErrorFunc` (via `MY_accumulateErr` in its cgo), which
only forwards the formatted message string. libxml2's native
`xmlErrorPtr` (with `line` / `int2` fields) is discarded inside the
binding before we ever see it. Switching to the structured-error
callback would require patching the binding — too heavy.

Pragmatic fix: post-process. After collecting the `[]error` from
`schema.Validate`, parse the QName out of the message with a regex —
typical shape `Element '{ns}name': …` or bare `Element 'name': …` —
and scan the source bytes for the first `<name[\s/>]` occurrence.
Byte offset → (line, column), both 1-based. Falls back to (0, 0) when
no element name can be extracted.

Covered by a new `TestLocateElementInSource` with four cases including
two fall-through paths (unrelated message; element not present in src).
The heuristic is not perfect (multiple identical tag names → we pick the
first), but for FictionBook's typical "missing / unexpected at
description level" errors it lands on the right line.

### Part B — Description.TitleInfo as *TitleInfo

Root cause: `Description.TitleInfo` was a value type (`TitleInfo`, not
`*TitleInfo`), so Go's encoding/xml always emitted the element on
marshal. Two string children (`BookTitle`, `Lang`) lacked `,omitempty`
so they emitted as `<book-title></book-title>` / `<lang></lang>` even
when zero-value. A file with no `<title-info>` therefore round-tripped
as an empty-but-present title-info, and the validator reported
`<book-title>` as unexpected (first child didn't match the XSD's
required-first `<genre>`) instead of telling the user their title-info
was missing entirely.

Fix: `*TitleInfo` with `,omitempty` + nil-guards at every access site.

Access sites updated:
- `internal/fb2/thumb/thumb.go` — nil check before `Coverpage` deref.
- `internal/fb2/export/html/html.go` — `writeHeader` reads
  `BookTitle`/`Lang` via a nil-tolerant local; `writeDescription`
  returns early when `TitleInfo == nil`.
- `frontend/src/description/DescriptionPanel.svelte` — wrapped
  `<TitleInfoForm bind:info={…}>` in `{#if fb.Description.TitleInfo}`
  with an "Add title info" prompt in the else branch (mirrors the
  existing SrcTitleInfo pattern). Refactored the two "add empty
  title-info object" code paths to share one `emptyTitleInfo()` helper.
- `frontend/src/fb2/types.ts` — `TitleInfo?: TitleInfo | null`.
- `frontend/src/editor/serialize.test.ts` — optional-chain the read.

Wails regen verified: `TitleInfo?: TitleInfo` propagated to
`wailsjs/go/models.ts` automatically.

### Documented invariant

CLAUDE.md Architecture section now carries an "Absent-section invariant"
note next to the existing "Lossless fallback invariant". Keeps future
readers from re-introducing the ghost-element bug.

### Verification

- `go build -tags xsd ./...` / `go vet -tags xsd ./...` clean.
- `go test -tags xsd ./...` — all existing packages green; new
  `TestLocateElementInSource` (4 sub-cases) passes.
- `wails build -tags xsd` — full bundle clean; regen pulled
  `TitleInfo?: TitleInfo` into the generated TS models as expected.
- `npm run check` — 0 errors, 0 warnings.
- `npm run test` — 54/54.
- UI flow again not clicked-through from dev env — Dmitry to open
  `book-broken.fb2` and confirm the post-fix behavior: XML pane should
  no longer show an empty `<title-info>`, and the error should now
  point at the `<description>` line (because `title-info` is *missing*,
  not present-but-wrong).

### Known heuristic limits (Part A)

- If a doc contains many elements with the same local name, we map every
  error about that tag to the first occurrence. Good enough for typical
  description-level errors; could mislead on body-level errors in long
  docs.
- Messages that don't quote an element (e.g. attribute-value errors) fall
  back to (0, 0). Acceptable — better than "always 0" across the board,
  and the message text still conveys the issue.
- When `lestrrat-go/libxml2` eventually exposes structured errors, swap
  the regex heuristic for the native `xmlErrorPtr` fields.

### Files modified

- `internal/fb2/doc/doc.go` — TitleInfo pointer + rationale comment
- `internal/fb2/thumb/thumb.go`, `internal/fb2/export/html/html.go` — nil guards
- `internal/fb2/xsd/xsd_libxml2.go` — line/col heuristic
- `internal/fb2/xsd/xsd_libxml2_test.go` — `TestLocateElementInSource`
- `frontend/src/fb2/types.ts` — TitleInfo optional
- `frontend/src/description/DescriptionPanel.svelte` — conditional + helper refactor
- `frontend/src/editor/serialize.test.ts` — optional chain
- `CLAUDE.md` — absent-section invariant
- `PROGRESS.md`, `wails.json`, `frontend/package.json`, `frontend/package-lock.json`

### Versions bumped

- `wails.json`                  0.0.30 → 0.0.31
- `frontend/package.json`       0.0.30 → 0.0.31
- `frontend/package-lock.json`  0.0.30 → 0.0.31

---

## Rev 30 — 2026-04-22 — Draggable resizer between XML and errors panes [dev]

Version: **0.0.30**

### Symptom

Rev 29's ValidationPanel capped the errors section at `max-height: 42%`
with no way to grow it. With a long error list (or a long first-line
message that wraps to many visual lines), the errors pane felt cramped
and half the rows were hidden behind an inner scrollbar, even though
the XML pane above had empty space to spare.

### Fix

Horizontal drag-handle between the XML pane and the errors pane:

- **Pointer events** (not old mouse events) with `setPointerCapture` so
  the drag follows the cursor even outside the handle, and touch / pen
  input work the same way. `touch-action: none` disables native scroll
  on touch.
- **Grid layout** changed from `2rem 1fr auto` to
  `2rem 1fr auto auto` (title, XML, resizer, errors). Errors pane keeps
  a CSS default of `height: 35%` with `min-height: 60px`; once the user
  drags, an inline `style="height: Npx"` takes over.
- **Keyboard support** — the handle is `role="separator"` +
  `aria-orientation="horizontal"` + `tabindex="0"`. Focus it and use
  ↑ / ↓ (10px step, or 40px with Shift) to adjust. `aria-label`
  explains the contract.
- **Body-level cursor / user-select** are forced to `ns-resize` / `none`
  during drag so the cursor stays consistent and text doesn't accidentally
  get selected if the pointer leaves the handle mid-drag. Reset on drag
  end and in `onDestroy`.
- Default errors-pane height raised from 42% max to 35% default (still
  user-adjustable). Worth re-tuning if it feels off in practice.

### A11y lint

Svelte's `a11y-no-noninteractive-tabindex` and
`a11y-no-noninteractive-element-interactions` fire on `<div role="separator">`.
The role is explicitly interactive per WAI-ARIA when paired with
keyboard handling — the lint is over-strict. Suppressed with two
`<!-- svelte-ignore -->` directives (same precedent as Rev 23's
TableDialog). Rest of the file still passes unsilenced.

### Verification

- `npm run check` — 0 errors, 0 warnings.
- `npm run test` — 54/54.
- UI not clicked-through from dev env (same limitation as Rev 29);
  Dmitry to sanity-test the drag on NixOS.

### Files modified

- `frontend/src/validation/ValidationPanel.svelte`
- `PROGRESS.md`, `wails.json`, `frontend/package.json`, `frontend/package-lock.json`

### Versions bumped

- `wails.json`                  0.0.29 → 0.0.30
- `frontend/package.json`       0.0.29 → 0.0.30
- `frontend/package-lock.json`  0.0.29 → 0.0.30

---

## Rev 29 — 2026-04-22 — XML source panel + clickable validation errors [dev]

Version: **0.0.29**

### Why

Before Rev 29 the Validate button produced a single status-bar line
with the first error truncated to 120 characters — *"XSD: N error(s)
— first: Element '{http://www.gribuser.ru/xml/fictionbook/2.0}document-
info': This element is not expected. Expected is ( {http:/"*. The
namespace URI alone ate half the budget; the rest of the errors weren't
shown at all. There was also no way to inspect the serialized XML of
the in-memory document, which is needed any time behaviour diverges
between "what the editor thinks the doc is" and "what the writer
produces".

### What

New right-side drawer (`frontend/src/validation/ValidationPanel.svelte`)
with two sections:

1. **XML source (read-only).** Line-numbered `<pre>` of
   `writer.Write(a.current)`. Monospace, syntax-neutral (no highlighter
   dependency added — deliberate scope cut).
2. **Validation errors.** Full list below the XML pane, each row a
   `<button>` showing `L<line>:<col>` + full wrapped message. Clicking
   scrolls the XML pane to that line and flashes it yellow for 2.5s.

Opens when Validate is clicked. Stays open; explicit × closes it.

### Go side

Two new `App` methods, both operating on the **in-memory** document so
unsaved edits are reflected:

- `App.SerializeCurrent() (string, error)` — serializes `a.current` via
  `writer.Write` into a string.
- `App.ValidateCurrent() ([]xsd.ValidationError, error)` — serializes
  then validates. Line numbers in returned errors align exactly with
  the `SerializeCurrent` output, so the click-to-jump mapping is
  trivial (no offset arithmetic).

The older `App.Validate(path)` stays for any future "validate a file
without opening it" flow; the UI no longer uses it.

### Frontend wiring

`validate()` in `App.svelte` rewritten to:

1. Push the latest PM state to Go via `UpdateDocument` (so serialize
   reflects current unsaved edits).
2. `Promise.all([SerializeCurrent(), ValidateCurrent()])` in parallel.
3. Open the panel with both results set.

`Validate` button's `disabled` condition loosened: was `!currentPath`
(required a saved file), now `!fb` (any loaded doc, saved or not).

### Type note on `ValidationError`

Wails generates TS types from the Go JSON tags, not Go field names, so
`xsd.ValidationError{Line,Column,Message}` with `json:"line"` etc. becomes
TS `{ line, column, message }` (lowercase). Old status-bar code already
used `.message` so the wire was correct; only my first-cut panel was
wrong — fixed before landing.

### Out of scope (on purpose)

- **Syntax highlighting for the XML pane.** Would need prismjs/highlight.js
  — real cost in bundle size for a developer-assist feature. Skipped.
- **Editable XML view.** Requires two-way sync between the textual XML
  and the ProseMirror schema, conflict resolution on partial docs, etc.
  Significantly more work; this rev stays read-only to ship the
  high-value portion immediately.
- **Validation on every keystroke.** Expensive (XSD + libxml2 per edit).
  Still on-demand via the Validate button; revisit once there's a
  real-world complaint.

### Verification

- `go build -tags xsd ./...` clean.
- `wails build -tags xsd` clean — binding regeneration picked up the two
  new methods (`SerializeCurrent`, `ValidateCurrent`) as expected.
- `npm run check` — 0 errors, 0 warnings.
- `npm run test` — 54/54 green (existing suite; no new tests added for
  the Svelte component yet, since there's no component-testing harness
  wired up — worth adding separately).
- **UI not visually verified from the dev environment** — I can type-check
  and build the bundle but can't click through the flow. Dmitry to test
  the golden path on NixOS: Open .fb2 → Validate → panel opens with XML
  + error list → click an error → XML pane scrolls and flashes the
  target line. Edge case: empty errors list should show "XSD valid ✓"
  in the errors area.

### Files added / modified

- `app.go` — new `SerializeCurrent`, `ValidateCurrent`
- `frontend/src/validation/ValidationPanel.svelte` (new)
- `frontend/src/App.svelte` — state, `validate()` flow, layout
- `CLAUDE.md` — short frontend-arch note about the panel
- `PROGRESS.md`, `wails.json`, `frontend/package.json`, `frontend/package-lock.json`

### Versions bumped

- `wails.json`                  0.0.28 → 0.0.29
- `frontend/package.json`       0.0.28 → 0.0.29
- `frontend/package-lock.json`  0.0.28 → 0.0.29

---

## Rev 28 — 2026-04-22 — Pin libxml2 to 2.13.x on Nix (binding vs. 2.15 ABI) [dev]

Version: **0.0.28**

### Symptom

After Rev 27 on NixOS, `wails dev -tags 'xsd webkit2_41'` failed at
`go mod tidy` / bindings stage:

```
# github.com/lestrrat-go/libxml2/clib
clib.go:1889:75: cannot use _cgo4 (variable of type *_Ctype_xmlNodePtr)
  as **_Ctype_struct__xmlNode value in argument to _Cfunc_xmlParseInNodeContext
```

(Plus a pair of cosmetic deprecation warnings about `xmlIndentTreeOutput` —
noise, not the cause.)

### Root cause

nixpkgs-unstable ships libxml2 **2.15.1**. Upstream libxml2 changed the
C signature of `xmlParseInNodeContext` somewhere in the 2.14 → 2.15 range
from accepting `xmlNodePtr` to `xmlNodePtr*` (double indirection). The
Go binding `github.com/lestrrat-go/libxml2` (which `-tags xsd` drags in
via `internal/fb2/xsd/xsd_libxml2.go`) was written against the old
signature and passes `&ret` where the new API wants a different shape.
Result: hard compile error, not just deprecation noise.

The binding itself hasn't been updated to match — its last commit (pseudo
version `v0.0.0-20260304224138-bb3877930cf7`, ~2026-03-04) still has the
old calling pattern. Other distros ship libxml2 2.9–2.12 which compiles
fine, so this is a nixpkgs-unstable / bleeding-edge issue, not a
universal Linux regression.

### Fix

`flake.nix` — Linux `linuxDeps`: `libxml2` → `libxml2_13` (2.13.9, the
last release before the ABI break). This lets the binding compile as it
always did; nothing else needs touching.

### Alternatives considered and rejected

1. **Bump `lestrrat-go/libxml2` in `go.mod`.** There's no newer pseudo-
   version that fixes the issue; the binding hasn't caught up. Trying a
   bleeding-edge commit from their main branch would couple us to an
   unstable ref with no tags.
2. **`go mod replace` with a local patch.** Would require maintaining a
   patched fork for an indirect dependency. High ongoing cost, low value.
3. **Drop XSD validation on Linux.** Breaks feature parity; `-tags xsd`
   is the canonical way to get real schema validation everywhere.
4. **Use a pure-Go XSD validator.** No production-grade library exists
   for full XML Schema 1.0 (what FictionBook.xsd requires). Not happening.

### Doc updates

- `CLAUDE.md` — new Platform-notes bullet "libxml2 pin on Nix" explaining
  why `libxml2_13` is pinned, when to revisit, and that the pin is
  Nix-specific (other distros aren't affected).
- `CLAUDE.md` — NixOS bullet now references `libxml2_13` instead of
  `libxml2` and cross-links to the pin note.

### Revisit trigger

When `github.com/lestrrat-go/libxml2` lands a commit that fixes
`xmlParseInNodeContext`'s calling convention for libxml2 2.14+,
bump `go.mod`, switch the flake back to `pkgs.libxml2`, and drop both
the inline comment in `flake.nix` and the pin note in `CLAUDE.md`.

### Verification

- `nix flake check --all-systems` — clean on all four target systems.
- Dmitry to re-run `wails dev -tags 'xsd webkit2_41'` on his NixOS box
  after `git pull` + `nix develop`.

### Files modified

- `flake.nix`
- `CLAUDE.md`
- `PROGRESS.md`, `wails.json`, `frontend/package.json`, `frontend/package-lock.json`

### Versions bumped

- `wails.json`                  0.0.27 → 0.0.28
- `frontend/package.json`       0.0.27 → 0.0.28
- `frontend/package-lock.json`  0.0.27 → 0.0.28

---

## Rev 27 — 2026-04-22 — Linux / NixOS fixes: webkit2_41 tag + GSettings schemas [dev]

Version: **0.0.27**

### Symptoms on NixOS

Two separate breakages surfaced when Dmitry first ran `wails dev` on his
NixOS box after Rev 26:

1. **Build failed** with `No package 'webkit2gtk-4.0' found` — pkg-config
   couldn't resolve the webkit dep even though `webkitgtk_4_1` was in the
   devshell.
2. **Binary built after fix #1, then crashed on Open/Save click** with
   `GLib-GIO-ERROR: Settings schema 'org.gtk.Settings.FileChooser' is not
   installed` → SIGTRAP during CGo.

### Root causes

**(1) Build-tag gap.** Wails v2's CGo directives are gated by a build tag:
`#cgo !webkit2_41 pkg-config: webkit2gtk-4.0` / `#cgo webkit2_41 pkg-config:
webkit2gtk-4.1`. Without `-tags webkit2_41`, the build asks for the older
`4.0` ABI (libsoup 2.x), which modern distros and nixpkgs don't ship.
Rev 26 noted the Nix dependency but not the tag — a half-landed fix.

**(2) GSettings discovery.** GTK's `GtkFileChooserNative` reads the
`org.gtk.Settings.FileChooser` schema at dialog-open time. On NixOS this
schema lives at `${gtk3}/share/gsettings-schemas/${gtk3.name}/glib-2.0/schemas/`,
but `XDG_DATA_DIRS` inside `nix develop` only points at the system's
`/run/current-system/sw/share` — which on a fresh NixOS box doesn't carry
GTK's schemas. The schema load fails, glib panics, the WKWebView host
process gets SIGTRAP.

### Fixes

**Flake (`flake.nix`):**

- Added `gsettings-desktop-schemas` to the Linux build inputs (for common
  GNOME schemas beyond GTK's own).
- Linux `shellHook` now exports `XDG_DATA_DIRS` prepended with the
  Nix-store schema paths for `gtk3`, `glib`, and `gsettings-desktop-schemas`.
  Guarded by `pkgs.lib.optionalString pkgs.stdenv.isLinux` so macOS is
  untouched.
- `shellHook` echo reminders now show the correct `-tags webkit2_41`
  invocations per platform (reminders include the tag on Linux, omit on
  macOS — where it's harmless anyway).

**Docs:**

- `CLAUDE.md` Commands section — both `wails dev` and `wails build`
  examples now include the tag, with a short note that it's a no-op on
  macOS.
- `CLAUDE.md` Platform notes — new bullet explaining the `webkit2_41`
  CGo tag requirement; expanded NixOS bullet describing the
  GSettings/XDG_DATA_DIRS issue with the exact error message (so a
  future reader googling the error string lands on the right place).
- `README.md` Nix section — updated command examples, added the "no-op on
  macOS" hint so readers don't strip the tag in a cross-platform setup.

### Build tag safety on macOS

Verified that `-tags webkit2_41` is harmless on macOS: all files that
reference the tag (`internal/frontend/desktop/linux/*.go`) are gated by
`//go:build linux`, so the tag is silently ignored in darwin builds.
This lets us document a single cross-platform command instead of forking
by OS.

### Verification

- `nix flake check --all-systems` — clean on all four target systems.
- Shell entry on darwin prints the macOS-flavoured hint (no tag);
  Linux path would print the tagged variant (can't verify from macOS,
  but `pkgs.stdenv.isLinux` logic is standard nixpkgs idiom).

### Files modified

- `flake.nix` (no `flake.lock` change — only Nix expression logic tweaked)
- `CLAUDE.md`, `README.md`
- `PROGRESS.md`, `wails.json`, `frontend/package.json`, `frontend/package-lock.json`

### Versions bumped

- `wails.json`                  0.0.26 → 0.0.27
- `frontend/package.json`       0.0.26 → 0.0.27
- `frontend/package-lock.json`  0.0.26 → 0.0.27

### Not attempted

- `wrapGAppsHook3` setup-hook approach: considered but rejected in favor
  of the explicit `XDG_DATA_DIRS` export. Setup-hook behaviour inside
  `mkShell` (vs. proper derivations) varies by nixpkgs version, and the
  explicit export is easier to audit and debug.
- `GDK_PIXBUF_MODULE_FILE` / `GIO_EXTRA_MODULES` exports: not needed yet
  (we don't load external pixbuf loaders or GIO VFS modules). Add if
  icon/theme rendering breaks.

---

## Rev 26 — 2026-04-22 — Nix flake with cross-platform dev shell [dev]

Version: **0.0.26**

### Why

Dmitry wants to run fbe-go on a NixOS box. Rather than an ad-hoc
`nix-shell -p …` command, add a reproducible `flake.nix` with a pinned
`flake.lock` so anyone with Nix/Lix can `nix develop` and get a working
build environment on Linux or macOS without touching system packages.

### What

New `flake.nix` exposes `devShells.default` for four systems —
`x86_64-linux`, `aarch64-linux`, `x86_64-darwin`, `aarch64-darwin`.
The shell includes:

- `go_1_25` (matches `go.mod` pin 1.25.0)
- `nodejs_22` (for the frontend build)
- On Linux only (via `pkgs.lib.optionals pkgs.stdenv.isLinux`):
  `pkg-config`, `gtk3`, `webkitgtk_4_1`, `libxml2`. macOS uses the
  system WKWebView + libxml2 from Xcode CLT, so those aren't needed.

`shellHook` installs the Wails CLI into `$GOPATH/bin` on first entry
(guarded by `command -v wails` so it's a one-time cost per shell). This
matches the canonical `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
instruction in `CLAUDE.md`, rather than using the older `wails 2.11.0`
packaged in nixpkgs.

`flake.lock` pins `nixpkgs-unstable` at commit `b86751bc…` (2026-04-16).

### Verification

- `nix flake check --all-systems` — all four target systems evaluate cleanly.
- Locally entered the darwin shell; `go`, `node`, `wails` all resolve.

### Docs

- `README.md` — new "Nix / NixOS" section under Prerequisites; bumped
  Go prerequisite from 1.24+ to 1.25+ (matches `go.mod` pin, was stale).
- `CLAUDE.md` — new "NixOS / Nix" platform note explaining flake layout
  and the "consider `nix flake update` after Wails bumps" hint.

### Files added / modified

- `flake.nix` (new), `flake.lock` (new)
- `README.md`, `CLAUDE.md`
- `PROGRESS.md`, `wails.json`, `frontend/package.json`, `frontend/package-lock.json`

### Versions bumped

- `wails.json`                  0.0.25 → 0.0.26
- `frontend/package.json`       0.0.25 → 0.0.26
- `frontend/package-lock.json`  0.0.25 → 0.0.26

### Out of scope

- A `packages.default` / `apps.default` output for `nix build` / `nix run`
  — building a Wails app as a pure Nix derivation is non-trivial (CGo +
  vite-generated frontend + `go:embed`), and dev-shell was the ask.

---

## Rev 25 — 2026-04-21 — Bump Wails v2.9.2 → v2.12.0; re-verify UTType crash; README status refresh [dev]

Version: **0.0.25**

### Upgrade

Bumped `github.com/wailsapp/wails/v2` from `v2.9.2` to `v2.12.0` in `go.mod`,
pulling in the usual transitive updates (`labstack/echo` v4.10.2 → v4.13.3,
`golang.org/x/*`, `go-webview2` 1.0.16 → 1.0.22, `samber/lo` v1.38.1 → v1.49.1,
new `git.sr.ht/~jackmordaunt/go-toast/v2` for the notifications API, etc.).

### Verification

- `go build -tags xsd ./...` — clean (CGo against v2.12.0 Obj-C sources compiles).
- `go vet ./...` — clean.
- `go test ./...` and `go test -tags xsd ./...` — all existing tests pass.
- `cd frontend && npm run check` — svelte-check: 0 errors, 0 warnings.
- `cd frontend && npm run test` — vitest: 54/54 green.
- `wails build -tags xsd` — full production bundle.

### Multi-dot dialog crash — **still present in v2.12.0**

Investigated whether the bump lets us restore `*.fb2.zip` in `PickFB2ToOpen`.
**It does not.** The `USE_NEW_FILTERS` code path in
`internal/frontend/desktop/darwin/WailsContext.m` (lines 594–607 of v2.12.0)
is byte-identical to v2.9.2:

```objc
UTType *t = [UTType typeWithFilenameExtension:filter];  // nil for "fb2.zip"
[contentTypes addObject:t];                              // NSInvalidArgumentException
```

No nil-guard was added upstream. Restoring the multi-dot pattern would
reintroduce the Rev 21 crash. Current workaround (`*.fb2` only; archives via
"All files") stays.

### Docs

- `README.md` — replaced the stale "Skeleton only" status with a reflection
  of actual Phase 3 MVP completion state; points readers at `PROGRESS.md`,
  `docs/PHASES.md`, `docs/OPERATIONS.md`.
- `CLAUDE.md` — widened the platform-note version range from "Wails v2.9.2"
  to "Wails v2.9.2–v2.12.0"; generalised the dialog-wrapper bullet to
  "Wails v2" (not version-specific); added a re-verified-on-v2.12.0 note so
  a future bump to v2.13+ triggers another check instead of silently assuming
  the bug got fixed.

### Files modified

- `go.mod`, `go.sum`
- `README.md`, `CLAUDE.md`
- `PROGRESS.md`, `wails.json`, `frontend/package.json`, `frontend/package-lock.json`

### Versions bumped

- `wails.json`                  0.0.24 → 0.0.25
- `frontend/package.json`       0.0.24 → 0.0.25
- `frontend/package-lock.json`  0.0.24 → 0.0.25

---

## Rev 24 — 2026-04-21 — Sync browser dev-tab with native window's open document [dev]

Version: **0.0.24**

### Symptom

Opening `http://localhost:34115` in a regular browser while a file was loaded
in the native Wails window showed `SAMPLE_BOOK` (Kobzar) instead of the file.
Same for any second JS context. The Wails dev-server hint *"To develop in
the browser and call your bound Go methods from Javascript, navigate to:
http://localhost:34115"* implies the browser tab should be useful for working
on the live document — but the tab always started from a fresh sample.

### Root cause

The Go-side `*App` struct (`app.go`) holds `current *doc.FictionBook` —
this state is shared across all JS contexts because they all hit the same
Go process. But the Svelte `fb` variable lives in each tab's JS heap
independently, and `App.svelte::onMount` unconditionally seeded
`fb = SAMPLE_BOOK` without ever asking Go what was open. So the second
context never saw the document already loaded by the first.

### Fix

`onMount` now opportunistically calls `App.CurrentDocument()` (already
exposed at `app.go:146`) when the Wails runtime is available. If Go
returns a document with at least one body, it becomes the initial `fb`;
otherwise we fall back to `SAMPLE_BOOK` as before.

`currentPath` is intentionally NOT synced. Two tabs holding the same
path could race on Save — last write wins, silently clobbering the
other context's edits. Without a path, the dev-tab's Save falls
through to `PickFB2ToSave` (Save-As), which is the safe default.

### Caveat

This syncs only on tab open / refresh, and reads only what's been
committed to Go (i.e., after Open or Save → `UpdateDocument`). Unsaved
edits made in the native window's PM-editor live in that window's
Svelte state and do NOT round-trip to Go until Save. Bridging unsaved
edits would need a different mechanism (Wails events on edit), out of
scope for this rev.

### Files modified

- `frontend/src/App.svelte`
- `PROGRESS.md`, `wails.json`, `frontend/package.json`, `frontend/package-lock.json`

### Versions bumped

- `wails.json`                  0.0.23 → 0.0.24
- `frontend/package.json`       0.0.23 → 0.0.24
- `frontend/package-lock.json`  0.0.23 → 0.0.24

---

## Rev 23 — 2026-04-21 — A11y warning + 5 long-standing TS errors [dev]

Version: **0.0.23**

### A11y — `TableDialog.svelte`

`vite-plugin-svelte` warned: *Non-interactive element `<div>` should not be
assigned mouse or keyboard event listeners*. Real target was the inner
`<div role="dialog" on:click|stopPropagation on:keydown|stopPropagation>` —
Svelte-a11y treats `role="dialog"` as non-interactive. The two
`stopPropagation` handlers are still useful (they stop clicks/keys inside the
dialog from reaching the backdrop's dismiss handler), so we silence the
warning rather than restructure:

```svelte
<!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
<div class="dialog" role="dialog" …>
```

### 5 TS errors flushed out by `npm run check`

Pre-existing, unrelated to Rev 22 — discovered when re-running svelte-check.

1. **`App.svelte:33`** — `// @ts-expect-error` on `App.OpenFile()` was unused
   (Wails-generated `doc.FictionBook` is now structurally compatible with
   our local `FictionBook` type). Removed. The two remaining
   `@ts-expect-error` markers on `UpdateDocument` (lines 73, 93) are still
   needed — that direction (our → Wails) still mismatches.

2. **`App.svelte:112`** — `App.Validate()` returns `xsd.ValidationError[]`
   with **lowercase** `line / column / message` fields. Local code declared
   `Array<{ Line, Column, Message }>` (PascalCase, never matched). Dropped
   the bogus annotation, switched the access to `errs[0].message`.

3. **`AuthorField.svelte:58 / :70`** — `bind:value={author.Email[i]}` /
   `…HomePage[i]` failed because both fields are `string[] | null | undefined`
   in `fb2/types.ts`. The reactive guards on lines 16–17 ensure they're set
   at runtime, but TS doesn't track Svelte reactivity. Tried
   `bind:value={author.Email![i]}` first — Svelte 4 template parser rejects
   `!` inside `bind:` directives ("Expected }"). Workaround: lift the
   non-null assertion to `<script>` via reactive locals:

   ```ts
   $: if (!author.Email)    author.Email    = [];
   $: if (!author.HomePage) author.HomePage = [];
   $: emails    = author.Email!;
   $: homepages = author.HomePage!;
   ```

   Template then uses `bind:value={emails[i]}` / `…homepages[i]`. Mutation
   propagates because `emails`/`homepages` are the same array references as
   `author.Email`/`author.HomePage`.

4. **`TitleInfoForm.svelte:94`** — passed prop `availableBinaryIDs` to
   `<CoverpageField>`, but the component expects `availableIDs`. Renamed at
   call site: `availableIDs={availableBinaryIDs}`.

After the fixes: `svelte-check` reports `0 errors and 0 warnings`.

### Files modified

- `frontend/src/editor/TableDialog.svelte`
- `frontend/src/App.svelte`
- `frontend/src/description/AuthorField.svelte`
- `frontend/src/description/TitleInfoForm.svelte`
- `PROGRESS.md`, `wails.json`, `frontend/package.json`, `frontend/package-lock.json`

### Versions bumped

- `wails.json`              0.0.22 → 0.0.23
- `frontend/package.json`   0.0.22 → 0.0.23
- `frontend/package-lock.json` 0.0.22 → 0.0.23

---

## Rev 22 — 2026-04-21 — Toolbar text wrap + outline navigation after first click [dev]

Version: **0.0.22**

### Bug 1 — toolbar labels wrapping onto two lines

`Toolbar.svelte` buttons with multi-glyph labels (`T-A`, `+ Title`, `+ Epigraph`,
`+ Annot.`, `+ T-A`) rendered stacked: first glyph on top line, rest on the
second. The button had `min-width: 2rem; height: 1.8rem` but no
`white-space: nowrap`, so the natural label width exceeded min-width and the
text wrapped. `T-A` additionally broke at the hyphen.

Fix: added `display: inline-flex; align-items: center; justify-content: center;
white-space: nowrap; line-height: 1` to `.toolbar button`.

### Bug 2 — only the first outline navigation worked, subsequent clicks scrolled out of view

`Editor.scrollToPath` computed the scroll delta against
`view.dom.getBoundingClientRect().top`, but scrolled a *different* element
(the nearest scrollable ancestor, which is the `<section>` in
`App.svelte`, not `view.dom`). `view.dom`'s rect moves with every scroll —
after the first click, its `top` is negative by the current `scrollTop`,
so the formula `coords.top - rootRect.top - 12` over-counted by exactly
the previous scroll distance, landing the target above the visible area.

The first click worked only because `scrollTop = 0` and `view.dom.top`
coincidentally equals the scrollable container's top in that state.

Fix: measure delta against `el.getBoundingClientRect().top` (the scrollable
container's rect), which is invariant under its own `scrollTop` changes:

```ts
const elRect = el.getBoundingClientRect();
el.scrollTop += coords.top - elRect.top - 12;
```

Also made the flash highlight use `parentElement` when `domAtPos` returns a
Text node, so sections get a visible flash instead of silent no-op.

### Files modified

- `frontend/src/editor/Toolbar.svelte`
- `frontend/src/editor/Editor.svelte`
- `PROGRESS.md`, `wails.json`, `frontend/package.json`

### Versions bumped

- `wails.json`            0.0.21 → 0.0.22
- `frontend/package.json` 0.0.21 → 0.0.22

---

## Rev 21 — 2026-04-21 — Drop `*.fb2.zip` filter (Wails v2 macOS UTType nil crash) [dev]

Version: **0.0.21**

### Symptom

Opening any FB2 crashed the `.app` hard with:

```
*** Terminating app due to uncaught exception 'NSInvalidArgumentException',
    reason: '*** -[__NSArrayM insertObject:atIndex:]: object cannot be nil'
StartCustomProtocolHandler + 11926
OpenFileDialog + 506
```

The Go-side recover() from Rev 20 could not catch it because the crash
happened in Objective-C, before the dialog even returned — so
`defer recover()` around `OpenFile` was never going to help for this one.

### Root cause

In Wails v2.9.2 (`internal/frontend/desktop/darwin/WailsContext.m`) the
native file-dialog helper splits the pattern on `;`, strips `*.`, and
feeds each token to `[UTType typeWithFilenameExtension:]`. The result
is added to an `NSMutableArray` **without a nil check**. The extension
`fb2.zip` (a multi-dot pattern) resolves to `nil` on macOS 11+, and
`addObject:nil` throws `NSInvalidArgumentException` from native code —
crashing the whole process.

### Fix

`PickFB2ToOpen` now passes `Pattern: "*.fb2"` only. Users who need to
open `.fb2.zip` archives select "All files" in the dialog's format
picker.

Doc comment explains the Wails bug for future-me.

Versions bumped 0.0.20 → 0.0.21.

---

## Rev 20 — 2026-04-21 — Robust Open (panic recovery + graceful schema fallback) [dev]

Version: **0.0.20**

### Symptom

Opening a real-world FB2 from `~/Documents/books` caused the `.app` to
"crash without logs". Go-side `parser.Parse` succeeded on all three test
books via the CLI, so the failure was downstream — either a Go panic
during Wails JSON marshaling or a ProseMirror schema violation during
`fb2ToPMDoc`.

### Fix

- **Go `App.OpenFile` recover.** Named return values + `defer recover()`
  convert any panic (from parser / encoding / Wails marshaling) into a
  normal error returned to the frontend, instead of crashing the webview.
- **Frontend `toPMDoc` guard.** `Editor.svelte` wraps `fb2ToPMDoc(fb)` in
  a try/catch. On schema failure, renders a placeholder doc
  ("Could not render this document" + the error message + a note that
  the original FB2 is preserved for Save As) so the app stays alive and
  lets the user at least re-export the raw file.
- **Better openFile diagnostics.** `App.svelte::openFile` now:
  - logs `[fbe] opening …` / `[fbe] parsed: N bodies, N binaries, title "…"`
    / `[fbe] openFile failed: …` with stack trace.
  - shows a progress status in the header ("Opening X…") that yields to
    the event loop before mounting a potentially huge PM doc.
  - surfaces the error message prominently instead of silently falling
    back to the sample book.

### How to debug a future hang

Launch the app from the terminal so stderr is visible:

```
/Users/dmitry.gordiyevsky/fbe-go/build/bin/fbe-go.app/Contents/MacOS/fbe
```

Go panics print there; frontend logs go to the webview's devtools (which
Wails enables in dev builds — for release, use `wails dev`).

Versions bumped 0.0.19 → 0.0.20.

---

## Rev 19 — 2026-04-21 — Fix native dialogs (Wails v2 exposes them Go-only) [dev]

Version: **0.0.19**

### Bug

Save / Save As / Export HTML failed with
`TypeError: w.runtime.SaveFileDialog is not a function`.

### Root cause

In Wails v2, the generated `wailsjs/runtime/runtime.js` exports window/log/
event helpers but **not** file-dialog helpers. `OpenFileDialog` and
`SaveFileDialog` are part of `github.com/wailsapp/wails/v2/pkg/runtime`
and can only be called from Go. The frontend had to route through a
Go-side wrapper. My earlier App.svelte code called them directly on the
JS-side runtime import, which was undefined.

### Fix

Added three Go methods on `App`:

```go
PickFB2ToOpen()           (path, error)
PickFB2ToSave(suggested)  (path, error)
PickHTMLToSave(suggested) (path, error)
```

Each invokes `wailsrt.OpenFileDialog` / `SaveFileDialog` with the right
title + filters and returns the chosen path (empty string on cancel).

`App.svelte` now calls these generated bindings directly; dropped the
`wails()` helper that imported `../wailsjs/runtime/runtime` for dialogs,
since it no longer needs the runtime module at all.

### Verified

- `wails build -tags xsd` regenerates bindings; `PickFB2ToOpen` /
  `PickFB2ToSave` / `PickHTMLToSave` appear in
  `wailsjs/go/main/App.d.ts`.
- Production build completes with zero warnings in 10.7 s.

Versions bumped 0.0.18 → 0.0.19.

---

## Rev 18 — 2026-04-21 — A11y + unused CSS cleanup [dev]

Version: **0.0.18**

Clears every vite-plugin-svelte warning the production build used to print:

- **Label ↔ control association** (was: 15 warnings across every form
  component). Added `frontend/src/lib/uid.ts` with a per-process counter
  so each component instance composes unique `id`s for its inputs, and
  every `<label>` now has `for={…}`. Affects `AuthorField`, `GenreField`,
  `DateField`, `SequenceField`, `TitleInfoForm`, `DocumentInfoForm`,
  `CustomInfoForm` (for `SrcURL` / `Type` / `Value`).
- **Backdrop a11y** in `TableDialog.svelte`: the dismiss-on-click `<div>`
  now has `role="button"`, `tabindex="-1"`, `aria-label`, and a `keydown`
  handler for Escape; the inner dialog keeps `role="dialog"` and stops
  click / keydown bubbling.
- **Unused CSS** removed from `TitleInfoForm.svelte` (the `.hint` and
  `code` selectors that remained after Rev 16 replaced the placeholder).

54/54 vitest still pass. Production build produces **zero warnings**.

Versions bumped 0.0.17 → 0.0.18.

---

## Rev 17 — 2026-04-21 — Speller (native webview + Hunspell interface) [dev]

Version: **0.0.17**

**Decision.** Use the webview's native OS spellchecker instead of shipping
Hunspell bytes with the .app. macOS (WKWebView) and Linux (WebKitGTK) both
surface red squiggles + right-click suggestions when the editable DOM
declares `spellcheck="true" lang="…"`.

- `Editor.svelte` sets `spellcheck="true"` and `lang={fb.TitleInfo.Lang}`
  on the PM view attributes. The lang attribute is re-evaluated when the
  loaded book changes, so switching to a Ukrainian book picks up `uk`
  dictionaries automatically.
- `internal/fb2/speller/speller.go` keeps the `Speller` interface and the
  no-op backend; adds a documented roadmap for the future
  `-tags speller_hunspell` CGo backend (empty stub file for now).
- `docs/OPERATIONS.md §9` rewritten to describe the native-spellcheck
  current state and the Hunspell plan for Phase 4.

Versions bumped 0.0.16 → 0.0.17.

---

## Rev 16 — 2026-04-21 — Rich annotation editor [dev]

Version: **0.0.16**

- `frontend/src/description/AnnotationEditor.svelte` — embedded ProseMirror
  instance for `<annotation>` rich-text editing. Uses a derived schema
  (`fb2Schema.spec.nodes.update("doc", …)`) so the root accepts
  `paragraph | subtitle | empty_line | cite | poem | table`. Marks (strong,
  emphasis, strike, sub, sup, code, link) reuse the main schema's mark
  specs so they round-trip cleanly.
- Two-way binding: converts `Annotation.Children` into PM nodes on mount,
  emits `change` with the re-serialized `Annotation` on every transaction.
- Paste handling reuses `editor/paste.ts` (Word cleanup, CRLF normalize).
- Keyboard: Mod-B / Mod-I / standard undo/redo.
- `TitleInfoForm.svelte` replaces the placeholder hint with a real
  `<AnnotationEditor>` bound to `info.Annotation`.

Versions bumped 0.0.15 → 0.0.16.

---

## Rev 15 — 2026-04-21 — HTML export [dev]

Version: **0.0.15**

- `internal/fb2/export/html` — full Go implementation replacing FBE's
  493-line XSLT (`FBE/ExportHTML/html.xsl`). Walks the typed FictionBook
  struct, emits a single self-contained HTML file with embedded CSS and
  base64 data: URLs for images. Handles description (cover, title,
  authors, annotation), nested sections with heading levels 2–6,
  epigraphs/cites/poems/stanzas/verses, subtitles, empty-lines, tables,
  inline and block images, every inline mark + link + style mark. Raw
  unknown elements surface as `<div data-unknown="…">` with their text
  content.
- `cmd/fbe export html FILE.fb2 OUT.html` now works.
- `App.ExportHTML(path)` exposed to the frontend; App.svelte adds an
  `Export HTML…` button.
- Two Go tests (blank.fb2, rich.fb2) assert key output markers.

Versions bumped 0.0.14 → 0.0.15.

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
