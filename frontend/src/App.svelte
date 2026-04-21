<script lang="ts">
  import { onMount } from "svelte";
  import DocumentTree from "./tree/DocumentTree.svelte";
  import Editor from "./editor/Editor.svelte";
  import Toolbar from "./editor/Toolbar.svelte";
  import { SAMPLE_BOOK } from "./fb2/sample";
  import type { FictionBook } from "./fb2/types";

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
    <span class="title">FictionBook Editor · <em>{filename}</em></span>
    {#if status}<span class="status">{status}</span>{/if}
    {#if error}<span class="err">{error}</span>{/if}
  </header>

  <Toolbar {editor} />

  <main>
    <aside><DocumentTree /></aside>
    <section><Editor bind:this={editor} {fb} /></section>
  </main>
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
