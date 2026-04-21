<script lang="ts">
  import { onMount } from "svelte";
  import DocumentTree from "./tree/DocumentTree.svelte";
  import Editor from "./editor/Editor.svelte";

  // Wails-generated bindings show up at ../wailsjs/go/main/App after `wails dev`.
  // Stubbed here so the app renders without the generated code.
  async function openFile() {
    // TODO: invoke window.runtime.OpenFileDialog then App.OpenFile(path)
    console.log("TODO: hook up file dialog");
  }

  onMount(() => {
    document.title = "FictionBook Editor (Go)";
  });
</script>

<div class="layout">
  <header>
    <button on:click={openFile}>Open…</button>
    <span class="title">FictionBook Editor</span>
  </header>

  <main>
    <aside><DocumentTree /></aside>
    <section><Editor /></section>
  </main>
</div>

<style>
  .layout {
    display: grid;
    grid-template-rows: 2.5rem 1fr;
    height: 100vh;
    font-family: -apple-system, Segoe UI, sans-serif;
  }
  header {
    display: flex;
    align-items: center;
    padding: 0 0.5rem;
    gap: 0.5rem;
    border-bottom: 1px solid #ddd;
  }
  .title {
    font-weight: 600;
  }
  main {
    display: grid;
    grid-template-columns: 260px 1fr;
    overflow: hidden;
  }
  aside {
    border-right: 1px solid #ddd;
    overflow: auto;
  }
  section {
    overflow: auto;
  }
</style>
