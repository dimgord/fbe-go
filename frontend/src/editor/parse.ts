/**
 * FB2 document (doc.FictionBook from Go) → ProseMirror doc.
 *
 * Handles: body, section (nested or flat), title, epigraph, annotation,
 * paragraph, subtitle, empty-line, poem/stanza/verse, cite, block + inline
 * image, table, and all inline marks (strong, emphasis, strikethrough, sub,
 * sup, code, style, link).
 *
 * Counterpart: serialize.ts (ProseMirror → FB2) — still a TODO for Phase 3.
 */
import { Node as PMNode, Mark, Fragment } from "prosemirror-model";
import type {
  FictionBook, Body, Section, Block, Paragraph, Inline,
  Poem, Stanza, Cite, Epigraph, Annotation,
  Table, Image,
} from "../fb2/types";
import { fb2Schema } from "./schema";

const N = fb2Schema.nodes;
const M = fb2Schema.marks;

export function fb2ToPMDoc(fb: FictionBook): PMNode {
  const bodies = (fb.Bodies ?? []).map(buildBody).filter((n): n is PMNode => n != null);
  if (bodies.length === 0) {
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
  for (const ep of body.Epigraph ?? []) {
    const n = buildEpigraph(ep);
    if (n) children.push(n);
  }
  if (body.Image) {
    children.push(buildBlockImage(body.Image));
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
  for (const ep of s.Epigraph ?? []) {
    const n = buildEpigraph(ep);
    if (n) children.push(n);
  }
  if (s.Image) {
    children.push(buildBlockImage(s.Image));
  }
  if (s.Annotation) {
    const n = buildAnnotation(s.Annotation);
    if (n) children.push(n);
  }

  // FB2: section has either nested sections OR flat block content, not both.
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
  const children = buildBlockList(blocks, { titleOnly: true });
  if (children.length === 0) children.push(N.paragraph.createAndFill()!);
  return N.title.create(null, children);
}

function buildEpigraph(ep: Epigraph): PMNode | null {
  const children: PMNode[] = [];
  for (const b of ep.Children ?? []) {
    const n = buildBlock(b);
    if (n) children.push(n);
  }
  for (const p of ep.TextAuthor ?? []) {
    children.push(buildTextAuthor(p));
  }
  if (children.length === 0) children.push(N.paragraph.createAndFill()!);
  return N.epigraph.create(null, children);
}

function buildAnnotation(a: Annotation): PMNode | null {
  const children: PMNode[] = [];
  for (const b of a.Children ?? []) {
    const n = buildBlock(b);
    if (n) children.push(n);
  }
  if (children.length === 0) children.push(N.paragraph.createAndFill()!);
  return N.annotation.create(null, children);
}

function buildCite(c: Cite): PMNode | null {
  const children: PMNode[] = [];
  for (const b of c.Children ?? []) {
    const n = buildBlock(b);
    if (n) children.push(n);
  }
  for (const p of c.TextAuthor ?? []) {
    children.push(buildTextAuthor(p));
  }
  if (children.length === 0) children.push(N.paragraph.createAndFill()!);
  return N.cite.create(null, children);
}

function buildPoem(p: Poem): PMNode | null {
  const children: PMNode[] = [];
  if (p.Title) {
    const t = buildTitle(p.Title.Children ?? []);
    if (t) children.push(t);
  }
  for (const ep of p.Epigraph ?? []) {
    const n = buildEpigraph(ep);
    if (n) children.push(n);
  }
  for (const s of p.Stanzas ?? []) {
    const n = buildStanza(s);
    if (n) children.push(n);
  }
  for (const ta of p.TextAuthor ?? []) {
    children.push(buildTextAuthor(ta));
  }
  if (children.length === 0) {
    children.push(N.stanza.create(null, [N.verse.createAndFill()!]));
  }
  return N.poem.create(null, children);
}

function buildStanza(s: Stanza): PMNode | null {
  const children: PMNode[] = [];
  if (s.Title) {
    const t = buildTitle(s.Title.Children ?? []);
    if (t) children.push(t);
  }
  if (s.Subtitle) {
    children.push(N.subtitle.create(null, buildInlines(s.Subtitle.Children ?? [])));
  }
  for (const v of s.Verses ?? []) {
    children.push(N.verse.create(null, buildInlines(v.Children ?? [])));
  }
  if ((s.Verses?.length ?? 0) === 0) {
    children.push(N.verse.createAndFill()!);
  }
  return N.stanza.create(null, children);
}

function buildTable(t: Table): PMNode | null {
  const rows: PMNode[] = [];
  for (const r of t.Rows ?? []) {
    const cells: PMNode[] = [];
    for (const c of r.Cells ?? []) {
      const header = c.XMLName?.Local === "th";
      cells.push(N.table_cell.create(
        {
          header,
          colspan: parseInt(c.ColSpan ?? "1", 10) || 1,
          rowspan: parseInt(c.RowSpan ?? "1", 10) || 1,
          align: c.Align ?? null,
          valign: c.VAlign ?? null,
        },
        buildInlines(c.Children ?? []),
      ));
    }
    if (cells.length === 0) {
      cells.push(N.table_cell.createAndFill()!);
    }
    rows.push(N.table_row.create(null, cells));
  }
  if (rows.length === 0) return null;
  return N.table.create({ id: t.ID ?? null }, rows);
}

function buildBlockImage(img: Image): PMNode {
  return N.image_block.create({
    href: img.Href,
    title: img.Title ?? "",
    alt: img.Alt ?? "",
  });
}

function buildTextAuthor(p: Paragraph): PMNode {
  return N.text_author.create(null, buildInlines(p.Children ?? []));
}

function buildBlockList(blocks: Block[], opts: { titleOnly?: boolean } = {}): PMNode[] {
  const out: PMNode[] = [];
  for (const b of blocks) {
    if (opts.titleOnly) {
      // Titles may only contain paragraphs and empty-lines per FictionBook.xsd.
      if (b.Paragraph) out.push(buildParagraph(b.Paragraph));
      else if (b.EmptyLine) out.push(N.empty_line.create());
      continue;
    }
    const n = buildBlock(b);
    if (n) out.push(n);
  }
  return out;
}

function buildBlock(b: Block): PMNode | null {
  if (b.Paragraph) return buildParagraph(b.Paragraph);
  if (b.EmptyLine) return N.empty_line.create();
  if (b.Subtitle)  return N.subtitle.create(null, buildInlines(b.Subtitle.Children ?? []));
  if (b.Poem)      return buildPoem(b.Poem);
  if (b.Cite)      return buildCite(b.Cite);
  if (b.Table)     return buildTable(b.Table);
  if (b.Image)     return buildBlockImage(b.Image);
  return null;
}

function buildParagraph(p: Paragraph): PMNode {
  const attrs = { id: p.ID ?? null, style: p.Style ?? null };
  const inlines = buildInlines(p.Children ?? []);
  if (inlines.length === 0) return N.paragraph.create(attrs);
  return N.paragraph.create(attrs, inlines);
}

function buildInlines(items: Inline[]): PMNode[] {
  const out: PMNode[] = [];
  for (const i of items) pushInline(i, [], out);
  return out;
}

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

export { Fragment };
