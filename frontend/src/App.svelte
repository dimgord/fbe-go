<script lang="ts">
  import { onMount } from "svelte";
  import DocumentTree from "./tree/DocumentTree.svelte";
  import Editor from "./editor/Editor.svelte";
  import Toolbar from "./editor/Toolbar.svelte";
  import DescriptionPanel from "./description/DescriptionPanel.svelte";
  import ValidationPanel from "./validation/ValidationPanel.svelte";
  import HelpDialog from "./help/HelpDialog.svelte";
  import { SAMPLE_BOOK } from "./fb2/sample";
  import type { FictionBook } from "./fb2/types";

  type View = "body" | "description";
  let view: View = "body";

  let fb: FictionBook | null = null;
  let filename = "(untitled)";
  let currentPath = "";
  let status = "";
  let error = "";
  let editor: Editor | undefined = undefined;

  // Validation / XML-source panel state.
  let showPanel = false;
  let xmlSource = "";
  let validationErrors: { line: number; column: number; message: string }[] = [];

  let showHelp = false;

  // Most-recently-used files, fed from settings.json on the Go side.
  let recentFiles: string[] = [];
  let recentMenuOpen = false;

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

  // Keyboard shortcut: Cmd-S / Ctrl-S saves.
  function onKeyDown(e: KeyboardEvent) {
    if ((e.metaKey || e.ctrlKey) && e.key === "s") {
      e.preventDefault();
      save(e.shiftKey); // Shift-Cmd-S → Save As
    }
  }

  onMount(() => {
    document.title = "FictionBook Editor (Go)";
    window.addEventListener("keydown", onKeyDown);
    void refreshRecent();
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
    return () => window.removeEventListener("keydown", onKeyDown);
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
    <button on:click={() => (showHelp = true)} title="Keyboard shortcuts and about">Help</button>
    <div class="view-toggle" role="tablist" aria-label="View">
      <button
        class:active={view === "body"}
        on:click={() => (view = "body")}
        role="tab"
        aria-selected={view === "body"}>Body</button>
      <button
        class:active={view === "description"}
        on:click={() => (view = "description")}
        role="tab"
        aria-selected={view === "description"}>Description</button>
    </div>
    <span class="title">FictionBook Editor · <em>{filename}</em></span>
    {#if status}<span class="status">{status}</span>{/if}
    {#if error}<span class="err">{error}</span>{/if}
  </header>

  {#if view === "body"}
    <Toolbar {editor} />
    <main class:with-panel={showPanel}>
      <aside>
        <DocumentTree {fb} on:navigate={(e) => editor?.scrollToPath(e.detail.path)} />
      </aside>
      <section><Editor bind:this={editor} {fb} /></section>
      {#if showPanel}
        <ValidationPanel
          {xmlSource}
          errors={validationErrors}
          on:close={() => (showPanel = false)}
        />
      {/if}
    </main>
  {:else if fb}
    <div class="spacer" />
    <div class="description-wrap with-panel-maybe" class:with-panel={showPanel}>
      <DescriptionPanel bind:fb />
      {#if showPanel}
        <ValidationPanel
          {xmlSource}
          errors={validationErrors}
          on:close={() => (showPanel = false)}
        />
      {/if}
    </div>
  {/if}
</div>

<HelpDialog bind:open={showHelp} />

<style>
  :global(body), :global(html) {
    margin: 0;
    height: 100%;
    background: #fafaf7;
    color: #222;
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
    background: #f1f1ec;
    border-bottom: 1px solid #d5d5cb;
  }
  header button {
    padding: 0.25rem 0.7rem;
    border: 1px solid #bbb;
    background: white;
    border-radius: 4px;
    cursor: pointer;
  }
  header button:hover:not(:disabled) { background: #fff8e5; }
  header button:disabled { opacity: 0.5; cursor: default; }

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
    background: #fffdf8;
    border: 1px solid #d5d5cb;
    border-radius: 4px;
    box-shadow: 0 6px 18px rgba(0, 0, 0, 0.18);
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
  button.recent-item:hover { background: #fff8e5; }
  button.recent-item .basename { font-weight: 600; color: #222; }
  button.recent-item .dir {
    color: #888;
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
    background: #fce6a0;
    font-weight: 600;
  }
  .description-wrap {
    overflow: hidden;
  }
  .spacer { height: 0; }
  .title { font-size: 0.9rem; color: #444; margin-left: 0.5rem; }
  .status { color: #2a7; font-size: 0.8rem; margin-left: auto; }
  .err { color: #a33; font-size: 0.8rem; margin-left: auto; }
  main {
    display: grid;
    grid-template-columns: 260px 1fr;
    overflow: hidden;
  }
  main.with-panel {
    grid-template-columns: 260px 1fr minmax(320px, 30%);
  }
  aside {
    border-right: 1px solid #d5d5cb;
    overflow: auto;
    background: #f5f5f0;
    font-size: 0.9rem;
  }
  section {
    overflow: auto;
    background: white;
  }
  .description-wrap.with-panel-maybe.with-panel {
    display: grid;
    grid-template-columns: 1fr minmax(320px, 30%);
  }
</style>
