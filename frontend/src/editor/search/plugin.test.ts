import { describe, expect, it } from "vitest";
import { EditorState, TextSelection } from "prosemirror-state";
import { Schema } from "prosemirror-model";
import type { EditorView } from "prosemirror-view";
import { findNext, findPrev, searchPlugin, searchPluginKey, setSearch } from "./plugin";

const schema = new Schema({
  nodes: { doc: { content: "p+" }, p: { content: "text*" }, text: {} },
});

/** Just enough of an EditorView for the commands: state + dispatch, nothing scrollable. */
function fakeView(text: string): EditorView {
  const doc = schema.node("doc", null, [schema.node("p", null, [schema.text(text)])]);
  const view = {
    state: EditorState.create({ doc, plugins: [searchPlugin()] }),
    isDestroyed: false,
    dom: { scrollHeight: 0, clientHeight: 0, parentElement: null },
    dispatch(tr: Parameters<EditorState["apply"]>[0]) {
      view.state = view.state.apply(tr);
    },
  };
  return view as unknown as EditorView;
}

const active = (v: EditorView) => searchPluginKey.getState(v.state)!.active;
const FLAGS = { caseSensitive: false, wholeWord: false, regex: false };

describe("search navigation", () => {
  it("visits the highlighted hit first, then advances on every call", () => {
    const v = fakeView("cat cat cat");
    setSearch(v, "cat", FLAGS);
    expect(active(v)).toBe(0);
    const seen = [1, 2, 3, 4].map(() => (findNext(v), active(v)));
    expect(seen).toEqual([0, 1, 2, 0]);
    const sel = v.state.selection;
    expect(sel instanceof TextSelection && [sel.from, sel.to]).toEqual([1, 4]);
  });

  it("goes backwards and wraps", () => {
    const v = fakeView("cat cat cat");
    setSearch(v, "cat", FLAGS);
    const seen = [1, 2, 3].map(() => (findPrev(v), active(v)));
    expect(seen).toEqual([0, 2, 1]);
  });

  it("reports nothing to do without matches", () => {
    const v = fakeView("dog");
    setSearch(v, "cat", FLAGS);
    expect(findNext(v)).toBe(false);
  });
});
