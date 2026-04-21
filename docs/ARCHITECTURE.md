# Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                       Wails Frontend (webview)                  │
│                                                                 │
│  ┌──────────────┐   ┌────────────────────┐   ┌──────────────┐   │
│  │ DocumentTree │   │ ProseMirror Editor │   │ Dialogs      │   │
│  │ (Svelte)     │   │ (schema.ts + cmds) │   │ metadata,    │   │
│  │              │   │                    │   │ settings,    │   │
│  └──────┬───────┘   └────────┬───────────┘   │ find, spell  │   │
│         │                    │                └──────┬───────┘   │
│         └──────── Svelte store (current doc) ───────┘           │
└─────────────────────────┬───────────────────────────────────────┘
                          │  Wails JSON-RPC
┌─────────────────────────┴───────────────────────────────────────┐
│                    Go backend (app.go)                          │
│                                                                 │
│  App.OpenFile / SaveFile / GetBinaryDataURL / LoadSettings …   │
└──┬────────┬────────┬────────┬────────┬────────┬────────┬───────┘
   │        │        │        │        │        │        │
┌──▼─┐  ┌──▼──┐  ┌──▼──┐  ┌──▼──┐  ┌──▼──┐  ┌──▼──┐  ┌──▼────┐
│doc │  │pars │  │writ │  │xsd  │  │zip  │  │thumb│  │export │
│    │  │er   │  │er   │  │     │  │fb2  │  │     │  │/html  │
└────┘  └─────┘  └─────┘  └─────┘  └─────┘  └─────┘  └───────┘
┌────┐  ┌─────┐  ┌─────┐
│binary│ │search│ │speller│   ┌───────────┐
│     │  │     │  │     │    │ settings  │
└─────┘  └─────┘  └─────┘    └───────────┘
                                    │
                              JSON at ~/.config/fbe/config.json

┌─────────────────────────────────────────┐
│   cmd/fbe — CLI (same Go packages)      │
│   fbe validate | thumb | pack | unpack  │
│        | info | export                  │
└─────────────────────────────────────────┘
```

## Data flow (open → edit → save)

1. **Open:** User picks a `.fb2` or `.fb2.zip`. Frontend calls `App.OpenFile(path)`.
2. **Parse:** Go `parser.Parse()` reads the XML (autodetect encoding, unwrap zip if needed), returns `*doc.FictionBook`.
3. **Marshal:** Wails JSON-encodes the FictionBook and hands it to the frontend.
4. **Hydrate:** Frontend's `editor/parse.ts::fb2ToPMDoc()` walks the FictionBook and produces a ProseMirror `Node` via `fb2Schema`.
5. **Mount:** Editor.svelte creates an `EditorView` with the doc; DocumentTree.svelte subscribes to a derived Svelte store that tracks outline nodes.
6. **Edit:** User triggers commands (`editor/commands.ts`) or types. ProseMirror updates the doc immutably; history plugin tracks undo.
7. **Serialize:** On save, `editor/serialize.ts::pmDocToFB2()` walks the current PM doc back to a FictionBook-shaped object.
8. **Validate (optional):** Frontend calls `App.Validate()` which runs `xsd.Validate()` and returns errors; UI surfaces them as inline markers.
9. **Write:** `App.SaveFile(path)` calls `writer.Write(fb)`. Output is indented canonical FB2 XML.

## Why ProseMirror and not TipTap?

TipTap is a batteries-included wrapper over ProseMirror with a nicer API. For this project:

- **Pro:** Faster development, many features (menus, collab, formatting) built-in.
- **Con:** Heavier bundle, opinions we may want to override (FB2 has oddities TipTap wasn't designed for: stanza/verse, recursive sections, inline vs. block image).

Default decision: **raw ProseMirror + selected addons** (`prosemirror-history`, `prosemirror-keymap`, `prosemirror-commands`, `prosemirror-inputrules`). Revisit after Phase 3 MVP — if the schema/command code is clean, stay raw; if boilerplate is painful, wrap with TipTap.

## Why separate `image_block` and `image_inline` schema nodes?

FB2 treats `<image>` as one element but its semantics differ by position:
- As a direct child of `<section>` or `<body>` → block, may have its own `<title>`
- Inside `<p>` / `<subtitle>` / `<text-author>` → inline, text-flow

In ProseMirror, a single node must be either block or inline. Splitting into two nodes lets the schema enforce FB2 validity at edit time (no inline images in wrong contexts) without a post-validation step.

## Why Go XSD instead of JS (ajv)?

- The CLI `fbe validate` needs to run headless, without a webview.
- Consistency: same validator for editor save and library-mode batch ops.
- FictionBook.xsd is ~500 lines with no tricky XSD features (no complex types, no imports). A custom pure-Go validator is feasible.

Implementation path: try `github.com/lestrrat-go/libxml2` first (CGo, but mature). If the deployment size hit is unacceptable, write a subset validator.

## Cross-platform strategy

**Target: macOS + Linux only.** Windows is out of scope — the original C++ FBE
remains the Windows story. Native platform code (thumbnailers, shell plugins)
may be written in Rust or C where Go is awkward.

| Concern | macOS | Linux |
|---|---|---|
| Webview | WKWebView | WebKitGTK |
| Binary size | ~12 MB | ~10 MB |
| File association | DMG + `Info.plist` UTIs | `.desktop` file + `mimeapps.list` |
| Thumbnailer | QuickLook generator (Swift, or Rust via `swift-bridge`) | GNOME thumbnailer shim (Bash + `fbe thumb` or Rust CLI) |
| Spellcheck | Hunspell via CGo | Hunspell (system or bundled) via CGo |

## What the original FBE had that this port deliberately drops

- **ActiveX plugins (ExportHTML as COM DLL)** — Go plugins are fragile and platform-specific. Export formats live as Go packages under `internal/fb2/export/*`.
- **Embedded IE MSHTML editor** — replaced by ProseMirror (cross-platform, web-based).
- **MSXML dependency** — replaced by Go's `encoding/xml` for parsing/writing; libxml2 only for XSD validation (if that path is chosen).
- **Windows Registry for settings** — replaced by JSON at the OS-standard config dir.
- **`FarMenu.ini`** (FBShell) — Far Manager integration is niche, dropped.
