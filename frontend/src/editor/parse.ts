/**
 * FB2 document (doc.FictionBook from Go) → ProseMirror doc.
 *
 * Phase 0 scope: body, section, title, paragraph + strong/emphasis/strikethrough/sub/sup/code/link marks.
 * Not yet: poem/stanza/cite/epigraph/annotation/image/table — those come in Phase 3.
 */
import { Node as PMNode, Mark, Fragment } from "prosemirror-model";
import type { FictionBook, Body, Section, Block, Paragraph, Inline } from "../fb2/types";
import { fb2Schema } from "./schema";

const N = fb2Schema.nodes;
const M = fb2Schema.marks;

export function fb2ToPMDoc(fb: FictionBook): PMNode {
  const bodies = (fb.Bodies ?? []).map(buildBody).filter((n): n is PMNode => n != null);
  if (bodies.length === 0) {
    // Empty fallback.
    return fb2Schema.topNodeType.createAndFill()!;
  }
  return N.doc.create(null, bodies);
}

function buildBody(body: Body): PMNode | null {
  const children: PMNode[] = [];
  if (body.Title) {
    const t = buildTitle(body.Title.Children ?? []);
    if (t) children.push(t);
  }
  for (const s of body.Sections ?? []) {
    const node = buildSection(s);
    if (node) children.push(node);
  }
  if (children.length === 0) {
    children.push(N.section.createAndFill()!);
  }
  return N.body.create({ name: body.Name ?? "" }, children);
}

function buildSection(s: Section): PMNode | null {
  const children: PMNode[] = [];

  if (s.Title) {
    const t = buildTitle(s.Title.Children ?? []);
    if (t) children.push(t);
  }

  // FB2 sections either nest OR contain flat blocks, not both.
  if (s.Sections && s.Sections.length > 0) {
    for (const sub of s.Sections) {
      const n = buildSection(sub);
      if (n) children.push(n);
    }
  } else {
    for (const b of s.Blocks ?? []) {
      const n = buildBlock(b);
      if (n) children.push(n);
    }
  }

  if (children.length === 0) {
    children.push(N.paragraph.createAndFill()!);
  }
  return N.section.create({ id: s.ID ?? null }, children);
}

function buildTitle(blocks: Block[]): PMNode | null {
  const children: PMNode[] = [];
  for (const b of blocks) {
    if (b.Paragraph) {
      children.push(buildParagraph(b.Paragraph));
    } else if (b.EmptyLine) {
      children.push(N.empty_line.create());
    }
  }
  if (children.length === 0) {
    children.push(N.paragraph.createAndFill()!);
  }
  return N.title.create(null, children);
}

function buildBlock(b: Block): PMNode | null {
  if (b.Paragraph) return buildParagraph(b.Paragraph);
  if (b.EmptyLine) return N.empty_line.create();
  if (b.Subtitle) return N.subtitle.create(null, buildInlines(b.Subtitle.Children ?? []));
  // TODO(phase-3): Poem, Cite, Table, Image
  return null;
}

function buildParagraph(p: Paragraph): PMNode {
  const attrs = { id: p.ID ?? null, style: p.Style ?? null };
  const inlines = buildInlines(p.Children ?? []);
  if (inlines.length === 0) {
    return N.paragraph.create(attrs);
  }
  return N.paragraph.create(attrs, inlines);
}

function buildInlines(items: Inline[]): PMNode[] {
  const out: PMNode[] = [];
  for (const i of items) {
    pushInline(i, [], out);
  }
  return out;
}

/** Walk one Inline node; emits text nodes with the accumulated marks, recurses into nested marks. */
function pushInline(i: Inline, marks: Mark[], out: PMNode[]): void {
  if (i.Text !== undefined && i.Text !== "") {
    out.push(fb2Schema.text(i.Text, marks));
  }

  const wrap = (markName: keyof typeof M, attrs: Record<string, unknown> = {}, children: Inline[] = []) => {
    const m = M[markName].create(attrs);
    for (const c of children) pushInline(c, marks.concat([m]), out);
  };

  if (i.Strong)        wrap("strong", {}, i.Strong.Children ?? []);
  if (i.Emphasis)      wrap("emphasis", {}, i.Emphasis.Children ?? []);
  if (i.Strikethrough) wrap("strikethrough", {}, i.Strikethrough.Children ?? []);
  if (i.Sub)           wrap("sub", {}, i.Sub.Children ?? []);
  if (i.Sup)           wrap("sup", {}, i.Sup.Children ?? []);
  if (i.Code)          wrap("code", {}, i.Code.Children ?? []);
  if (i.Style)         wrap("style", { name: i.Style.Name }, i.Style.Children ?? []);
  if (i.A)             wrap("link", { href: i.A.Href, type: i.A.Type ?? "" }, i.A.Children ?? []);

  if (i.Image) {
    out.push(N.image_inline.create({
      href: i.Image.Href,
      title: i.Image.Title ?? "",
      alt: i.Image.Alt ?? "",
    }));
  }
}

// Utility export — kept in case other modules want to build a Fragment directly.
export { Fragment };
