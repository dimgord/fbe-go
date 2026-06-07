<script lang="ts">
  import { createEventDispatcher, onDestroy, tick } from "svelte";
  import XmlEditor from "./XmlEditor.svelte";

  export let xmlSource: string = "";
  export let errors: { line: number; column: number; message: string }[] = [];
  /** App's effective theme passed through to CM6. */
  export let theme: "light" | "dark" = "light";
  /** Last Apply attempt's error message (empty = no error). Surfaces
   *  inline above the editor so the user sees WHY their Apply was
   *  rejected without having to scan the App-level status bar. */
  export let applyError: string = "";
  /** True when an Undo target is available (parent captured a
   *  pre-Apply snapshot). Controls visibility of the "Undo Apply"
   *  banner — separate from edit-mode toggle so the banner stays
   *  visible even after the user clicks Done. */
  export let canUndoApply: boolean = false;
  // Initial height from persisted settings (px). `null` / `0` = use CSS
  // default (45%). Not bound — parent only seeds the initial value; the
  // panel reports changes via the `resize` event.
  export let initialErrorsHeight: number | null = null;

  const dispatch = createEventDispatcher<{
    close: void;
    resize: { height: number };
    /** Fired when the user clicks Apply in edit mode. Parent should
     *  call App.ParseXMLString → on success swap fb; on failure surface
     *  an error and KEEP edit mode open. */
    apply: { xml: string };
    /** Fired when the user clicks "Undo Apply". Parent restores the
     *  pre-Apply fb snapshot. */
    "undo-apply": void;
    /** Fired on every CM6 edit. Parent uses this for dirty tracking. */
    "xml-dirty": { dirty: boolean };
  }>();

  let xmlPane: HTMLDivElement | undefined;
  let panelEl: HTMLDivElement | undefined;
  let highlightedLine: number | null = null;

  // Edit-mode state. Exported as a bind:able prop so the parent
  // (App.svelte) can flip it off after a successful Apply →
  // ParseXMLString → UpdateDocument cycle. On parse failure the parent
  // leaves it `true` and surfaces the error inline; the user's text
  // stays in the CM6 doc.
  export let editMode = false;
  let xmlEditor: XmlEditor | undefined;
  let editedXml = "";
  let lastBaselineXml = "";  // what we entered editMode with
  $: editDirty = editMode && editedXml !== lastBaselineXml;
  $: dispatch("xml-dirty", { dirty: editDirty });

  // Three-way sync of editMode + xmlSource into CM6:
  //   - editMode flips false→true: seed CM6 with current xmlSource and
  //     capture it as the dirty-tracking baseline.
  //   - editMode flips true→false: reset working buffer.
  //   - editMode stays true but xmlSource changes externally (parent
  //     re-ran SerializeCurrent after a successful Apply): refresh CM6
  //     to the new (writer-normalized) form and reset baseline so the
  //     dirty mark clears. Without this last branch, the user's Apply
  //     would commit fine but the editor would still show their
  //     pre-normalization text and the dirty mark would stay lit.
  let prevEditMode = false;
  let lastSeenXmlSource = "";
  $: {
    if (editMode && !prevEditMode) {
      lastBaselineXml = xmlSource;
      editedXml = xmlSource;
      lastSeenXmlSource = xmlSource;
    } else if (!editMode && prevEditMode) {
      editedXml = "";
      lastBaselineXml = "";
      lastSeenXmlSource = "";
    } else if (editMode && xmlSource !== lastSeenXmlSource) {
      lastBaselineXml = xmlSource;
      lastSeenXmlSource = xmlSource;
      xmlEditor?.setDoc(xmlSource);
      // CM6 docChanged fires from setDoc → onEditorChange updates
      // editedXml → editDirty drops to false on next reactive pass.
    }
    prevEditMode = editMode;
  }

  function enterEditMode() { editMode = true; }
  /** "Done" — exits edit mode unconditionally. Any unapplied CM6
   *  edits are dropped on the floor (the in-memory FictionBook stays
   *  untouched since no Apply ran). No confirm prompt because Wails'
   *  WKWebView silently swallows `window.confirm` returns under some
   *  build configurations — bug-reported as "Done doesn't exit".
   *  If the user wants to keep their text, Apply is the explicit
   *  commit; Done is the explicit abandon. */
  function doneEdit() { editMode = false; }
  /** "Apply" — pushes the current CM6 buffer to the parent, which
   *  parses and commits to the in-memory FictionBook. On success the
   *  parent's re-Serialize bumps xmlSource → the reactive above
   *  refreshes CM6 to the normalized form and clears the baseline
   *  (so editDirty → false). On failure the parent leaves editMode
   *  true and surfaces the error; the user's text is preserved
   *  verbatim in CM6 for them to fix and retry. */
  function applyEdit() {
    const next = xmlEditor?.getXml() ?? editedXml;
    dispatch("apply", { xml: next });
  }
  function onEditorChange(e: CustomEvent<string>) {
    editedXml = e.detail;
  }

  // Errors-pane height in pixels. `null` = "use default CSS" (45% of panel).
  // Switches to a concrete number once the user starts dragging the resizer.
  let errorsHeight: number | null = initialErrorsHeight && initialErrorsHeight > 0
    ? initialErrorsHeight
    : null;
  let dragging = false;

  $: xmlLines = xmlSource.split("\n");

  async function gotoLine(n: number) {
    highlightedLine = n;
    await tick();
    const el = xmlPane?.querySelector(`#xml-line-${n}`);
    el?.scrollIntoView({ block: "center", behavior: "smooth" });
    setTimeout(() => {
      if (highlightedLine === n) highlightedLine = null;
    }, 2500);
  }

  function clamp(v: number, lo: number, hi: number) {
    return Math.max(lo, Math.min(hi, v));
  }

  function panelBounds() {
    if (!panelEl) return null;
    const r = panelEl.getBoundingClientRect();
    // Leave room for title (2rem ≈ 32px) and at least 60px for the XML pane.
    return { top: r.top, bottom: r.bottom, min: 60, max: r.height - 32 - 60 };
  }

  function startDrag(e: PointerEvent) {
    e.preventDefault();
    dragging = true;
    const target = e.currentTarget as HTMLElement;
    target.setPointerCapture(e.pointerId);
    document.body.style.cursor = "ns-resize";
    document.body.style.userSelect = "none";
  }

  function onDrag(e: PointerEvent) {
    if (!dragging) return;
    const b = panelBounds();
    if (!b) return;
    errorsHeight = clamp(b.bottom - e.clientY, b.min, b.max);
  }

  function endDrag(e: PointerEvent) {
    if (!dragging) return;
    dragging = false;
    const target = e.currentTarget as HTMLElement;
    if (target.hasPointerCapture(e.pointerId)) target.releasePointerCapture(e.pointerId);
    document.body.style.cursor = "";
    document.body.style.userSelect = "";
    if (errorsHeight !== null) {
      dispatch("resize", { height: errorsHeight });
    }
  }

  function onResizerKey(e: KeyboardEvent) {
    const b = panelBounds();
    if (!b) return;
    const current = errorsHeight ?? Math.round(panelEl!.getBoundingClientRect().height * 0.45);
    const step = e.shiftKey ? 40 : 10;
    let changed = false;
    if (e.key === "ArrowUp") {
      e.preventDefault();
      errorsHeight = clamp(current + step, b.min, b.max);
      changed = true;
    } else if (e.key === "ArrowDown") {
      e.preventDefault();
      errorsHeight = clamp(current - step, b.min, b.max);
      changed = true;
    }
    if (changed && errorsHeight !== null) {
      dispatch("resize", { height: errorsHeight });
    }
  }

  onDestroy(() => {
    document.body.style.cursor = "";
    document.body.style.userSelect = "";
  });
</script>

<div class="panel" bind:this={panelEl}>
  <div class="panel-title">
    <span>
      XML source{xmlLines.length > 0 ? ` · ${xmlLines.length} lines` : ""}
      {#if editDirty}<span class="dirty-mark" title="Unsaved edits">●</span>{/if}
    </span>
    <div class="title-actions">
      {#if editMode}
        <button
          class="action apply"
          on:click={applyEdit}
          disabled={!editDirty}
          title="Parse XML and replace the in-memory document; editor stays open"
        >Apply</button>
        <button
          class="action"
          on:click={doneEdit}
          title={editDirty ? "Close editor (prompts about unapplied edits)" : "Close editor"}
        >Done</button>
      {:else}
        <button
          class="action"
          on:click={enterEditMode}
          disabled={!xmlSource}
          title="Edit raw XML"
        >Edit XML…</button>
      {/if}
      <button on:click={() => dispatch("close")} title="Close panel">×</button>
    </div>
  </div>

  {#if editMode}
    <div class="edit-mode-wrap">
      {#if applyError}
        <div class="apply-error" role="alert">
          <strong>Apply failed — your text is preserved.</strong>
          <div class="apply-error-msg">{applyError}</div>
        </div>
      {/if}
      {#if canUndoApply}
        <div class="undo-banner">
          <span>Last Apply succeeded. Lost something?</span>
          <button class="action" on:click={() => dispatch("undo-apply")}>Undo Apply</button>
        </div>
      {/if}
      <div class="edit-host">
        <XmlEditor
          bind:this={xmlEditor}
          initial={lastBaselineXml}
          {theme}
          on:change={onEditorChange}
        />
      </div>
    </div>
  {:else}
    <div class="xml" bind:this={xmlPane}>
      {#each xmlLines as line, i}
        <div
          id={`xml-line-${i + 1}`}
          class="xml-line"
          class:hl={highlightedLine === i + 1}
        >
          <span class="ln">{i + 1}</span>
          <span class="content">{line}</span>
        </div>
      {/each}
    </div>
  {/if}

  {#if errors.length > 0}
    <!-- svelte-ignore a11y-no-noninteractive-tabindex -->
    <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
    <div
      class="resizer"
      class:dragging
      role="separator"
      aria-orientation="horizontal"
      aria-label="Resize errors pane (drag, or arrow keys when focused; Shift = larger step)"
      tabindex="0"
      on:pointerdown={startDrag}
      on:pointermove={onDrag}
      on:pointerup={endDrag}
      on:pointercancel={endDrag}
      on:keydown={onResizerKey}
    ></div>

    <div
      class="errors"
      style={errorsHeight !== null ? `height: ${errorsHeight}px;` : ""}
    >
      <div class="errors-title">
        <span class="dot">●</span>
        {errors.length} XSD error{errors.length === 1 ? "" : "s"}
      </div>
      <ul>
        {#each errors as e}
          <li>
            <button
              type="button"
              on:click={() => gotoLine(e.line)}
              title="Jump to line {e.line}"
            >
              <span class="pos">L{e.line}:{e.column}</span>
              <span class="msg">{e.message}</span>
            </button>
          </li>
        {/each}
      </ul>
    </div>
  {:else if xmlSource}
    <div class="errors ok">
      <span class="dot">✓</span> XSD valid
    </div>
  {/if}
</div>

<style>
  .panel {
    display: grid;
    grid-template-rows: 2rem 1fr auto auto;
    height: 100%;
    min-height: 0;
    border-left: 1px solid var(--border);
    background: var(--bg-surface);
    color: var(--fg);
    font-family: "SF Mono", Menlo, Consolas, monospace;
    font-size: 0.78rem;
  }

  .panel-title {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 0.25rem 0 0.6rem;
    background: var(--bg-chrome);
    border-bottom: 1px solid var(--border);
    font-family: -apple-system, "Segoe UI", sans-serif;
    font-size: 0.8rem;
    font-weight: 600;
    color: var(--fg);
  }
  .panel-title button {
    border: none;
    background: transparent;
    font-size: 1.1rem;
    line-height: 1;
    padding: 0.15rem 0.5rem;
    color: var(--fg-muted);
    cursor: pointer;
    border-radius: 3px;
  }
  .panel-title button:hover {
    background: var(--bg-hover);
    color: var(--fg-strong);
  }
  .title-actions {
    display: flex;
    align-items: center;
    gap: 0.25rem;
  }
  .title-actions .action {
    font-size: 0.78rem;
    font-weight: 600;
    padding: 0.2rem 0.55rem;
    border: 1px solid var(--border-button, var(--border));
    background: var(--bg-surface);
    color: var(--fg);
    line-height: 1.1;
  }
  .title-actions .action:hover:not(:disabled) {
    background: var(--bg-hover);
  }
  .title-actions .action:disabled {
    opacity: 0.45;
    cursor: default;
  }
  .title-actions .action.apply {
    background: var(--bg-active, #fce6a0);
    color: var(--fg-strong);
  }
  .title-actions .action.apply:hover:not(:disabled) {
    background: var(--bg-active-hover, #f5da7c);
  }
  .dirty-mark {
    color: var(--warn-fg, #c97a00);
    margin-left: 0.35rem;
    font-size: 0.7rem;
    vertical-align: middle;
  }
  /* edit-mode-wrap is a flex column inside the panel's `1fr` grid
     row. Without this wrapper, apply-error + undo-banner + XmlEditor
     are direct grid children and each consumes its own (implicit) row,
     which pushes the editor off-screen and breaks the resizer/errors
     rows below. The wrap stays in the `1fr` slot; banners take their
     natural height; the editor gets the leftover via flex:1. */
  .edit-mode-wrap {
    display: flex;
    flex-direction: column;
    min-height: 0;
    height: 100%;
    overflow: hidden;
  }
  .edit-host {
    flex: 1 1 auto;
    min-height: 0;
    display: flex;
    flex-direction: column;
  }
  .apply-error {
    background: var(--bg-errors, #fff5f5);
    border-bottom: 1px solid var(--err-border, #f3c5c5);
    color: var(--err-fg, #a13030);
    padding: 0.55rem 0.8rem;
    font-family: -apple-system, "Segoe UI", sans-serif;
    font-size: 0.82rem;
    flex: 0 0 auto;
  }
  .apply-error-msg {
    margin-top: 0.2rem;
    font-family: "SF Mono", Menlo, Consolas, monospace;
    font-size: 0.74rem;
    white-space: pre-wrap;
  }
  .undo-banner {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.5rem;
    padding: 0.35rem 0.7rem;
    background: var(--bg-ok, #f2faf4);
    border-bottom: 1px solid var(--border);
    color: var(--fg-secondary);
    font-family: -apple-system, "Segoe UI", sans-serif;
    font-size: 0.78rem;
    flex: 0 0 auto;
  }
  .undo-banner .action {
    font-weight: 600;
    padding: 0.18rem 0.55rem;
    font-size: 0.76rem;
    border: 1px solid var(--border-button, var(--border));
    background: var(--bg-surface);
    color: var(--fg);
    border-radius: 3px;
    cursor: pointer;
  }
  .undo-banner .action:hover {
    background: var(--bg-hover);
  }

  .xml {
    overflow: auto;
    padding: 0.25rem 0;
    min-height: 0;
  }
  .xml-line {
    display: grid;
    grid-template-columns: 3.25rem 1fr;
    gap: 0.35rem;
    padding: 0 0.25rem;
    white-space: pre;
    line-height: 1.35;
    /* The XML pane is non-virtualized — every line of the source is
       rendered as its own grid div. A typical FB2 (Облачный атлас) has
       7 000+ lines, and without these hints the browser reflows all of
       them on every mousemove of the window resize handle. The actual
       cost dwarfs the editor's contenteditable reflow.
         - `content-visibility: auto` skips layout/paint for off-screen
           lines until they scroll into view.
         - `contain-intrinsic-size: auto 1.35em` reserves the right
           placeholder height so the scrollbar geometry stays correct
           without actually doing layout. */
    content-visibility: auto;
    contain-intrinsic-size: auto 1.35em;
  }
  .xml-line .ln {
    color: var(--fg-muted-soft);
    text-align: right;
    user-select: none;
    border-right: 1px solid var(--border);
    padding-right: 0.35rem;
  }
  .xml-line.hl {
    background: var(--highlight);
    transition: background 0.4s ease-out;
  }
  .xml-line.hl .ln {
    color: var(--warn-fg);
    font-weight: 600;
  }

  .resizer {
    position: relative;
    height: 6px;
    background: var(--border);
    cursor: ns-resize;
    border-top: 1px solid var(--border-strong);
    border-bottom: 1px solid var(--border-strong);
    touch-action: none;
  }
  .resizer::before {
    content: "";
    position: absolute;
    left: 50%;
    top: 50%;
    transform: translate(-50%, -50%);
    width: 32px;
    height: 2px;
    background: var(--fg-muted);
    border-radius: 1px;
    box-shadow: 0 3px 0 var(--fg-muted);
  }
  .resizer:hover,
  .resizer.dragging,
  .resizer:focus-visible {
    background: var(--border-strong);
    outline: none;
  }
  .resizer:focus-visible::before {
    background: var(--fg-secondary);
    box-shadow: 0 3px 0 var(--fg-secondary);
  }

  .errors {
    /* Default size when the user hasn't dragged the resizer. Inline `height`
       on the element overrides this once drag starts.
       Was 35% before Rev 45 — two multi-line XSD messages (libxml2 wraps
       to 3+ lines once the namespace URI is inlined) barely fit, so the
       second error hid behind the scroll and users had to drag the
       resizer before they noticed it was there. 45% holds 2–3 typical
       error rows without dragging. min-height stays at 60px so the
       resizer's panelBounds.min (60) isn't fought by the CSS clamp. */
    height: 45%;
    min-height: 60px;
    overflow: auto;
    background: var(--bg-errors);
    font-family: -apple-system, "Segoe UI", sans-serif;
    font-size: 0.8rem;
  }
  .errors-title {
    position: sticky;
    top: 0;
    z-index: 1;
    background: var(--bg-errors-title);
    color: var(--danger);
    padding: 0.3rem 0.6rem;
    border-bottom: 1px solid var(--danger-border);
    font-size: 0.72rem;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    font-weight: 700;
  }
  .errors .dot { margin-right: 0.25rem; }

  .errors.ok {
    height: auto;
    color: var(--ok);
    padding: 0.5rem 0.75rem;
    background: var(--bg-ok);
    border-top: 1px solid var(--ok-border);
    font-weight: 600;
  }

  .errors ul {
    list-style: none;
    margin: 0;
    padding: 0;
  }
  .errors li {
    border-bottom: 1px solid var(--border);
  }
  .errors li button {
    all: unset;
    display: grid;
    grid-template-columns: 4.5rem 1fr;
    gap: 0.5rem;
    align-items: start;
    width: 100%;
    padding: 0.3rem 0.6rem;
    box-sizing: border-box;
    cursor: pointer;
    text-align: left;
  }
  .errors li button:hover { background: var(--bg-hover); }
  .errors li button:focus-visible {
    outline: 2px solid var(--danger);
    outline-offset: -2px;
    background: var(--bg-hover);
  }
  .errors li .pos {
    color: var(--fg-muted);
    font-family: "SF Mono", Menlo, Consolas, monospace;
    font-size: 0.72rem;
  }
  .errors li .msg {
    color: var(--danger);
    word-break: break-word;
    white-space: normal;
  }
</style>
