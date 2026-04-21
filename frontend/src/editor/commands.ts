/**
 * FB2 editing commands for ProseMirror.
 *
 * Each command follows the ProseMirror convention:
 *   (state, dispatch?) => boolean
 * When `dispatch` is omitted the command only reports availability.
 *
 * See docs/OPERATIONS.md for the FBE → command mapping.
 */
import type { EditorState, Transaction } from "prosemirror-state";
import type { NodeType, Attrs } from "prosemirror-model";
import {
  toggleMark as pmToggleMark,
  setBlockType as pmSetBlockType,
} from "prosemirror-commands";
import { fb2Schema } from "./schema";

export type Command = (
  state: EditorState,
  dispatch?: (tr: Transaction) => void,
) => boolean;

// ── Inline mark toggles ─────────────────────────────────────────────────────

const M = fb2Schema.marks;

/** Toggle <strong> on the selection. Keyboard: Mod-b. */
export const toggleStrong: Command        = pmToggleMark(M.strong);

/** Toggle <emphasis> on the selection. Keyboard: Mod-i. */
export const toggleEmphasis: Command      = pmToggleMark(M.emphasis);

/** Toggle <strikethrough>. */
export const toggleStrikethrough: Command = pmToggleMark(M.strikethrough);

/** Toggle <sub>. */
export const toggleSub: Command           = pmToggleMark(M.sub);

/** Toggle <sup>. */
export const toggleSup: Command           = pmToggleMark(M.sup);

/** Toggle inline <code>. */
export const toggleCode: Command          = pmToggleMark(M.code);

/** Toggle a link with the given href. Pass empty href to remove. */
export function toggleLink(href: string): Command {
  if (!href) {
    // Remove link mark from the range.
    return (state, dispatch) => {
      const { from, to } = state.selection;
      if (from === to) return false;
      if (dispatch) {
        const tr = state.tr.removeMark(from, to, M.link);
        dispatch(tr);
      }
      return true;
    };
  }
  return pmToggleMark(M.link, { href, type: "" });
}

/** Apply a named <style name="…"> to the selection. */
export function applyStyleMark(name: string): Command {
  return pmToggleMark(M.style, { name });
}

// ── Paragraph style (block-type) commands ───────────────────────────────────

const N = fb2Schema.nodes;

/** Turn the current paragraph back into a plain <p>. */
export const styleNormal: Command       = pmSetBlockType(N.paragraph);

/** Convert the current block into a <subtitle>. */
export const styleSubtitle: Command     = pmSetBlockType(N.subtitle);

/** Convert the current block into a <text-author> (typically at end of poem/cite/epigraph). */
export const styleTextAuthor: Command   = pmSetBlockType(N.text_author);

/** Convert the current block into an <empty-line>. */
export const insertEmptyLine: Command = (state, dispatch) => {
  if (!N.empty_line) return false;
  if (dispatch) {
    const tr = state.tr.replaceSelectionWith(N.empty_line.create());
    dispatch(tr);
  }
  return true;
};

// ── Helpers that return the block/mark status under the cursor ──────────────

/** Is the given mark active at the current selection head? */
export function isMarkActive(state: EditorState, markName: keyof typeof M): boolean {
  const mark = M[markName];
  const { from, $from, to, empty } = state.selection;
  if (empty) return !!mark.isInSet(state.storedMarks ?? $from.marks());
  return state.doc.rangeHasMark(from, to, mark);
}

/** Is the block at the cursor of the given type? */
export function isBlockActive(state: EditorState, nodeType: NodeType, attrs?: Attrs): boolean {
  const { $from, to } = state.selection;
  if (to > $from.end()) return false;
  return $from.parent.hasMarkup(nodeType, attrs ?? null);
}

// ── Structural stubs kept for Phase 3 ───────────────────────────────────────
// Not yet implemented — see OPERATIONS.md for reference FBE code paths.

export const insertPoem:   Command = () => false;  // FBEview.cpp:903
export const insertCite:   Command = () => false;  // FBEview.cpp:1048
export const addEpigraph:  Command = () => false;  // main.js:2050
export const addAnnotation: Command = () => false; // main.js:2142
export const cloneContainer: Command = () => false;
export const mergeContainers: Command = () => false;
export const removeOuterContainer: Command = () => false;
