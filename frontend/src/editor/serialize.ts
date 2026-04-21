/**
 * ProseMirror doc → FB2 document (doc.FictionBook on the Go side).
 *
 * Counterpart to parse.ts. The result shape mirrors Go's doc.FictionBook so it
 * can be passed to App.UpdateDocument() then App.SaveFile() verbatim.
 *
 * Reference: FBE/FBDoc.cpp::SaveToFile and the main.js GetDesc / GetBinaries
 * helpers (FBE/main.js:1525, 1539).
 */
import type { Node as PMNode } from "prosemirror-model";

type FictionBook = unknown; // to be replaced by Wails-generated type

export function pmDocToFB2(_doc: PMNode): FictionBook {
  // TODO: walk the doc, produce the FictionBook struct.
  return {};
}
