/**
 * ProseMirror doc → FictionBook (the shape Go expects via Wails).
 *
 * Inverse of parse.ts. Works at the JSON-shape level so the result can be
 * passed verbatim to `App.UpdateDocument(fb)` followed by `App.SaveFile(path)`.
 *
 * Notes:
 *  - Description (title-info, document-info, etc.) is not editable in
 *    Phase 3 yet — it's preserved from the originally-loaded FictionBook.
 *    Callers should merge the serialized bodies with the original description.
 *  - Binaries are likewise preserved from the original.
 */
import type { Node as PMNode, Mark } from "prosemirror-model";
import type {
  FictionBook, Body, Section, Block, Paragraph, Inline,
  Title, Epigraph, Annotation, Cite, Poem, Stanza,
  Table, Cell, Image,
} from "../fb2/types";

export function pmDocToFB2(doc: PMNode, base: FictionBook): FictionBook {
  const bodies: Body[] = [];
  doc.forEach((node) => {
    if (node.type.name === "body") {
      bodies.push(buildBody(node));
    }
  });
  return {
    ...base,
    Bodies: bodies,
  };
}

function buildBody(node: PMNode): Body {
  const body: Body = {
    Name: node.attrs.name || undefined,
    Sections: [],
    Epigraph: [],
  };
  node.forEach((child) => {
    switch (child.type.name) {
      case "title":
        body.Title = buildTitle(child);
        break;
      case "epigraph":
        body.Epigraph!.push(buildEpigraph(child));
        break;
      case "image_block":
        body.Image = buildImage(child);
        break;
      case "section":
        body.Sections.push(buildSection(child));
        break;
    }
  });
  if (body.Epigraph?.length === 0) delete body.Epigraph;
  return body;
}

function buildSection(node: PMNode): Section {
  const out: Section = {
    ID: node.attrs.id ?? undefined,
    Epigraph: [],
    Body: [],
  };
  node.forEach((child) => {
    switch (child.type.name) {
      case "title":
        out.Title = buildTitle(child);
        break;
      case "epigraph":
        out.Epigraph!.push(buildEpigraph(child));
        break;
      case "image_block":
        // Image only counts as the section's <image> header slot when it's
        // the first post-title child; elsewhere it's a body image. We
        // conservatively route image_block into Body — the (rare) cases
        // where it's actually the header image still round-trip correctly
        // because FB2 readers accept <image> at block level inside sections.
        out.Body!.push({ Image: buildImage(child) });
        break;
      case "annotation":
        out.Annotation = buildAnnotation(child);
        break;
      case "section":
        out.Body!.push({ Section: buildSection(child) });
        break;
      default: {
        const blk = buildBlock(child);
        if (blk) out.Body!.push(blk);
      }
    }
  });
  if (out.Epigraph?.length === 0) delete out.Epigraph;
  if (out.Body?.length === 0) delete out.Body;
  return out;
}

function buildTitle(node: PMNode): Title {
  const children: Block[] = [];
  node.forEach((child) => {
    const blk = buildBlock(child);
    if (blk) children.push(blk);
  });
  return { Children: children };
}

function buildEpigraph(node: PMNode): Epigraph {
  const children: Block[] = [];
  const textAuthor: Paragraph[] = [];
  node.forEach((child) => {
    if (child.type.name === "text_author") {
      textAuthor.push({ Children: buildInlines(child) });
    } else {
      const blk = buildBlock(child);
      if (blk) children.push(blk);
    }
  });
  const out: Epigraph = { Children: children };
  if (textAuthor.length) out.TextAuthor = textAuthor;
  return out;
}

function buildAnnotation(node: PMNode): Annotation {
  const children: Block[] = [];
  node.forEach((child) => {
    const blk = buildBlock(child);
    if (blk) children.push(blk);
  });
  return { Children: children };
}

function buildCite(node: PMNode): Cite {
  const children: Block[] = [];
  const textAuthor: Paragraph[] = [];
  node.forEach((child) => {
    if (child.type.name === "text_author") {
      textAuthor.push({ Children: buildInlines(child) });
    } else {
      const blk = buildBlock(child);
      if (blk) children.push(blk);
    }
  });
  const out: Cite = { Children: children };
  if (textAuthor.length) out.TextAuthor = textAuthor;
  return out;
}

function buildPoem(node: PMNode): Poem {
  const stanzas: Stanza[] = [];
  const textAuthor: Paragraph[] = [];
  let title: Title | undefined;
  const epigraphs: Epigraph[] = [];
  node.forEach((child) => {
    switch (child.type.name) {
      case "title":
        title = buildTitle(child);
        break;
      case "epigraph":
        epigraphs.push(buildEpigraph(child));
        break;
      case "stanza":
        stanzas.push(buildStanza(child));
        break;
      case "text_author":
        textAuthor.push({ Children: buildInlines(child) });
        break;
    }
  });
  const poem: Poem = { Stanzas: stanzas };
  if (title) poem.Title = title;
  if (epigraphs.length) poem.Epigraph = epigraphs;
  if (textAuthor.length) poem.TextAuthor = textAuthor;
  return poem;
}

function buildStanza(node: PMNode): Stanza {
  const verses: Paragraph[] = [];
  let title: Title | undefined;
  let subtitle: Paragraph | undefined;
  node.forEach((child) => {
    switch (child.type.name) {
      case "title":
        title = buildTitle(child);
        break;
      case "subtitle":
        subtitle = { Children: buildInlines(child) };
        break;
      case "verse":
        verses.push({ Children: buildInlines(child) });
        break;
    }
  });
  const out: Stanza = { Verses: verses };
  if (title) out.Title = title;
  if (subtitle) out.Subtitle = subtitle;
  return out;
}

function buildTable(node: PMNode): Table {
  const rows: Table["Rows"] = [];
  node.forEach((row) => {
    const cells: Cell[] = [];
    row.forEach((cell) => {
      const c: Cell = {
        XMLName: { Local: cell.attrs.header ? "th" : "td" },
        Children: buildInlines(cell),
      };
      if (cell.attrs.colspan && cell.attrs.colspan !== 1) c.ColSpan = String(cell.attrs.colspan);
      if (cell.attrs.rowspan && cell.attrs.rowspan !== 1) c.RowSpan = String(cell.attrs.rowspan);
      if (cell.attrs.align) c.Align = cell.attrs.align;
      if (cell.attrs.valign) c.VAlign = cell.attrs.valign;
      cells.push(c);
    });
    rows.push({ Cells: cells });
  });
  return { Rows: rows, ID: node.attrs.id || undefined };
}

function buildImage(node: PMNode): Image {
  return {
    Href: node.attrs.href || "",
    Title: node.attrs.title || undefined,
    Alt: node.attrs.alt || undefined,
  };
}

function buildBlock(node: PMNode): Block | null {
  switch (node.type.name) {
    case "paragraph":
      return { Paragraph: buildParagraph(node) };
    case "subtitle":
      return { Subtitle: { Children: buildInlines(node) } };
    case "empty_line":
      return { EmptyLine: {} };
    case "poem":
      return { Poem: buildPoem(node) };
    case "cite":
      return { Cite: buildCite(node) };
    case "table":
      return { Table: buildTable(node) };
    case "image_block":
      return { Image: buildImage(node) };
    case "section":
      return { Section: buildSection(node) };
    case "raw_block":
      return decodeRaw(node.attrs.raw, "Block");
  }
  return null;
}

// decodeRaw parses the JSON blob stashed in a raw_{block,inline} node back
// into the wrapper shape the Go writer expects. Returns null if the attr is
// empty or malformed — the block is silently dropped rather than corrupting
// the whole document.
function decodeRaw(raw: unknown, kind: "Block" | "Inline"): Block | Inline | null {
  if (typeof raw !== "string" || raw === "") return null;
  try {
    const parsed = JSON.parse(raw);
    if (!parsed || typeof parsed !== "object") return null;
    return { Raw: parsed } as Block | Inline;
  } catch {
    return null;
  }
}

function buildParagraph(node: PMNode): Paragraph {
  const p: Paragraph = { Children: buildInlines(node) };
  if (node.attrs.id) p.ID = node.attrs.id;
  if (node.attrs.style) p.Style = node.attrs.style;
  return p;
}

function buildInlines(node: PMNode): Inline[] {
  const out: Inline[] = [];
  node.forEach((child) => {
    if (child.isText) {
      pushTextWithMarks(child.text ?? "", child.marks, out);
    } else if (child.type.name === "image_inline") {
      out.push({
        Image: {
          Href: child.attrs.href || "",
          Title: child.attrs.title || undefined,
          Alt: child.attrs.alt || undefined,
        },
      });
    } else if (child.type.name === "raw_inline") {
      const raw = decodeRaw(child.attrs.raw, "Inline") as Inline | null;
      if (raw) out.push(raw);
    }
  });
  return out;
}

/**
 * Convert a PM text node with marks into an Inline. We nest marks
 * right-to-left so the outermost element corresponds to the first mark.
 * Order-within-marks: strong > emphasis > strikethrough > sub > sup > code > style > link.
 */
function pushTextWithMarks(text: string, marks: readonly Mark[], out: Inline[]): void {
  if (marks.length === 0) {
    out.push({ Text: text });
    return;
  }
  const ordered = [...marks].sort((a, b) => markOrder(a) - markOrder(b));
  let inner: Inline = { Text: text };
  for (let i = ordered.length - 1; i >= 0; i--) {
    inner = wrapMark(ordered[i], inner);
  }
  out.push(inner);
}

function markOrder(m: Mark): number {
  switch (m.type.name) {
    case "strong":        return 1;
    case "emphasis":      return 2;
    case "strikethrough": return 3;
    case "sub":           return 4;
    case "sup":           return 5;
    case "code":          return 6;
    case "style":         return 7;
    case "link":          return 8;
    default:              return 99;
  }
}

function wrapMark(m: Mark, child: Inline): Inline {
  const children: Inline[] = [child];
  switch (m.type.name) {
    case "strong":        return { Strong: { Children: children } };
    case "emphasis":      return { Emphasis: { Children: children } };
    case "strikethrough": return { Strikethrough: { Children: children } };
    case "sub":           return { Sub: { Children: children } };
    case "sup":           return { Sup: { Children: children } };
    case "code":          return { Code: { Children: children } };
    case "style":         return { Style: { Name: m.attrs.name ?? "", Children: children } };
    case "link":          return { A: { Href: m.attrs.href ?? "", Type: m.attrs.type ?? "", Children: children } };
  }
  return child;
}
