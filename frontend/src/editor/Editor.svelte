<script lang="ts">
  import { onDestroy, onMount, createEventDispatcher } from "svelte";
  import { EditorState, Plugin, type Transaction } from "prosemirror-state";
  import { EditorView } from "prosemirror-view";
  import { history } from "prosemirror-history";
  import { keydownHandler, keymap } from "prosemirror-keymap";
  import { baseKeymap } from "prosemirror-commands";
  import { Node as PMNode } from "prosemirror-model";
  import { fb2Schema } from "./schema";
  import { errorDoc, fb2ToPMDoc } from "./parse";
  import { pmDocToFB2 } from "./serialize";
  import { cleanPastedHTML, cleanPastedText } from "./paste";
  import { searchPlugin } from "./search/plugin";
  import { noteLinksPlugin, noteLinksBack } from "./noteLinks";
  import { scrollContainer, scrollForcePlugin, scrollPosIntoView } from "./scroll";
  import { positionAt, type BookPosition } from "../tree/position";
  import { findNodePos } from "../tree/outline";
  import { dirtyPlugin, markClean } from "./dirtyTracker";
  import { createImageViews } from "./imageView";
  import { toggleLink, insertTableCmd } from "./commands";
  import { EDITOR_COMMANDS, type CommandId } from "./commandRegistry";
  import { HOTKEY_ACTIONS, toPMKey } from "../settings/hotkeys";
  import type { Command } from "prosemirror-state";
  import TableDialog from "./TableDialog.svelte";
  import type { FictionBook, Binary } from "../fb2/types";

  export let fb: FictionBook | null = null;

  /** User-configurable hotkey map, loaded from settings.Hotkeys. A new object
      takes effect at once, see `activeKeymap`. Empty / missing actions are
      simply not bound. */
  export let hotkeys: Record<string, string> = {};

  /** Export the EditorView so the toolbar in App.svelte can dispatch commands. */
  export let view: EditorView | undefined = undefined;

  /** Inferred from the document's title-info <lang> for native spellcheck routing. */
  $: lang = fb?.Description?.TitleInfo?.Lang || "en";

  const dispatch = createEventDispatcher<{ dirty: boolean; position: BookPosition }>();
  // What sits at the top of the visible text, reported once per frame at most.
  const POSITION_PROBE_Y = 40;
  let positionFrame = 0;
  let lastPosition = "";
  const POSITION_SETTLE_MS = 150;
  let positionSettle: ReturnType<typeof setTimeout> | undefined;

  function schedulePosition(): void {
    if (!positionFrame) positionFrame = requestAnimationFrame(reportPosition);
  }

  /** The document position at the top of the visible text. */
  function topPos(): number | null {
    if (!view || view.isDestroyed) return null;
    const box = (scrollContainer(view) ?? (view.dom as HTMLElement)).getBoundingClientRect();
    return view.posAtCoords({ left: box.left + box.width / 2, top: box.top + POSITION_PROBE_Y })?.pos ?? null;
  }

  // Where the reader was while body.resizing hides the editor. Hiding collapses the scroller to
  // zero, and a width change reflows the text, so the place is kept as a document position.
  let heldPos: number | null = null;

  export function holdPosition(): void {
    heldPos ??= topPos();
  }

  export function releasePosition(): void {
    const pos = heldPos;
    heldPos = null;
    if (pos === null || !view) return;
    scrollPosIntoView(view, pos, "top", POSITION_PROBE_Y);
    schedulePosition();
  }

  function reportPosition(): void {
    positionFrame = 0;
    if (!view || view.isDestroyed || heldPos !== null) return;
    const pos = topPos();
    if (pos === null) return;
    const next = positionAt(view.state.doc, pos);
    const key = `${next.path.join(".")}@${next.percent}`;
    if (key !== lastPosition) {
      lastPosition = key;
      dispatch("position", next);
    }
  }

  function onAnyScroll(e: Event): void {
    if (!view || !(e.target instanceof Node) || !e.target.contains(view.dom)) return;
    schedulePosition();
    // A jump lands on sections that are only just rendering: measure again once it settles.
    clearTimeout(positionSettle);
    positionSettle = setTimeout(schedulePosition, POSITION_SETTLE_MS);
  }

  onMount(() => {
    window.addEventListener("scroll", onAnyScroll, true); // scroll does not bubble
    return () => {
      window.removeEventListener("scroll", onAnyScroll, true);
      cancelAnimationFrame(positionFrame);
      clearTimeout(positionSettle);
    };
  });

  /** The current state is saved: undoing back to it now reads as clean. */
  export function markSaved(): void {
    if (view) markClean(view);
  }

  /** Serialize current PM doc back to a FictionBook, merged with the original description + binaries. */
  export function currentFB(): FictionBook | null {
    if (!view || !fb) return null;
    return pmDocToFB2(view.state.doc, fb);
  }

  /**
   * Scroll the editor to the section identified by an outline path like
   * [bodyIdx, sectionIdx, subIdx, ...]. Uses ProseMirror's coordsAtPos so it
   * works even when the section has no DOM id.
   *
   * Delta is measured from the scrollable container's rect (not view.dom's),
   * because when view.dom is NOT itself the scrollable element its rect moves
   * with each scroll — using it would cause second-and-later clicks to land
   * above the visible area.
   */
  export function scrollToPath(path: number[]): void {
    if (!view || path.length === 0) return;
    const pos = findNodePos(view.state.doc, path);
    if (pos == null) return;
    scrollPosIntoView(view, pos, "top", 12);
    // Brief highlight on the nearest enclosing element (domAtPos may return a Text node).
    const dom = view.domAtPos(pos);
    const flashTarget =
      dom.node instanceof HTMLElement ? dom.node : dom.node.parentElement;
    if (flashTarget) {
      flashTarget.classList.add("outline-flash");
      setTimeout(() => flashTarget.classList.remove("outline-flash"), 700);
    }
  }

  let container: HTMLDivElement;

  let binaries: Binary[] = [];
  $: binaries = fb?.Binaries ?? [];
  const images = createImageViews(() => binaries);
  $: binaries, images.refresh();

  function buildKeymap(hk: Record<string, string>): Record<string, Command> {
    const bindings: Record<string, Command> = {
      "Mod-z": EDITOR_COMMANDS.Undo,
      "Mod-y": EDITOR_COMMANDS.Redo,
      "Mod-Shift-z": EDITOR_COMMANDS.Redo,
    };
    for (const action of HOTKEY_ACTIONS) {
      if (!action.editor) continue;
      const raw = hk[action.id];
      if (!raw) continue;
      const pmKey = toPMKey(raw);
      if (pmKey) bindings[pmKey] = EDITOR_COMMANDS[action.id];
    }
    // F3 / Shift-F3 stay hardcoded as Windows-convention aliases — users can
    // still rebind FindNext to something else without losing these.
    bindings["F3"] = EDITOR_COMMANDS.FindNext;
    bindings["Shift-F3"] = EDITOR_COMMANDS.FindPrev;
    return bindings;
  }

  function mount(doc: PMNode) {
    view?.destroy();
    const state = EditorState.create({
      schema: fb2Schema,
      doc,
      plugins: [
        history(),
        new Plugin({ props: { handleKeyDown: (v, e) => activeKeymap(v, e) } }),
        keymap({ "Mod-[": noteLinksBack }),
        keymap(baseKeymap),
        searchPlugin(),
        scrollForcePlugin(),
        noteLinksPlugin(),
        dirtyPlugin((dirty) => dispatch("dirty", dirty)),
        new Plugin({
          view: () => ({
            update(v, prev) {
              if (v.state.doc !== prev.doc) schedulePosition();
            },
          }),
        }),
      ],
    });
    view = new EditorView(container, {
      state,
      attributes: { spellcheck: "true", lang },
      transformPastedHTML: cleanPastedHTML,
      transformPastedText: cleanPastedText,
      nodeViews: images.nodeViews,
    });
    schedulePosition();
  }

  /** Settings → Apply swaps the bindings here. Never `reconfigure` for this: PM destroys every
   *  plugin view on it, which reset the dirty baseline and blanked the search bar. */
  $: activeKeymap = keydownHandler(buildKeymap(hotkeys));

  /**
   * Build a PM doc from the FictionBook; on schema failure, fall back to a
   * "parse error" placeholder doc so the app stays alive instead of blank
   * white-screening when a real book trips an unexpected content rule.
   */
  function toPMDoc(src: FictionBook | null): PMNode {
    if (!src) return fb2Schema.topNodeType.createAndFill()!;
    try {
      return fb2ToPMDoc(src);
    } catch (err) {
      console.error("[fbe] fb2ToPMDoc failed:", err, src);
      return errorDoc((err as Error).message || String(err));
    }
  }

  // Re-render attrs when language changes so the webview re-evaluates spellcheck
  // against the new dictionary.
  $: if (view) {
    view.setProps({
      attributes: { spellcheck: "true", lang },
    });
  }

  // Dedupe mount on fb identity, not just "fb is truthy" — otherwise onMount
  // + the reactive block both fire on initial load, creating two EditorView
  // instances and leaking the first. The leaked view's docView stays linked
  // to the DOM, and later Svelte scheduler flushes can call methods on it
  // (e.g. via a sibling's dispatch through bind:view), triggering
  // `TypeError: null is not an object (evaluating 'this.docView.matchesNode')`.
  let lastMountedFB: FictionBook | null = null;

  $: if (container && fb && lastMountedFB !== fb) {
    lastMountedFB = fb;
    mount(toPMDoc(fb));
  }

  onDestroy(() => {
    view?.destroy();
    // Null out so `bind:view` propagates "no live view" to the parent and
    // siblings (SearchBar, BinaryManagerDialog) don't dispatch against a
    // destroyed EditorView after a view-switch.
    view = undefined;
    lastMountedFB = null;
  });

  /** Run a command from outside the editor, such as a toolbar button. */
  export function run(id: CommandId): void {
    if (!view) return;
    EDITOR_COMMANDS[id](view.state, view.dispatch, view);
    view.focus();
  }

  export function execLink(): void {
    if (!view) return;
    const href = prompt("Link URL (leave empty to remove):") ?? "";
    toggleLink(href)(view.state, view.dispatch);
    view.focus();
  }

  let tableDialogOpen = false;
  export function openTableDialog(): void {
    tableDialogOpen = true;
  }
  function onTableInsert(e: CustomEvent<{ rows: number; cols: number; header: boolean }>) {
    if (!view) return;
    insertTableCmd(e.detail.rows, e.detail.cols, e.detail.header)(view.state, view.dispatch);
    view.focus();
  }
</script>

<div bind:this={container} class="editor" />
<TableDialog bind:open={tableDialogOpen} on:insert={onTableInsert} />

<style>
  .editor {
    padding: 1rem 2rem;
    max-width: 820px;
    margin: 0 auto;
    min-height: 100%;
  }
  :global(.ProseMirror) {
    outline: none;
    line-height: 1.65;
    font-family: var(--editor-font-family, "Trebuchet MS", -apple-system, sans-serif);
    font-size: var(--editor-font-size, 16px);
  }
  /* Window-resize fast path. A typical FB2 holds 3 000+ block elements;
     without these hints the browser reflows every contenteditable child
     on every mousemove of the window edge.
     1. `contain: layout style` — children can't affect outer layout, so
        the reflow walk stops at the section boundary.
     2. `content-visibility: auto` on leaf sections — skip rendering them
        off-screen. Each carries an estimated `contain-intrinsic-size`
        (schema.ts), so the scrollbar stays accurate without paying for
        layout on hidden content. This is where the big win comes from on
        large books — Облачный атлас (~7 000 lines XML, ~3 000 PM
        blocks) goes from "the resize handle drags with a 500 ms lag"
        to "drags at 60 fps".
     Caveats accepted:
     - Native spellcheck may skip off-screen sections until they scroll
       in. WebKit re-spellchecks on visibility change, so this catches up.
     - find-in-page on Safari can be quirky with content-visibility, but
       modern WebKit handles it.
     - Selection across sections still works — ProseMirror uses logical
       positions, not DOM rect coordinates. */
  :global(.ProseMirror div.section) {
    contain: layout style;
  }
  :global(.ProseMirror div.section.leaf) {
    content-visibility: auto; /* height estimate: schema.ts, estimateHeight */
  }
  :global(.ProseMirror div.section.cv-visible) {
    content-visibility: visible;
  }
  :global(.ProseMirror div.poem),
  :global(.ProseMirror div.table),
  :global(.ProseMirror div.cite),
  :global(.ProseMirror div.epigraph),
  :global(.ProseMirror div.annotation) {
    contain: layout style;
  }
  :global(.ProseMirror p) {
    margin: 0 0 0.6em 0;
  }
  :global(.ProseMirror p.empty-line) {
    height: 1em;
  }
  :global(.ProseMirror p.subtitle) {
    font-weight: 600;
    font-size: 1.15em;
    margin-top: 1em;
  }
  :global(.ProseMirror p.text-author) {
    text-align: right;
    font-style: italic;
  }
  :global(.ProseMirror div.title) {
    margin: 1.5em 0 1em 0;
    text-align: center;
  }
  :global(.ProseMirror div.title p) {
    font-size: 1.6em;
    font-weight: 600;
  }
  :global(.ProseMirror div.section) {
    margin-bottom: 1em;
  }
  :global(.ProseMirror div.epigraph) {
    margin: 1em 0 1em 3em;
    font-style: italic;
    color: var(--fg-secondary);
  }
  :global(.ProseMirror div.cite) {
    margin: 1em 0 1em 2em;
    padding-left: 1em;
    border-left: 3px solid var(--border-input);
  }
  :global(.ProseMirror div.annotation) {
    margin: 1em 2em;
    font-size: 0.95em;
    color: var(--fg-secondary);
  }
  :global(.ProseMirror div.poem) {
    margin: 1.5em 0 1.5em 2em;
  }
  :global(.ProseMirror div.stanza) {
    margin-bottom: 1em;
  }
  :global(.ProseMirror p.v) {
    margin: 0;
  }
  :global(.ProseMirror div.table) {
    display: table;
    border-collapse: collapse;
    margin: 1em 0;
    border: 1px solid var(--border);
  }
  :global(.ProseMirror div.tr) {
    display: table-row;
  }
  :global(.ProseMirror p.td),
  :global(.ProseMirror p.th) {
    display: table-cell;
    padding: 0.3em 0.7em;
    border: 1px solid var(--border);
    margin: 0;
  }
  :global(.ProseMirror p.th) {
    background: var(--bg-chrome);
    font-weight: 600;
  }
  :global(.ProseMirror span.code),
  :global(.ProseMirror code) {
    font-family: var(--font-mono);
    font-size: 0.92em;
    background: var(--bg-chrome);
    padding: 0.1em 0.3em;
    border-radius: 3px;
  }
  :global(.ProseMirror a) {
    color: var(--fg-link);
  }
  :global(.ProseMirror div.image) {
    text-align: center;
    margin: 1em 0;
  }
  :global(.ProseMirror img) {
    max-width: 100%;
    height: auto;
  }
  /* Dangling image ref — binary id not found in current fb.Binaries.
     Show a visible placeholder so users can spot broken references
     instead of getting a blank gap. */
  :global(.ProseMirror img.missing) {
    display: inline-block;
    min-width: 80px;
    min-height: 40px;
    background: var(--warn-bg-a);
    border: 1px dashed var(--warn);
    color: var(--warn-fg);
  }
  :global(.ProseMirror img.missing::after) {
    content: attr(alt) " (missing)";
    display: block;
    padding: 0.5em;
    font-family: var(--font-mono);
    font-size: 0.8em;
    color: var(--warn-fg);
  }
  :global(.ProseMirror .outline-flash) {
    transition: background-color 0.3s ease;
    background: var(--highlight);
  }

  /* Search/replace highlighting — inactive matches get a pale wash, the
     currently-focused hit stands out with a stronger accent so the user can
     track ◀/▶ navigation at a glance. Palette vars are defined in App.svelte
     for both light and dark themes. */
  :global(.ProseMirror .search-match) {
    background: var(--search-match-bg);
    border-radius: 2px;
  }
  :global(.ProseMirror .search-match-active) {
    background: var(--search-match-active-bg);
    outline: 1px solid var(--search-match-active-border);
  }

  /* Lossless fallback placeholders for unknown FB2 elements (see schema.ts
     raw_block / raw_inline). Content is non-editable but the node can be
     selected and deleted. Dashed border + hatched background make it
     obvious these aren't real FB2 elements the editor understands. */
  :global(.ProseMirror .raw-block) {
    display: block;
    margin: 0.75em 0;
    padding: 0.35em 0.6em;
    background:
      repeating-linear-gradient(
        45deg,
        var(--warn-bg-a) 0 6px,
        var(--warn-bg-b) 6px 12px
      );
    border: 1px dashed var(--warn);
    color: var(--warn-fg);
    font-family: var(--font-mono);
    font-size: 0.85em;
    border-radius: 3px;
    user-select: none;
  }
  :global(.ProseMirror .raw-inline) {
    display: inline-block;
    margin: 0 0.1em;
    padding: 0 0.35em;
    background: var(--warn-bg-inline);
    border: 1px dashed var(--warn);
    color: var(--warn-fg);
    font-family: var(--font-mono);
    font-size: 0.85em;
    border-radius: 3px;
    user-select: none;
    vertical-align: baseline;
  }
  :global(.ProseMirror .raw-block.ProseMirror-selectednode),
  :global(.ProseMirror .raw-inline.ProseMirror-selectednode) {
    outline: 2px solid var(--warn);
    outline-offset: 1px;
  }
</style>
