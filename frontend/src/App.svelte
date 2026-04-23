<script lang="ts">
  import { onMount } from "svelte";
  import DocumentTree from "./tree/DocumentTree.svelte";
  import Editor from "./editor/Editor.svelte";
  import Toolbar from "./editor/Toolbar.svelte";
  import DescriptionPanel from "./description/DescriptionPanel.svelte";
  import ValidationPanel from "./validation/ValidationPanel.svelte";
  import HelpDialog from "./help/HelpDialog.svelte";
  import SettingsDialog from "./settings/SettingsDialog.svelte";
  import SearchBar from "./editor/search/SearchBar.svelte";
  import BinaryManagerDialog from "./binary/BinaryManagerDialog.svelte";
  import { installExternalLinkHandler } from "./runtime/externalLink";
  import { configurePaste } from "./editor/paste";
  import { SAMPLE_BOOK } from "./fb2/sample";
  import type { FictionBook } from "./fb2/types";
  import { HOTKEY_ACTIONS, matchesEvent } from "./settings/hotkeys";

  type View = "body" | "description";
  let view: View = "body";

  let fb: FictionBook | null = null;
  let filename = "(untitled)";
  let currentPath = "";
  let status = "";
  let error = "";
  let editor: Editor | undefined = undefined;
  /** Bound to Editor.view so SearchBar (and any other sibling) gets a
      reactive reference to the live PM EditorView. */
  let editorView: import("prosemirror-view").EditorView | undefined = undefined;

  // Validation / XML-source panel state.
  let showPanel = false;
  let xmlSource = "";
  let validationErrors: { line: number; column: number; message: string }[] = [];
  // Initial errors-pane height in px (null = CSS default 45%). Populated
  // from settings.panes.validationErrorsHeight on mount; updated when
  // ValidationPanel dispatches `resize`.
  let initialErrorsHeight: number | null = null;

  // Outline-sidebar and validation-panel column widths (px). Null = use
  // CSS default (260px and minmax(320px, 30%) respectively).
  let outlineWidth: number | null = null;
  let panelWidth: number | null = null;
  let mainEl: HTMLElement | undefined;
  let descWrapEl: HTMLElement | undefined;
  let draggingOutline = false;
  let draggingPanel = false;

  const OUTLINE_MIN = 150;
  const OUTLINE_MAX = 500;
  const PANEL_MIN = 260;

  let showHelp = false;
  let showSettings = false;
  let showBinaries = false;

  /** User-configurable keyboard shortcuts, loaded from settings.Hotkeys on
      mount and refreshed after Settings → Apply. Passed into Editor as a
      prop so its PM keymap rebuilds on change, and consulted here for the
      window-level actions (Save, Find, dialogs). */
  let hotkeys: Record<string, string> = {};

  // Search/Replace inline bar state. Bar is non-modal and only shown in the
  // body view because description is edited in a separate PM instance.
  let searchOpen = false;
  let searchMode: "find" | "replace" = "find";
  function openSearch(mode: "find" | "replace") {
    searchMode = mode;
    searchOpen = true;
  }

  function onSettingsApplied(
    e: CustomEvent<{
      theme: Theme;
      settings: {
        font: { family: string; size: number };
        nbspChar: string;
        hotkeys?: Record<string, string>;
      };
    }>
  ) {
    // Sync live runtime state with what the dialog just wrote to disk.
    theme = e.detail.theme;
    applyEditorFont(e.detail.settings.font);
    configurePaste({ nbspChar: e.detail.settings.nbspChar });
    // Fresh object identity so Editor's reactive reconfigure fires.
    if (e.detail.settings.hotkeys) {
      hotkeys = { ...e.detail.settings.hotkeys };
    }
    // Refresh recent-files list in case the dialog cleared it.
    void refreshRecent();
  }

  // applyEditorFont sets CSS custom properties on <html> so the editor's
  // `:global(.ProseMirror) { font-family: var(--editor-font-family); … }`
  // picks them up. Empty / invalid values fall through to the
  // var()-declared defaults in Editor.svelte.
  function applyEditorFont(font: { family?: string; size?: number }) {
    if (typeof document === "undefined") return;
    const root = document.documentElement;
    if (font.family && font.family.trim()) {
      root.style.setProperty("--editor-font-family", font.family);
    } else {
      root.style.removeProperty("--editor-font-family");
    }
    if (typeof font.size === "number" && font.size >= 8 && font.size <= 48) {
      root.style.setProperty("--editor-font-size", `${font.size}px`);
    } else {
      root.style.removeProperty("--editor-font-size");
    }
  }

  // Most-recently-used files, fed from settings.json on the Go side.
  let recentFiles: string[] = [];
  let recentMenuOpen = false;

  // Theme: "system" follows prefers-color-scheme, "light"/"dark" pin it.
  // Stored in settings.Theme on the Go side; defaults to "system".
  type Theme = "system" | "light" | "dark";
  let theme: Theme = "system";
  // Track the OS preference so "system" effective theme updates live.
  let systemDark = typeof window !== "undefined" &&
    window.matchMedia("(prefers-color-scheme: dark)").matches;

  $: effectiveTheme = theme === "system" ? (systemDark ? "dark" : "light") : theme;
  $: if (typeof document !== "undefined") {
    document.documentElement.setAttribute("data-theme", effectiveTheme);
  }

  async function cycleTheme() {
    const next: Theme = theme === "system" ? "light" : theme === "light" ? "dark" : "system";
    theme = next;
    await patchSettings((s) => { s.theme = next; });
  }

  // patchSettings: load, mutate, save. Used by theme cycle, view toggle,
  // and validation-pane resize. Silent on error — persistence is a
  // convenience, not correctness.
  async function patchSettings(mutate: (s: any) => void) {
    const App = await wailsApp();
    if (!App) return;
    try {
      const s = await App.LoadSettings();
      mutate(s);
      await App.SaveSettings(s);
    } catch (e) {
      console.warn("[fbe] settings save failed:", e);
    }
  }

  function switchView(v: View) {
    view = v;
    void patchSettings((s) => { s.lastView = v; });
  }

  function onPanelResize(e: CustomEvent<{ height: number }>) {
    const h = Math.max(0, Math.round(e.detail.height));
    void patchSettings((s) => {
      if (!s.panes) s.panes = { outlineWidth: 0, validationWidth: 0, validationErrorsHeight: 0 };
      s.panes.validationErrorsHeight = h;
    });
  }

  function clamp(v: number, lo: number, hi: number) {
    return Math.max(lo, Math.min(hi, v));
  }

  // --- Outline sidebar resizer (left edge, drags right-ward) ---
  function startDragOutline(e: PointerEvent) {
    e.preventDefault();
    draggingOutline = true;
    (e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
    document.body.style.cursor = "ew-resize";
    document.body.style.userSelect = "none";
  }
  function onDragOutline(e: PointerEvent) {
    if (!draggingOutline || !mainEl) return;
    const rect = mainEl.getBoundingClientRect();
    outlineWidth = clamp(e.clientX - rect.left, OUTLINE_MIN, OUTLINE_MAX);
  }
  function endDragOutline(e: PointerEvent) {
    if (!draggingOutline) return;
    draggingOutline = false;
    const el = e.currentTarget as HTMLElement;
    if (el.hasPointerCapture(e.pointerId)) el.releasePointerCapture(e.pointerId);
    document.body.style.cursor = "";
    document.body.style.userSelect = "";
    if (outlineWidth !== null) {
      const w = Math.round(outlineWidth);
      void patchSettings((s) => {
        if (!s.panes) s.panes = { outlineWidth: 0, validationWidth: 0, validationErrorsHeight: 0 };
        s.panes.outlineWidth = w;
      });
    }
  }

  // --- Validation-panel resizer (drags toward left = panel wider,
  //     toward right = panel narrower) ---
  function startDragPanel(e: PointerEvent) {
    e.preventDefault();
    draggingPanel = true;
    (e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
    document.body.style.cursor = "ew-resize";
    document.body.style.userSelect = "none";
  }
  function onDragPanel(e: PointerEvent) {
    if (!draggingPanel) return;
    // In body view mainEl is the parent; in description view descWrapEl is.
    const parent = (view === "body" ? mainEl : descWrapEl) ?? undefined;
    if (!parent) return;
    const rect = parent.getBoundingClientRect();
    // Cap max at 70% of parent so the editor surface can't collapse to
    // zero. Min keeps the errors list legible.
    panelWidth = clamp(rect.right - e.clientX, PANEL_MIN, rect.width * 0.7);
  }
  function endDragPanel(e: PointerEvent) {
    if (!draggingPanel) return;
    draggingPanel = false;
    const el = e.currentTarget as HTMLElement;
    if (el.hasPointerCapture(e.pointerId)) el.releasePointerCapture(e.pointerId);
    document.body.style.cursor = "";
    document.body.style.userSelect = "";
    if (panelWidth !== null) {
      const w = Math.round(panelWidth);
      void patchSettings((s) => {
        if (!s.panes) s.panes = { outlineWidth: 0, validationWidth: 0, validationErrorsHeight: 0 };
        s.panes.validationWidth = w;
      });
    }
  }

  function onResizerKeyH(
    e: KeyboardEvent,
    get: () => number,
    set: (v: number) => void,
    min: number,
    max: number,
    persist: (w: number) => void,
  ) {
    const step = e.shiftKey ? 40 : 10;
    let changed = false;
    if (e.key === "ArrowLeft") {
      e.preventDefault();
      set(clamp(get() - step, min, max));
      changed = true;
    } else if (e.key === "ArrowRight") {
      e.preventDefault();
      set(clamp(get() + step, min, max));
      changed = true;
    }
    if (changed) persist(Math.round(get()));
  }

  function onOutlineResizerKey(e: KeyboardEvent) {
    onResizerKeyH(
      e,
      () => outlineWidth ?? 260,
      (v) => (outlineWidth = v),
      OUTLINE_MIN,
      OUTLINE_MAX,
      (w) => patchSettings((s) => {
        if (!s.panes) s.panes = { outlineWidth: 0, validationWidth: 0, validationErrorsHeight: 0 };
        s.panes.outlineWidth = w;
      }),
    );
  }

  function onPanelResizerKey(e: KeyboardEvent) {
    const parent = view === "body" ? mainEl : descWrapEl;
    if (!parent) return;
    const rect = parent.getBoundingClientRect();
    onResizerKeyH(
      e,
      () => panelWidth ?? Math.round(rect.width * 0.3),
      (v) => (panelWidth = v),
      PANEL_MIN,
      rect.width * 0.7,
      (w) => patchSettings((s) => {
        if (!s.panes) s.panes = { outlineWidth: 0, validationWidth: 0, validationErrorsHeight: 0 };
        s.panes.validationWidth = w;
      }),
    );
  }

  function themeIcon(t: Theme): string {
    switch (t) {
      case "light":  return "☀";
      case "dark":   return "☾";
      default:       return "◐";
    }
  }

  async function wailsApp() {
    return await import("../wailsjs/go/main/App").catch(() => null);
  }

  async function refreshRecent() {
    const App = await wailsApp();
    if (!App) return;
    try { recentFiles = (await App.RecentFiles()) ?? []; } catch { /* ignore */ }
  }

  // `preset` lets the Recent-files menu call openFile with a specific path
  // (skipping the native file-picker). When omitted we fall back to
  // `App.PickFB2ToOpen()` as before.
  async function openFile(preset?: string) {
    error = ""; status = "";
    recentMenuOpen = false;
    try {
      const App = await wailsApp();
      if (!App) throw new Error("Wails bindings not available — running in plain vite dev. Loaded bundled sample.");
      const path = preset ?? await App.PickFB2ToOpen();
      if (!path) return;
      console.log(`[fbe] opening ${path}`);
      status = `Opening ${path.split(/[\\/]/).pop()}…`;
      const parsed: FictionBook = await App.OpenFile(path);
      console.log(
        `[fbe] parsed: ${parsed.Bodies?.length ?? 0} bodies, ` +
        `${parsed.Binaries?.length ?? 0} binaries, ` +
        `title "${parsed.Description?.TitleInfo?.BookTitle ?? ""}"`,
      );
      // Defer to next tick so the status update renders before we potentially
      // block the UI thread converting a huge doc into ProseMirror nodes.
      await new Promise((r) => setTimeout(r, 0));
      fb = parsed;
      currentPath = path;
      filename = path.split(/[\\/]/).pop() ?? path;
      status = `Opened ${filename}`;
      setTimeout(() => (status = ""), 3000);
      void refreshRecent();
    } catch (e) {
      const msg = (e as Error).message || String(e);
      console.error("[fbe] openFile failed:", e);
      error = `Open failed: ${msg}`;
      if (preset) {
        // Recent-file pointed at something that no longer exists (or is
        // unreadable) — purge it so the menu doesn't keep offering a dead
        // entry.
        const App = await wailsApp();
        try { await App?.RemoveFromRecent(preset); } catch { /* ignore */ }
        void refreshRecent();
      } else {
        fb = SAMPLE_BOOK;
        currentPath = "";
        filename = "blank.fb2 (sample)";
      }
    }
  }

  async function save(saveAs: boolean) {
    error = ""; status = "";
    try {
      const App = await wailsApp();
      if (!App) throw new Error("Wails bindings not available — save works only in the desktop app.");
      if (!editor) throw new Error("Editor not ready.");
      const current = editor.currentFB();
      if (!current) throw new Error("No document loaded.");

      let path = currentPath;
      if (saveAs || !path) {
        const suggested = filename.endsWith(".fb2") ? filename : "untitled.fb2";
        path = await App.PickFB2ToSave(suggested);
        if (!path) return;
      }
      // @ts-expect-error — Wails-generated type uses doc.FictionBook shape.
      await App.UpdateDocument(current);
      await App.SaveFile(path);
      currentPath = path;
      filename = path.split(/[\\/]/).pop() ?? path;
      status = `Saved ${filename}`;
      setTimeout(() => (status = ""), 3000);
      void refreshRecent();
    } catch (e) {
      error = (e as Error).message;
    }
  }

  async function exportHTML() {
    error = ""; status = "";
    try {
      const App = await wailsApp();
      if (!App) throw new Error("Wails bindings not available.");
      if (!editor) throw new Error("Editor not ready.");
      const current = editor.currentFB();
      if (current) {
        // @ts-expect-error
        await App.UpdateDocument(current);
      }
      const suggested = (filename || "untitled").replace(/\.fb2(\.zip)?$/i, "") + ".html";
      const path = await App.PickHTMLToSave(suggested);
      if (!path) return;
      await App.ExportHTML(path);
      status = `Exported ${path.split(/[\\/]/).pop() ?? path}`;
      setTimeout(() => (status = ""), 3000);
    } catch (e) {
      error = (e as Error).message;
    }
  }

  async function validate() {
    error = ""; status = "";
    try {
      const App = await wailsApp();
      if (!App) throw new Error("Wails bindings not available.");
      if (!fb) throw new Error("No document loaded.");

      // Push the latest editor state to Go (if we're in body view) so the
      // serialized XML and the validation result reflect unsaved edits.
      const current = (view === "body" && editor) ? editor.currentFB() : fb;
      if (current) {
        // @ts-expect-error — Wails-generated type uses doc.FictionBook shape.
        await App.UpdateDocument(current);
      }

      const [xml, errs] = await Promise.all([
        App.SerializeCurrent(),
        App.ValidateCurrent(),
      ]);

      xmlSource = xml ?? "";
      validationErrors = errs ?? [];
      showPanel = true;

      status = errs && errs.length > 0
        ? `XSD: ${errs.length} error(s)`
        : "XSD valid ✓";
      setTimeout(() => (status = ""), 4000);
    } catch (e) {
      error = (e as Error).message;
    }
  }

  // Table of app-level actions that need a window-level keyboard listener:
  // either they sit outside the editor DOM (SearchBar, dialogs) or they
  // must run in both views (Save). Editor-internal commands (Bold, etc.)
  // are handled by ProseMirror's keymap inside Editor.svelte, built from
  // the same hotkeys map — see Editor.svelte::buildKeymap.
  const APP_ACTIONS: Record<string, () => void> = {
    Save:         () => save(false),
    SaveAs:       () => save(true),
    Find:         () => { if (view === "body") openSearch("find"); },
    Replace:      () => { if (view === "body") openSearch("replace"); },
    InsertTable:  () => { if (view === "body") editor?.openTableDialog(); },
    OpenBinaries: () => { if (view === "body") showBinaries = true; },
    OpenSettings: () => { showSettings = true; },
    OpenHelp:     () => { showHelp = true; },
  };

  function onKeyDown(e: KeyboardEvent) {
    // Walk the catalog rather than the APP_ACTIONS keys so actions without
    // a binding are cheap no-ops. Stop at the first match — once an action
    // fires, we preventDefault and return so the event doesn't also reach
    // the editor's PM keymap and double-dispatch.
    for (const action of HOTKEY_ACTIONS) {
      if (action.editor) continue; // PM handles editor-level bindings
      const accel = hotkeys[action.id];
      if (!accel) continue;
      if (matchesEvent(e, accel)) {
        const handler = APP_ACTIONS[action.id];
        if (handler) {
          e.preventDefault();
          handler();
        }
        return;
      }
    }
  }

  onMount(() => {
    document.title = "FictionBook Editor (Go)";
    window.addEventListener("keydown", onKeyDown);
    // Route every external <a href> click (editor content, Help, …) through
    // Wails runtime so the webview doesn't navigate away from the editor.
    const detachExternalLinks = installExternalLinkHandler();
    void refreshRecent();

    // Live-follow OS color-scheme changes while theme === "system".
    const mq = window.matchMedia("(prefers-color-scheme: dark)");
    const onSystemChange = (e: MediaQueryListEvent) => { systemDark = e.matches; };
    mq.addEventListener("change", onSystemChange);

    // Load persisted preferences: theme, last-open view, validation-pane
    // errors-height. Settings-read errors are swallowed — we'd rather use
    // in-memory defaults than block mount.
    void (async () => {
      const App = await wailsApp();
      if (!App) return;
      try {
        const s = await App.LoadSettings();
        const t = s?.theme;
        if (t === "light" || t === "dark" || t === "system") {
          theme = t;
        }
        if (s?.lastView === "body" || s?.lastView === "description") {
          view = s.lastView;
        }
        const h = s?.panes?.validationErrorsHeight;
        if (typeof h === "number" && h > 0) {
          initialErrorsHeight = h;
        }
        const ow = s?.panes?.outlineWidth;
        if (typeof ow === "number" && ow >= OUTLINE_MIN && ow <= OUTLINE_MAX) {
          outlineWidth = ow;
        }
        const pw = s?.panes?.validationWidth;
        if (typeof pw === "number" && pw >= PANEL_MIN) {
          panelWidth = pw;
        }
        if (s?.font) {
          applyEditorFont({ family: s.font.family, size: s.font.size });
        }
        if (s?.nbspChar) {
          configurePaste({ nbspChar: s.nbspChar });
        }
        if (s?.hotkeys) {
          // Fresh object so Editor's reactive reconfigure runs once on first
          // load even if the Wails binding happens to return the same ref.
          hotkeys = { ...s.hotkeys };
        }
      } catch { /* leave defaults */ }
    })();
    // Pick up whatever Go already has open (so opening :34115 in a browser
    // tab while a file is loaded in the native window shows that file
    // instead of the sample). Path is intentionally NOT synced — Save in
    // the dev-tab should land in Save-As to avoid two contexts racing on
    // the same path.
    void (async () => {
      const App = await wailsApp();
      if (App) {
        try {
          const current = await App.CurrentDocument();
          if (current && current.Bodies && current.Bodies.length > 0) {
            fb = current as FictionBook;
            filename = "(opened in native window)";
            return;
          }
        } catch { /* fall through to sample */ }
      }
      fb = SAMPLE_BOOK;
      filename = "blank.fb2 (sample)";
    })();
    return () => {
      window.removeEventListener("keydown", onKeyDown);
      mq.removeEventListener("change", onSystemChange);
      detachExternalLinks();
    };
  });
</script>

<div class="layout">
  <header>
    <div class="open-group">
      <button on:click={() => openFile()}>Open…</button>
      <button
        class="recent-toggle"
        title="Recent files"
        disabled={recentFiles.length === 0}
        on:click={() => (recentMenuOpen = !recentMenuOpen)}
      >▾</button>
      {#if recentMenuOpen}
        <!-- svelte-ignore a11y-click-events-have-key-events -->
        <!-- svelte-ignore a11y-no-static-element-interactions -->
        <div class="recent-backdrop" on:click={() => (recentMenuOpen = false)}></div>
        <ul class="recent-menu" role="menu">
          {#each recentFiles as path}
            <li role="menuitem">
              <button
                type="button"
                class="recent-item"
                on:click={() => openFile(path)}
                title={path}
              >
                <span class="basename">{path.split(/[\\/]/).pop() ?? path}</span>
                <span class="dir">{path.slice(0, -((path.split(/[\\/]/).pop() ?? path).length))}</span>
              </button>
            </li>
          {/each}
        </ul>
      {/if}
    </div>
    <button on:click={() => save(false)} disabled={!editor}>Save</button>
    <button on:click={() => save(true)} disabled={!editor}>Save As…</button>
    <button on:click={validate} disabled={!fb}>Validate</button>
    <button on:click={exportHTML} disabled={!editor}>Export HTML…</button>
    <button on:click={() => (showSettings = true)} title="Settings" aria-label="Open settings">⚙</button>
    <button on:click={() => (showHelp = true)} title="Keyboard shortcuts and about">Help</button>
    <button
      class="theme-toggle"
      on:click={cycleTheme}
      title={`Theme: ${theme}${theme === "system" ? ` (effective: ${effectiveTheme})` : ""} — click to cycle`}
      aria-label="Cycle theme (system → light → dark)"
    >{themeIcon(theme)}</button>
    <div class="view-toggle" role="tablist" aria-label="View">
      <button
        class:active={view === "body"}
        on:click={() => switchView("body")}
        role="tab"
        aria-selected={view === "body"}>Body</button>
      <button
        class:active={view === "description"}
        on:click={() => switchView("description")}
        role="tab"
        aria-selected={view === "description"}>Description</button>
    </div>
    <span class="title">FictionBook Editor · <em>{filename}</em></span>
    {#if status}<span class="status">{status}</span>{/if}
    {#if error}<span class="err">{error}</span>{/if}
  </header>

  {#if view === "body"}
    <Toolbar {editor} on:openBinaries={() => (showBinaries = true)} />
    {#if searchOpen}
      <SearchBar
        view={editorView}
        bind:mode={searchMode}
        on:close={() => (searchOpen = false)}
      />
    {/if}
    <main
      bind:this={mainEl}
      class:with-panel={showPanel}
      style={`${outlineWidth !== null ? `--outline-w: ${outlineWidth}px;` : ""}${panelWidth !== null && showPanel ? `--panel-w: ${panelWidth}px;` : ""}`}
    >
      <aside>
        <DocumentTree {fb} on:navigate={(e) => editor?.scrollToPath(e.detail.path)} />
      </aside>
      <!-- svelte-ignore a11y-no-noninteractive-tabindex -->
      <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
      <div
        class="v-resizer"
        class:dragging={draggingOutline}
        role="separator"
        aria-orientation="vertical"
        aria-label="Resize outline sidebar"
        tabindex="0"
        on:pointerdown={startDragOutline}
        on:pointermove={onDragOutline}
        on:pointerup={endDragOutline}
        on:pointercancel={endDragOutline}
        on:keydown={onOutlineResizerKey}
      ></div>
      <section><Editor bind:this={editor} bind:view={editorView} {fb} {hotkeys} /></section>
      {#if showPanel}
        <!-- svelte-ignore a11y-no-noninteractive-tabindex -->
        <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
        <div
          class="v-resizer"
          class:dragging={draggingPanel}
          role="separator"
          aria-orientation="vertical"
          aria-label="Resize validation panel"
          tabindex="0"
          on:pointerdown={startDragPanel}
          on:pointermove={onDragPanel}
          on:pointerup={endDragPanel}
          on:pointercancel={endDragPanel}
          on:keydown={onPanelResizerKey}
        ></div>
        <ValidationPanel
          {xmlSource}
          errors={validationErrors}
          {initialErrorsHeight}
          on:close={() => (showPanel = false)}
          on:resize={onPanelResize}
        />
      {/if}
    </main>
  {:else if fb}
    <div class="spacer" />
    <div
      bind:this={descWrapEl}
      class="description-wrap with-panel-maybe"
      class:with-panel={showPanel}
      style={panelWidth !== null && showPanel ? `--panel-w: ${panelWidth}px;` : ""}
    >
      <DescriptionPanel bind:fb />
      {#if showPanel}
        <!-- svelte-ignore a11y-no-noninteractive-tabindex -->
        <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
        <div
          class="v-resizer"
          class:dragging={draggingPanel}
          role="separator"
          aria-orientation="vertical"
          aria-label="Resize validation panel"
          tabindex="0"
          on:pointerdown={startDragPanel}
          on:pointermove={onDragPanel}
          on:pointerup={endDragPanel}
          on:pointercancel={endDragPanel}
          on:keydown={onPanelResizerKey}
        ></div>
        <ValidationPanel
          {xmlSource}
          errors={validationErrors}
          {initialErrorsHeight}
          on:close={() => (showPanel = false)}
          on:resize={onPanelResize}
        />
      {/if}
    </div>
  {/if}
</div>

<HelpDialog bind:open={showHelp} />
<SettingsDialog bind:open={showSettings} on:apply={onSettingsApplied} />
<BinaryManagerDialog bind:open={showBinaries} bind:fb view={editorView} />

<style>
  /* Theme palette. Applied via [data-theme="light|dark"] on <html>; the
     default (no attribute) also resolves to light so server-rendered
     previews look right. Components reference these vars; hard-coded
     hex should stay rare and documented. */
  :global(:root),
  :global([data-theme="light"]) {
    color-scheme: light;
    --bg-app:          #fafaf7;
    --bg-surface:      #ffffff;   /* editor paper */
    --bg-chrome:       #f1f1ec;   /* header, panel-title, toolbar */
    --bg-sidebar:      #f5f5f0;
    --bg-card:         #fffdf8;   /* dialogs, menus */
    --bg-hover:        #fff8e5;
    --bg-active:       #fce6a0;
    --bg-active-hover: #f5da7c;
    --bg-errors:       #fffaf0;
    --bg-errors-title: #fdecec;
    --bg-ok:           #f2faf4;

    --fg:              #222;
    --fg-strong:       #111;
    --fg-secondary:    #444;
    --fg-muted:        #888;
    --fg-muted-soft:   #aaa;
    --fg-link:         #1a5490;

    --border:          #d5d5cb;
    --border-strong:   #bfbfb2;
    --border-input:    #ccc;
    --border-button:   #bbb;

    --danger:          #a33;
    --danger-border:   #edc7c7;
    --ok:              #2a7;
    --ok-border:       #cfe7d6;
    --warn:            #b58f00;   /* raw-block accent */
    --warn-fg:         #7a5a10;
    --warn-bg-a:       #fef7d8;
    --warn-bg-b:       #fcebad;
    --warn-bg-inline:  #fef0bc;

    --highlight:       #fce6a0;
    --shadow:          rgba(0, 0, 0, 0.25);
    --backdrop:        rgba(0, 0, 0, 0.35);   /* modal dim */

    /* Search/replace match highlighting — light wash on inactive hits,
       saturated orange on the currently-focused one (matches VS Code). */
    --search-match-bg:            #ffe89a;
    --search-match-active-bg:     #ffb347;
    --search-match-active-border: #d07b0a;
  }

  :global([data-theme="dark"]) {
    color-scheme: dark;
    --bg-app:          #1a1a1a;
    --bg-surface:      #242422;
    --bg-chrome:       #2a2a27;
    --bg-sidebar:      #242422;
    --bg-card:         #2a2a27;
    --bg-hover:        #3a3630;
    --bg-active:       #5a4a10;
    --bg-active-hover: #6b5814;
    --bg-errors:       #2a2420;
    --bg-errors-title: #3a2222;
    --bg-ok:           #1a2a22;

    --fg:              #e4e4de;
    --fg-strong:       #f5f5ef;
    --fg-secondary:    #c0c0ba;
    --fg-muted:        #8a8a82;
    --fg-muted-soft:   #5a5a52;
    --fg-link:         #7fb6e6;

    --border:          #3a3a35;
    --border-strong:   #4a4a42;
    --border-input:    #4a4a42;
    --border-button:   #4a4a42;

    --danger:          #e88;
    --danger-border:   #5a2a2a;
    --ok:              #7ec99f;
    --ok-border:       #2a4a38;
    --warn:            #ebc550;
    --warn-fg:         #f5d878;
    --warn-bg-a:       #3a2e10;
    --warn-bg-b:       #4a3818;
    --warn-bg-inline:  #3a2e10;

    --highlight:       #5a4a10;
    --shadow:          rgba(0, 0, 0, 0.6);
    --backdrop:        rgba(0, 0, 0, 0.55);   /* modal dim — stronger in dark mode */

    --search-match-bg:            #6b5814;
    --search-match-active-bg:     #c88420;
    --search-match-active-border: #e8a850;
  }

  :global(body), :global(html) {
    margin: 0;
    height: 100%;
    background: var(--bg-app);
    color: var(--fg);
  }
  .layout {
    display: grid;
    grid-template-rows: 2.5rem auto 1fr;
    height: 100vh;
    font-family: -apple-system, "Segoe UI", sans-serif;
  }
  header {
    display: flex;
    align-items: center;
    padding: 0 0.75rem;
    gap: 0.5rem;
    background: var(--bg-chrome);
    border-bottom: 1px solid var(--border);
  }
  header button {
    padding: 0.25rem 0.7rem;
    border: 1px solid var(--border-button);
    background: var(--bg-surface);
    color: var(--fg);
    border-radius: 4px;
    cursor: pointer;
  }
  header button:hover:not(:disabled) { background: var(--bg-hover); }
  header button:disabled { opacity: 0.5; cursor: default; }

  /* Theme cycle icon-button (lives on the right edge, near Help). */
  header button.theme-toggle {
    padding: 0.25rem 0.5rem;
    font-size: 0.95rem;
    line-height: 1;
  }

  /* Recent-files split-button: Open…<dropdown-caret>. Visually the two
     buttons share a border so they read as one control. */
  .open-group {
    position: relative;
    display: inline-flex;
    gap: 0;
  }
  .open-group > button:first-child {
    border-top-right-radius: 0;
    border-bottom-right-radius: 0;
  }
  .open-group > button.recent-toggle {
    border-top-left-radius: 0;
    border-bottom-left-radius: 0;
    border-left-width: 0;
    padding: 0.25rem 0.45rem;
    font-size: 0.7rem;
    line-height: 1;
  }
  .recent-backdrop {
    position: fixed;
    inset: 0;
    background: transparent;
    z-index: 49;
  }
  .recent-menu {
    position: absolute;
    top: 100%;
    left: 0;
    z-index: 50;
    list-style: none;
    margin: 2px 0 0;
    padding: 0.25rem 0;
    min-width: 22rem;
    max-width: 38rem;
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: 4px;
    box-shadow: 0 6px 18px var(--shadow);
    font-size: 0.85rem;
  }
  .recent-menu li { margin: 0; padding: 0; }
  button.recent-item {
    all: unset;
    display: block;
    width: 100%;
    padding: 0.3rem 0.7rem;
    cursor: pointer;
    box-sizing: border-box;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  button.recent-item:hover { background: var(--bg-hover); }
  button.recent-item .basename { font-weight: 600; color: var(--fg); }
  button.recent-item .dir {
    color: var(--fg-muted);
    font-size: 0.78rem;
    margin-left: 0.4rem;
  }

  .view-toggle {
    display: inline-flex;
    gap: 0;
    margin-left: 0.5rem;
  }
  .view-toggle button {
    border-radius: 0;
    border-right-width: 0;
  }
  .view-toggle button:first-child { border-radius: 4px 0 0 4px; }
  .view-toggle button:last-child { border-radius: 0 4px 4px 0; border-right-width: 1px; }
  .view-toggle button.active {
    background: var(--bg-active);
    font-weight: 600;
  }
  .description-wrap {
    overflow: hidden;
  }
  .spacer { height: 0; }
  .title { font-size: 0.9rem; color: var(--fg-secondary); margin-left: 0.5rem; }
  .status { color: var(--ok); font-size: 0.8rem; margin-left: auto; }
  .err { color: var(--danger); font-size: 0.8rem; margin-left: auto; }
  main {
    display: grid;
    /* outline | resizer | editor. Resizer is a 6px track so the cursor
       target is big enough to grab comfortably. */
    grid-template-columns: var(--outline-w, 260px) 6px 1fr;
    overflow: hidden;
  }
  main.with-panel {
    /* outline | resizer | editor | resizer | validation-panel. */
    grid-template-columns:
      var(--outline-w, 260px)
      6px
      1fr
      6px
      var(--panel-w, minmax(320px, 30%));
  }
  aside {
    overflow: auto;
    background: var(--bg-sidebar);
    font-size: 0.9rem;
  }
  section {
    overflow: auto;
    background: var(--bg-surface);
  }
  .description-wrap.with-panel-maybe.with-panel {
    display: grid;
    grid-template-columns: 1fr 6px var(--panel-w, minmax(320px, 30%));
  }

  /* Vertical drag handle (separator between two columns). Shares its
     styling approach with ValidationPanel's horizontal resizer but lives
     here because these two handles are App.svelte's responsibility. */
  .v-resizer {
    background: var(--border);
    cursor: ew-resize;
    border-left: 1px solid var(--border-strong);
    border-right: 1px solid var(--border-strong);
    touch-action: none;
    position: relative;
  }
  .v-resizer::before {
    content: "";
    position: absolute;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    width: 2px;
    height: 32px;
    background: var(--fg-muted);
    border-radius: 1px;
    box-shadow: 3px 0 0 var(--fg-muted);
  }
  .v-resizer:hover,
  .v-resizer.dragging,
  .v-resizer:focus-visible {
    background: var(--border-strong);
    outline: none;
  }
  .v-resizer:focus-visible::before {
    background: var(--fg-secondary);
    box-shadow: 3px 0 0 var(--fg-secondary);
  }
</style>
