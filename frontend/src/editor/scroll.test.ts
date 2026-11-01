import { describe, expect, it } from "vitest";
import { EditorState, TextSelection } from "prosemirror-state";
import { DecorationSet } from "prosemirror-view";
import { fb2ToPMDoc } from "./parse";
import { fb2Schema } from "./schema";
import { blockMarkPlugin, blockMarkTr } from "./scroll";
import { SAMPLE_BOOK } from "../fb2/sample";

describe("block mark", () => {
  const plugin = blockMarkPlugin();
  const state = EditorState.create({ schema: fb2Schema, doc: fb2ToPMDoc(SAMPLE_BOOK), plugins: [plugin] });
  const marked = (s: EditorState) => (plugin.getState(s) as DecorationSet).find().length;

  it("marks one block and clears", () => {
    const on = state.apply(blockMarkTr(state, 1 + firstBlock(state)));
    expect(marked(on)).toBe(1);
    expect(marked(on.apply(blockMarkTr(on, null)))).toBe(0);
  });

  it("clears when a caret is placed in the text", () => {
    const on = state.apply(blockMarkTr(state, 1 + firstBlock(state)));
    const moved = on.apply(on.tr.setSelection(TextSelection.atEnd(on.doc)));
    expect(marked(moved)).toBe(0);
  });
});

function firstBlock(state: EditorState): number {
  let found = -1;
  state.doc.descendants((n, pos) => {
    if (found < 0 && n.isTextblock) found = pos;
    return found < 0;
  });
  return found - 1;
}
