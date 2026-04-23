# Packaging files

Freedesktop integration assets for Linux distributions. On macOS the
`.app` bundle carries its file associations, UTI declarations, and
thumbnail icon in its own `Info.plist` (rendered by Wails from
`wails.json`'s `fileAssociations`), so nothing here is needed on macOS.

| File                  | Purpose                                                                                   | Install location                                                          |
| --------------------- | ----------------------------------------------------------------------------------------- | ------------------------------------------------------------------------- |
| `fbe-go.desktop`      | Launcher entry; registers the app for `.fb2` / `.fb2.zip` MIME types.                     | `/usr/share/applications/` (system) or `~/.local/share/applications/`     |
| `fbe-go-mime.xml`     | Shared MIME registration — declares `application/x-fictionbook+xml` with `<FictionBook`   | `/usr/share/mime/packages/` (system) or `~/.local/share/mime/packages/`   |
|                       | magic detection and `.fb2` glob; also registers compressed `application/x-zip-compressed-fb2`. |                                                                        |
| `fbe-go.thumbnailer`  | Tells GNOME Files / Nautilus / Nemo / Caja / KDE Dolphin to call `fbe thumb` for previews.| `/usr/share/thumbnailers/` (system) or `~/.local/share/thumbnailers/`     |

## Per-user install (no root needed)

```sh
install -Dm644 packaging/fbe-go.desktop      ~/.local/share/applications/fbe-go.desktop
install -Dm644 packaging/fbe-go-mime.xml     ~/.local/share/mime/packages/fbe-go.xml
install -Dm644 packaging/fbe-go.thumbnailer  ~/.local/share/thumbnailers/fbe-go.thumbnailer

# refresh freedesktop caches
update-mime-database   ~/.local/share/mime
update-desktop-database ~/.local/share/applications
```

Log out / back in (or `pkill nautilus`) for the file manager to pick up
the new thumbnailer. The `fbe` CLI must be on `PATH` — either install
the AppImage somewhere PATH-visible or symlink the AppImage's embedded
`fbe` into `~/.local/bin/`.

## System-wide install

Replace `~/.local/share` with `/usr/share` in the paths above and run
the update commands without the trailing directory argument (or with
`/usr/share/mime`, `/usr/share/applications`).

## Why no QuickLook `.appex` on macOS?

Full QuickLook preview extensions require a separate Xcode project
(`.appex` bundle with `NSExtension` plist, `QLPreviewingController`
subclass, code-signing). The current `wails.json` `fileAssociations`
entry gets us:

- The `.fb2` icon in Finder.
- Double-click opens the file in fbe-go.
- Spotlight sees `.fb2` as "FictionBook 2.x document".

…which covers 90 % of the QuickLook value. The thumbnail preview pane
is deferred to a future rev that ships the Swift extension.
