<script lang="ts">
  import { createEventDispatcher, onMount } from "svelte";
  import pkg from "../../package.json";

  export let open = false;

  const dispatch = createEventDispatcher<{ close: void }>();

  const version = pkg.version;
  const isMac = typeof navigator !== "undefined" && /Mac|iPhone|iPad/i.test(navigator.platform);
  const mod = isMac ? "⌘" : "Ctrl";
  const shift = isMac ? "⇧" : "Shift";

  // External-link routing lives in src/runtime/externalLink.ts; App.svelte
  // installs a document-wide capture-phase handler, so every <a href> click
  // in this dialog (and anywhere else in the app) already routes through
  // Wails runtime.BrowserOpenURL without a per-link on:click wrapper.

  // Native right-click → "Copy Link Address" is unreliable in WKWebView /
  // WebKitGTK production builds (context menu behavior varies by OS and is
  // often suppressed in release bundles). Explicit copy buttons next to
  // each link are the portable answer.
  let copiedUrl: string | null = null;
  let copiedTimer: ReturnType<typeof setTimeout> | null = null;

  async function copyUrl(url: string) {
    let ok = false;
    try {
      if (navigator.clipboard && navigator.clipboard.writeText) {
        await navigator.clipboard.writeText(url);
        ok = true;
      }
    } catch { /* fall through to textarea fallback */ }
    if (!ok) {
      // Fallback for older webviews without the async Clipboard API.
      const ta = document.createElement("textarea");
      ta.value = url;
      ta.style.position = "fixed";
      ta.style.opacity = "0";
      document.body.appendChild(ta);
      ta.select();
      try { ok = document.execCommand("copy"); } catch { ok = false; }
      document.body.removeChild(ta);
    }
    if (ok) {
      copiedUrl = url;
      if (copiedTimer) clearTimeout(copiedTimer);
      copiedTimer = setTimeout(() => { copiedUrl = null; }, 1500);
    }
  }

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
        <p><strong>Version {version}-beta</strong> · MIT-licensed ·
          <a href="https://github.com/dimgord/fbe-go/blob/main/LICENSE"
             target="_blank" rel="noreferrer noopener">LICENSE</a>
          ·
          <a href="https://github.com/dimgord/fbe-go/blob/main/NOTICE.md"
             target="_blank" rel="noreferrer noopener">NOTICE</a>
        </p>
        <p>
          A Go + <a href="https://wails.io" target="_blank" rel="noreferrer noopener">Wails v2</a> port of the
          classic Windows FictionBook Editor, targeting macOS and Linux.
          Edits FB2 (FictionBook 2.x) documents in a ProseMirror-backed
          WYSIWYG editor; full round-trip fidelity including unknown
          elements, XSD validation (libxml2), and HTML export.
        </p>
        <p class="credits">
          Independent rewrite — thanks to Dmitry Gribov (FB2 spec + XSD),
          the classic FBE team, and the Wails / ProseMirror / libxml2
          maintainers. Full credits in NOTICE.
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
        <ul class="links">
          {#each [
            { label: "Source — github.com/dimgord/fbe-go", url: "https://github.com/dimgord/fbe-go" },
            { label: "FictionBook 2.x specification",       url: "http://www.fictionbook.org/index.php/Eng:FictionBook" },
            { label: "Original FBE (Windows)",              url: "https://github.com/evpobr/fictionbookeditor" },
          ] as link}
            <li>
              <a
                href={link.url}
                target="_blank"
                rel="noreferrer noopener">{link.label}</a>
              <button
                type="button"
                class="copy-url"
                title="Copy URL to clipboard"
                on:click={() => copyUrl(link.url)}
                aria-label={`Copy URL: ${link.url}`}
              >{copiedUrl === link.url ? "✓ copied" : "copy"}</button>
            </li>
          {/each}
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
    background: var(--backdrop);
    display: grid;
    place-items: center;
    z-index: 100;
  }
  .dialog {
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: 6px;
    padding: 1rem 1.4rem 1.2rem;
    min-width: 28rem;
    max-width: 36rem;
    max-height: 80vh;
    overflow: auto;
    box-shadow: 0 8px 24px var(--shadow);
    font-family: -apple-system, "Segoe UI", sans-serif;
    font-size: 0.9rem;
    color: var(--fg);
    /* Explicitly opt-in to text selection so users can copy the version
       string, kbd labels, and link text. Rest of the app (editor surface,
       raw-block placeholders, resizer handle) sets `user-select: none` on
       its chrome; without this override some of that could inherit. */
    user-select: text;
    -webkit-user-select: text;
    cursor: auto;
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
    color: var(--fg-secondary);
    text-transform: uppercase;
    letter-spacing: 0.6px;
  }
  .close {
    border: none;
    background: transparent;
    font-size: 1.2rem;
    line-height: 1;
    padding: 0.1rem 0.45rem;
    color: var(--fg-muted);
    cursor: pointer;
    border-radius: 3px;
  }
  .close:hover { background: var(--bg-hover); color: var(--fg-strong); }

  .about p { margin: 0.35rem 0; line-height: 1.45; }
  .about p.credits { font-size: 0.82rem; color: var(--fg-secondary); }
  .about a, section a {
    color: var(--fg-link);
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
    color: var(--fg-secondary);
  }
  kbd {
    display: inline-block;
    padding: 0.08rem 0.4rem;
    border: 1px solid var(--border);
    border-bottom-width: 2px;
    border-radius: 3px;
    background: var(--bg-chrome);
    font-family: "SF Mono", Menlo, Consolas, monospace;
    font-size: 0.78rem;
    color: var(--fg);
  }
  ul {
    margin: 0.2rem 0 0 1.2rem;
    padding: 0;
    line-height: 1.55;
  }
  ul.links {
    list-style: none;
    margin-left: 0;
  }
  ul.links li {
    display: flex;
    align-items: baseline;
    gap: 0.5rem;
    padding: 0.1rem 0;
  }
  ul.links li a {
    flex: 1;
    min-width: 0;
    word-break: break-all;
  }
  button.copy-url {
    flex: none;
    padding: 0.1rem 0.5rem;
    font-size: 0.72rem;
    font-family: "SF Mono", Menlo, Consolas, monospace;
    color: var(--fg-secondary);
    background: var(--bg-chrome);
    border: 1px solid var(--border);
    border-radius: 3px;
    cursor: pointer;
    line-height: 1.3;
    min-width: 4.5rem;
    text-align: center;
  }
  button.copy-url:hover { background: var(--bg-hover); color: var(--fg); }
  button.copy-url:active { background: var(--bg-active); }
  .actions {
    display: flex;
    justify-content: flex-end;
    gap: 0.5rem;
    margin-top: 1rem;
  }
  button {
    padding: 0.35rem 0.9rem;
    border: 1px solid var(--border-button);
    background: var(--bg-surface);
    color: var(--fg);
    border-radius: 4px;
    cursor: pointer;
    font: inherit;
  }
  button:hover { background: var(--bg-hover); }
  button.primary {
    background: var(--bg-active);
    font-weight: 600;
    border-color: var(--warn);
  }
  button.primary:hover { background: var(--bg-active-hover); }
</style>
