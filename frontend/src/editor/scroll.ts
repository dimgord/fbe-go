import type { EditorView } from "prosemirror-view";

/** The ancestor that owns the editor's scrollbar (`<section>` in App, not `view.dom`). */
export function scrollContainer(view: EditorView): HTMLElement | null {
  let el: HTMLElement | null = view.dom as HTMLElement;
  while (el && el.scrollHeight <= el.clientHeight) el = el.parentElement;
  return el;
}

/**
 * Scroll `pos` into the editor's viewport and keep correcting for a few frames.
 * Sections use `content-visibility: auto` with a 600px placeholder, so a one-shot
 * jump lands off-target: sections passed on the way render at their real height
 * and shift the text. "top" pins the position `margin` px below the top edge;
 * "nearest" only scrolls when it sits within `margin` px of an edge or outside.
 */
export function scrollPosIntoView(
  view: EditorView,
  pos: number,
  mode: "top" | "nearest" = "nearest",
  margin = 80,
): void {
  const el = scrollContainer(view);
  if (!el) return;
  let frames = 0;
  const step = () => {
    if ((view as unknown as { docView: unknown }).docView === null) return; // view destroyed
    let coords: { top: number; bottom: number };
    try {
      coords = view.coordsAtPos(Math.min(pos, view.state.doc.content.size));
    } catch {
      return;
    }
    const rect = el.getBoundingClientRect();
    let delta = 0;
    if (mode === "top") delta = coords.top - rect.top - margin;
    else if (coords.top < rect.top + margin) delta = coords.top - rect.top - margin;
    else if (coords.bottom > rect.bottom - margin) delta = coords.bottom - rect.bottom + margin;
    if (Math.abs(delta) < 2) return;
    el.scrollTop += delta;
    if (++frames < 8) requestAnimationFrame(step);
  };
  step();
}
