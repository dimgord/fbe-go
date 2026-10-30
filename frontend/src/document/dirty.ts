/** Content-only comparison of FictionBook values, for unsaved-changes tracking. */
import type { FictionBook } from "../fb2/types";

/**
 * The Go parser and the TS serializer build the same document with different key order, and
 * Go emits null or "" for absent fields where TS omits them; the description forms also fill in
 * empty defaults on first mount. So keys are sorted, and null, "", [] and {} count as absent:
 * the result depends only on content, not on which side produced the object.
 */
export function canonicalize(v: unknown): unknown {
  if (v === null || v === undefined) return undefined;
  if (typeof v === "string") return v === "" ? undefined : v;
  if (Array.isArray(v)) {
    const out = v.map(canonicalize).filter((x) => x !== undefined);
    return out.length === 0 ? undefined : out;
  }
  if (typeof v === "object") {
    const src = v as Record<string, unknown>;
    const out: Record<string, unknown> = {};
    for (const k of Object.keys(src).sort()) {
      const cv = canonicalize(src[k]);
      if (cv !== undefined) out[k] = cv;
    }
    return Object.keys(out).length === 0 ? undefined : out;
  }
  return v;
}

export function stableStringify(value: unknown): string {
  return JSON.stringify(canonicalize(value)) ?? "";
}

/** The snapshot compared to decide whether description edits are unsaved. */
export function descriptionSnapshot(fb: FictionBook | null): string {
  return stableStringify(fb?.Description);
}
