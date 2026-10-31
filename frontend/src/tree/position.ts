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
  /** Index among all text blocks; xmlBlockLines[block] is its line in the XML. */
  block: number;
}

/** Start positions of every text block, cached because the position updates while scrolling. */
const blockStarts = new WeakMap<PMNode, number[]>();

function textBlocks(doc: PMNode): number[] {
  let starts = blockStarts.get(doc);
  if (!starts) {
    const found: number[] = [];
    doc.descendants((n, pos) => {
      if (n.isTextblock) found.push(pos);
      return !n.isTextblock;
    });
    blockStarts.set(doc, found);
    starts = found;
  }
  return starts;
}

/** Index of the last text block starting at or before pos. */
function blockIndexAt(doc: PMNode, pos: number): number {
  const starts = textBlocks(doc);
  let lo = 0;
  let hi = starts.length - 1;
  while (lo < hi) {
    const mid = Math.ceil((lo + hi) / 2);
    if (starts[mid] <= pos) lo = mid;
    else hi = mid - 1;
  }
  return lo;
}

/** A position just inside text block `index`, for scrolling to it. */
export function blockPos(doc: PMNode, index: number): number | null {
  const start = textBlocks(doc)[index];
  return start === undefined ? null : start + 1;
}

/** Index of the last block starting at or above a 1-based XML line. */
export function blockAtLine(blockLines: number[], line: number): number {
  let lo = 0;
  let hi = blockLines.length - 1;
  while (lo < hi) {
    const mid = Math.ceil((lo + hi) / 2);
    if (blockLines[mid] <= line) lo = mid;
    else hi = mid - 1;
  }
  return lo;
}

// The elements that become PM text blocks, in document order.
const BLOCK_TAG = /<(?:p|subtitle|v|text-author|date|td|th)(?=[\s>/])/g;

/** How far past the expected tag to look: <p>s inside a preserved unknown element shift the count. */
const ALIGN_WINDOW = 12;
const PROBE_CHARS = 12;

/** 1-based line of every block tag from the first <body> up to the binaries. */
function blockTagLines(xml: string): number[] {
  const start = xml.indexOf("<body");
  if (start < 0) return [];
  const end = xml.indexOf("<binary", start);
  const lines: number[] = [];
  let line = 1;
  let scanned = 0;
  BLOCK_TAG.lastIndex = start;
  for (let m = BLOCK_TAG.exec(xml); m && (end < 0 || m.index < end); m = BLOCK_TAG.exec(xml)) {
    for (let i = xml.indexOf("\n", scanned); i >= 0 && i < m.index; i = xml.indexOf("\n", i + 1)) {
      line++;
      scanned = i + 1;
    }
    lines.push(line);
  }
  return lines;
}

/** A line's text as the reader sees it: tags dropped, the writer's escapes undone. */
function plainText(line: string): string {
  return line
    .replace(/<[^>]+>/g, "")
    .replace(/&#39;/g, "'")
    .replace(/&#34;|&quot;/g, '"')
    .replace(/&lt;/g, "<")
    .replace(/&gt;/g, ">")
    .replace(/&amp;/g, "&");
}

/**
 * 1-based XML line of each text block. Blocks pair up with tags in order, and their text corrects
 * the pairing where the two sequences differ, as they do inside preserved unknown elements.
 */
export function xmlBlockLines(doc: PMNode, xml: string): number[] {
  const tags = blockTagLines(xml);
  const lines = xml.split("\n");
  const out: number[] = [];
  let next = 0;
  doc.descendants((n) => {
    if (!n.isTextblock) return true;
    const probe = n.textContent.trim().slice(0, PROBE_CHARS);
    let hit = next;
    if (probe) {
      const last = Math.min(next + ALIGN_WINDOW, tags.length - 1);
      for (let t = next; t <= last; t++) {
        if (plainText(lines[tags[t] - 1]).includes(probe)) {
          hit = t;
          break;
        }
      }
    }
    if (hit < tags.length) out.push(tags[hit]);
    next = hit + 1;
    return false;
  });
  return out;
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
  return {
    path,
    trail,
    percent: size > 0 ? Math.round((pos / size) * 100) : 0,
    block: blockIndexAt(doc, pos),
  };
}
