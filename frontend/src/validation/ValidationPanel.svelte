<script lang="ts">
  import { createEventDispatcher, onDestroy } from "svelte";
  import XmlEditor from "./XmlEditor.svelte";
  import Resizer from "../ui/Resizer.svelte";

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
  /** XML line of the text being read; the pane follows it. */
  export let followLine: number | null = null;

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
    /** The user scrolled the XML; payload is the line at the top. */
    "scroll-line": number;
  }>();

  let panelEl: HTMLDivElement | undefined;

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

  // CM6 shows xmlSource in both modes. A new xmlSource (Validate, or a successful Apply, which
  // brings the writer-normalized form) replaces its text and clears the dirty mark. Entering edit
  // mode changes nothing but the baseline, and leaving it drops unapplied edits.
  let shownXml = xmlSource;
  $: if (xmlSource !== shownXml) {
    shownXml = xmlSource;
    lastBaselineXml = xmlSource;
    xmlEditor?.setDoc(xmlSource);
    if (followLine) xmlEditor?.scrollToLine(followLine);
  }
  let prevEditMode = false;
  $: {
    if (editMode && !prevEditMode) {
      lastBaselineXml = xmlSource;
      editedXml = xmlSource;
    } else if (!editMode && prevEditMode && xmlEditor && xmlEditor.getXml() !== xmlSource) {
      xmlEditor.setDoc(xmlSource);
    }
    prevEditMode = editMode;
  }

  // Keep the XML on the line being read, unless the user is working in it.
  $: if (xmlEditor && followLine && !editDirty && !xmlEditor.hasFocus()) {
    xmlEditor.scrollToLine(followLine);
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

  $: lineCount = xmlSource ? xmlSource.split("\n").length : 0;

  function panelBounds() {
    if (!panelEl) return null;
    const r = panelEl.getBoundingClientRect();
    // Leave room for title (2rem ≈ 32px) and at least 60px for the XML pane.
    return { top: r.top, bottom: r.bottom, min: 60, max: r.height - 32 - 60 };
  }

  const errorsResizer = {
    current: () => errorsHeight ?? Math.round((panelEl?.getBoundingClientRect().height ?? 0) * 0.45),
    bounds: () => panelBounds() ?? { min: 0, max: 0 },
    fromPointer: (e: PointerEvent) => (panelBounds()?.bottom ?? 0) - e.clientY,
  };

  onDestroy(() => {
    document.body.style.cursor = "";
    document.body.style.userSelect = "";
  });
</script>

<div class="panel" bind:this={panelEl}>
  <div class="panel-title">
    <span>
      XML source{lineCount > 0 ? ` · ${lineCount} lines` : ""}
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

  <div class="edit-mode-wrap">
    {#if editMode}
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
    {/if}
    <div class="edit-host">
      <XmlEditor
        bind:this={xmlEditor}
        initial={xmlSource}
        readOnly={!editMode}
        {theme}
        on:change={onEditorChange}
        on:scrollline={(e) => !editDirty && dispatch("scroll-line", e.detail)}
      />
    </div>
  </div>

  {#if errors.length > 0}
    <Resizer
      orientation="horizontal"
      label="Resize errors pane (drag, or arrow keys when focused; Shift = larger step)"
      {...errorsResizer}
      bind:dragging
      on:change={(e) => (errorsHeight = e.detail)}
      on:commit={(e) => dispatch("resize", { height: e.detail })}
    />

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
              on:click={() => xmlEditor?.gotoLine(e.line)}
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
    font-family: var(--font-mono);
    font-size: 0.78rem;
  }

  .panel-title {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 0.25rem 0 0.6rem;
    background: var(--bg-chrome);
    border-bottom: 1px solid var(--border);
    font-family: var(--font-ui);
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
    background: var(--bg-active);
    color: var(--fg-strong);
  }
  .title-actions .action.apply:hover:not(:disabled) {
    background: var(--bg-active-hover);
  }
  .dirty-mark {
    color: var(--warn-fg);
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
    background: var(--bg-errors);
    border-bottom: 1px solid var(--danger-border);
    color: var(--danger);
    padding: 0.55rem 0.8rem;
    font-family: var(--font-ui);
    font-size: 0.82rem;
    flex: 0 0 auto;
  }
  .apply-error-msg {
    margin-top: 0.2rem;
    font-family: var(--font-mono);
    font-size: 0.74rem;
    white-space: pre-wrap;
  }
  .undo-banner {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.5rem;
    padding: 0.35rem 0.7rem;
    background: var(--bg-ok);
    border-bottom: 1px solid var(--border);
    color: var(--fg-secondary);
    font-family: var(--font-ui);
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
    font-family: var(--font-ui);
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
    font-family: var(--font-mono);
    font-size: 0.72rem;
  }
  .errors li .msg {
    color: var(--danger);
    word-break: break-word;
    white-space: normal;
  }
</style>
