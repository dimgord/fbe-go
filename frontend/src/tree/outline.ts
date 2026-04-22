/**
 * Build an outline tree from a FictionBook value.
 *
 * Each node has:
 *  - label: displayed text (extracted from section's <title>, or a placeholder)
 *  - path:  index array ([body, section, ...subsections]) that lets the editor
 *           navigate to the matching ProseMirror node
 *  - kind:  which FB2 element the node corresponds to
 *  - children: nested outline nodes
 */
import type { FictionBook, Section, Title, Inline } from "../fb2/types";

export type OutlineKind = "body" | "section";

export interface OutlineNode {
  label: string;
  kind: OutlineKind;
  path: number[];
  children: OutlineNode[];
}

export function buildOutline(fb: FictionBook | null): OutlineNode[] {
  if (!fb) return [];
  const out: OutlineNode[] = [];
  const bodies = fb.Bodies ?? [];
  for (let bi = 0; bi < bodies.length; bi++) {
    const body = bodies[bi];
    out.push({
      kind: "body",
      label: titleText(body.Title) || (body.Name ? `body[${body.Name}]` : `body ${bi + 1}`),
      path: [bi],
      children: (body.Sections ?? []).map((s, si) => buildSection(s, [bi, si], si + 1)),
    });
  }
  return out;
}

function buildSection(s: Section, path: number[], index: number): OutlineNode {
  const label = titleText(s.Title) || `section ${index}`;
  // Rev 37: subsections now live inside Section.Body as Block entries with
  // a non-null Section variant. Filter them out and keep their index within
  // the subsection-only sequence so the outline path still maps to the
  // editor's PM section children.
  let subIdx = 0;
  const subs: OutlineNode[] = [];
  for (const b of s.Body ?? []) {
    if (!b.Section) continue;
    subs.push(buildSection(b.Section, [...path, subIdx], subIdx + 1));
    subIdx++;
  }
  return {
    kind: "section",
    label,
    path,
    children: subs,
  };
}

function titleText(t: Title | null | undefined): string {
  if (!t) return "";
  const parts: string[] = [];
  for (const block of t.Children ?? []) {
    if (block.Paragraph) parts.push(flattenInlines(block.Paragraph.Children ?? []));
  }
  return parts.join(" ").trim();
}

function flattenInlines(items: Inline[]): string {
  let out = "";
  for (const i of items) {
    if (i.Text) out += i.Text;
    if (i.Strong?.Children)        out += flattenInlines(i.Strong.Children);
    if (i.Emphasis?.Children)      out += flattenInlines(i.Emphasis.Children);
    if (i.Strikethrough?.Children) out += flattenInlines(i.Strikethrough.Children);
    if (i.Sub?.Children)           out += flattenInlines(i.Sub.Children);
    if (i.Sup?.Children)           out += flattenInlines(i.Sup.Children);
    if (i.Code?.Children)          out += flattenInlines(i.Code.Children);
    if (i.Style?.Children)         out += flattenInlines(i.Style.Children);
    if (i.A?.Children)             out += flattenInlines(i.A.Children);
  }
  return out;
}
