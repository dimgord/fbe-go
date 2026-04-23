# Implementation phases

Single contributor, rough estimates. Multiply by ~1.5 if unfamiliar with Wails / ProseMirror.

## Phase 0 — PoC (1 week)

Goal: prove the Wails + ProseMirror stack handles an FB2 document.

- [ ] `wails init` into this directory, reconcile with existing skeleton
- [ ] Implement `parser.Parse` end-to-end for UTF-8 FB2 (skip encoding auto-detect for now)
- [ ] Implement `fb2ToPMDoc` for body + sections + paragraphs + basic marks (strong, emphasis)
- [ ] Render coverpage as `<img>` via `App.GetBinaryDataURL`
- [ ] Read-only editor (no editing yet)

**Exit criterion:** open `testdata/*.fb2` and see the book rendered correctly in the Wails window.

## Phase 1 — Core Go library (3–4 weeks)

Goal: the Go side is complete and usable as a CLI, independent of UI.

- [ ] Encoding auto-detection (win-1251, koi8-r, utf-8, utf-16)
- [ ] Parser handles all FB2 elements (per FictionBook.xsd)
- [ ] Writer produces byte-for-byte canonical output compatible with FBE readers
- [ ] XSD validation wired up (libxml2 CGo path)
- [ ] `.fb2.zip` pack/unpack
- [ ] Binary extraction + thumbnail
- [ ] Golden-file tests: round-trip 20 real-world .fb2 files
- [ ] `cmd/fbe` CLI covers: validate, thumb, pack, unpack, info, export html
- [ ] Replace FBV.exe in user's workflow

**Exit criterion:** `fbe validate` produces the same verdicts as FBV on a 100-file corpus.

## Phase 2 — Wails shell + read-only viewer (2 weeks)

Goal: desktop app opens/closes files, browses library.

- [ ] File open/save native dialogs
- [ ] Recent files list (with thumbnails)
- [ ] Settings dialog (interface language, fonts, colors — read-only first)
- [ ] Dark mode
- [ ] Document tree (outline) — read-only
- [ ] Metadata panel — read-only display of title-info, etc.
- [ ] Window/pane sizes persisted

**Exit criterion:** comfortable to browse a library of .fb2 files; read a book end-to-end.

## Phase 3 — Editor MVP (6–8 weeks)

Goal: basic editing parity with FBE for the most-used operations.

Order of implementation (lowest → highest risk):
1. Inline marks (strong, emphasis, sub, sup, strikethrough, code) — 2 days
2. Paragraph style commands (normal, subtitle, text-author, code) — 3 days
3. Empty line, split-section — 2 days
4. Add title / add body / add epigraph / add annotation / add text-author — 1 week
5. Clone container (sections, poems, etc.) — 2 days
6. Insert block image / inline image — 2 days
7. **Insert poem** (selection → stanza transform) — 3 days
8. **Insert cite** — 2 days
9. **Merge containers** (6 sub-cases for sections) — 3 days
10. **Insert table** — 3 days
11. Save + XSD validate + error UI — 3 days

**Exit criterion:** can type and save a non-trivial .fb2 that FBE (the original) opens cleanly.

## Phase 4 — Feature parity (4 weeks)

- [ ] Speller (Hunspell via CGo, PM decoration plugin)
- [ ] Search / Replace (in-editor + CLI)
- [ ] Hotkeys (configurable, from settings.json)
- [ ] Paste handling (Word, HTML, plain text, images from clipboard)
- [ ] Description form (full metadata editor — 7 sections)
- [ ] Binary manager (upload, rename, preview, delete)
- [ ] HTML export (Go templates)
- [~] Scripts compatibility — **deferred to post-1.0**. FBE ships hundreds
      of `.js` macros, and reimplementing the `apiRunCmd` / `apiProcessCmd`
      surface plus a migration path is a separate-project-scale effort
      (est. 4–6 weeks of undefined scope). Revisit when there is concrete
      user demand with named scripts to port. See OPERATIONS.md §10.

**Exit criterion:** a returning FBE user finds everything they used *except
scripts* — that is explicitly punted.

## Phase 5 — Platform polish (1.5 weeks)

**Target platforms: macOS + Linux only.** Windows is out of scope — the original
C++ FBE remains the Windows story. Platform-native bits (thumbnailers, QuickLook)
may be written in Rust or C where Go is awkward.

- [x] macOS DMG (Rev 65) + UTI / file associations (Rev 66). QuickLook
      `.appex` preview extension deferred — needs a separate Xcode project
      with code-signing; parked until hardware refresh.
- [x] Linux AppImage (Rev 65) + `.desktop` (Rev 65) + GNOME thumbnailer
      shim calling `fbe thumb` + shared-MIME registration (Rev 66).
- [x] CI builds for macOS (universal .app: arm64 + x86_64) and Linux
      (x86_64) (Revs 63–65). Linux arm64 deferred — GitHub's hosted
      runners are x86_64-only; would need self-hosted or
      `docker/setup-qemu-action` cross-build.
- [ ] Auto-update (optional — Wails has no built-in; consider Sparkle for macOS)

**Exit criterion:** downloadable `.dmg` / `.AppImage`, drag-and-drop in Finder
and Linux file managers shows cover thumbnails.

## Total estimate

| Phase | Weeks | Cumulative |
|---|---|---|
| 0 | 1 | 1 |
| 1 | 4 | 5 |
| 2 | 2 | 7 |
| 3 | 8 | 15 |
| 4 | 4 | 19 |
| 5 | 1.5 | 20.5 |

**~5 months of single-contributor focused work.** In reality, expect 8–10 calendar months with research detours and testing.

## Milestones for external communication

- **M1 (week 5):** "Go CLI replaces FBV" — announce on FB2 forums, get feedback from library maintainers
- **M2 (week 7):** "Cross-platform FB2 reader" — read-only beta on macOS + Linux
- **M3 (week 15):** "Editor alpha" — invite original FBE community to try writing
- **M4 (week 20.5):** "1.0" — feature parity, installers, thumbnailers (macOS + Linux)
