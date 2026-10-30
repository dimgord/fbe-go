import { describe, expect, it } from "vitest";
import { history, undo } from "prosemirror-history";
import { EditorState } from "prosemirror-state";
import type { EditorView } from "prosemirror-view";
import { fb2Schema } from "./schema";
import { dirtyPlugin, markClean } from "./dirtyTracker";

/** A view stand-in that runs plugin view updates the way EditorView does. */
function harness() {
  const events: boolean[] = [];
  const plugin = dirtyPlugin((d) => events.push(d));
  const state = EditorState.create({ schema: fb2Schema, plugins: [history(), plugin] });
  const view = {
    state,
    dispatch(tr: Parameters<EditorState["apply"]>[0]) {
      const prev = view.state;
      view.state = prev.apply(tr);
      pluginView.update!(view as unknown as EditorView, prev);
    },
  };
  const pluginView = plugin.spec.view!(view as unknown as EditorView);
  const type = () => view.dispatch(view.state.tr.insertText("x", 3));
  const back = () => undo(view.state, view.dispatch);
  return { view: view as unknown as EditorView, events, type, back };
}

describe("dirtyPlugin", () => {
  it("is dirty after an edit and clean after undoing it", () => {
    const h = harness();
    h.type();
    expect(h.events).toEqual([true]);
    h.back();
    expect(h.events).toEqual([true, false]);
  });

  it("treats the saved point as clean, and undoing past it as dirty", () => {
    const h = harness();
    h.type();
    markClean(h.view);
    expect(h.events).toEqual([true, false]);
    h.back();
    expect(h.events).toEqual([true, false, true]);
  });
});
