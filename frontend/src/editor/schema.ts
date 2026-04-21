/**
 * ProseMirror schema for FictionBook 2.
 *
 * Mapping derived from FBE/fb2.xsl (see docs/OPERATIONS.md for the full list).
 * The schema intentionally stays close to the FB2 element model so that
 * serialize.ts / parse.ts can be near 1:1 translations.
 */
import { Schema, type NodeSpec, type MarkSpec } from "prosemirror-model";

const nodes: Record<string, NodeSpec> = {
  // Top of the editable document: one or more <body> containers.
  doc: { content: "body+" },

  // Block containers (each has optional title/epigraph/image/annotation children).
  body: {
    content: "(title | epigraph | image | annotation | section)+",
    group: "structural",
    attrs: { name: { default: "" } },
    toDOM: (n) => ["div", { class: "body", "data-name": n.attrs.name }, 0],
    parseDOM: [{ tag: "div.body", getAttrs: (dom) => ({ name: (dom as HTMLElement).getAttribute("data-name") || "" }) }],
  },

  section: {
    content: "(title | epigraph | image | annotation)* (section+ | block+)",
    group: "structural",
    attrs: { id: { default: null } },
    toDOM: (n) => ["div", { class: "section", id: n.attrs.id }, 0],
    parseDOM: [{ tag: "div.section" }],
  },

  title: {
    content: "(paragraph | empty_line)+",
    group: "structural",
    toDOM: () => ["div", { class: "title" }, 0],
    parseDOM: [{ tag: "div.title" }],
  },

  epigraph: {
    content: "(paragraph | poem | cite | empty_line)+ text_author*",
    group: "structural",
    toDOM: () => ["div", { class: "epigraph" }, 0],
    parseDOM: [{ tag: "div.epigraph" }],
  },

  cite: {
    content: "(paragraph | poem | empty_line | subtitle | table)+ text_author*",
    group: "block",
    toDOM: () => ["div", { class: "cite" }, 0],
    parseDOM: [{ tag: "div.cite" }],
  },

  poem: {
    content: "title? epigraph* stanza+ text_author* date?",
    group: "block",
    toDOM: () => ["div", { class: "poem" }, 0],
    parseDOM: [{ tag: "div.poem" }],
  },

  stanza: {
    content: "title? subtitle? verse+",
    toDOM: () => ["div", { class: "stanza" }, 0],
    parseDOM: [{ tag: "div.stanza" }],
  },

  annotation: {
    content: "(paragraph | poem | cite | subtitle | empty_line | table)+",
    group: "structural",
    toDOM: () => ["div", { class: "annotation" }, 0],
    parseDOM: [{ tag: "div.annotation" }],
  },

  // Block-level leaf-ish items.
  paragraph: {
    content: "inline*",
    group: "block",
    attrs: { id: { default: null }, style: { default: null } },
    toDOM: (n) => ["p", { id: n.attrs.id, "data-style": n.attrs.style }, 0],
    parseDOM: [{ tag: "p", getAttrs: (dom) => ({
      id: (dom as HTMLElement).id || null,
      style: (dom as HTMLElement).getAttribute("data-style"),
    }) }],
  },

  // FB2 <v> (verse line) — same shape as paragraph but in stanza context.
  verse: {
    content: "inline*",
    toDOM: () => ["p", { class: "v" }, 0],
    parseDOM: [{ tag: "p.v" }],
  },

  subtitle: {
    content: "inline*",
    group: "block",
    toDOM: () => ["p", { class: "subtitle" }, 0],
    parseDOM: [{ tag: "p.subtitle" }],
  },

  text_author: {
    content: "inline*",
    toDOM: () => ["p", { class: "text-author" }, 0],
    parseDOM: [{ tag: "p.text-author" }],
  },

  date: {
    content: "inline*",
    attrs: { value: { default: "" } },
    toDOM: (n) => ["p", { class: "date", "data-value": n.attrs.value }, 0],
    parseDOM: [{ tag: "p.date" }],
  },

  empty_line: {
    group: "block",
    toDOM: () => ["p", { class: "empty-line" }],
    parseDOM: [{ tag: "p.empty-line" }],
  },

  // FB2 table — rows are divs, cells are paragraphs with class td/th.
  table: {
    content: "table_row+",
    group: "block",
    toDOM: () => ["div", { class: "table" }, 0],
    parseDOM: [{ tag: "div.table" }],
  },

  table_row: {
    content: "table_cell+",
    toDOM: () => ["div", { class: "tr" }, 0],
    parseDOM: [{ tag: "div.tr" }],
  },

  table_cell: {
    content: "inline*",
    attrs: {
      header: { default: false },
      colspan: { default: 1 },
      rowspan: { default: 1 },
      align: { default: null },
      valign: { default: null },
    },
    toDOM: (n) => ["p", {
      class: n.attrs.header ? "th" : "td",
      "data-colspan": n.attrs.colspan,
      "data-rowspan": n.attrs.rowspan,
      "data-align": n.attrs.align,
      "data-valign": n.attrs.valign,
    }, 0],
    parseDOM: [
      { tag: "p.td", getAttrs: () => ({ header: false }) },
      { tag: "p.th", getAttrs: () => ({ header: true }) },
    ],
  },

  // Images come in block and inline flavors; store as separate node types to
  // make schema constraints explicit (FB2 allows inline images only inside p,
  // subtitle, text-author — see fb2.xsl line 224).
  image_block: {
    group: "block",
    attrs: { href: { default: "" }, title: { default: "" }, alt: { default: "" } },
    atom: true,
    draggable: true,
    toDOM: (n) => ["div", { class: "image", "data-href": n.attrs.href, title: n.attrs.title },
      ["img", { src: `fb2://binary${n.attrs.href}`, alt: n.attrs.alt }]],
    parseDOM: [{ tag: "div.image" }],
  },

  image_inline: {
    group: "inline",
    inline: true,
    atom: true,
    attrs: { href: { default: "" }, title: { default: "" }, alt: { default: "" } },
    toDOM: (n) => ["span", { class: "image", "data-href": n.attrs.href, title: n.attrs.title },
      ["img", { src: `fb2://binary${n.attrs.href}`, alt: n.attrs.alt }]],
    parseDOM: [{ tag: "span.image" }],
  },

  text: { group: "inline" },
};

const marks: Record<string, MarkSpec> = {
  // Inline FB2 types. Mapping from fb2.xsl lines 197-221.
  strong: {
    toDOM: () => ["strong", 0],
    parseDOM: [{ tag: "strong" }, { tag: "b" }, { style: "font-weight", getAttrs: (v) => /^(bold(er)?|[5-9]\d{2,})$/.test(v as string) && null }],
  },
  emphasis: {
    toDOM: () => ["em", 0],
    parseDOM: [{ tag: "em" }, { tag: "i" }, { style: "font-style=italic" }],
  },
  strikethrough: {
    toDOM: () => ["s", 0],
    parseDOM: [{ tag: "s" }, { tag: "strike" }, { tag: "del" }],
  },
  sub: { toDOM: () => ["sub", 0], parseDOM: [{ tag: "sub" }] },
  sup: { toDOM: () => ["sup", 0], parseDOM: [{ tag: "sup" }] },
  code: {
    toDOM: () => ["span", { class: "code" }, 0],
    parseDOM: [{ tag: "span.code" }, { tag: "code" }],
  },
  style: {
    attrs: { name: { default: "" } },
    toDOM: (m) => ["span", { class: "style", "data-name": m.attrs.name }, 0],
    parseDOM: [{ tag: "span.style", getAttrs: (d) => ({ name: (d as HTMLElement).dataset.name || "" }) }],
  },
  link: {
    attrs: { href: { default: "" }, type: { default: "" } },
    toDOM: (m) => ["a", { href: m.attrs.href, "data-type": m.attrs.type }, 0],
    parseDOM: [{ tag: "a[href]", getAttrs: (d) => ({
      href: (d as HTMLElement).getAttribute("href") || "",
      type: (d as HTMLElement).getAttribute("data-type") || "",
    }) }],
    inclusive: false,
  },
};

export const fb2Schema = new Schema({ nodes, marks });
