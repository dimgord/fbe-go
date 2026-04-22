# PROGRESS

Revision log for `fbe-go`. Every commit that changes behavior or shape of the
project must add an entry here and bump the version in `wails.json` and
`frontend/package.json` (keep them in sync).

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
