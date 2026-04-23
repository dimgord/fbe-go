<script lang="ts">
  import { onDestroy } from "svelte";
  import { EditorState, type Transaction } from "prosemirror-state";
  import { EditorView } from "prosemirror-view";
  import { history, redo, undo } from "prosemirror-history";
  import { keymap } from "prosemirror-keymap";
  import { baseKeymap } from "prosemirror-commands";
  import { Node as PMNode } from "prosemirror-model";
  import { fb2Schema } from "./schema";
  import { fb2ToPMDoc } from "./parse";
  import { pmDocToFB2 } from "./serialize";
  import { cleanPastedHTML, cleanPastedText } from "./paste";
  import { searchPlugin, findNext as searchFindNext, findPrev as searchFindPrev } from "./search/plugin";
  import {
    toggleStrong, toggleEmphasis, toggleStrikethrough,
    toggleSub, toggleSup, toggleCode, toggleLink,
    styleNormal, styleSubtitle, styleTextAuthor,
    insertEmptyLine,
    cloneContainer, removeOuterContainer,
    addTitle, addEpigraph, addAnnotation, addTextAuthor,
    insertCite, insertPoem, insertTableCmd,
    mergeContainers,
  } from "./commands";
  import { HOTKEY_ACTIONS, toPMKey } from "../settings/hotkeys";
  import type { Command } from "prosemirror-state";
  import TableDialog from "./TableDialog.svelte";
  import type { FictionBook, Binary } from "../fb2/types";
  import type { NodeView } from "prosemirror-view";

  export let fb: FictionBook | null = null;

  /** User-configurable hotkey map, loaded from settings.Hotkeys. Passing a
      new object identity triggers a reconfigure — history / document survive,
      but the keymap plugin is swapped. Empty / missing actions are simply
      not bound. */
  export let hotkeys: Record<string, string> = {};

  /** Export the EditorView so the toolbar in App.svelte can dispatch commands. */
  export let view: EditorView | undefined = undefined;

  /** Inferred from the document's title-info <lang> for native spellcheck routing. */
  $: lang = fb?.Description?.TitleInfo?.Lang || "en";

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
    const coords = view.coordsAtPos(pos);
    let el: HTMLElement | null = view.dom as HTMLElement;
    while (el && el.scrollHeight <= el.clientHeight) el = el.parentElement;
    if (el) {
      const elRect = el.getBoundingClientRect();
      el.scrollTop += coords.top - elRect.top - 12;
    }
    // Brief highlight on the nearest enclosing element (domAtPos may return a Text node).
    const dom = view.domAtPos(pos);
    const flashTarget =
      dom.node instanceof HTMLElement ? dom.node : dom.node.parentElement;
    if (flashTarget) {
      flashTarget.classList.add("outline-flash");
      setTimeout(() => flashTarget.classList.remove("outline-flash"), 700);
    }
  }

  /** Walk the doc by outline path: [bodyIdx, sectionIdx, sub, …]. Returns the position of that node. */
  function findNodePos(doc: PMNode, path: number[]): number | null {
    if (path.length === 0) return null;
    // Step 0: the bodyIdx-th body child of doc.
    let cursor = doc;
    let absPos = 0;
    let remaining = path.slice();
    // Descend through body → section → sub…
    // First, pick body.
    const bodyIdx = remaining.shift()!;
    let i = 0;
    for (let c = 0; c < cursor.childCount; c++) {
      const child = cursor.child(c);
      if (child.type.name === "body") {
        if (i === bodyIdx) {
          absPos += 1; // enter body
          cursor = child;
          break;
        }
        i++;
      }
      absPos += child.nodeSize;
    }
    // Then descend into sections.
    while (remaining.length > 0) {
      const want = remaining.shift()!;
      let sIdx = 0;
      let found = false;
      for (let c = 0; c < cursor.childCount; c++) {
        const child = cursor.child(c);
        if (child.type.name === "section") {
          if (sIdx === want) {
            absPos += 1; // enter section
            cursor = child;
            found = true;
            break;
          }
          sIdx++;
        }
        absPos += child.nodeSize;
      }
      if (!found) return null;
    }
    return absPos;
  }

  let container: HTMLDivElement;

  // Image rendering: schema.ts' toDOM generates `fb2://binary${href}` which
  // relies on a custom asset-server handler we never wired up — images would
  // otherwise render as broken icons. We resolve href → `data:` URL on the
  // fly via NodeViews, closed over a shared binariesRef that we refresh
  // whenever fb.Binaries identity changes (file open, upload, rename,
  // delete). `refreshImageViews()` iterates every active NodeView and
  // re-resolves its src — cheap, since each view only reassigns img.src.
  //
  // Serialization is unaffected: pmDocToFB2 only reads node.attrs.href.
  let binariesRef: { current: Binary[] } = { current: [] };
  $: binariesRef.current = fb?.Binaries ?? [];
  const imageViews = new Set<ImageNodeView>();

  interface ImageNodeView extends NodeView {
    refresh(): void;
  }

  function resolveBinary(href: string | undefined): string {
    if (!href) return "";
    const id = href.replace(/^#/, "");
    const bin = binariesRef.current.find(b => b.ID === id);
    return bin ? `data:${bin.ContentType};base64,${bin.Data}` : "";
  }

  function refreshImageViews(): void {
    for (const v of imageViews) v.refresh();
  }

  // Re-resolve every image whenever the binaries array changes identity —
  // new upload / delete / rename cascades here and images pick up fresh
  // data: URLs without needing a PM remount (which would blow undo history).
  $: void fb?.Binaries, refreshImageViews();

  function createImageView(node: PMNode): ImageNodeView {
    const blockLevel = node.type.name === "image_block";
    const wrap = document.createElement(blockLevel ? "div" : "span");
    wrap.className = "image";
    if (node.attrs.href) wrap.setAttribute("data-href", node.attrs.href);
    if (node.attrs.title) wrap.title = node.attrs.title;

    const img = document.createElement("img");
    img.alt = node.attrs.alt || "";
    const resolved = resolveBinary(node.attrs.href);
    if (resolved) {
      img.src = resolved;
    } else {
      img.classList.add("missing");
    }
    wrap.appendChild(img);

    // Keep a mutable reference to the current node so refresh() uses the
    // latest href (after setNodeMarkup) without needing another update()
    // round-trip.
    let current = node;
    const nv: ImageNodeView = {
      dom: wrap,
      update(next) {
        if (next.type !== current.type) return false;
        current = next;
        wrap.setAttribute("data-href", next.attrs.href || "");
        wrap.title = next.attrs.title || "";
        img.alt = next.attrs.alt || "";
        const src = resolveBinary(next.attrs.href);
        img.src = src;
        img.classList.toggle("missing", !src);
        return true;
      },
      refresh() {
        const src = resolveBinary(current.attrs.href);
        img.src = src;
        img.classList.toggle("missing", !src);
      },
      destroy() {
        imageViews.delete(nv);
      },
    };
    imageViews.add(nv);
    return nv;
  }

  /** Map of action id → PM Command. Non-PM actions (Save, Find, dialogs) are
      handled at the App level and not listed here. Keep in sync with the
      `editor: true` rows of HOTKEY_ACTIONS. */
  const EDITOR_COMMANDS: Record<string, Command> = {
    ToggleStrong: toggleStrong,
    ToggleEmphasis: toggleEmphasis,
    ToggleStrikethrough: toggleStrikethrough,
    ToggleSub: toggleSub,
    ToggleSup: toggleSup,
    ToggleCode: toggleCode,
    StyleNormal: styleNormal,
    StyleSubtitle: styleSubtitle,
    StyleTextAuthor: styleTextAuthor,
    InsertEmptyLine: insertEmptyLine,
    CloneContainer: cloneContainer,
    RemoveOuterContainer: removeOuterContainer,
    AddTitle: addTitle,
    AddEpigraph: addEpigraph,
    AddAnnotation: addAnnotation,
    AddTextAuthor: addTextAuthor,
    InsertCite: insertCite,
    InsertPoem: insertPoem,
    MergeContainers: mergeContainers,
    FindNext: (_s, _d, v) => (v ? searchFindNext(v) : false),
    FindPrev: (_s, _d, v) => (v ? searchFindPrev(v) : false),
  };

  /** Build a PM keymap record from the current hotkeys map. Always includes
      the undo/redo stack — those are hardcoded because rebinding them
      invariably breaks in ways users don't expect. Also keeps F3 / Shift-F3
      aliases for FindNext / FindPrev (legacy Windows affordance; independent
      of the user's Ctrl-G remap). */
  function buildKeymap(hk: Record<string, string>): Record<string, Command> {
    const bindings: Record<string, Command> = {
      "Mod-z": undo,
      "Mod-y": redo,
      "Mod-Shift-z": redo,
    };
    for (const action of HOTKEY_ACTIONS) {
      if (!action.editor) continue;
      const raw = hk[action.id];
      if (!raw) continue;
      const pmKey = toPMKey(raw);
      const cmd = EDITOR_COMMANDS[action.id];
      if (!pmKey || !cmd) continue;
      bindings[pmKey] = cmd;
    }
    // F3 / Shift-F3 stay hardcoded as Windows-convention aliases — users can
    // still rebind FindNext to something else without losing these.
    bindings["F3"] = (_s, _d, v) => (v ? searchFindNext(v) : false);
    bindings["Shift-F3"] = (_s, _d, v) => (v ? searchFindPrev(v) : false);
    return bindings;
  }

  function mount(doc: PMNode) {
    view?.destroy();
    const state = EditorState.create({
      schema: fb2Schema,
      doc,
      plugins: [
        history(),
        keymap(buildKeymap(hotkeys)),
        keymap(baseKeymap),
        searchPlugin(),
      ],
    });
    view = new EditorView(container, {
      state,
      attributes: { spellcheck: "true", lang },
      transformPastedHTML: cleanPastedHTML,
      transformPastedText: cleanPastedText,
      nodeViews: {
        image_block: (node) => createImageView(node),
        image_inline: (node) => createImageView(node),
      },
    });
    lastHotkeys = hotkeys;
  }

  /** Rebuild just the user-facing keymap without re-mounting — preserves
      document state, selection, and undo history via PluginKey continuity
      in prosemirror-history (historyKey) and our searchPlugin (searchPluginKey).
      Called when the parent writes a new `hotkeys` object (Settings → Apply). */
  let lastHotkeys: Record<string, string> | null = null;
  $: if (view && hotkeys !== lastHotkeys) {
    lastHotkeys = hotkeys;
    view.updateState(view.state.reconfigure({
      plugins: [
        history(),
        keymap(buildKeymap(hotkeys)),
        keymap(baseKeymap),
        searchPlugin(),
      ],
    }));
  }

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
      const msg = (err as Error).message || String(err);
      const N = fb2Schema.nodes;
      const body = N.body.create(null, [
        N.section.create(null, [
          N.title.create(null, [
            N.paragraph.create(null, fb2Schema.text("Could not render this document")),
          ]),
          N.paragraph.create(null, fb2Schema.text(msg)),
          N.paragraph.create(null, fb2Schema.text("The raw FB2 is still loaded — Save As will write it back unchanged.")),
        ]),
      ]);
      return N.doc.create(null, [body]);
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

  export function exec(cmd: (state: EditorState, dispatch?: (tr: Transaction) => void) => boolean): void {
    if (!view) return;
    cmd(view.state, view.dispatch);
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

  // Re-export commands so App.svelte can import-and-bind from the same place.
  export {
    toggleStrong, toggleEmphasis, toggleStrikethrough,
    toggleSub, toggleSup, toggleCode,
    styleNormal, styleSubtitle, styleTextAuthor,
    insertEmptyLine,
    cloneContainer, removeOuterContainer,
    addTitle, addEpigraph, addAnnotation, addTextAuthor,
    insertCite, insertPoem,
    mergeContainers,
    undo, redo,
  };
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
    font-family: "SF Mono", Menlo, monospace;
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
    font-family: "SF Mono", Menlo, Consolas, monospace;
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
    font-family: "SF Mono", Menlo, Consolas, monospace;
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
    font-family: "SF Mono", Menlo, Consolas, monospace;
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
