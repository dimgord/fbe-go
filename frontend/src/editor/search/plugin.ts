import { Plugin, PluginKey, TextSelection, type EditorState, type Transaction } from "prosemirror-state";
import { Decoration, DecorationSet, type EditorView } from "prosemirror-view";
import type { Node as PMNode } from "prosemirror-model";
import { writable, type Readable } from "svelte/store";

export interface SearchFlags {
  caseSensitive: boolean;
  wholeWord: boolean;
  regex: boolean;
}

export interface Match {
  from: number;
  to: number;
}

export interface SearchState {
  pattern: string;
  flags: SearchFlags;
  matches: Match[];
  /** Index into `matches` of the currently-focused hit, or -1 when no matches. */
  active: number;
  /** True when the last `set-query` was a valid pattern; false on regex syntax errors. */
  valid: boolean;
  decorations: DecorationSet;
}

const DEFAULT_FLAGS: SearchFlags = { caseSensitive: false, wholeWord: false, regex: false };

const EMPTY_STATE: SearchState = {
  pattern: "",
  flags: DEFAULT_FLAGS,
  matches: [],
  active: -1,
  valid: true,
  decorations: DecorationSet.empty,
};

export const searchPluginKey = new PluginKey<SearchState>("fbe-search");

type SearchMeta =
  | { type: "set-query"; pattern: string; flags: SearchFlags }
  | { type: "clear" }
  | { type: "next" }
  | { type: "prev" };

/**
 * Mirror of the plugin state as a Svelte store so UI components get reactive
 * updates without having to subscribe to EditorView transactions themselves.
 * The plugin's `view()` hook keeps this in sync with the canonical PM state.
 */
const searchStoreInternal = writable<SearchState>(EMPTY_STATE);
export const searchStore: Readable<SearchState> = { subscribe: searchStoreInternal.subscribe };

function escapeRegex(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

function buildRegex(pattern: string, flags: SearchFlags): RegExp | null {
  if (!pattern) return null;
  try {
    let source = flags.regex ? pattern : escapeRegex(pattern);
    let jsFlags = flags.caseSensitive ? "g" : "gi";
    if (flags.wholeWord) {
      // JS regex `\b` is ASCII-only — `\bслово\b` matches nothing in
      // Cyrillic (or any non-Latin) text because letters outside [A-Za-z0-9_]
      // aren't considered word characters by default. Substitute with
      // Unicode-aware lookarounds requiring `u` flag.
      source = `(?<![\\p{L}\\p{N}_])(?:${source})(?![\\p{L}\\p{N}_])`;
      jsFlags += "u";
    }
    return new RegExp(source, jsFlags);
  } catch {
    return null;
  }
}

/**
 * Walk every text leaf in the doc and collect regex matches, translating
 * local text offsets into absolute PM positions. Matches that span multiple
 * text nodes (e.g. across mark boundaries within a paragraph) are not
 * detected — per-node scanning covers the common case and avoids building
 * a per-block concat map; regex users who need cross-mark matching can opt
 * into a `.*?` pattern.
 */
function scanMatches(doc: PMNode, re: RegExp): Match[] {
  const out: Match[] = [];
  doc.descendants((node, pos) => {
    if (!node.isText || !node.text) return;
    re.lastIndex = 0;
    let m: RegExpExecArray | null;
    while ((m = re.exec(node.text)) !== null) {
      if (m[0].length === 0) {
        // Zero-width match (e.g. `\b` alone) — advance manually so the loop
        // terminates instead of spinning at the same offset.
        re.lastIndex++;
        continue;
      }
      out.push({ from: pos + m.index, to: pos + m.index + m[0].length });
    }
  });
  return out;
}

function buildDecorations(doc: PMNode, matches: Match[], active: number): DecorationSet {
  if (matches.length === 0) return DecorationSet.empty;
  const decos = matches.map((m, i) =>
    Decoration.inline(m.from, m.to, {
      class: i === active ? "search-match search-match-active" : "search-match",
    }),
  );
  return DecorationSet.create(doc, decos);
}

function nextActiveForCursor(matches: Match[], cursor: number): number {
  if (matches.length === 0) return -1;
  const idx = matches.findIndex((m) => m.from >= cursor);
  return idx === -1 ? 0 : idx;
}

export function searchPlugin(): Plugin<SearchState> {
  return new Plugin<SearchState>({
    key: searchPluginKey,
    state: {
      init: () => EMPTY_STATE,
      apply(tr: Transaction, prev: SearchState, _oldState: EditorState, newState: EditorState): SearchState {
        const meta = tr.getMeta(searchPluginKey) as SearchMeta | undefined;

        // Doc edit with a live query → rescan so highlights + counts stay fresh.
        if (tr.docChanged && prev.pattern && !meta) {
          const re = buildRegex(prev.pattern, prev.flags);
          if (!re) return prev;
          const matches = scanMatches(newState.doc, re);
          const active = matches.length === 0
            ? -1
            : Math.min(Math.max(prev.active, 0), matches.length - 1);
          return {
            ...prev,
            matches,
            active,
            decorations: buildDecorations(newState.doc, matches, active),
          };
        }

        if (!meta) {
          // No search-related meta — just map existing decorations through
          // the transaction so they follow any structural changes.
          return { ...prev, decorations: prev.decorations.map(tr.mapping, tr.doc) };
        }

        switch (meta.type) {
          case "set-query": {
            if (!meta.pattern) {
              return { ...EMPTY_STATE, flags: meta.flags };
            }
            const re = buildRegex(meta.pattern, meta.flags);
            if (!re) {
              return {
                pattern: meta.pattern,
                flags: meta.flags,
                matches: [],
                active: -1,
                valid: false,
                decorations: DecorationSet.empty,
              };
            }
            const matches = scanMatches(newState.doc, re);
            const active = nextActiveForCursor(matches, newState.selection.from);
            return {
              pattern: meta.pattern,
              flags: meta.flags,
              matches,
              active,
              valid: true,
              decorations: buildDecorations(newState.doc, matches, active),
            };
          }
          case "clear":
            return { ...EMPTY_STATE, flags: prev.flags };
          case "next": {
            if (prev.matches.length === 0) return prev;
            const active = (prev.active + 1) % prev.matches.length;
            return {
              ...prev,
              active,
              decorations: buildDecorations(newState.doc, prev.matches, active),
            };
          }
          case "prev": {
            if (prev.matches.length === 0) return prev;
            const active = (prev.active - 1 + prev.matches.length) % prev.matches.length;
            return {
              ...prev,
              active,
              decorations: buildDecorations(newState.doc, prev.matches, active),
            };
          }
        }
        return prev;
      },
    },
    props: {
      decorations(state) {
        return searchPluginKey.getState(state)?.decorations ?? DecorationSet.empty;
      },
    },
    view() {
      return {
        update(view, prevEditorState) {
          const cur = searchPluginKey.getState(view.state);
          const old = searchPluginKey.getState(prevEditorState);
          if (cur && cur !== old) {
            searchStoreInternal.set(cur);
          }
        },
        destroy() {
          searchStoreInternal.set(EMPTY_STATE);
        },
      };
    },
  });
}

// ── Imperative commands. The SearchBar component calls these directly on the
//    EditorView; it's cleaner than wrapping every dispatch in a PM command. ──

export function setSearch(view: EditorView, pattern: string, flags: SearchFlags): void {
  view.dispatch(view.state.tr.setMeta(searchPluginKey, { type: "set-query", pattern, flags }));
}

export function clearSearch(view: EditorView): void {
  view.dispatch(view.state.tr.setMeta(searchPluginKey, { type: "clear" }));
}

export function findNext(view: EditorView): boolean {
  const st = searchPluginKey.getState(view.state);
  if (!st || st.matches.length === 0) return false;
  view.dispatch(view.state.tr.setMeta(searchPluginKey, { type: "next" }));
  scrollActiveIntoView(view);
  return true;
}

export function findPrev(view: EditorView): boolean {
  const st = searchPluginKey.getState(view.state);
  if (!st || st.matches.length === 0) return false;
  view.dispatch(view.state.tr.setMeta(searchPluginKey, { type: "prev" }));
  scrollActiveIntoView(view);
  return true;
}

function scrollActiveIntoView(view: EditorView): void {
  // Read the state AFTER dispatch — active has just advanced.
  const st = searchPluginKey.getState(view.state);
  if (!st || st.active < 0) return;
  const m = st.matches[st.active];
  if (!m) return;

  // Move selection so typing / replace target the hit.
  view.dispatch(view.state.tr.setSelection(TextSelection.create(view.state.doc, m.from, m.to)));

  // PM's built-in `tr.scrollIntoView()` scrolls whatever container IT
  // considers scrollable, which in our layout is typically the wrong one
  // — `<section>` wraps the editor and owns the scrollbar, not
  // `view.dom`. Walk up until we find a scrollable ancestor (same idiom
  // as Editor.scrollToPath) and nudge it manually. Only scroll when the
  // hit is near the viewport edges so already-visible matches don't cause
  // the page to jump on every ▶ click.
  const coords = view.coordsAtPos(m.from);
  let el: HTMLElement | null = view.dom as HTMLElement;
  while (el && el.scrollHeight <= el.clientHeight) el = el.parentElement;
  if (!el) return;

  const rect = el.getBoundingClientRect();
  const MARGIN = 80;
  if (coords.top < rect.top + MARGIN) {
    el.scrollTop += coords.top - rect.top - MARGIN;
  } else if (coords.bottom > rect.bottom - MARGIN) {
    el.scrollTop += coords.bottom - rect.bottom + MARGIN;
  }
}

/**
 * Replace the currently-active match with `replacement`. Moves the active
 * pointer forward to what was the next match (now at the same index, because
 * the replacement invalidated the old one). Returns true if a replacement
 * happened.
 */
export function replaceActive(view: EditorView, replacement: string): boolean {
  const st = searchPluginKey.getState(view.state);
  if (!st || st.active < 0) return false;
  const m = st.matches[st.active];
  if (!m) return false;
  // tr.insertText handles both empty-replacement (pure delete) and non-empty
  // (delete + insert) without the schema issue of constructing an empty text
  // node manually.
  view.dispatch(view.state.tr.insertText(replacement, m.from, m.to));
  // The doc-edit branch of `apply` above re-scanned matches and clamped
  // `active` — that leaves it pointing at what used to be the next match, so
  // the user can chain Replace → Replace → … on incremental hits.
  scrollActiveIntoView(view);
  return true;
}

/**
 * Replace every match in the document with `replacement`. Returns the number
 * of replacements performed. Applies replacements in reverse document order
 * so earlier positions stay valid as later ones shrink/grow.
 */
export function replaceAll(view: EditorView, replacement: string): number {
  const st = searchPluginKey.getState(view.state);
  if (!st || st.matches.length === 0) return 0;
  const tr = view.state.tr;
  const ordered = [...st.matches].sort((a, b) => b.from - a.from);
  for (const m of ordered) {
    tr.insertText(replacement, m.from, m.to);
  }
  view.dispatch(tr);
  return ordered.length;
}
