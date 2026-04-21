/**
 * FB2 editing commands for ProseMirror.
 *
 * Every entry here replaces a C++ method in FBEview.cpp or a JS function in
 * FBE/main.js. See docs/OPERATIONS.md for the full mapping.
 *
 * Contract: each command has signature (state, dispatch?) => boolean.
 * - When `dispatch` is omitted, the command reports availability (FBE's `fCheck=true`).
 * - When `dispatch` is provided, the command performs its transaction.
 */
import type { EditorState, Transaction } from "prosemirror-state";
import type { NodeType } from "prosemirror-model";
import { fb2Schema } from "./schema";

type Command = (state: EditorState, dispatch?: (tr: Transaction) => void) => boolean;

// --- Containers ---

/** Replaces FBEview::InsertPoem (FBE/FBEview.cpp:903). Wraps selected paragraphs in <poem><stanza>...</stanza></poem>. */
export const insertPoem: Command = (_state, _dispatch) => {
  // TODO
  return false;
};

/** Replaces FBEview::InsertCite (FBE/FBEview.cpp:1048). */
export const insertCite: Command = () => false;

/** Replaces FBEview::InsertTable (FBE/FBEview.cpp:3556). */
export const insertTable = (rows: number, cols: number, header: boolean): Command => () => false;

/** Replaces main.js::AddEpigraph (FBE/main.js:2050). */
export const addEpigraph: Command = () => false;

/** Replaces main.js::AddAnnotation (FBE/main.js:2142). */
export const addAnnotation: Command = () => false;

/** Replaces main.js::AddTA (FBE/main.js:2168). Adds text-author to the enclosing poem/cite/epigraph. */
export const addTextAuthor: Command = () => false;

/** Replaces main.js::AddTitle (FBE/main.js:1766). */
export const addTitle: Command = () => false;

/** Replaces main.js::AddBody (FBE/main.js:1894). */
export const addBody: Command = () => false;

/** Replaces main.js::AddImage (FBE/main.js:2030). Block image at section start. */
export const addImage: Command = () => false;

// --- Insert at cursor ---

/** Replaces main.js::InsImage (FBE/main.js:1971). */
export const insertImage = (_href: string): Command => () => false;

/** Replaces main.js::InsInlineImage (FBE/main.js:2001). */
export const insertInlineImage = (_href: string): Command => () => false;

// --- Structural manipulation ---

/** Replaces main.js::CloneContainer (FBE/main.js:1940). Duplicates section/poem/stanza/cite/epigraph. */
export const cloneContainer: Command = () => false;

/** Replaces main.js::MergeContainers (FBE/main.js:2216). Joins adjacent same-type sections/stanzas/cites. */
export const mergeContainers: Command = () => false;

/** Replaces main.js::RemoveOuterContainer (FBE/main.js:2357). Pops child sections up a level. */
export const removeOuterContainer: Command = () => false;

// --- Paragraph styles ---

/** Replaces main.js::StyleNormal. Removes text-author/subtitle/code classes. */
export const styleNormal: Command = () => false;

/** Replaces main.js::StyleSubtitle (FBE/main.js:1699). */
export const styleSubtitle: Command = () => false;

/** Replaces main.js::StyleTextAuthor (FBE/main.js:1693). */
export const styleTextAuthor: Command = () => false;

/** Replaces main.js::StyleCode (FBE/main.js:1705). */
export const styleCode: Command = () => false;

// --- Inline marks ---

/** Toggle an inline mark by schema name. */
export function toggleMark(name: keyof typeof fb2Schema.marks): Command {
  // TODO: thin wrapper around prosemirror-commands.toggleMark with availability check.
  return () => false;
}

// --- Splits ---

/** Replaces FBEview split command (ID_EDIT_SPLIT). Breaks section at caret. */
export const splitSection: Command = () => false;

// --- Notes / footnotes ---

/** Inline note reference (<a type="note">). Creates a new body with name="notes". */
export const insertFootnote = (_id: string): Command => () => false;

// Utility — unused parameter suppression to keep TSC happy while stubs exist.
function _unused<T>(x: T): T { return x; }
_unused<NodeType>(fb2Schema.nodes.paragraph);
