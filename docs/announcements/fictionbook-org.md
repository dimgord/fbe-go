# Announcement — fictionbook.org forum

Audience: broader FB2 user community (readers, library maintainers,
typesetters). Tone: feature-focused, screenshot-friendly,
download-link-prominent.

Suggested venue: a single forum post on fictionbook.org (or its
successor / mirror). Inline screenshots make a big difference here —
see the placeholder list at the bottom for which shots to take before
posting.

---

## English

> **Title:** fbe-go 1.0 — FB2 editor for macOS and Linux (open-source, MIT)
>
> ![Editor — body view with outline tree](https://raw.githubusercontent.com/dimgord/fbe-go/main/docs/announcements/screenshots/editor-outline.png)
>
> If you've ever wanted FBE on macOS or Linux — fbe-go 1.0 is out
> today. It's a from-scratch Go port targeting these two platforms
> (Windows users keep using the original FBE).
>
> **What works out of the box:**
>
> - Open / edit / save / validate / export FB2 documents, including
>   `.fb2.zip` archives.
> - Full structural editing — sections, poems, cites, tables,
>   epigraphs, annotations, images, footnotes.
> - Description form covering every FB2 metadata block (title-info,
>   src-title-info, publish-info, document-info, custom-info).
> - Binary manager for embedded images / cover art with
>   reference-cascade rename.
> - Search and Replace with regex + Unicode whole-word.
> - XSD validation against the bundled FictionBook 2.x schemas, with
>   clickable error navigation.
> - HTML export.
> - Native webview spellcheck routed via the document's `<lang>`.
> - Cmd/Ctrl-click on a footnote jumps to the note;
>   Cmd/Ctrl-[ goes back.
>
> ![Description form — Title info tab with annotation editor](https://raw.githubusercontent.com/dimgord/fbe-go/main/docs/announcements/screenshots/description-form.png)
>
> ![Validation panel — XML source + clickable XSD error](https://raw.githubusercontent.com/dimgord/fbe-go/main/docs/announcements/screenshots/validation-panel.png)
>
> **Tested on a 166-file real-world corpus** — every XSD-valid source
> survives the parse → edit → save round-trip as XSD-valid output (the
> `fidelityBroken == 0` gating invariant).
>
> **Download (latest):**
>
> - macOS (universal, signed + notarized):
>   [fbe-go macOS DMG](https://github.com/dimgord/fbe-go/releases/latest)
> - Linux (x86_64 AppImage, requires `webkit2gtk-4.1`):
>   [fbe-go Linux AppImage](https://github.com/dimgord/fbe-go/releases/latest)
> - Linux (freedesktop tarball with `.desktop` + GNOME thumbnailer):
>   [fbe-go Linux tarball](https://github.com/dimgord/fbe-go/releases/latest)
>
> Source code: https://github.com/dimgord/fbe-go
> Bug reports / feature requests:
> https://github.com/dimgord/fbe-go/issues

---

## Українська

> **Title:** fbe-go 1.0 — FB2-редактор для macOS і Linux (open-source, MIT)
>
> ![Редактор — body view з outline-деревом](https://raw.githubusercontent.com/dimgord/fbe-go/main/docs/announcements/screenshots/editor-outline.png)
>
> Якщо колись хотіли FBE на macOS чи Linux — fbe-go 1.0 вийшов
> сьогодні. Це Go-порт з нуля для цих двох платформ
> (Windows-користувачі продовжують використовувати оригінальний FBE).
>
> **Що працює з коробки:**
>
> - Open / edit / save / validate / export FB2-документів, включно з
>   `.fb2.zip`-архівами.
> - Повне структурне редагування — section, poem, cite, table,
>   epigraph, annotation, image, footnote.
> - Description-форма, що покриває всі FB2-метадата-блоки (title-info,
>   src-title-info, publish-info, document-info, custom-info).
> - Binary manager для вбудованих зображень / cover art з
>   cascade-rename посилань.
> - Search / Replace з regex та Unicode whole-word.
> - XSD-валідація проти вбудованих схем FictionBook 2.x з клікабельною
>   навігацією помилок.
> - HTML-експорт.
> - Native-webview spellcheck, що роутиться через `<lang>` документа.
> - Cmd/Ctrl-click по footnote стрибає на ноту;
>   Cmd/Ctrl-[ повертає назад.
>
> ![Description-форма — Title info з annotation editor](https://raw.githubusercontent.com/dimgord/fbe-go/main/docs/announcements/screenshots/description-form.png)
>
> ![Validation-панель — XML source + клікабельна XSD-помилка](https://raw.githubusercontent.com/dimgord/fbe-go/main/docs/announcements/screenshots/validation-panel.png)
>
> **Протестовано на 166-файловому реальному корпусі** — кожен
> XSD-валідний source виживає parse → edit → save як XSD-валідний
> output (`fidelityBroken == 0` — кардинальний інваріант).
>
> **Завантажити (latest):**
>
> - macOS (universal, signed + notarized):
>   [fbe-go macOS DMG](https://github.com/dimgord/fbe-go/releases/latest)
> - Linux (x86_64 AppImage, потребує `webkit2gtk-4.1`):
>   [fbe-go Linux AppImage](https://github.com/dimgord/fbe-go/releases/latest)
> - Linux (freedesktop tarball з `.desktop` + GNOME thumbnailer):
>   [fbe-go Linux tarball](https://github.com/dimgord/fbe-go/releases/latest)
>
> Source: https://github.com/dimgord/fbe-go
> Bug-репорти / feature-запити:
> https://github.com/dimgord/fbe-go/issues

---

## Screenshots — what to capture before posting

Three frames to grab on the main editor surface (macOS preferred —
Retina makes a noticeable difference for crisp UI screenshots, ~1200 px
wide is plenty):

1. **Editor + outline tree** — open a real `.fb2` (e.g. Nevelichka drama)
   with several sections; outline tree visible on the left. This is
   the "look-and-feel" hero shot.
2. **Description form** — Title info tab, with name fields filled and
   the annotation editor expanded so the rich-text controls are
   visible. Demonstrates "we're not just a body editor — full
   metadata form too."
3. **Validation panel** — read-only XML source pane on one side, the
   clickable error list on the other (open a deliberately-broken FB2
   to surface 1-2 errors). Highlights the unique feature.

Optional bonus shots: Binary Manager with the COVER badge visible;
Search bar active mid-search; Settings → Keyboard shortcuts tab.

Save as PNG, name them descriptively
(`editor-outline.png`, `description-form.png`, `validation-panel.png`)
and replace the `[SCREENSHOT: …]` placeholders before posting.
