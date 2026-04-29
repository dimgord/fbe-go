# Announcement — original FBE community (github.com/evpobr/fictionbookeditor)

Audience: developers / contributors of the original Windows FBE fork.
Tone: technical, neighbourly — we extend, not replace.

**Suggested venue:** a single **Issue** on
`github.com/evpobr/fictionbookeditor` (Discussions are disabled there
as of 2026-04). Title prefix `[Announcement]` so triage tooling sorts
it away from real bugs.

**Language:** EN body below is the safe default — the tracker is mostly
English-speaking. UA version is included as a parallel option if you
want to target the Ukrainian-speaking subset of the FB2 audience
explicitly; either alone is fine, posting both in the same issue would
be noise.

**Repo state heads-up (2026-04):** the original repo's last code push
was 2023-06-29; only one user (`alzhur`) is active in recent issues.
The announcement may sit quietly. That's fine — keep it short, polite,
and easy to ignore for anyone not interested.

---

## English

> **Title:** [Announcement] fbe-go 1.0 — a macOS / Linux port of FBE
>
> *Not a bug or feature request — community announcement, no action
> needed from maintainers. Closing on sight is fine if it doesn't fit
> here; happy to move it elsewhere on request.*
>
> ---
>
> Hi everyone — long-time FBE user here. I just shipped a Go +
> [Wails v2](https://wails.io) reimplementation of FBE specifically for
> **macOS and Linux**. Windows is explicitly out of scope — the original
> FBE remains the canonical Windows version, this just extends FB2
> editing to two more platforms.
>
> **Why:** I needed an FB2 editor on macOS for personal library work
> and nothing existed. So I rewrote the core in pure Go (parser /
> writer / zip / thumbnailer / XSD validation via libxml2) and replaced
> MSHTML `contentEditable` with [ProseMirror](https://prosemirror.net)
> hosted in a system webview through Wails.
>
> **Feature parity with original FBE:**
>
> - Every structural operation: insert / clone / merge sections, poems,
>   cites, tables, epigraphs, annotations, empty-lines, titles,
>   text-authors, images.
> - Inline marks, paragraph styles, paste cleanup (Word / Google Docs).
> - Description form (full title-info + src-title-info + publish-info +
>   document-info), with a rich-text annotation editor.
> - Binary manager (upload / rename / delete embedded images, with
>   reference-cascade rename).
> - Search / Replace with regex + Unicode whole-word.
> - XSD validation against the bundled schemas, clickable error list,
>   read-only XML source panel.
> - HTML export.
> - Native-webview spellcheck routed by the document's `<lang>`.
>
> **Round-trip fidelity:** unknown FB2 elements survive parse → edit →
> save unchanged via a `RawElement` fallback. Tested on a 166-file
> real-world corpus, `fidelityBroken == 0` is the gating invariant.
>
> **Distribution:**
>
> - macOS: signed + notarized universal DMG (arm64 + x86_64),
>   drag-to-Applications, no Gatekeeper warning.
> - Linux: x86_64 AppImage (requires system `webkit2gtk-4.1`) +
>   freedesktop tarball with `.desktop` file and GNOME thumbnailer.
> - CLI (`fbe`) ships alongside the desktop app:
>   `validate / pack / unpack / thumb / info / export html`.
>
> **Source:** https://github.com/dimgord/fbe-go (MIT)
> **Download:** https://github.com/dimgord/fbe-go/releases/latest
>
> Issues and Discussions are open. Bug reports / feature requests /
> corpus contributions all welcome — especially if you have an FB2
> file that round-trips cleanly through the original FBE but breaks
> here.

---

## Українська

> **Title:** [Announcement] fbe-go 1.0 — порт FBE для macOS і Linux
>
> *Не баг і не feature request — community-анонс, ніяких дій від
> maintainer'ів не потрібно. Закрити одразу як невідповідне теж OK;
> можу перенести деінде за проханням.*
>
> ---
>
> Привіт усім — давній користувач FBE. Щойно випустив реімплементацію
> FBE на Go + [Wails v2](https://wails.io) спеціально для **macOS і
> Linux**. Windows свідомо поза рамками — оригінальний FBE залишається
> канонічним Windows-варіантом, ми просто додаємо ще дві платформи до
> FB2-екосистеми.
>
> **Чому:** мені потрібен був FB2-редактор на macOS для роботи з
> власною бібліотекою, і нічого не існувало. Тож я переписав ядро на
> чистому Go (парсер / writer / zip / thumbnailer / XSD-валідація через
> libxml2) і замінив MSHTML `contentEditable` на
> [ProseMirror](https://prosemirror.net) у системному webview через
> Wails.
>
> **Паритет фіч з оригінальним FBE:**
>
> - Усі структурні операції: insert / clone / merge для section, poem,
>   cite, table, epigraph, annotation, empty-line, title, text-author,
>   image.
> - Inline-марки, paragraph styles, очистка вставки (Word /
>   Google Docs).
> - Description-форма (title-info + src-title-info + publish-info +
>   document-info), з rich-text-редактором annotation'у.
> - Binary manager (завантаження / перейменування / видалення вбудованих
>   зображень з cascade-rename посилань).
> - Search / Replace з regex та Unicode whole-word.
> - XSD-валідація проти вбудованих схем, клікабельний список помилок,
>   read-only XML-панель.
> - HTML-експорт.
> - Native-webview spellcheck, що роутиться через `<lang>` документа.
>
> **Round-trip-вірність:** невідомі FB2-елементи виживають
> parse → edit → save без змін через `RawElement` fallback.
> Протестовано на 166-файловому реальному корпусі,
> `fidelityBroken == 0` — кардинальний інваріант.
>
> **Дистрибуція:**
>
> - macOS: підписаний і notarized universal DMG (arm64 + x86_64),
>   drag-to-Applications, без Gatekeeper-попередження.
> - Linux: x86_64 AppImage (потребує системний `webkit2gtk-4.1`) +
>   freedesktop tarball з `.desktop`-файлом і GNOME-thumbnailer.
> - CLI (`fbe`) йде поряд з desktop-додатком:
>   `validate / pack / unpack / thumb / info / export html`.
>
> **Source:** https://github.com/dimgord/fbe-go (MIT)
> **Download:** https://github.com/dimgord/fbe-go/releases/latest
>
> Issues і Discussions відкриті. Bug-репорти / feature-запити /
> корпус-контрибуції — все вітається, особливо якщо маєте FB2-файл, що
> чисто round-trip'ить через оригінальний FBE, але ламається тут.
