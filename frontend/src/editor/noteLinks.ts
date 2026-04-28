/**
 * Cmd/Ctrl+click on an internal link (`href` starting with `#`) jumps to the
 * node carrying the matching `id` attribute, scrolls it to the top of the
 * viewport, and remembers the originating position so the user can return.
 *
 * Cmd/Ctrl+[ pops one step back through the navigation stack — same gesture
 * as "back" in JetBrains IDEs and most code editors. The stack survives
 * across multiple successive jumps.
 *
 * Plain click is left untouched so the cursor still lands inside link text
 * for editing. The modifier convention matches browsers and IDEs (Cmd-click
 * on Mac, Ctrl-click on Linux).
 *
 * Primary use case: FB2 footnote references —
 * `<a type="note" l:href="#n1">` pointing at `<section id="n1">` inside
 * `<body name="notes">` — but the resolver works for any internal anchor
 * (epigraph IDs, paragraph IDs, etc.).
 */
import {
  Plugin,
  PluginKey,
  TextSelection,
  type Command,
  type EditorState,
  type Transaction,
} from "prosemirror-state";
import type { EditorView } from "prosemirror-view";

type StackMeta =
  | { type: "push"; from: number }
  | { type: "pop" };

const noteLinksKey = new PluginKey<number[]>("noteLinks");

export function noteLinksPlugin(): Plugin {
  return new Plugin<number[]>({
    key: noteLinksKey,
    state: {
      init: () => [],
      apply(tr: Transaction, value: number[]): number[] {
        const meta = tr.getMeta(noteLinksKey) as StackMeta | undefined;
        if (meta?.type === "push") return [...value, meta.from];
        if (meta?.type === "pop") return value.slice(0, -1);
        return value;
      },
    },
    props: {
      handleClick(view: EditorView, pos: number, event: MouseEvent): boolean {
        if (!event.metaKey && !event.ctrlKey) return false;
        const targetPos = resolveLinkTarget(view.state, pos);
        if (targetPos < 0) return false;
        jump(view, targetPos, { type: "push", from: view.state.selection.from });
        return true;
      },
    },
  });
}

/** Pop one entry off the navigation stack and jump back. Bind to Cmd/Ctrl+[. */
export const noteLinksBack: Command = (state, dispatch, view) => {
  if (!view) return false;
  const stack = noteLinksKey.getState(state);
  if (!stack || stack.length === 0) return false;
  const target = stack[stack.length - 1];
  if (!dispatch) return true;
  jump(view, target, { type: "pop" });
  return true;
};

/** Resolve the click position to an internal-anchor target, or -1. */
function resolveLinkTarget(state: EditorState, pos: number): number {
  const node = state.doc.nodeAt(pos);
  if (!node) return -1;
  const link = node.marks.find((m) => m.type.name === "link");
  if (!link) return -1;
  const href: string = (link.attrs.href as string | undefined) ?? "";
  if (!href.startsWith("#")) return -1;
  const id = href.slice(1);
  if (!id) return -1;

  let targetPos = -1;
  state.doc.descendants((n, p) => {
    if (targetPos >= 0) return false;
    if (n.attrs && (n.attrs as Record<string, unknown>).id === id) {
      targetPos = p;
      return false;
    }
    return true;
  });
  return targetPos;
}

/** Move selection to `targetPos`, push/pop stack via meta, and scroll the
 *  target DOM element to the top of the viewport. PM's tr.scrollIntoView()
 *  alone uses minimal scrolling — that lands long target nodes (entire
 *  notes) on the LAST visible line, hiding the body. Calling scrollIntoView
 *  on the DOM directly with block:"start" puts the node header at the top. */
function jump(view: EditorView, targetPos: number, meta: StackMeta) {
  const $pos = view.state.doc.resolve(targetPos + 1);
  const sel = TextSelection.near($pos);
  view.dispatch(view.state.tr.setSelection(sel).setMeta(noteLinksKey, meta));
  // Schedule after the dispatch so PM has updated DOM positions.
  // Prefer nodeDOM (gives the block element when target is a node-boundary
  // like a section), fall back to domAtPos which works for ANY position
  // including text-positions inside a paragraph (the "back" jump case).
  queueMicrotask(() => {
    let el: HTMLElement | null = null;
    const blockDom = view.nodeDOM(targetPos);
    if (blockDom instanceof HTMLElement) {
      el = blockDom;
    } else {
      const dom = view.domAtPos(sel.from);
      if (dom?.node) {
        el = dom.node.nodeType === Node.ELEMENT_NODE
          ? (dom.node as HTMLElement)
          : dom.node.parentElement;
      }
    }
    el?.scrollIntoView({ block: "start" });
  });
}
