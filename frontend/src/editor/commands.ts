/**
 * FB2 editing commands for ProseMirror.
 *
 * Each command follows the ProseMirror convention:
 *   (state, dispatch?) => boolean
 * When `dispatch` is omitted the command only reports availability.
 *
 * See docs/OPERATIONS.md for the FBE → command mapping.
 */
import type { EditorState, Transaction } from "prosemirror-state";
import type { NodeType, Attrs } from "prosemirror-model";
import {
  toggleMark as pmToggleMark,
  setBlockType as pmSetBlockType,
} from "prosemirror-commands";
import { fb2Schema } from "./schema";

export type Command = (
  state: EditorState,
  dispatch?: (tr: Transaction) => void,
) => boolean;

// ── Inline mark toggles ─────────────────────────────────────────────────────

const M = fb2Schema.marks;

/** Toggle <strong> on the selection. Keyboard: Mod-b. */
export const toggleStrong: Command        = pmToggleMark(M.strong);

/** Toggle <emphasis> on the selection. Keyboard: Mod-i. */
export const toggleEmphasis: Command      = pmToggleMark(M.emphasis);

/** Toggle <strikethrough>. */
export const toggleStrikethrough: Command = pmToggleMark(M.strikethrough);

/** Toggle <sub>. */
export const toggleSub: Command           = pmToggleMark(M.sub);

/** Toggle <sup>. */
export const toggleSup: Command           = pmToggleMark(M.sup);

/** Toggle inline <code>. */
export const toggleCode: Command          = pmToggleMark(M.code);

/** Toggle a link with the given href. Pass empty href to remove. */
export function toggleLink(href: string): Command {
  if (!href) {
    // Remove link mark from the range.
    return (state, dispatch) => {
      const { from, to } = state.selection;
      if (from === to) return false;
      if (dispatch) {
        const tr = state.tr.removeMark(from, to, M.link);
        dispatch(tr);
      }
      return true;
    };
  }
  return pmToggleMark(M.link, { href, type: "" });
}

/** Apply a named <style name="…"> to the selection. */
export function applyStyleMark(name: string): Command {
  return pmToggleMark(M.style, { name });
}

// ── Paragraph style (block-type) commands ───────────────────────────────────

const N = fb2Schema.nodes;

/** Turn the current paragraph back into a plain <p>. */
export const styleNormal: Command       = pmSetBlockType(N.paragraph);

/** Convert the current block into a <subtitle>. */
export const styleSubtitle: Command     = pmSetBlockType(N.subtitle);

/** Convert the current block into a <text-author> (typically at end of poem/cite/epigraph). */
export const styleTextAuthor: Command   = pmSetBlockType(N.text_author);

/** Convert the current block into an <empty-line>. */
export const insertEmptyLine: Command = (state, dispatch) => {
  if (!N.empty_line) return false;
  if (dispatch) {
    const tr = state.tr.replaceSelectionWith(N.empty_line.create());
    dispatch(tr);
  }
  return true;
};

// ── Helpers that return the block/mark status under the cursor ──────────────

/** Is the given mark active at the current selection head? */
export function isMarkActive(state: EditorState, markName: keyof typeof M): boolean {
  const mark = M[markName];
  const { from, $from, to, empty } = state.selection;
  if (empty) return !!mark.isInSet(state.storedMarks ?? $from.marks());
  return state.doc.rangeHasMark(from, to, mark);
}

/** Is the block at the cursor of the given type? */
export function isBlockActive(state: EditorState, nodeType: NodeType, attrs?: Attrs): boolean {
  const { $from, to } = state.selection;
  if (to > $from.end()) return false;
  return $from.parent.hasMarkup(nodeType, attrs ?? null);
}

// ── Structural commands ─────────────────────────────────────────────────────

import type { Node as PMNode, ResolvedPos } from "prosemirror-model";

/** Walk up the ancestor chain from $pos; return the first parent matching nodeName. */
function findAncestor(
  $pos: ResolvedPos,
  nodeName: string,
): { node: PMNode; depth: number; before: number; after: number } | null {
  for (let d = $pos.depth; d > 0; d--) {
    const node = $pos.node(d);
    if (node.type.name === nodeName) {
      return {
        node,
        depth: d,
        before: $pos.before(d),
        after: $pos.after(d),
      };
    }
  }
  return null;
}

/** First ancestor whose name is in the given set. */
function findAncestorAny(
  $pos: ResolvedPos,
  names: string[],
): { node: PMNode; depth: number; before: number; after: number } | null {
  for (let d = $pos.depth; d > 0; d--) {
    const node = $pos.node(d);
    if (names.includes(node.type.name)) {
      return {
        node,
        depth: d,
        before: $pos.before(d),
        after: $pos.after(d),
      };
    }
  }
  return null;
}

/**
 * Clone the surrounding section/poem/stanza/cite/epigraph and insert the copy
 * directly after it. Replaces main.js::CloneContainer (FBE/main.js:1940).
 */
export const cloneContainer: Command = (state, dispatch) => {
  const { $from } = state.selection;
  const target = findAncestorAny($from, ["section", "poem", "stanza", "cite", "epigraph"]);
  if (!target) return false;
  if (!dispatch) return true;

  // Deep copy via JSON round-trip so marks/attrs/content all duplicate cleanly.
  const clone = fb2Schema.nodeFromJSON(target.node.toJSON());
  const tr = state.tr.insert(target.after, clone);
  dispatch(tr.scrollIntoView());
  return true;
};

/**
 * Remove the surrounding section, moving its child sections (if any) up to the
 * parent. Replaces main.js::RemoveOuterContainer (FBE/main.js:2357).
 *
 * Only operates on a section whose contents are entirely nested sections
 * (matches FBE's IsCtSection check).
 */
export const removeOuterContainer: Command = (state, dispatch) => {
  const { $from } = state.selection;
  const target = findAncestor($from, "section");
  if (!target) return false;

  // Are all children of the target section themselves sections (plus optional title/epigraph/image/annotation)?
  let onlyStructural = true;
  target.node.forEach((child) => {
    const n = child.type.name;
    if (n !== "section" && n !== "title" && n !== "epigraph" && n !== "image_block" && n !== "annotation") {
      onlyStructural = false;
    }
  });
  if (!onlyStructural) return false;

  if (!dispatch) return true;

  // Extract just the section children of target; drop title/epigraph/image/annotation for simplicity.
  const moved: PMNode[] = [];
  target.node.forEach((child) => {
    if (child.type.name === "section") moved.push(child);
  });
  if (moved.length === 0) return false; // Nothing to promote.

  const tr = state.tr.replaceWith(target.before, target.after, moved);
  dispatch(tr.scrollIntoView());
  return true;
};

/**
 * Add an empty <title> to the enclosing section / body / poem / stanza if it
 * doesn't already have one. Replaces main.js::AddTitle (FBE/main.js:1766).
 * Simplified: always inserts an empty title (doesn't consume selection text).
 */
export const addTitle: Command = (state, dispatch) => {
  const { $from } = state.selection;
  const target = findAncestorAny($from, ["section", "body", "poem", "stanza"]);
  if (!target) return false;

  // Check if a title already exists as the first (or second, after image in body) child.
  let hasTitle = false;
  target.node.forEach((child) => {
    if (child.type.name === "title") hasTitle = true;
  });
  if (hasTitle) return false;

  if (!dispatch) return true;

  const titleNode = N.title.create(null, [N.paragraph.createAndFill()!]);
  // Insert at the start of target's content (position = before + 1).
  const tr = state.tr.insert(target.before + 1, titleNode);
  dispatch(tr.scrollIntoView());
  return true;
};

/**
 * Add an <epigraph> to the enclosing body / section / poem. Inserts after any
 * existing title but before the first section/block content.
 * Replaces main.js::AddEpigraph (FBE/main.js:2050). Simplified: empty epigraph.
 */
export const addEpigraph: Command = (state, dispatch) => {
  const { $from } = state.selection;
  const target = findAncestorAny($from, ["body", "section", "poem"]);
  if (!target) return false;

  if (!dispatch) return true;

  const epigraph = N.epigraph.create(null, [N.paragraph.createAndFill()!]);
  const insertPos = firstInsertionPointAfterHeader(target.node, target.before, ["title"]);
  const tr = state.tr.insert(insertPos, epigraph);
  dispatch(tr.scrollIntoView());
  return true;
};

/**
 * Add an <annotation> to the enclosing section if it doesn't already have one.
 * Inserts after title/epigraph/image but before section/block content.
 * Replaces main.js::AddAnnotation (FBE/main.js:2142).
 */
export const addAnnotation: Command = (state, dispatch) => {
  const { $from } = state.selection;
  const target = findAncestor($from, "section");
  if (!target) return false;

  // Don't add a second annotation.
  let hasAnnotation = false;
  target.node.forEach((child) => {
    if (child.type.name === "annotation") hasAnnotation = true;
  });
  if (hasAnnotation) return false;

  if (!dispatch) return true;

  const ann = N.annotation.create(null, [N.paragraph.createAndFill()!]);
  const insertPos = firstInsertionPointAfterHeader(target.node, target.before, [
    "title",
    "epigraph",
    "image_block",
  ]);
  const tr = state.tr.insert(insertPos, ann);
  dispatch(tr.scrollIntoView());
  return true;
};

/**
 * Append a <text-author> trailer to the enclosing poem / epigraph / cite.
 * Replaces main.js::AddTA (FBE/main.js:2168).
 */
export const addTextAuthor: Command = (state, dispatch) => {
  const { $from } = state.selection;
  const target = findAncestorAny($from, ["poem", "epigraph", "cite"]);
  if (!target) return false;

  // Insert position: end of the container's content.
  const endPos = target.after - 1;

  if (!dispatch) return true;

  const ta = N.text_author.createAndFill()!;
  const tr = state.tr.insert(endPos, ta);
  dispatch(tr.scrollIntoView());
  return true;
};

/**
 * For a container node starting at `containerBefore`, return the position
 * *after* the initial run of children whose type names are in `headerTypes`.
 * Used by addEpigraph / addAnnotation to preserve canonical element order.
 */
function firstInsertionPointAfterHeader(
  container: PMNode,
  containerBefore: number,
  headerTypes: string[],
): number {
  let offset = 1; // enter container
  for (let i = 0; i < container.childCount; i++) {
    const child = container.child(i);
    if (headerTypes.includes(child.type.name)) {
      offset += child.nodeSize;
    } else {
      break;
    }
  }
  return containerBefore + offset;
}

/**
 * Wrap the block range covered by the selection in a <cite>. Empty-line blocks
 * are kept; non-paragraph blocks (subtitles, nested poems, tables) are flattened
 * out because cite's FB2 schema is stricter than arbitrary block content.
 *
 * Matches FBEview.cpp:1048 InsertCite (simplified: we don't do ExpandTxtRange
 * equivalent, relying on ProseMirror's blockRange() to give us the containing
 * blocks).
 */
export const insertCite: Command = (state, dispatch) => {
  const { $from, $to } = state.selection;

  // Parent must allow <cite>: section / annotation / history / poem.
  const parent = findAncestorAny($from, ["section", "poem", "annotation", "history"]);
  if (!parent) return false;

  const range = $from.blockRange($to);
  if (!range) return false;

  // Must be at the parent's depth (not inside a nested poem/cite/etc.).
  const rangeParent = range.parent;
  if (!["section", "poem", "annotation", "history"].includes(rangeParent.type.name)) return false;

  if (!dispatch) return true;

  // Collect paragraphs / empty-lines from the range. Drop incompatible block types.
  const citeChildren: PMNode[] = [];
  for (let i = range.startIndex; i < range.endIndex; i++) {
    const child = rangeParent.child(i);
    if (child.type.name === "paragraph" || child.type.name === "empty_line") {
      citeChildren.push(child);
    } else if (child.type.name === "subtitle") {
      citeChildren.push(child);
    }
    // Skip poem/table/image — they don't fit inside cite's "block" group.
  }
  if (citeChildren.length === 0) {
    citeChildren.push(N.paragraph.createAndFill()!);
  }

  const cite = N.cite.create(null, citeChildren);
  const tr = state.tr.replaceRangeWith(range.start, range.end, cite);
  dispatch(tr.scrollIntoView());
  return true;
};

/**
 * Convert the block range covered by the selection into a <poem><stanza><v>…+.
 *
 * Each paragraph becomes a `<v>` verse. `<empty-line>` blocks split the content
 * into separate stanzas (so "two paragraphs, blank line, two more paragraphs"
 * becomes two stanzas of two verses each).
 *
 * Matches FBEview.cpp:903 InsertPoem (simplified: FBE's original also strips
 * leading/trailing blank lines from each stanza — we replicate that).
 */
export const insertPoem: Command = (state, dispatch) => {
  const { $from, $to } = state.selection;

  const parent = findAncestorAny($from, ["section", "epigraph", "annotation", "history", "cite"]);
  if (!parent) return false;

  const range = $from.blockRange($to);
  if (!range) return false;

  const rangeParent = range.parent;
  if (!["section", "epigraph", "annotation", "history", "cite"].includes(rangeParent.type.name)) return false;

  if (!dispatch) return true;

  // Group paragraphs into stanzas, split by empty-lines.
  const stanzas: PMNode[][] = [[]];
  for (let i = range.startIndex; i < range.endIndex; i++) {
    const child = rangeParent.child(i);
    if (child.type.name === "paragraph") {
      stanzas[stanzas.length - 1].push(N.verse.create(null, child.content));
    } else if (child.type.name === "empty_line") {
      // Start a new stanza if the current one is non-empty.
      if (stanzas[stanzas.length - 1].length > 0) stanzas.push([]);
    }
    // subtitles / tables / nested poems inside the range: skip silently.
  }

  // Drop trailing empty stanza.
  if (stanzas[stanzas.length - 1].length === 0) stanzas.pop();

  // If no verses at all, produce one empty stanza/verse for editing.
  const stanzaNodes: PMNode[] =
    stanzas.length === 0
      ? [N.stanza.create(null, [N.verse.createAndFill()!])]
      : stanzas.map((verses) => N.stanza.create(null, verses));

  const poem = N.poem.create(null, stanzaNodes);
  const tr = state.tr.replaceRangeWith(range.start, range.end, poem);
  dispatch(tr.scrollIntoView());
  return true;
};

/**
 * Insert a `rows × cols` table at the cursor. When `header` is true, the first
 * row uses `<th>` cells. Matches FBEview.cpp:3556 InsertTable.
 *
 * The command returns a factory so the toolbar can bind its dimensions from a
 * dialog. `insertTableCmd()` without arguments yields a sensible 3×3 default.
 */
export function insertTableCmd(rows = 3, cols = 3, header = true): Command {
  return (state, dispatch) => {
    if (rows < 1 || cols < 1) return false;

    const { $from, $to } = state.selection;
    const parent = findAncestorAny($from, [
      "section", "epigraph", "annotation", "history", "cite",
    ]);
    if (!parent) return false;

    // Make sure the insertion point is at the block level of a valid container,
    // not inside an inline run where a <table> would be invalid.
    const range = $from.blockRange($to);
    if (!range) return false;
    if (!["section", "epigraph", "annotation", "history", "cite"].includes(range.parent.type.name)) {
      return false;
    }

    if (!dispatch) return true;

    const table = buildTableNode(rows, cols, header);
    // If there's a non-empty selection across blocks, replace it; otherwise
    // insert at the current block's end so we don't split the caret paragraph.
    let tr = state.tr;
    if (state.selection.empty) {
      tr = tr.insert(range.end, table);
    } else {
      tr = tr.replaceRangeWith(range.start, range.end, table);
    }
    dispatch(tr.scrollIntoView());
    return true;
  };
}

/** Convenience: 3×3 with header row, for menus that want a zero-arg command. */
export const insertTable: Command = insertTableCmd();

function buildTableNode(rows: number, cols: number, header: boolean): PMNode {
  const rowsNodes: PMNode[] = [];
  for (let r = 0; r < rows; r++) {
    const isHeader = header && r === 0;
    const cells: PMNode[] = [];
    for (let c = 0; c < cols; c++) {
      cells.push(
        N.table_cell.create({
          header: isHeader,
          colspan: 1,
          rowspan: 1,
          align: null,
          valign: null,
        }),
      );
    }
    rowsNodes.push(N.table_row.create(null, cells));
  }
  return N.table.create(null, rowsNodes);
}

/**
 * Merge the current container with its immediate next sibling of the same type.
 * Supports section, stanza, and cite containers. Matches main.js:2216 semantics
 * across the four structural combinations plus the cite and stanza special cases.
 *
 * Section combinations:
 *  - flat+flat: concat block content; unwrap nx's title/epigraph/annotation
 *    wrappers (title's paragraphs become subtitles, matching FBE).
 *  - flat+nested: flatten nx's nested sections into cp's block content.
 *  - nested+flat: promote nx's flat content into a new subsection appended to cp.
 *  - nested+nested: concat the nested-section children; nx's header items
 *    (title/epigraph/image/annotation) are dropped to keep the merged section
 *    valid (user can re-add if needed).
 *
 * Stanza: verse lists concatenate.
 * Cite: children concatenate; cp's text_author paragraphs demote to
 *       plain paragraphs (matching FBE's removeAttribute("className") step).
 */
export const mergeContainers: Command = (state, dispatch) => {
  const { $from } = state.selection;
  const target = findAncestorAny($from, ["section", "stanza", "cite"]);
  if (!target) return false;

  // Locate cp's index within its parent; find next sibling of same type.
  const cpDepth = target.depth;
  const parent = $from.node(cpDepth - 1);
  const cpIndex = $from.index(cpDepth - 1);
  const nx = cpIndex + 1 < parent.childCount ? parent.child(cpIndex + 1) : null;
  if (!nx || nx.type.name !== target.node.type.name) return false;

  if (!dispatch) return true;

  const merged = mergeTwo(target.node, nx);
  if (!merged) return false;

  // Replace [cp.start, nx.end] with the merged node.
  const cpBefore = target.before;
  const nxAfter = cpBefore + target.node.nodeSize + nx.nodeSize;
  const tr = state.tr.replaceWith(cpBefore, nxAfter, merged);
  dispatch(tr.scrollIntoView());
  return true;
};

/** Produce a merged node of the same type as cp, combining content with nx. */
function mergeTwo(cp: PMNode, nx: PMNode): PMNode | null {
  switch (cp.type.name) {
    case "section": return mergeSections(cp, nx);
    case "stanza":  return mergeStanzas(cp, nx);
    case "cite":    return mergeCites(cp, nx);
  }
  return null;
}

/** True if a section's top-level children are only nested sections (plus optional title/epigraph/annotation/image headers). */
function isNestedSection(section: PMNode): boolean {
  let hasSection = false;
  let hasFlat = false;
  section.forEach((child) => {
    const n = child.type.name;
    if (n === "section") hasSection = true;
    else if (n === "title" || n === "epigraph" || n === "annotation" || n === "image_block") {
      /* header items — neutral */
    } else hasFlat = true;
  });
  return hasSection && !hasFlat;
}

function mergeSections(cp: PMNode, nx: PMNode): PMNode {
  const cpNested = isNestedSection(cp);
  const nxNested = isNestedSection(nx);

  // Start with cp's own children, then add nx's processed children.
  const children: PMNode[] = [];
  cp.forEach((c) => children.push(c));

  if (cpNested && nxNested) {
    // Both nested: keep cp's headers + sections, append nx's nested sections. Drop nx's headers.
    nx.forEach((c) => {
      if (c.type.name === "section") children.push(c);
    });
  } else if (cpNested && !nxNested) {
    // Nested + flat: wrap nx's flat content in a new section and append.
    const demoted: PMNode[] = [];
    nx.forEach((c) => {
      // Skip nx's headers; keep only its flat block children.
      const n = c.type.name;
      if (n !== "title" && n !== "epigraph" && n !== "annotation" && n !== "image_block") {
        demoted.push(c);
      }
    });
    if (demoted.length === 0) demoted.push(N.paragraph.createAndFill()!);
    children.push(N.section.create(null, demoted));
  } else if (!cpNested && nxNested) {
    // Flat + nested: flatten nx's nested sections into blocks and append to cp.
    nx.forEach((c) => {
      if (c.type.name === "section") {
        flattenSectionInto(c, children);
      }
      // Skip nx's own headers; the nested sections' headers are handled in flattenSectionInto.
    });
  } else {
    // Both flat: append nx's block content, unwrapping title/epigraph/annotation headers.
    nx.forEach((c) => {
      const n = c.type.name;
      if (n === "title") {
        // Title paragraphs become subtitles.
        c.forEach((p) => {
          if (p.type.name === "paragraph") {
            children.push(N.subtitle.create(null, p.content));
          } else {
            children.push(p);
          }
        });
      } else if (n === "epigraph" || n === "annotation") {
        // Unwrap: promote inner blocks as-is.
        c.forEach((inner) => children.push(inner));
      } else if (n === "image_block") {
        children.push(c);
      } else {
        // Flat block content: paragraph / empty_line / subtitle / poem / cite / table / image_block.
        children.push(c);
      }
    });
  }

  return N.section.create(cp.attrs, children);
}

/** For a flat-style merge where we're absorbing a nested section: promote its
 * nested blocks (and recurse for sub-sections) so the result stays flat. */
function flattenSectionInto(section: PMNode, out: PMNode[]): void {
  section.forEach((c) => {
    const n = c.type.name;
    if (n === "title") {
      c.forEach((p) => {
        if (p.type.name === "paragraph") out.push(N.subtitle.create(null, p.content));
        else out.push(p);
      });
    } else if (n === "epigraph" || n === "annotation") {
      c.forEach((inner) => out.push(inner));
    } else if (n === "section") {
      flattenSectionInto(c, out);
    } else {
      out.push(c);
    }
  });
}

function mergeStanzas(cp: PMNode, nx: PMNode): PMNode {
  const children: PMNode[] = [];
  cp.forEach((c) => children.push(c));
  nx.forEach((c) => {
    if (c.type.name === "verse") children.push(c);
    // Drop nx's title/subtitle (cp's header is kept).
  });
  return N.stanza.create(cp.attrs, children);
}

function mergeCites(cp: PMNode, nx: PMNode): PMNode {
  // Demote cp's text_author children to plain paragraphs before merging.
  const children: PMNode[] = [];
  cp.forEach((c) => {
    if (c.type.name === "text_author") {
      children.push(N.paragraph.create(null, c.content));
    } else {
      children.push(c);
    }
  });
  nx.forEach((c) => children.push(c));
  return N.cite.create(cp.attrs, children);
}
