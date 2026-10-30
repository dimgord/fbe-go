/**
 * Body edits are tracked through PM's undo stack: dirty while the depth differs from a baseline.
 * The baseline sits in plugin state rather than in a closure set by the plugin view, because a
 * view can be recreated while the state lives on.
 */
import { undoDepth } from "prosemirror-history";
import { Plugin, PluginKey, type EditorState, type Transaction } from "prosemirror-state";
import type { EditorView } from "prosemirror-view";

const key = new PluginKey<number>("fbe-dirty");

export function dirtyPlugin(onChange: (dirty: boolean) => void): Plugin<number> {
  let last = false;
  return new Plugin<number>({
    key,
    state: {
      init: (_config, state) => undoDepth(state),
      apply: (tr: Transaction, baseline: number, _old: EditorState, next: EditorState) =>
        tr.getMeta(key) ? undoDepth(next) : baseline,
    },
    view: () => ({
      update(view, prev) {
        const baseline = key.getState(view.state);
        if (view.state.doc === prev.doc && baseline === key.getState(prev)) return;
        const dirty = undoDepth(view.state) !== baseline;
        if (dirty !== last) {
          last = dirty;
          onChange(dirty);
        }
      },
    }),
  });
}

/** Make the current state the clean one, so undoing back to it reads as clean. */
export function markClean(view: EditorView): void {
  view.dispatch(view.state.tr.setMeta(key, true));
}
