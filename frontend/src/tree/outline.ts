/**
 * The outline, built from the live ProseMirror document so that it can never disagree with
 * what is being edited.
 *
 * Each node has:
 *  - label: displayed text (the section's <title>, or a placeholder)
 *  - path:  index array ([body, section, ...subsections]) of the matching PM node
 *  - kind:  which FB2 element the node corresponds to
 *  - children: nested outline nodes
 */
import type { Node as PMNode } from "prosemirror-model";

export type OutlineKind = "body" | "section";

export interface OutlineNode {
  label: string;
  kind: OutlineKind;
  path: number[];
  children: OutlineNode[];
}

function titleText(node: PMNode): string {
  for (let i = 0; i < node.childCount; i++) {
    const child = node.child(i);
    if (child.type.name === "title") return child.textBetween(0, child.content.size, " ").trim();
  }
  return "";
}

function sections(parent: PMNode, path: number[]): OutlineNode[] {
  const out: OutlineNode[] = [];
  parent.forEach((child) => {
    if (child.type.name !== "section") return;
    const childPath = [...path, out.length];
    out.push({
      kind: "section",
      label: titleText(child) || `section ${out.length + 1}`,
      path: childPath,
      children: sections(child, childPath),
    });
  });
  return out;
}

export function buildOutline(doc: PMNode | null): OutlineNode[] {
  if (!doc) return [];
  const out: OutlineNode[] = [];
  doc.forEach((body) => {
    if (body.type.name !== "body") return;
    const path = [out.length];
    const name = body.attrs.name as string;
    out.push({
      kind: "body",
      label: titleText(body) || (name ? `body[${name}]` : `body ${out.length + 1}`),
      path,
      children: sections(body, path),
    });
  });
  return out;
}

/** The position of the body or section at an outline path, counting same-type siblings. */
export function findNodePos(doc: PMNode, path: number[]): number | null {
  let cursor = doc;
  let pos = 0;
  for (let depth = 0; depth < path.length; depth++) {
    const type = depth === 0 ? "body" : "section";
    let seen = 0;
    let found = false;
    for (let c = 0; c < cursor.childCount && !found; c++) {
      const child = cursor.child(c);
      if (child.type.name === type && seen++ === path[depth]) {
        pos += 1;
        cursor = child;
        found = true;
      } else {
        pos += child.nodeSize;
      }
    }
    if (!found) return null;
  }
  return path.length > 0 ? pos : null;
}
