import type { FictionBook } from "../fb2/types";
import type { EditorView } from "prosemirror-view";

/**
 * FB2 images reference binaries by `l:href="#id"` (or, less commonly, bare
 * `id`). The href form is preserved on rewrite so docs that use the bare
 * style stay that way, though most real-world FB2 uses the `#` prefix.
 */
function hrefMatches(href: string | undefined, id: string): boolean {
  if (!href) return false;
  return href.replace(/^#/, "") === id;
}

function rewriteHref(href: string, oldId: string, newId: string): string {
  if (!hrefMatches(href, oldId)) return href;
  return href.startsWith("#") ? "#" + newId : newId;
}

export interface RenameResult {
  /** Number of refs rewritten inside `fb.Description.*.Coverpage.Images`. */
  coverpageRefs: number;
  /** Number of PM image nodes whose `href` attr was rewritten. */
  editorRefs: number;
  /** Whether the target binary itself was renamed. */
  binaryRenamed: boolean;
}

/**
 * Rename a binary and update every reference to its old ID:
 *   - The binary entry in `fb.Binaries`.
 *   - `Coverpage.Images[].Href` in `Description.TitleInfo` /
 *     `Description.SrcTitleInfo` (the only description sections that can
 *     reference binaries per FB2 schema).
 *   - Every `image_block` / `image_inline` node in the live PM document.
 *
 * Does NOT walk `fb.Bodies` — during an edit session the PM doc is the
 * source of truth for body content; walking `fb.Bodies` would miss edits
 * that haven't been serialized back yet. The PM transaction is dispatched
 * inside a single tr so the whole rename is one undo step.
 *
 * Returns counts; the caller should `fb = fb` to trigger Svelte reactivity
 * on the description mutations (in-place edits don't trip Svelte 4's
 * assignment-based reactivity).
 */
export function renameBinary(
  fb: FictionBook,
  view: EditorView | undefined,
  oldId: string,
  newId: string,
): RenameResult {
  const result: RenameResult = { coverpageRefs: 0, editorRefs: 0, binaryRenamed: false };
  if (!oldId || !newId || oldId === newId) return result;

  if (fb.Binaries) {
    for (const b of fb.Binaries) {
      if (b.ID === oldId) {
        b.ID = newId;
        result.binaryRenamed = true;
      }
    }
  }

  for (const ti of [fb.Description.TitleInfo, fb.Description.SrcTitleInfo]) {
    if (!ti?.Coverpage?.Images) continue;
    for (const img of ti.Coverpage.Images) {
      const updated = rewriteHref(img.Href, oldId, newId);
      if (updated !== img.Href) {
        img.Href = updated;
        result.coverpageRefs++;
      }
    }
  }

  if (view && (view as unknown as { docView: unknown }).docView !== null) {
    const tr = view.state.tr;
    let edits = 0;
    view.state.doc.descendants((node, pos) => {
      if (node.type.name !== "image_block" && node.type.name !== "image_inline") return;
      const href = node.attrs.href as string | undefined;
      if (!href) return;
      const updated = rewriteHref(href, oldId, newId);
      if (updated !== href) {
        tr.setNodeMarkup(pos, node.type, { ...node.attrs, href: updated });
        edits++;
      }
    });
    if (edits > 0) view.dispatch(tr);
    result.editorRefs = edits;
  }

  return result;
}

export interface RefCounts {
  coverpageRefs: number;
  editorRefs: number;
}

/** Count every reference to a binary — used to warn on Delete. */
export function countBinaryRefs(
  fb: FictionBook,
  view: EditorView | undefined,
  id: string,
): RefCounts {
  let coverpageRefs = 0;
  for (const ti of [fb.Description.TitleInfo, fb.Description.SrcTitleInfo]) {
    if (!ti?.Coverpage?.Images) continue;
    for (const img of ti.Coverpage.Images) {
      if (hrefMatches(img.Href, id)) coverpageRefs++;
    }
  }

  let editorRefs = 0;
  if (view && (view as unknown as { docView: unknown }).docView !== null) {
    view.state.doc.descendants((node) => {
      if (node.type.name !== "image_block" && node.type.name !== "image_inline") return;
      if (hrefMatches(node.attrs.href, id)) editorRefs++;
    });
  }

  return { coverpageRefs, editorRefs };
}

/** Returns true if any coverpage across description sections references `id`. */
export function isCoverBinary(fb: FictionBook, id: string): boolean {
  for (const ti of [fb.Description.TitleInfo, fb.Description.SrcTitleInfo]) {
    const first = ti?.Coverpage?.Images?.[0];
    if (first && hrefMatches(first.Href, id)) return true;
  }
  return false;
}
