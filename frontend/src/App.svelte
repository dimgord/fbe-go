<script lang="ts">
  import { onMount } from "svelte";
  import DocumentTree from "./tree/DocumentTree.svelte";
  import Editor from "./editor/Editor.svelte";
  import Toolbar from "./editor/Toolbar.svelte";
  import DescriptionPanel from "./description/DescriptionPanel.svelte";
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

  async function wails() {
    const App = await import("../wailsjs/go/main/App").catch(() => null);
    const runtime = await import("../wailsjs/runtime/runtime").catch(() => null);
    return App && runtime ? { App, runtime } : null;
  }

  async function openFile() {
    error = ""; status = "";
    try {
      const w = await wails();
      if (!w) throw new Error("Wails bindings not available — running in plain vite dev. Loaded bundled sample.");
      const path: string = await w.runtime.OpenFileDialog({
        Title: "Open FB2 file",
        Filters: [{ DisplayName: "FictionBook (*.fb2;*.fb2.zip)", Pattern: "*.fb2;*.fb2.zip" }],
      });
      if (!path) return;
      // @ts-expect-error — Wails-generated type uses doc.FictionBook shape.
      fb = await w.App.OpenFile(path);
      currentPath = path;
      filename = path.split(/[\\/]/).pop() ?? path;
    } catch (e) {
      error = (e as Error).message;
      fb = SAMPLE_BOOK;
      currentPath = "";
      filename = "blank.fb2 (sample)";
    }
  }

  async function save(saveAs: boolean) {
    error = ""; status = "";
    try {
      const w = await wails();
      if (!w) throw new Error("Wails bindings not available — save works only in the desktop app.");
      if (!editor) throw new Error("Editor not ready.");
      const current = editor.currentFB();
      if (!current) throw new Error("No document loaded.");

      let path = currentPath;
      if (saveAs || !path) {
        path = await w.runtime.SaveFileDialog({
          Title: saveAs ? "Save As" : "Save FB2",
          DefaultFilename: filename.endsWith(".fb2") ? filename : "untitled.fb2",
          Filters: [{ DisplayName: "FictionBook (*.fb2)", Pattern: "*.fb2" }],
        });
        if (!path) return;
      }
      // @ts-expect-error — Wails-generated type uses doc.FictionBook shape.
      await w.App.UpdateDocument(current);
      await w.App.SaveFile(path);
      currentPath = path;
      filename = path.split(/[\\/]/).pop() ?? path;
      status = `Saved ${filename}`;
      // Clear status after 3s.
      setTimeout(() => (status = ""), 3000);
    } catch (e) {
      error = (e as Error).message;
    }
  }

  async function exportHTML() {
    error = ""; status = "";
    try {
      const w = await wails();
      if (!w) throw new Error("Wails bindings not available.");
      if (!editor) throw new Error("Editor not ready.");
      // Refresh the Go-side fb with the current PM doc before export.
      const current = editor.currentFB();
      if (current) {
        // @ts-expect-error — doc.FictionBook shape
        await w.App.UpdateDocument(current);
      }
      const defaultName = (filename || "untitled").replace(/\.fb2(\.zip)?$/i, "") + ".html";
      const path: string = await w.runtime.SaveFileDialog({
        Title: "Export HTML",
        DefaultFilename: defaultName,
        Filters: [{ DisplayName: "HTML (*.html)", Pattern: "*.html" }],
      });
      if (!path) return;
      // @ts-expect-error
      await w.App.ExportHTML(path);
      status = `Exported ${path.split(/[\\/]/).pop() ?? path}`;
      setTimeout(() => (status = ""), 3000);
    } catch (e) {
      error = (e as Error).message;
    }
  }

  async function validate() {
    error = ""; status = "";
    try {
      const w = await wails();
      if (!w || !currentPath) throw new Error("Open a saved file first.");
      // @ts-expect-error
      const errs: Array<{ Line: number; Column: number; Message: string }> =
        await w.App.Validate(currentPath);
      if (!errs || errs.length === 0) {
        status = "XSD valid ✓";
      } else {
        status = `XSD: ${errs.length} error(s) — first: ${errs[0].Message.slice(0, 120)}`;
      }
      setTimeout(() => (status = ""), 6000);
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
    fb = SAMPLE_BOOK;
    filename = "blank.fb2 (sample)";
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  });
</script>

<div class="layout">
  <header>
    <button on:click={openFile}>Open…</button>
    <button on:click={() => save(false)} disabled={!editor}>Save</button>
    <button on:click={() => save(true)} disabled={!editor}>Save As…</button>
    <button on:click={validate} disabled={!currentPath}>Validate</button>
    <button on:click={exportHTML} disabled={!editor}>Export HTML…</button>
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
    <main>
      <aside>
        <DocumentTree {fb} on:navigate={(e) => editor?.scrollToPath(e.detail.path)} />
      </aside>
      <section><Editor bind:this={editor} {fb} /></section>
    </main>
  {:else if fb}
    <div class="spacer" />
    <div class="description-wrap">
      <DescriptionPanel bind:fb />
    </div>
  {/if}
</div>

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
</style>
