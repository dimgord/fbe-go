import { Plugin, PluginKey } from "prosemirror-state";
import { Decoration, DecorationSet, type EditorView } from "prosemirror-view";
import type { Node as PMNode } from "prosemirror-model";

const forceKey = new PluginKey<DecorationSet>("fbe-scroll-force");

/**
 * Holds the `cv-visible` class on the sections around a scroll target. A section outside the
 * viewport (`content-visibility: auto`) occupies a 600px placeholder while its children still
 * measure at full size, so coordinates inside it point far past the end of the document. A
 * decoration rather than a DOM class: ProseMirror re-parses nodes whose attributes change under it.
 */
export function scrollForcePlugin(): Plugin<DecorationSet> {
  return new Plugin<DecorationSet>({
    key: forceKey,
    state: {
      init: () => DecorationSet.empty,
      apply(tr, prev) {
        const next = tr.getMeta(forceKey) as DecorationSet | undefined;
        return next ?? prev.map(tr.mapping, tr.doc);
      },
    },
    props: {
      decorations: (state) => forceKey.getState(state),
    },
  });
}

function sectionsAround(doc: PMNode, pos: number): DecorationSet {
  const $pos = doc.resolve(pos);
  const decos: Decoration[] = [];
  for (let d = 1; d <= $pos.depth; d++) {
    if ($pos.node(d).type.name === "section") {
      decos.push(Decoration.node($pos.before(d), $pos.after(d), { class: "cv-visible" }));
    }
  }
  return DecorationSet.create(doc, decos);
}

/** The ancestor that owns the editor's scrollbar (`<section>` in App, not `view.dom`). */
export function scrollContainer(view: EditorView): HTMLElement | null {
  let el: HTMLElement | null = view.dom as HTMLElement;
  while (el && el.scrollHeight <= el.clientHeight) el = el.parentElement;
  return el;
}

function isAlive(view: EditorView): boolean {
  return (view as unknown as { docView: unknown }).docView !== null;
}

let currentRun = 0;

/**
 * Scroll `pos` into the editor's viewport: render the sections around it, scroll, then re-measure
 * for a few frames in case layout still moves. "top" pins the position `margin` px below the top
 * edge; "nearest" only scrolls when it sits within `margin` px of an edge or outside.
 */
export function scrollPosIntoView(
  view: EditorView,
  pos: number,
  mode: "top" | "nearest" = "nearest",
  margin = 80,
): void {
  const el = scrollContainer(view);
  if (!el) return;
  const run = ++currentRun;
  const force = forceKey.get(view.state) !== undefined;
  if (force) view.dispatch(view.state.tr.setMeta(forceKey, sectionsAround(view.state.doc, pos)));

  const finish = () => {
    if (!force) return;
    // Two frames: the section must have painted as on-screen before `auto` takes over again.
    requestAnimationFrame(() => requestAnimationFrame(() => {
      if (run === currentRun && isAlive(view)) {
        view.dispatch(view.state.tr.setMeta(forceKey, DecorationSet.empty));
      }
    }));
  };

  let frames = 0;
  const step = () => {
    if (run !== currentRun || !isAlive(view)) return;
    let coords: { top: number; bottom: number };
    try {
      coords = view.coordsAtPos(Math.min(pos, view.state.doc.content.size));
    } catch {
      finish();
      return;
    }
    const rect = el.getBoundingClientRect();
    let delta = 0;
    if (mode === "top") delta = coords.top - rect.top - margin;
    else if (coords.top < rect.top + margin) delta = coords.top - rect.top - margin;
    else if (coords.bottom > rect.bottom - margin) delta = coords.bottom - rect.bottom + margin;
    if (Math.abs(delta) < 2 || frames >= 8) {
      finish();
      return;
    }
    frames++;
    el.scrollTop += delta;
    requestAnimationFrame(step);
  };
  step();
}
