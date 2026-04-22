# Third-party notices

fbe-go itself is licensed under the MIT License — see [LICENSE](LICENSE) at
the repo root. This file lists the upstream works that ship inside the binary
(bundled) or that fbe-go depends on at build / runtime, together with their
own licenses and attribution notices.

## Bundled files

### FictionBook 2.0 XSD schemas

The files under `internal/fb2/xsd/` — `FictionBook.xsd`,
`FictionBookLinks.xsd`, `genres.xsd`, `xml.xsd` — are the canonical
FictionBook 2.0 XML schemas authored by **Dmitry Gribov**. They ship inside
every fbe-go binary (via `go:embed`) so the editor can validate FB2
documents without a network round-trip.

Licensed under the 2-clause BSD License, reproduced here as required by
clause 2 for binary redistributions:

> Copyright (c) 2004, Dmitry Gribov
> All rights reserved.
>
> Redistribution and use in source and binary forms, with or without
> modification, are permitted provided that the following conditions are met:
>
>  * Redistributions of source code must retain the above copyright notice,
>    this list of conditions and the following disclaimer.
>  * Redistributions in binary form must reproduce the above copyright notice,
>    this list of conditions and the following disclaimer in the documentation
>    and/or other materials provided with the distribution.
>
> THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
> AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
> IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
> ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE
> LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
> CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
> SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
> INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
> CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
> ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF
> THE POSSIBILITY OF SUCH DAMAGE.

## Go runtime dependencies

Pinned via `go.mod`; inspect `go.sum` for exact versions. All MIT-licensed
unless noted.

- **[Wails v2](https://github.com/wailsapp/wails)** — MIT · © Lea Anthony and
  the Wails contributors. Desktop-app framework binding Go to a native
  webview. Its own set of transitive MIT/BSD dependencies
  (`labstack/echo`, `samber/lo`, `go-webview2`, `go-ole`, etc.) ship inside
  the Wails v2 module tree.
- **[lestrrat-go/libxml2](https://github.com/lestrrat-go/libxml2)** — MIT ·
  © Daisuke Maki. Go CGo bindings to libxml2; used by `-tags xsd` for
  real schema validation.
- **[golang.org/x/text](https://pkg.go.dev/golang.org/x/text)**,
  **[golang.org/x/net](https://pkg.go.dev/golang.org/x/net)**,
  **[golang.org/x/crypto](https://pkg.go.dev/golang.org/x/crypto)**,
  **[golang.org/x/sys](https://pkg.go.dev/golang.org/x/sys)** — BSD-3 ·
  © The Go Authors.
- **Go standard library** — BSD-3 · © The Go Authors.

## Native C libraries

- **[libxml2](https://gitlab.gnome.org/GNOME/libxml2)** — MIT ·
  © Daniel Veillard. Linked via `lestrrat-go/libxml2` when built with
  `-tags xsd`. On Linux pinned to 2.13.x (see `flake.nix` / `CLAUDE.md`
  for the binding-compatibility rationale).
- **GTK 3** — LGPL-2.1+ · © The GTK team. Loaded at runtime by
  Wails on Linux via `webkit2gtk-4.1`.
- **WebKitGTK** — LGPL-2 / BSD-style · © Apple + KDE + GNOME + others.
- **WKWebView / Cocoa frameworks** — macOS system frameworks from Apple
  (no redistribution).

## Frontend (`frontend/`) dependencies

Pinned via `frontend/package.json` / `frontend/package-lock.json`. All
MIT unless noted.

- **[Svelte 4](https://github.com/sveltejs/svelte)** — MIT · © Rich Harris
  et al.
- **[Vite](https://github.com/vitejs/vite)** — MIT · © Yuxi (Evan) You
  and contributors.
- **[ProseMirror](https://prosemirror.net/)** (`prosemirror-model`,
  `prosemirror-state`, `prosemirror-view`, `prosemirror-commands`,
  `prosemirror-history`, `prosemirror-keymap`, `prosemirror-schema-basic`,
  `prosemirror-transform`) — MIT · © Marijn Haverbeke et al.
- **Vitest**, **svelte-check**, **TypeScript** — MIT / Apache-2.0.

## Inspiration (no code reuse)

- **[FictionBook Editor (classic FBE)](https://github.com/evpobr/fictionbookeditor)**
  — the Windows-only C++/WTL/MSHTML/MSXML application whose user
  experience and set of FB2 operations inspired fbe-go. **fbe-go is an
  independent rewrite in Go + Wails + ProseMirror; no source code from
  FBE was reused.** The FB2 operations catalog under `docs/OPERATIONS.md`
  cross-references FBE's behavior as the spec, not as a source.

## Nix / NixOS

`flake.nix` at repo root pins `nixpkgs` via `flake.lock` (nixpkgs is
MIT-licensed). Listed dependencies (`go_1_25`, `nodejs_22`, `gtk3`,
`webkitgtk_4_1`, `libxml2_13`, `gsettings-desktop-schemas`,
`pkg-config`) are the standard nixpkgs-packaged versions of upstream
works already covered above.
