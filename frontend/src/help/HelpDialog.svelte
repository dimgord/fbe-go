<script lang="ts">
  import { createEventDispatcher, onMount } from "svelte";
  import pkg from "../../package.json";

  export let open = false;

  const dispatch = createEventDispatcher<{ close: void }>();

  const version = pkg.version;
  const isMac = typeof navigator !== "undefined" && /Mac|iPhone|iPad/i.test(navigator.platform);
  const mod = isMac ? "⌘" : "Ctrl";
  const shift = isMac ? "⇧" : "Shift";

  // Keyboard shortcuts — kept in sync by hand with Editor.svelte's keymap
  // and App.svelte's Cmd-S handler. If you change a binding, update this
  // table too.
  const shortcuts: Array<{ keys: string; action: string }> = [
    { keys: `${mod}-S`, action: "Save" },
    { keys: `${shift}-${mod}-S`, action: "Save As…" },
    { keys: `${mod}-Z`, action: "Undo" },
    { keys: `${mod}-Y  /  ${shift}-${mod}-Z`, action: "Redo" },
    { keys: `${mod}-B`, action: "Bold (strong)" },
    { keys: `${mod}-I`, action: "Italic (emphasis)" },
    { keys: `${shift}-${mod}-S`, action: "Strikethrough" },
    { keys: `${mod}-,`, action: "Subscript" },
    { keys: `${mod}-.`, action: "Superscript" },
  ];

  function close() {
    open = false;
    dispatch("close");
  }

  function onKey(e: KeyboardEvent) {
    if (!open) return;
    if (e.key === "Escape") {
      e.preventDefault();
      close();
    }
  }

  onMount(() => {
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  });
</script>

{#if open}
  <div
    class="backdrop"
    role="button"
    tabindex="-1"
    aria-label="Dismiss help"
    on:click={close}
    on:keydown={(e) => e.key === "Escape" && close()}>
    <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
    <div
      class="dialog"
      role="dialog"
      aria-modal="true"
      aria-labelledby="hd-title"
      on:click|stopPropagation
      on:keydown|stopPropagation>
      <header>
        <h3 id="hd-title">FictionBook Editor (Go)</h3>
        <button type="button" class="close" on:click={close} title="Close">×</button>
      </header>

      <section class="about">
        <p><strong>Version {version}-beta</strong></p>
        <p>
          A Go + <a href="https://wails.io" target="_blank" rel="noreferrer">Wails v2</a> port of the
          classic Windows FictionBook Editor, targeting macOS and Linux.
          Edits FB2 (FictionBook 2.x) documents in a ProseMirror-backed
          WYSIWYG editor; full round-trip fidelity including unknown
          elements, XSD validation (libxml2), and HTML export.
        </p>
      </section>

      <section>
        <h4>Keyboard shortcuts</h4>
        <table>
          {#each shortcuts as s}
            <tr>
              <td class="keys"><kbd>{s.keys}</kbd></td>
              <td>{s.action}</td>
            </tr>
          {/each}
        </table>
      </section>

      <section>
        <h4>Resources</h4>
        <ul>
          <li><a href="https://github.com/dimgord/fbe-go" target="_blank" rel="noreferrer">github.com/dimgord/fbe-go</a></li>
          <li><a href="http://www.fictionbook.org/index.php/Eng:FictionBook" target="_blank" rel="noreferrer">FictionBook 2.x specification</a></li>
          <li><a href="https://github.com/evpobr/fictionbookeditor" target="_blank" rel="noreferrer">Original FBE (Windows)</a></li>
        </ul>
      </section>

      <div class="actions">
        <button type="button" class="primary" on:click={close}>Close</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.35);
    display: grid;
    place-items: center;
    z-index: 100;
  }
  .dialog {
    background: #fffdf8;
    border: 1px solid #d5d5cb;
    border-radius: 6px;
    padding: 1rem 1.4rem 1.2rem;
    min-width: 28rem;
    max-width: 36rem;
    max-height: 80vh;
    overflow: auto;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.25);
    font-family: -apple-system, "Segoe UI", sans-serif;
    font-size: 0.9rem;
    color: #222;
  }
  header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 0.4rem;
  }
  h3 {
    margin: 0;
    font-size: 1.05rem;
  }
  h4 {
    margin: 1rem 0 0.4rem;
    font-size: 0.9rem;
    color: #555;
    text-transform: uppercase;
    letter-spacing: 0.6px;
  }
  .close {
    border: none;
    background: transparent;
    font-size: 1.2rem;
    line-height: 1;
    padding: 0.1rem 0.45rem;
    color: #666;
    cursor: pointer;
    border-radius: 3px;
  }
  .close:hover { background: #e8e4d8; color: #111; }

  .about p { margin: 0.35rem 0; line-height: 1.45; }
  .about a, section a {
    color: #1a5490;
    text-decoration: none;
  }
  .about a:hover, section a:hover { text-decoration: underline; }

  table {
    border-collapse: collapse;
    width: 100%;
    font-size: 0.85rem;
  }
  td {
    padding: 0.18rem 0.45rem;
    vertical-align: baseline;
  }
  td.keys {
    white-space: nowrap;
    color: #444;
  }
  kbd {
    display: inline-block;
    padding: 0.08rem 0.4rem;
    border: 1px solid #c9c9bd;
    border-bottom-width: 2px;
    border-radius: 3px;
    background: #f5f3ea;
    font-family: "SF Mono", Menlo, Consolas, monospace;
    font-size: 0.78rem;
    color: #333;
  }
  ul {
    margin: 0.2rem 0 0 1.2rem;
    padding: 0;
    line-height: 1.55;
  }
  .actions {
    display: flex;
    justify-content: flex-end;
    gap: 0.5rem;
    margin-top: 1rem;
  }
  button {
    padding: 0.35rem 0.9rem;
    border: 1px solid #bbb;
    background: white;
    border-radius: 4px;
    cursor: pointer;
    font: inherit;
  }
  button:hover { background: #fff8e5; }
  button.primary {
    background: #fce6a0;
    font-weight: 600;
    border-color: #b89a3e;
  }
  button.primary:hover { background: #f5da7c; }
</style>
