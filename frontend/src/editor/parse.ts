/**
 * FB2 document (doc.FictionBook from the Go side) → ProseMirror doc.
 *
 * Counterpart to serialize.ts. See docs/ARCHITECTURE.md for the data-flow diagram.
 */
import { fb2Schema } from "./schema";
import type { Node as PMNode } from "prosemirror-model";

// Type imported from Wails bindings once generated; mirror of doc.FictionBook.
type FictionBook = unknown;

/** Convert a parsed FB2 document (from Go) into a ProseMirror doc. */
export function fb2ToPMDoc(_fb: FictionBook): PMNode {
  // TODO: walk fb.bodies / sections / paragraphs and build nodes via fb2Schema.nodes.*.
  return fb2Schema.topNodeType.createAndFill()!;
}
