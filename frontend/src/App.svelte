<script lang="ts">
  import { onMount, tick } from "svelte";
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
  import { descriptionSnapshot } from "./document/dirty";
  import { basename } from "./lib/path";
  import AppHeader from "./ui/AppHeader.svelte";
  import SearchBar from "./editor/search/SearchBar.svelte";
  import Resizer from "./ui/Resizer.svelte";
  import PositionBar from "./tree/PositionBar.svelte";
  import { blockAtLine, xmlBlockLines, type BookPosition } from "./tree/position";
  import BinaryManagerDialog from "./binary/BinaryManagerDialog.svelte";
  import UpdateBanner from "./updates/UpdateBanner.svelte";
  import type { updates as UpdatesNS } from "../wailsjs/go/models";
  import { installExternalLinkHandler } from "./runtime/externalLink";
  import { configurePaste } from "./editor/paste";
  import { SAMPLE_BOOK } from "./fb2/sample";
  import type { FictionBook } from "./fb2/types";
  import { HOTKEY_ACTIONS, matchesEvent } from "./settings/hotkeys";

  type View = "body" | "description";
  let view: View = "body";
  let position: BookPosition | null = null;
  /** XML line per text block, for the panel to follow the reader. */
  let blockLines: number[] = [];

  let fb: FictionBook | null = null;
  let filename = "(untitled)";
  let currentPath = "";
  let status = "";
  let statusTimer: ReturnType<typeof setTimeout> | undefined;

  /** A transient message. A newer one cancels the older one's timer, so it is not cut short. */
  function showStatus(message: string, ms: number): void {
    clearTimeout(statusTimer);
    status = message;
    statusTimer = setTimeout(() => (status = ""), ms);
  }
  let error = "";

  // Dirty-tracking is hybrid because the body editor and the description
  // form have fundamentally different shapes:
  //   - Body lives in ProseMirror; the parser and PM serializer build
  //     STRUCTURALLY DIFFERENT FictionBook.Bodies trees for the same
  //     content (Section.Image vs inline image inside Section.Body[],
  //     Body.Title field vs no-Title-field, etc.). Stringifying both
  //     and comparing diverges even with no edits, so we can't snapshot
  //     the body. Track via PM's undo stack instead — Editor's
  //     dirtyPlugin emits `dirty: true|false` based on undo depth vs a
  //     baseline captured at mount / save.
  //   - Description has no PM involvement; both clean and current
  //     readings come straight from `fb.Description`. The parser /
  //     defaulters / mutations differ only in empty fields, which the
  //     canonical snapshot ignores (see document/dirty.ts).
  //
  // isDirty = bodyDirty (from PM) OR description snapshot mismatch. Cleared on
  // open / save / sample fallback.
  let bodyDirty = false;
  let cleanDescSnapshot: string | null = null;

  function isCurrentlyDirty(): boolean {
    if (bodyDirty) return true;
    if (xmlDirty) return true;
    if (cleanDescSnapshot === null) return false;
    return descriptionSnapshot(fb) !== cleanDescSnapshot;
  }
  /** The current description becomes the clean baseline, and body edits count as saved. */
  function captureCleanSnapshot() {
    cleanDescSnapshot = descriptionSnapshot(fb);
    bodyDirty = false;
  }
  function onEditorDirty(e: CustomEvent<boolean>) {
    bodyDirty = e.detail;
  }

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
      if (synced) fb = synced;
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

  $: blockLines = editorView && xmlSource ? xmlBlockLines(editorView.state.doc, xmlSource) : [];

  // --- XML editing (raw FB2 source pane) ---
  // editMode is shared between the body-view and description-view
  // ValidationPanel instances (only one is mounted at a time).
  // xmlDirty is true while the CM6 editor has unsaved edits — feeds
  // into isCurrentlyDirty() so Save / Cmd-Q honour it.
  let xmlEditMode = false;
  let xmlDirty = false;
  let xmlApplyError = "";
  // One-deep Apply undo. Captured pre-Apply so the user can roll back
  // a single commit if it turns out the parsed FB lost content despite
  // the data-loss guard. Cleared on next Apply or on panel close.
  let xmlUndoFb: FictionBook | null = null;
  let xmlUndoXml = "";
  function onXmlDirty(e: CustomEvent<{ dirty: boolean }>) {
    xmlDirty = e.detail.dirty;
    // Note: NOT clearing xmlApplyError on every keystroke — the user
    // is actively fixing the typo and needs the error as context.
    // It clears at the start of the next Apply attempt (success or
    // failure both reset the message) and on editMode transitions.
  }

  // editMode-transition mirror. Clears any stale Apply error whenever
  // the editor opens or closes — covers "Done → Edit XML" re-entry
  // (where the banner would otherwise show the previous attempt's
  // error) and Edit XML → Done → ... cycles.
  let prevXmlEditMode = false;
  $: if (xmlEditMode !== prevXmlEditMode) {
    xmlApplyError = "";
    prevXmlEditMode = xmlEditMode;
  }
  /** Close handler for ValidationPanel. Unconditional close — Wails
   *  WKWebView silently swallows `window.confirm()` returns under
   *  some configurations, so a prompt-guarded close ended up never
   *  actually closing (bug-reported as "× doesn't exit"). The
   *  document state isn't touched by closing the panel — only the
   *  CM6 working buffer is discarded, which the user can re-open via
   *  Validate. The global Cmd-Q unsaved-changes guard still catches
   *  any genuine in-memory FB changes via isCurrentlyDirty(). */
  function onPanelClose() {
    showPanel = false;
    xmlEditMode = false;
    xmlDirty = false;
    xmlApplyError = "";
    xmlUndoFb = null;
    xmlUndoXml = "";
  }
  async function onApplyXml(e: CustomEvent<{ xml: string }>) {
    xmlApplyError = "";
    const App = backend();
    if (!App) {
      xmlApplyError = "Wails bindings unavailable";
      return;
    }
    try {
      // Go-side ParseXMLString now parses + round-trips + checks for
      // structurally significant element loss. A typo like `<autho>`
      // instead of `<author>` would have wiped the body silently under
      // the original parser.Parse; now it throws back here with a
      // descriptive error and the user's CM6 text is preserved.
      const parsed = await App.ParseXMLString(e.detail.xml);
      if (!parsed) throw new Error("parser returned null");
      // SNAPSHOT for one-deep Apply-undo — captured BEFORE we touch fb
      // so we can roll back even if the data-loss guard missed
      // something subtler than a structural drop.
      xmlUndoFb = fb ? structuredClone(fb) : null;
      xmlUndoXml = xmlSource;
      // Wails generates doc.FictionBook structurally identical to our FB.
      fb = parsed;
      await App.UpdateDocument(fb);
      try { xmlSource = await App.SerializeCurrent(); } catch { /* leave as-is */ }
      try { validationErrors = (await App.ValidateCurrent()) ?? []; } catch { /* leave as-is */ }
      captureCleanSnapshot();
      // Explicit re-clear post-success. The clear at the top of this
      // function happens BEFORE the await chain; if Svelte reactivity
      // races with the panel's re-render, an explicit set here pins
      // the empty state once the success path is committed.
      xmlApplyError = "";
      showStatus("XML applied — Undo Apply is available in the panel", 5000);
    } catch (err) {
      xmlApplyError = (err as Error).message || String(err);
    }
  }
  /** One-deep Apply undo. Restores fb + xmlSource to the snapshot
   *  taken right before the last successful Apply. Snapshot is cleared
   *  after Undo so a stale double-undo isn't possible. */
  async function onUndoApplyXml() {
    if (!xmlUndoFb) return;
    fb = xmlUndoFb;
    xmlSource = xmlUndoXml;
    xmlUndoFb = null;
    xmlUndoXml = "";
    const App = backend();
    if (App) {
      try { await App.UpdateDocument(fb); } catch { /* ignore */ }
      try { validationErrors = (await App.ValidateCurrent()) ?? []; } catch { /* ignore */ }
    }
    captureCleanSnapshot();
    showStatus("Apply undone", 3000);
  }

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
    void refreshRecent();
  }

  // Most-recently-used files, fed from settings.json on the Go side.
  let recentFiles: string[] = [];

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
  const mainBox = () => mainEl?.getBoundingClientRect();

  const outlineResizer = {
    current: () => outlineWidth ?? DEFAULT_OUTLINE_WIDTH,
    bounds: () => ({ min: OUTLINE_MIN, max: OUTLINE_MAX }),
    fromPointer: (e: PointerEvent) => e.clientX - (mainBox()?.left ?? 0),
  };
  const panelResizer = {
    current: () => panelWidth ?? Math.round((mainBox()?.width ?? 0) * 0.3),
    // Capped so the editor surface cannot collapse to nothing.
    bounds: () => ({ min: PANEL_MIN, max: (mainBox()?.width ?? 0) * 0.7 }),
    fromPointer: (e: PointerEvent) => (mainBox()?.right ?? 0) - e.clientX,
  };

  async function refreshRecent() {
    const App = backend();
    if (!App) return;
    try { recentFiles = (await App.RecentFiles()) ?? []; } catch { /* ignore */ }
  }

  // `preset` lets the Recent-files menu call openFile with a specific path
  // (skipping the native file-picker). When omitted we fall back to
  // `App.PickFB2ToOpen()` as before.
  async function openFile(preset?: string) {
    error = ""; status = "";
    try {
      const App = backend();
      if (!App) throw new Error("Wails bindings not available — running in plain vite dev. Loaded bundled sample.");
      // Guard the in-memory replacement: if the current document has
      // unsaved edits, prompt before discarding them. The native Save /
      // Discard / Cancel dialog lives on the Go side
      // (App.ConfirmUnsavedChanges) so it matches the OS look exactly.
      if (isCurrentlyDirty()) {
        const choice = await App.ConfirmUnsavedChanges();
        if (choice === "cancel") return;
        if (choice === "save") {
          await save(false);
          if (isCurrentlyDirty()) return; // save failed or was cancelled at file picker
        }
        // discard: fall through, the next fb assignment overwrites the doc
      }
      const path = preset ?? await App.PickFB2ToOpen();
      if (!path) return;
      console.log(`[fbe] opening ${path}`);
      status = `Opening ${basename(path)}…`;
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
      filename = basename(path);
      showStatus(`Opened ${filename}`, 3000);
      captureCleanSnapshot();
      void refreshRecent();
    } catch (e) {
      const msg = (e as Error).message || String(e);
      console.error("[fbe] openFile failed:", e);
      error = `Open failed: ${msg}`;
      if (preset) {
        // Recent-file pointed at something that no longer exists (or is
        // unreadable) — purge it so the menu doesn't keep offering a dead
        // entry.
        const App = backend();
        try { await App?.RemoveFromRecent(preset); } catch { /* ignore */ }
        void refreshRecent();
      } else {
        fb = SAMPLE_BOOK;
        currentPath = "";
        filename = "blank.fb2 (sample)";
        captureCleanSnapshot();
      }
    }
  }

  /** The document as it stands now: the live editor in body view, `fb` otherwise (the switch away
   *  from body flushed into it, and the description forms edit it directly). */
  function liveDoc(): FictionBook | null {
    return (view === "body" && editor) ? editor.currentFB() : fb;
  }

  async function save(saveAs: boolean) {
    error = ""; status = "";
    try {
      const App = backend();
      if (!App) throw new Error("Wails bindings not available — save works only in the desktop app.");
      const current = liveDoc();
      if (!current) throw new Error("No document loaded.");

      let path = currentPath;
      if (saveAs || !path) {
        const suggested = filename.endsWith(".fb2") ? filename : "untitled.fb2";
        path = await App.PickFB2ToSave(suggested);
        if (!path) return;
      }
      await App.UpdateDocument(current);
      await App.SaveFile(path);
      currentPath = path;
      filename = basename(path);
      showStatus(`Saved ${filename}`, 3000);
      captureCleanSnapshot();
      // Re-baseline PM's undo stack so subsequent undos that return to
      // this just-saved point are recognised as clean.
      editor?.markSaved();
      void refreshRecent();
    } catch (e) {
      error = (e as Error).message;
    }
  }

  async function exportHTML() {
    error = ""; status = "";
    try {
      const App = backend();
      if (!App) throw new Error("Wails bindings not available.");
      const current = liveDoc();
      if (current) {
        await App.UpdateDocument(current);
      }
      const suggested = (filename || "untitled").replace(/\.fb2(\.zip)?$/i, "") + ".html";
      const path = await App.PickHTMLToSave(suggested);
      if (!path) return;
      await App.ExportHTML(path);
      showStatus(`Exported ${basename(path)}`, 3000);
    } catch (e) {
      error = (e as Error).message;
    }
  }

  async function validate() {
    error = ""; status = "";
    try {
      const App = backend();
      if (!App) throw new Error("Wails bindings not available.");
      if (!fb) throw new Error("No document loaded.");

      // Push the latest editor state to Go (if we're in body view) so the
      // serialized XML and the validation result reflect unsaved edits.
      const current = liveDoc();
      if (current) {
        await App.UpdateDocument(current);
      }

      const [xml, errs] = await Promise.all([
        App.SerializeCurrent(),
        App.ValidateCurrent(),
      ]);

      xmlSource = xml ?? "";
      validationErrors = errs ?? [];
      showPanel = true;

      showStatus(errs && errs.length > 0
        ? `XSD: ${errs.length} error(s)`
        : "XSD valid ✓", 4000);
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
    void refreshRecent();

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
        App.emit("app:dirty-response", isCurrentlyDirty());
      });
      const offSave = App.on("app:save-and-quit", async () => {
        try {
          await save(false);
        } catch (e) {
          console.error("[fbe] save-and-quit: save failed", e);
          return;
        }
        if (!isCurrentlyDirty()) {
          try { await App.ForceQuit(); } catch { /* app is exiting */ }
        }
      });
      // File drop: Go's OnFileDrop forwards the first dropped path; we
      // reuse the existing openFile() flow so the unsaved-changes guard,
      // status text, and recent-files update all behave identically to
      // a regular File→Open. Without this listener the drop would be
      // silently swallowed (WebView drop is disabled in main.go).
      const offDrop = App.on("app:file-drop", (path: string) => {
        if (typeof path === "string" && path) void openFile(path);
      });
      if (typeof offCheck === "function") detachListeners.push(offCheck);
      if (typeof offSave === "function") detachListeners.push(offSave);
      // OS "open this file" (double-click / Open With) while the app is already running.
      const offOpen = App.on("app:open-file", (path: string) => {
        if (typeof path === "string" && path) void openFile(path);
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
    // Pick up whatever Go already has open (so opening :34115 in a browser
    // tab while a file is loaded in the native window shows that file
    // instead of the sample). Path is intentionally NOT synced — Save in
    // the dev-tab should land in Save-As to avoid two contexts racing on
    // the same path.
    void (async () => {
      const launched = await launchFile;
      if (launched) {
        await openFile(launched);
        if (fb) return;
        // open failed (error is shown) — fall through to the sample
      }
      const App = backend();
      if (App) {
        try {
          const current = await App.CurrentDocument();
          if (current && current.Bodies && current.Bodies.length > 0) {
            fb = current as FictionBook;
            filename = "(opened in native window)";
            captureCleanSnapshot();
            return;
          }
        } catch { /* fall through to sample */ }
      }
      fb = SAMPLE_BOOK;
      filename = "blank.fb2 (sample)";
      captureCleanSnapshot();
    })();
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
    hasDocument={!!fb}
    {filename}
    {status}
    {error}
    {recentFiles}
    {view}
    on:open={(e) => openFile(e.detail)}
    on:save={(e) => save(e.detail)}
    on:validate={validate}
    on:export={exportHTML}
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
    class:with-panel={showPanel}
    style={`${outlineWidth !== null ? `--outline-w: ${outlineWidth}px;` : ""}${panelWidth !== null && showPanel ? `--panel-w: ${panelWidth}px;` : ""}`}
  >
    {#if view === "body"}
      <aside>
        <DocumentTree {fb} activePath={position?.path ?? null} on:navigate={(e) => editor?.scrollToPath(e.detail.path)} />
      </aside>
      <Resizer
        label="Resize outline sidebar"
        {...outlineResizer}
        bind:dragging={draggingOutline}
        on:change={(e) => (outlineWidth = e.detail)}
        on:commit={(e) => persistPane("outlineWidth", e.detail)}
      />
      <section>
        <Editor bind:this={editor} bind:view={editorView} {fb} {hotkeys} on:dirty={onEditorDirty} on:position={(e) => (position = e.detail)} />
        <PositionBar {position} />
      </section>
    {:else if fb}
      <DescriptionPanel bind:fb />
    {/if}
    {#if showPanel}
      <Resizer
        label="Resize validation panel"
        {...panelResizer}
        bind:dragging={draggingPanel}
        on:change={(e) => (panelWidth = e.detail)}
        on:commit={(e) => persistPane("validationWidth", e.detail)}
      />
      <ValidationPanel
        {xmlSource}
        followLine={view === "body" && position ? blockLines[position.block] ?? null : null}
        errors={validationErrors}
        {initialErrorsHeight}
        theme={$effectiveTheme}
        applyError={xmlApplyError}
        canUndoApply={xmlUndoFb !== null}
        bind:editMode={xmlEditMode}
        on:close={onPanelClose}
        on:resize={(e) => persistPane("validationErrorsHeight", Math.max(0, e.detail.height))}
        on:apply={onApplyXml}
        on:undo-apply={onUndoApplyXml}
        on:xml-dirty={onXmlDirty}
        on:scroll-line={(e) => editor?.scrollToBlock(blockAtLine(blockLines, e.detail))}
      />
    {/if}
  </main>
</div>

<HelpDialog bind:open={showHelp} />
<SettingsDialog bind:open={showSettings} on:apply={onSettingsApplied} />
<BinaryManagerDialog bind:open={showBinaries} bind:fb view={editorView} />

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
    height: 100vh;
    font-family: var(--font-ui);
  }
  .spacer { height: 0; }
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
  main.description {
    grid-template-columns: 1fr;
  }
  main.description.with-panel {
    grid-template-columns: 1fr 6px var(--panel-w, minmax(320px, 30%));
  }
</style>
