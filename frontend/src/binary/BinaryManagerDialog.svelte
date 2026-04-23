<script lang="ts">
  import { createEventDispatcher, onMount } from "svelte";
  import type { EditorView } from "prosemirror-view";
  import type { FictionBook, Binary } from "../fb2/types";
  import { renameBinary, countBinaryRefs, isCoverBinary } from "./refs";

  export let open = false;
  export let fb: FictionBook | null = null;
  export let view: EditorView | undefined = undefined;

  const dispatch = createEventDispatcher<{ close: void }>();

  // Local inline-rename state: `renaming` is the Binary instance being
  // renamed (identity-compared, not ID, so a cancel-mid-flight leaves the
  // real ID untouched). `draftId` holds the editing buffer.
  let renaming: Binary | null = null;
  let draftId = "";
  let renameInput: HTMLInputElement | undefined;

  // Focus the rename input on entry without tripping the Svelte a11y lint
  // (which flags `autofocus`). Rename is always user-initiated, so grabbing
  // focus here is an explicit response to their click — accessible.
  $: if (renaming && renameInput) {
    renameInput.focus();
    renameInput.select();
  }

  // Wails v2 WKWebView doesn't wire window.alert/confirm/prompt, so the
  // upload "name your binary" step, the delete confirmation, and any
  // validation error all live as in-component mini-overlays.
  let uploadStaged: { bin: Binary; draftId: string; path: string; error: string } | null = null;
  let uploadIdInput: HTMLInputElement | undefined;
  let deleting: { b: Binary; coverRefs: number; editorRefs: number } | null = null;
  let errorMessage = "";

  $: if (uploadStaged && uploadIdInput) {
    uploadIdInput.focus();
    uploadIdInput.select();
  }

  function cancel() {
    renaming = null;
    open = false;
    dispatch("close");
  }

  async function wailsApp() {
    return await import("../../wailsjs/go/main/App").catch(() => null);
  }

  function beginRename(b: Binary) {
    renaming = b;
    draftId = b.ID;
  }

  function cancelRename() {
    renaming = null;
  }

  function commitRename(b: Binary) {
    const newId = draftId.trim();
    if (!newId) return;
    if (newId === b.ID) { renaming = null; return; }
    if (!isValidId(newId)) {
      errorMessage = `"${newId}" is not a valid id. Use letters, digits, underscore, dot, hyphen; must start with a letter or underscore.`;
      return;
    }
    if (fb?.Binaries?.some(x => x !== b && x.ID === newId)) {
      errorMessage = `A binary with id "${newId}" already exists.`;
      return;
    }
    if (!fb) return;
    renameBinary(fb, view, b.ID, newId);
    // Svelte 4 assignment-based reactivity: in-place mutation of fb.Binaries
    // and fb.Description won't re-render the dialog without this nudge.
    fb = fb;
    renaming = null;
  }

  function beginDelete(b: Binary) {
    if (!fb) return;
    const refs = countBinaryRefs(fb, view, b.ID);
    deleting = { b, coverRefs: refs.coverpageRefs, editorRefs: refs.editorRefs };
  }

  function cancelDelete() { deleting = null; }

  function confirmDelete() {
    if (!deleting || !fb) return;
    const target = deleting.b;
    fb.Binaries = (fb.Binaries ?? []).filter(x => x !== target);
    fb = fb;
    deleting = null;
  }

  async function upload() {
    if (!fb) return;
    const App = await wailsApp();
    if (!App) { errorMessage = "Wails bindings unavailable — run `wails dev` or a proper build."; return; }
    let path: string;
    try {
      path = await App.PickImageToUpload();
    } catch (e) {
      errorMessage = `File picker failed: ${e}`;
      return;
    }
    if (!path) return;

    let bin: Binary | null;
    try {
      bin = await App.ReadImageBinary(path) as Binary | null;
    } catch (e) {
      errorMessage = `Could not read image: ${e}`;
      return;
    }
    if (!bin) return;

    // Stage the binary and prompt for an id via the in-component overlay.
    // Suggested id: sanitized filename stem, de-duplicated against existing ids.
    const existingIds = new Set((fb.Binaries ?? []).map(b => b.ID));
    const filenameGuess = path.split(/[\\/]/).pop()?.replace(/\.[^.]+$/, "") ?? "image";
    let suggested = filenameGuess.replace(/[^A-Za-z0-9._-]/g, "_");
    if (/^[^A-Za-z_]/.test(suggested)) suggested = "_" + suggested;
    while (existingIds.has(suggested)) suggested += "_1";
    uploadStaged = { bin, draftId: suggested, path, error: "" };
  }

  function cancelUpload() { uploadStaged = null; }

  function confirmUpload() {
    if (!uploadStaged || !fb) return;
    const trimmed = uploadStaged.draftId.trim();
    if (!isValidId(trimmed)) {
      uploadStaged = { ...uploadStaged, error: `"${trimmed}" is not a valid id. Use letters, digits, underscore, dot, hyphen; must start with a letter or underscore.` };
      return;
    }
    if ((fb.Binaries ?? []).some(b => b.ID === trimmed)) {
      uploadStaged = { ...uploadStaged, error: `A binary with id "${trimmed}" already exists.` };
      return;
    }
    uploadStaged.bin.ID = trimmed;
    fb.Binaries = [...(fb.Binaries ?? []), uploadStaged.bin];
    fb = fb;
    uploadStaged = null;
  }

  function onKey(e: KeyboardEvent) {
    if (!open) return;
    if (e.key === "Escape") {
      e.preventDefault();
      // Layered escape: innermost overlay first, dialog last.
      if (errorMessage) errorMessage = "";
      else if (uploadStaged) cancelUpload();
      else if (deleting) cancelDelete();
      else if (renaming) cancelRename();
      else cancel();
    }
  }

  onMount(() => {
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  });

  // FB2 binary ids follow XML ID rules: start with letter or `_`, then any
  // mix of letters / digits / `-_.`. Real FB2 readers are looser but this
  // keeps output portable.
  function isValidId(id: string): boolean {
    return /^[A-Za-z_][A-Za-z0-9._-]*$/.test(id);
  }

  function fmtBytes(n: number): string {
    if (n < 1024) return `${n} B`;
    if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
    return `${(n / 1024 / 1024).toFixed(1)} MB`;
  }

  /** base64 payload size → raw byte count. */
  function base64Size(data: string): number {
    if (!data) return 0;
    const padding = /=+$/.exec(data)?.[0]?.length ?? 0;
    return Math.floor((data.length * 3) / 4) - padding;
  }

  function dataURL(b: Binary): string {
    return `data:${b.ContentType};base64,${b.Data}`;
  }
</script>

{#if open}
  <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <div
    class="backdrop"
    role="button"
    tabindex="-1"
    aria-label="Dismiss dialog"
    on:click={cancel}
    on:keydown={(e) => e.key === "Escape" && cancel()}>
    <div
      class="dialog"
      role="dialog"
      aria-modal="true"
      aria-labelledby="bm-title"
      on:click|stopPropagation
      on:keydown|stopPropagation>
      <header>
        <h3 id="bm-title">Binaries</h3>
        <button type="button" class="close" title="Close (Esc)" on:click={cancel}>✕</button>
      </header>

      <div class="toolbar">
        <button type="button" class="primary" on:click={upload}>+ Upload image…</button>
        <span class="count">
          {fb?.Binaries?.length ?? 0}
          {(fb?.Binaries?.length ?? 0) === 1 ? "binary" : "binaries"}
        </span>
      </div>

      <div class="list">
        {#if !fb?.Binaries?.length}
          <p class="empty">No binaries in this document. Upload an image to get started.</p>
        {:else}
          {#each fb.Binaries as b (b)}
            <div class="row" class:renaming={renaming === b}>
              <div class="thumb">
                {#if b.ContentType.startsWith("image/")}
                  <img src={dataURL(b)} alt={b.ID} />
                {:else}
                  <span class="placeholder">{b.ContentType.split("/")[1] ?? "?"}</span>
                {/if}
              </div>

              <div class="meta">
                <div class="id-line">
                  {#if renaming === b}
                    <input
                      type="text"
                      bind:this={renameInput}
                      bind:value={draftId}
                      on:keydown={(e) => {
                        if (e.key === "Enter") { e.preventDefault(); commitRename(b); }
                        else if (e.key === "Escape") { e.preventDefault(); cancelRename(); }
                      }} />
                    <button type="button" class="small primary" on:click={() => commitRename(b)}>Save</button>
                    <button type="button" class="small" on:click={cancelRename}>Cancel</button>
                  {:else}
                    <span class="id" title={b.ID}>{b.ID}</span>
                    {#if isCoverBinary(fb, b.ID)}
                      <span class="badge cover" title="Referenced as the book cover">COVER</span>
                    {/if}
                  {/if}
                </div>
                <div class="info">{b.ContentType} · {fmtBytes(base64Size(b.Data))}</div>
              </div>

              {#if renaming !== b}
                <div class="actions">
                  <button type="button" title="Rename id" on:click={() => beginRename(b)}>✎</button>
                  <button type="button" class="danger" title="Delete" on:click={() => beginDelete(b)}>🗑</button>
                </div>
              {/if}
            </div>
          {/each}
        {/if}
      </div>

      <footer>
        <button type="button" on:click={cancel}>Close</button>
      </footer>

      <!-- Upload "name your binary" sub-overlay. -->
      {#if uploadStaged}
        <div class="sub-backdrop" on:click={cancelUpload} on:keydown={() => {}} role="button" tabindex="-1" aria-label="Cancel upload">
          <div class="sub-dialog" role="dialog" aria-modal="true" aria-labelledby="up-title" on:click|stopPropagation on:keydown|stopPropagation>
            <h4 id="up-title">Name this binary</h4>
            <div class="preview-row">
              <div class="thumb small">
                {#if uploadStaged.bin.ContentType.startsWith("image/")}
                  <img src={dataURL(uploadStaged.bin)} alt="preview" />
                {/if}
              </div>
              <div class="meta">
                <div class="info">{uploadStaged.path.split(/[\\/]/).pop()}</div>
                <div class="info muted">{uploadStaged.bin.ContentType} · {fmtBytes(base64Size(uploadStaged.bin.Data))}</div>
              </div>
            </div>
            <label for="up-id">Binary id</label>
            <input
              id="up-id"
              type="text"
              bind:this={uploadIdInput}
              bind:value={uploadStaged.draftId}
              on:keydown={(e) => {
                if (e.key === "Enter") { e.preventDefault(); confirmUpload(); }
                else if (e.key === "Escape") { e.preventDefault(); cancelUpload(); }
              }} />
            {#if uploadStaged.error}
              <p class="err">{uploadStaged.error}</p>
            {/if}
            <div class="sub-actions">
              <button type="button" on:click={cancelUpload}>Cancel</button>
              <button type="button" class="primary" on:click={confirmUpload}>Add</button>
            </div>
          </div>
        </div>
      {/if}

      <!-- Delete confirmation sub-overlay. -->
      {#if deleting}
        <div class="sub-backdrop" on:click={cancelDelete} on:keydown={() => {}} role="button" tabindex="-1" aria-label="Cancel delete">
          <div class="sub-dialog" role="alertdialog" aria-modal="true" aria-labelledby="del-title" on:click|stopPropagation on:keydown|stopPropagation>
            <h4 id="del-title">Delete binary?</h4>
            <p>
              Remove <code>{deleting.b.ID}</code> from the document?
            </p>
            {#if deleting.coverRefs + deleting.editorRefs > 0}
              <p class="warn">
                {deleting.coverRefs + deleting.editorRefs} reference{deleting.coverRefs + deleting.editorRefs === 1 ? "" : "s"}
                ({deleting.coverRefs} in cover, {deleting.editorRefs} in body) will become dangling — the corresponding images will render as broken until you fix or remove them.
              </p>
            {/if}
            <div class="sub-actions">
              <button type="button" on:click={cancelDelete}>Cancel</button>
              <button type="button" class="danger" on:click={confirmDelete}>Delete</button>
            </div>
          </div>
        </div>
      {/if}

      <!-- Generic error toast — used by rename validation + wails import. -->
      {#if errorMessage}
        <div class="sub-backdrop" on:click={() => (errorMessage = "")} on:keydown={() => {}} role="button" tabindex="-1" aria-label="Dismiss error">
          <div class="sub-dialog error" role="alertdialog" aria-modal="true" on:click|stopPropagation on:keydown|stopPropagation>
            <p>{errorMessage}</p>
            <div class="sub-actions">
              <button type="button" class="primary" on:click={() => (errorMessage = "")}>OK</button>
            </div>
          </div>
        </div>
      {/if}
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
    padding: 0;
    width: min(640px, 92vw);
    max-height: min(80vh, 720px);
    display: flex;
    flex-direction: column;
    box-shadow: 0 8px 24px var(--shadow);
    color: var(--fg);
  }
  header {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.75rem 1rem;
    border-bottom: 1px solid var(--border);
  }
  header h3 {
    margin: 0;
    font-size: 1rem;
    flex: 1;
  }
  .close {
    width: 1.8rem;
    height: 1.8rem;
    background: transparent;
    color: var(--fg-secondary);
    border: none;
    cursor: pointer;
    border-radius: 3px;
    font-size: 0.9rem;
  }
  .close:hover { background: var(--bg-hover); color: var(--fg); }

  .toolbar {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.75rem 1rem;
    border-bottom: 1px solid var(--border);
    background: var(--bg-chrome);
  }
  .toolbar .count {
    margin-left: auto;
    color: var(--fg-secondary);
    font-size: 0.85em;
    font-variant-numeric: tabular-nums;
  }

  .list {
    flex: 1;
    overflow-y: auto;
    padding: 0.5rem 1rem;
  }
  .empty {
    text-align: center;
    color: var(--fg-secondary);
    padding: 2rem 1rem;
    margin: 0;
    font-size: 0.9em;
  }

  .row {
    display: grid;
    grid-template-columns: 4.5rem 1fr auto;
    align-items: center;
    gap: 0.75rem;
    padding: 0.5rem 0;
    border-bottom: 1px solid var(--border);
  }
  .row:last-child { border-bottom: none; }

  .thumb {
    width: 4.5rem;
    height: 4.5rem;
    background: var(--bg-surface);
    border: 1px solid var(--border);
    border-radius: 3px;
    overflow: hidden;
    display: grid;
    place-items: center;
  }
  .thumb img {
    max-width: 100%;
    max-height: 100%;
    object-fit: contain;
  }
  .thumb .placeholder {
    font-family: "SF Mono", Menlo, Consolas, monospace;
    font-size: 0.75rem;
    color: var(--fg-secondary);
    text-transform: uppercase;
  }

  .meta {
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 0.2rem;
  }
  .id-line {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    min-width: 0;
  }
  .id {
    font-family: "SF Mono", Menlo, Consolas, monospace;
    font-weight: 600;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .id-line input[type="text"] {
    flex: 1;
    min-width: 0;
    padding: 0.2rem 0.4rem;
    border: 1px solid var(--border-input);
    background: var(--bg-surface);
    color: var(--fg);
    border-radius: 3px;
    font: inherit;
    font-family: "SF Mono", Menlo, Consolas, monospace;
  }
  .badge.cover {
    padding: 0.05rem 0.35rem;
    font-size: 0.65rem;
    letter-spacing: 0.04em;
    font-weight: 700;
    color: var(--bg);
    background: var(--warn);
    border-radius: 3px;
  }
  .info {
    color: var(--fg-secondary);
    font-size: 0.8em;
  }

  .actions {
    display: flex;
    gap: 0.25rem;
  }
  .actions button {
    width: 1.8rem;
    height: 1.8rem;
    display: grid;
    place-items: center;
    background: var(--bg-surface);
    color: var(--fg);
    border: 1px solid var(--border-button);
    border-radius: 3px;
    cursor: pointer;
    font-size: 0.9rem;
  }
  .actions button:hover { background: var(--bg-hover); }
  .actions button.danger:hover {
    background: var(--bg-errors);
    color: var(--danger);
    border-color: var(--danger-border);
  }

  footer {
    display: flex;
    justify-content: flex-end;
    gap: 0.5rem;
    padding: 0.75rem 1rem;
    border-top: 1px solid var(--border);
  }
  footer button,
  .id-line button.small {
    padding: 0.3rem 0.8rem;
    border: 1px solid var(--border-button);
    background: var(--bg-surface);
    color: var(--fg);
    border-radius: 3px;
    cursor: pointer;
    font: inherit;
  }
  .id-line button.small {
    padding: 0.2rem 0.6rem;
    font-size: 0.85em;
  }
  footer button:hover,
  .id-line button.small:hover { background: var(--bg-hover); }

  button.primary {
    background: var(--bg-active);
    border-color: var(--warn);
    font-weight: 600;
    padding: 0.3rem 0.9rem;
    border-radius: 3px;
    color: var(--fg);
    cursor: pointer;
    font: inherit;
    border-width: 1px;
    border-style: solid;
  }
  button.primary:hover { background: var(--bg-active-hover); }

  /* Sub-overlays for upload / delete / error. Positioned over the main
     dialog; the main dialog backdrop is still there so there's a double
     dim layer, which matches how most "confirm inside a modal" patterns
     look. Z-index relative to the .dialog so they only darken the dialog
     content, not the whole page. */
  .sub-backdrop {
    position: absolute;
    inset: 0;
    background: var(--backdrop);
    display: grid;
    place-items: center;
    border-radius: 6px;
  }
  .dialog { position: relative; }
  .sub-dialog {
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: 6px;
    padding: 1rem 1.2rem;
    width: min(420px, 85%);
    box-shadow: 0 4px 16px var(--shadow);
    color: var(--fg);
  }
  .sub-dialog h4 {
    margin: 0 0 0.6rem 0;
    font-size: 0.95rem;
  }
  .sub-dialog p {
    margin: 0 0 0.6rem 0;
    font-size: 0.9em;
    line-height: 1.4;
  }
  .sub-dialog p.warn {
    color: var(--warn-fg);
    background: var(--warn-bg-a);
    border: 1px solid var(--warn);
    padding: 0.4rem 0.6rem;
    border-radius: 3px;
  }
  .sub-dialog p.err {
    color: var(--danger);
    background: var(--bg-errors);
    border: 1px solid var(--danger-border);
    padding: 0.4rem 0.6rem;
    border-radius: 3px;
    font-size: 0.85em;
  }
  .sub-dialog.error p {
    color: var(--danger);
  }
  .sub-dialog label {
    display: block;
    font-size: 0.85em;
    color: var(--fg-secondary);
    margin-bottom: 0.2rem;
  }
  .sub-dialog input[type="text"] {
    width: 100%;
    box-sizing: border-box;
    padding: 0.3rem 0.5rem;
    border: 1px solid var(--border-input);
    background: var(--bg-surface);
    color: var(--fg);
    border-radius: 3px;
    font: inherit;
    font-family: "SF Mono", Menlo, Consolas, monospace;
  }
  .sub-dialog code {
    font-family: "SF Mono", Menlo, Consolas, monospace;
    font-size: 0.9em;
    background: var(--bg-chrome);
    padding: 0.05em 0.3em;
    border-radius: 2px;
  }

  .preview-row {
    display: flex;
    gap: 0.75rem;
    align-items: center;
    margin-bottom: 0.75rem;
  }
  .thumb.small {
    width: 3rem;
    height: 3rem;
  }
  .info.muted { color: var(--fg-secondary); font-size: 0.8em; }

  .sub-actions {
    display: flex;
    justify-content: flex-end;
    gap: 0.5rem;
    margin-top: 0.75rem;
  }
  .sub-actions button {
    padding: 0.3rem 0.9rem;
    border: 1px solid var(--border-button);
    background: var(--bg-surface);
    color: var(--fg);
    border-radius: 3px;
    cursor: pointer;
    font: inherit;
  }
  .sub-actions button:hover { background: var(--bg-hover); }
  .sub-actions button.danger {
    background: var(--danger);
    color: var(--bg);
    border-color: var(--danger-border);
  }
  .sub-actions button.danger:hover { opacity: 0.9; }
</style>
