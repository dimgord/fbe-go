<script lang="ts">
  import { onMount } from "svelte";
  import DocumentTree from "./tree/DocumentTree.svelte";
  import Editor from "./editor/Editor.svelte";
  import Toolbar from "./editor/Toolbar.svelte";
  import DescriptionPanel from "./description/DescriptionPanel.svelte";
  import ValidationPanel from "./validation/ValidationPanel.svelte";
  import HelpDialog from "./help/HelpDialog.svelte";
  import SettingsDialog from "./settings/SettingsDialog.svelte";
  import { backend } from "./runtime/backend";
  import { applyEditorFont, effectiveTheme, isTheme, nextTheme, theme, type Theme } from "./ui/theme";
  import { patchSettings, persistPane } from "./settings/store";
  import { createDocumentSession } from "./document/session";
  import AppHeader from "./ui/AppHeader.svelte";
  import SearchBar from "./editor/search/SearchBar.svelte";
  import Resizer from "./ui/Resizer.svelte";
  import PositionBar from "./tree/PositionBar.svelte";
  import { blockAtLine, xmlBlockLines, type BookPosition } from "./tree/position";
  import type { OutlineNode } from "./tree/outline";
  import type { Node as PMNode } from "prosemirror-model";
  import BinaryManagerDialog from "./binary/BinaryManagerDialog.svelte";
  import UpdateBanner from "./updates/UpdateBanner.svelte";
  import type { updates as UpdatesNS } from "../wailsjs/go/models";
  import { installExternalLinkHandler } from "./runtime/externalLink";
  import { configurePaste } from "./editor/paste";
  import { HOTKEY_ACTIONS, matchesEvent } from "./settings/hotkeys";

  type View = "body" | "description";
  let view: View = "body";
  let position: BookPosition | null = null;
  /** XML line per text block, for the panel to follow the reader. */
  let blockLines: number[] = [];

  let editor: Editor | undefined = undefined;
  /** Bound to Editor.view so SearchBar (and any other sibling) gets a
      reactive reference to the live PM EditorView. */
  let editorView: import("prosemirror-view").EditorView | undefined = undefined;

  const session = createDocumentSession({
    liveBody: () => (view === "body" && editor ? editor.currentFB() : null),
    onSaved: () => editor?.markSaved(),
  });
  const {
    doc, filename, status, error, recentFiles,
    panelOpen, xmlSource, xmlErrors, applyError, canUndoApply,
  } = session;

  /** When the user leaves the "body" view (tab-switch to description, or
   *  any other reason `view` changes), serialize the live PM doc back
   *  into `fb` so subsequent re-mounts of Editor (and any save) see the
   *  user's body edits. Without this, switching tabs unmounts Editor via
   *  `{#if view === "body"}`, the PM state is gone, and on return the
   *  freshly-mounted Editor rebuilds from a stale `fb` — the body edits
   *  silently disappear. */
  let prevView: View = view;
  $: {
    if (prevView === "body" && view !== "body" && editor) {
      const synced = editor.currentFB();
      if (synced) $doc = synced;
    }
    prevView = view;
  }

  // While ANY internal divider (outline resizer, validation-panel resizer)
  // is being dragged, set body.resizing so the heavy contenteditable + XML
  // pane drop out of layout via the global CSS rule. window.resize events
  // DO NOT fire when the user drags a divider — the OS window stays the
  // same size; only an internal grid-column width changes — so the
  // onWindowResize debounce path below never catches these. This reactive
  // mirror catches them at the source: draggingOutline/draggingPanel are
  // bound to the Resizers, which set them on actual pointerdown/up,
  // which is as close to "while mouse is held" as the browser can give us.
  $: if (typeof document !== "undefined") setResizing(draggingOutline || draggingPanel);

  /** Hide the heavy editor for a resize. Hiding empties its scroller, so the place is kept. */
  function setResizing(on: boolean): void {
    if (on === document.body.classList.contains("resizing")) return;
    if (on) editor?.holdPosition();
    document.body.classList.toggle("resizing", on);
    if (!on) editor?.releasePosition();
  }

  // The XML pane keeps up with the document while it is open, except while the user edits the XML
  // itself: that buffer is theirs until Apply or Done.
  let outline: OutlineNode[] = [];
  let settledDoc: PMNode | null = null;
  let caretBlock: number | null = null;
  function onDocChange(e: CustomEvent<PMNode>) {
    settledDoc = e.detail;
    if (!xmlEditMode) void session.refreshXml();
  }
  // Above the first block is the description: nothing in the text matches it.
  function onXmlCaret(line: number) {
    if (view !== "body" || !blockLines.length) return;
    caretBlock = null;
    editor?.markBlock(line < blockLines[0] ? null : blockAtLine(blockLines, line));
  }
  $: if (!$panelOpen) editor?.markBlock(null);
  $: blockLines = settledDoc && $xmlSource ? xmlBlockLines(settledDoc, $xmlSource) : [];

  // XML panel. editMode lives here so that a Done in the panel and closing it agree.
  let xmlEditMode = false;
  function onXmlDirty(e: CustomEvent<{ dirty: boolean }>) {
    session.setXmlDirty(e.detail.dirty);
  }
  // A stale Apply error must not greet the user on the next edit session.
  let prevXmlEditMode = false;
  $: {
    if (xmlEditMode !== prevXmlEditMode) {
      $applyError = "";
      if (!xmlEditMode) void session.refreshXml();
    }
    prevXmlEditMode = xmlEditMode;
  }
  function onPanelClose() {
    xmlEditMode = false;
    session.closePanel();
  }

  // Initial errors-pane height in px (null = CSS default 45%). Populated
  // from settings.panes.validationErrorsHeight on mount; updated when
  // ValidationPanel dispatches `resize`.
  let initialErrorsHeight: number | null = null;

  // Outline-sidebar and validation-panel column widths (px). Null = use
  // CSS default (260px and minmax(320px, 30%) respectively).
  let outlineWidth: number | null = null;
  let panelWidth: number | null = null;
  let mainEl: HTMLElement | undefined;
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

  /** Update banner state. Populated asynchronously shortly after mount by
      a non-blocking GitHub Releases poll. Null = no banner (either no
      update, check failed, or user dismissed this session). */
  let updateInfo: UpdatesNS.Info | null = null;

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
    theme.set(e.detail.theme);
    applyEditorFont(e.detail.settings.font);
    configurePaste({ nbspChar: e.detail.settings.nbspChar });
    // Fresh object identity so Editor's reactive reconfigure fires.
    if (e.detail.settings.hotkeys) {
      hotkeys = { ...e.detail.settings.hotkeys };
    }
    // Refresh recent-files list in case the dialog cleared it.
    void session.refreshRecent();
  }


  $: if (typeof document !== "undefined") {
    document.documentElement.setAttribute("data-theme", $effectiveTheme);
  }

  function cycleTheme() {
    const next = nextTheme($theme);
    theme.set(next);
    void patchSettings((s) => { s.theme = next; });
  }

  function switchView(v: View) {
    view = v;
    void patchSettings((s) => { s.lastView = v; });
  }

  // --- Pane resizers. ---
  const DEFAULT_OUTLINE_WIDTH = 260;
  const RESIZER_PX = 6;
  const TEXT_MIN = 240;
  const mainBox = () => mainEl?.getBoundingClientRect();

  const outlineResizer = {
    current: () => outlineWidth ?? DEFAULT_OUTLINE_WIDTH,
    bounds: () => ({ min: OUTLINE_MIN, max: OUTLINE_MAX }),
    fromPointer: (e: PointerEvent) => e.clientX - (mainBox()?.left ?? 0),
  };
  const panelResizer = {
    current: () => panelWidth ?? Math.round((mainBox()?.width ?? 0) * 0.3),
    // Stops where the grid would start taking width from the text.
    bounds: () => {
      const outline = view === "body" ? (outlineWidth ?? DEFAULT_OUTLINE_WIDTH) + RESIZER_PX : 0;
      return { min: PANEL_MIN, max: Math.max(PANEL_MIN, (mainBox()?.width ?? 0) - outline - RESIZER_PX - TEXT_MIN) };
    },
    fromPointer: (e: PointerEvent) => (mainBox()?.right ?? 0) - e.clientX,
  };

  // Table of app-level actions that need a window-level keyboard listener:
  // either they sit outside the editor DOM (SearchBar, dialogs) or they
  // must run in both views (Save). Editor-internal commands (Bold, etc.)
  // are handled by ProseMirror's keymap inside Editor.svelte, built from
  // the same hotkeys map — see Editor.svelte::buildKeymap.
  const APP_ACTIONS: Record<string, () => void> = {
    Save:         () => session.save(false),
    SaveAs:       () => session.save(true),
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

  // Live window-resize damping. With a 7 000-line book in the editor and
  // the XML source panel open (~7 000 more divs), every continuous resize
  // event from a window-edge drag triggers a full reflow of both DOM
  // trees. Even with `content-visibility: auto` containment, on slower
  // machines this can run 200-500 ms per frame — the resize handle drags
  // with visible lag and snaps when released.
  //
  // Mitigation: detect a resize burst (≥2 events in <120 ms), toggle a
  // body-level `resizing` class that hides the heavy panes via CSS
  // `display: none`, then remove the class after 150 ms of resize-event
  // silence (i.e. after the user releases the window edge). The browser
  // can repaint a near-empty layout at 60 fps regardless of book size;
  // the brief "blink" of hidden content during drag is the trade-off.
  // Approximates the user request "resize only when mouse button is
  // released" — browsers don't expose mouseup on the window chrome,
  // so debounced silence is the next best proxy.
  let resizeTimer: number | undefined;
  function onWindowResize() {
    setResizing(true);
    if (resizeTimer !== undefined) window.clearTimeout(resizeTimer);
    resizeTimer = window.setTimeout(() => {
      setResizing(false);
      resizeTimer = undefined;
    }, 150);
  }

  onMount(() => {
    document.title = "FictionBook Editor (Go)";
    window.addEventListener("keydown", onKeyDown);
    window.addEventListener("resize", onWindowResize);
    // Route every external <a href> click (editor content, Help, …) through
    // Wails runtime so the webview doesn't navigate away from the editor.
    const detachExternalLinks = installExternalLinkHandler();
    void session.refreshRecent();

    // Wire the unsaved-changes plumbing:
    //   - "app:check-dirty" — Go OnBeforeClose asks whether the live doc
    //     differs from the last clean snapshot. We answer with a bool
    //     via "app:dirty-response".
    //   - "app:save-and-quit" — Go's "Save" dialog branch dispatched a
    //     save request. Run the save, then ForceQuit if it succeeded.
    let detachListeners: Array<() => void> = [];
    // Resolves with the file the OS launched us with ("" if none) once the listeners below are
    // registered, so a later Finder "Open With" event is never emitted into the void.
    let launchReady: (p: string) => void = () => {};
    const launchFile = new Promise<string>((r) => (launchReady = r));
    void (async () => {
      const App = backend();
      if (!App) { launchReady(""); return; }
      const offCheck = App.on("app:check-dirty", () => {
        App.emit("app:dirty-response", session.isDirty());
      });
      const offSave = App.on("app:save-and-quit", async () => {
        try {
          await session.save(false);
        } catch (e) {
          console.error("[fbe] save-and-quit: save failed", e);
          return;
        }
        if (!session.isDirty()) {
          try { await App.ForceQuit(); } catch { /* app is exiting */ }
        }
      });
      // File drop: Go's OnFileDrop forwards the first dropped path; we
      // reuse the normal open flow so the unsaved-changes guard,
      // status text, and recent-files update all behave identically to
      // a regular File→Open. Without this listener the drop would be
      // silently swallowed (WebView drop is disabled in main.go).
      const offDrop = App.on("app:file-drop", (path: string) => {
        if (typeof path === "string" && path) void session.open(path);
      });
      if (typeof offCheck === "function") detachListeners.push(offCheck);
      if (typeof offSave === "function") detachListeners.push(offSave);
      // OS "open this file" (double-click / Open With) while the app is already running.
      const offOpen = App.on("app:open-file", (path: string) => {
        if (typeof path === "string" && path) void session.open(path);
      });
      if (typeof offDrop === "function") detachListeners.push(offDrop);
      if (typeof offOpen === "function") detachListeners.push(offOpen);
      let p = "";
      try { p = (await App.TakeLaunchFile()) ?? ""; } catch { /* older backend */ }
      launchReady(p);
    })();

    // Load persisted preferences: theme, last-open view, validation-pane
    // errors-height. Settings-read errors are swallowed — we'd rather use
    // in-memory defaults than block mount.
    void (async () => {
      const App = backend();
      if (!App) return;
      try {
        const s = await App.LoadSettings();
        if (isTheme(s?.theme)) theme.set(s.theme);
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

    // Non-blocking update check. Fired after a small delay so it doesn't
    // compete with the settings-load / document-load round-trips for the
    // first paint. A network flake just means "no banner this launch",
    // which is the right trade-off — the banner's job is to be polite.
    const updateTimer = window.setTimeout(async () => {
      const App = backend();
      if (!App) return;
      try {
        const info = await App.CheckForUpdate();
        if (info && info.available) {
          updateInfo = info;
        }
      } catch (e) {
        console.info("[fbe] update check failed:", e);
      }
    }, 800);
    void launchFile.then(session.loadInitial);
    return () => {
      window.removeEventListener("keydown", onKeyDown);
      window.removeEventListener("resize", onWindowResize);
      if (resizeTimer !== undefined) window.clearTimeout(resizeTimer);
      detachExternalLinks();
      window.clearTimeout(updateTimer);
      detachListeners.forEach((off) => off());
    };
  });
</script>

<div class="layout">
  <UpdateBanner info={updateInfo} on:dismiss={() => (updateInfo = null)} />
  <AppHeader
    hasDocument={!!$doc}
    filename={$filename}
    status={$status}
    error={$error}
    recentFiles={$recentFiles}
    {view}
    on:open={(e) => session.open(e.detail)}
    on:save={(e) => session.save(e.detail)}
    on:validate={session.validate}
    on:export={session.exportHtml}
    on:settings={() => (showSettings = true)}
    on:help={() => (showHelp = true)}
    on:theme={cycleTheme}
    on:view={(e) => switchView(e.detail)}
  />

  {#if view === "body"}
    <Toolbar {editor} {hotkeys} on:openBinaries={() => (showBinaries = true)} />
    {#if searchOpen}
      <SearchBar
        view={editorView}
        bind:mode={searchMode}
        on:close={() => (searchOpen = false)}
      />
    {/if}
  {:else}
    <div class="spacer" />
  {/if}
  <main
    bind:this={mainEl}
    class:description={view !== "body"}
    class:with-panel={$panelOpen}
    style={`${outlineWidth !== null ? `--outline-w: ${outlineWidth}px;` : ""}${panelWidth !== null && $panelOpen ? `--panel-w: ${panelWidth}px;` : ""}`}
  >
    {#if view === "body"}
      <aside>
        <DocumentTree tree={outline} activePath={position?.path ?? null} on:navigate={(e) => editor?.scrollToPath(e.detail.path)} />
      </aside>
      <Resizer
        label="Resize outline sidebar"
        {...outlineResizer}
        bind:dragging={draggingOutline}
        on:change={(e) => (outlineWidth = e.detail)}
        on:commit={(e) => persistPane("outlineWidth", e.detail)}
      />
      <section>
        <Editor bind:this={editor} bind:view={editorView} fb={$doc} {hotkeys} on:dirty={(e) => session.setBodyDirty(e.detail)} on:position={(e) => (position = e.detail)} on:notice={(e) => session.showStatus(e.detail, 6000)} on:caret={(e) => (caretBlock = e.detail)} on:outline={(e) => (outline = e.detail)} on:docchange={onDocChange} />
        <PositionBar {position} />
      </section>
    {:else if $doc}
      <DescriptionPanel bind:fb={$doc} />
    {/if}
    {#if $panelOpen}
      <Resizer
        label="Resize validation panel"
        {...panelResizer}
        bind:dragging={draggingPanel}
        on:change={(e) => (panelWidth = e.detail)}
        on:commit={(e) => persistPane("validationWidth", e.detail)}
      />
      <ValidationPanel
        xmlSource={$xmlSource}
        followLine={view === "body" && position ? blockLines[position.block] ?? null : null}
        caretLine={view === "body" && caretBlock !== null ? blockLines[caretBlock] ?? null : null}
        errors={$xmlErrors}
        {initialErrorsHeight}
        theme={$effectiveTheme}
        applyError={$applyError}
        canUndoApply={$canUndoApply}
        bind:editMode={xmlEditMode}
        on:close={onPanelClose}
        on:resize={(e) => persistPane("validationErrorsHeight", Math.max(0, e.detail.height))}
        on:apply={(e) => session.applyXml(e.detail.xml)}
        on:undo-apply={session.undoApply}
        on:xml-dirty={onXmlDirty}
        on:scroll-line={(e) => editor?.scrollToBlock(blockAtLine(blockLines, e.detail))}
        on:caret-line={(e) => onXmlCaret(e.detail)}
      />
    {/if}
  </main>
</div>

<HelpDialog bind:open={showHelp} />
<SettingsDialog bind:open={showSettings} on:apply={onSettingsApplied} />
<BinaryManagerDialog bind:open={showBinaries} bind:fb={$doc} view={editorView} />

<style>
  /* Resize-burst damping (see onWindowResize + the dragging reactive
     mirror in <script>). When EITHER the user drags a window edge OR
     drags an internal divider (outline / validation-panel resizer),
     `body.resizing` is on. We hide the heavy contenteditable + XML
     pane via `display: none` so neither participates in the layout
     reflow — the browser can repaint a near-empty layout at 60 fps
     regardless of book size. The class clears on pointerup
     (internal-divider case) or 150 ms after the last window.resize
     event (window-edge case). The brief blink during drag is the
     trade-off for a non-laggy handle.

     Selector form: a single `:global(body.resizing .X)` rather than
     `:global(body.resizing) :global(.X)` — Svelte 4 only emits a
     descendant combinator when both ends are inside the SAME
     `:global()` modifier. The two-modifier form silently scopes the
     `.X` part to the component, which then never matches the
     globally-mounted ProseMirror / XML-line divs. */
  :global(body.resizing .ProseMirror) {
    display: none !important;
  }

  .layout {
    display: grid;
    grid-template-rows: 2.5rem auto 1fr;
    grid-template-columns: minmax(0, 1fr); /* never wider than the window, whatever the content */
    height: 100vh;
    font-family: var(--font-ui);
  }
  .spacer { height: 0; }
  /* Saved widths are upper limits: if they no longer fit the window these columns give way
     (the 1fr text column keeps its minimum), so a wide panel cannot squeeze the text to nothing. */
  main {
    display: grid;
    grid-template-columns: minmax(150px, var(--outline-w, 260px)) 6px minmax(240px, 1fr);
    overflow: hidden;
  }
  main.with-panel {
    grid-template-columns:
      minmax(150px, var(--outline-w, 260px))
      6px
      minmax(240px, 1fr)
      6px
      minmax(260px, var(--panel-w, 30%));
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
  main.description {
    grid-template-columns: 1fr;
  }
  main.description.with-panel {
    grid-template-columns: minmax(240px, 1fr) 6px minmax(260px, var(--panel-w, 30%));
  }
</style>
