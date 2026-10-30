/**
 * Where a document position sits in the book, in the outline's terms: the path that
 * tree/outline.ts assigns to the enclosing section, and the titles above it.
 */
import type { Node as PMNode } from "prosemirror-model";

export interface BookPosition {
  /** Outline path of the innermost body or section around the position. */
  path: number[];
  /** Titles from the outermost titled level down to the innermost; untitled levels are skipped. */
  trail: string[];
  /** How far through the document, 0–100. */
  percent: number;
}

/** Index of `parent.child(index)` among its siblings of the same type. */
function indexAmongType(parent: PMNode, index: number): number {
  const type = parent.child(index).type;
  let n = 0;
  for (let i = 0; i < index; i++) if (parent.child(i).type === type) n++;
  return n;
}

function titleOf(node: PMNode): string {
  let text = "";
  node.forEach((child) => {
    if (!text && child.type.name === "title") {
      text = child.textBetween(0, child.content.size, " ").trim(); // a title may be several paragraphs
    }
  });
  return text;
}

export function positionAt(doc: PMNode, pos: number): BookPosition {
  const $pos = doc.resolve(Math.min(pos, doc.content.size));
  const path: number[] = [];
  const trail: string[] = [];
  for (let d = 1; d <= $pos.depth; d++) {
    const node = $pos.node(d);
    if (node.type.name !== "body" && node.type.name !== "section") break;
    path.push(indexAmongType($pos.node(d - 1), $pos.index(d - 1)));
    const title = titleOf(node);
    if (title) trail.push(title);
  }
  const size = doc.content.size;
  return { path, trail, percent: size > 0 ? Math.round((pos / size) * 100) : 0 };
}
