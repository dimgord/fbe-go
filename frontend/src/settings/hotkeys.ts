// Canonical catalog of user-bindable keyboard shortcuts. Mirrors
// internal/fb2/settings/settings.go::DefaultHotkeys — keep action ids and
// default accelerators in sync (the Go side is the source of truth for the
// on-disk format; this file is the source of truth for which actions exist
// and what they do).
//
// Accelerator format on disk (and for display in the UI):
//
//   "Ctrl+Shift+P", "Ctrl+,", "F3", "Shift+F3"
//
// Modifiers appear in canonical order Ctrl, Alt, Shift, Meta separated by `+`.
// The trailing key token is a single printable character (A-Z, 0-9, punctuation)
// or a named key (F1–F12, Enter, Tab, Escape, etc.). We treat `Ctrl` and
// `Cmd`/`Meta` as synonyms at parse time so bindings typed on macOS don't
// have to be re-entered on Linux; ProseMirror's `Mod-` prefix does the same
// at keymap-build time.

export type HotkeyCategory = "File" | "Edit" | "Format" | "Paragraph" | "Blocks" | "Dialogs";

export interface HotkeyAction {
  id: string;
  label: string;
  category: HotkeyCategory;
  /** True when the command is an editor (PM-level) command. False for
      window-level handlers (Save, SaveAs, Find, Replace, Dialogs). */
  editor: boolean;
}

export const HOTKEY_ACTIONS: readonly HotkeyAction[] = [
  // File
  { id: "Save",                 label: "Save",                    category: "File",       editor: false },
  { id: "SaveAs",               label: "Save As…",                category: "File",       editor: false },

  // Edit — search
  { id: "Find",                 label: "Find",                    category: "Edit",       editor: false },
  { id: "Replace",              label: "Find & Replace",          category: "Edit",       editor: false },
  { id: "FindNext",             label: "Find Next",               category: "Edit",       editor: true  },
  { id: "FindPrev",             label: "Find Previous",           category: "Edit",       editor: true  },

  // Format — inline marks
  { id: "ToggleStrong",         label: "Bold",                    category: "Format",     editor: true  },
  { id: "ToggleEmphasis",       label: "Italic",                  category: "Format",     editor: true  },
  { id: "ToggleStrikethrough",  label: "Strikethrough",           category: "Format",     editor: true  },
  { id: "ToggleSub",            label: "Subscript",               category: "Format",     editor: true  },
  { id: "ToggleSup",            label: "Superscript",             category: "Format",     editor: true  },
  { id: "ToggleCode",           label: "Code (inline)",           category: "Format",     editor: true  },

  // Paragraph style
  { id: "StyleNormal",          label: "Normal paragraph",        category: "Paragraph",  editor: true  },
  { id: "StyleSubtitle",        label: "Subtitle",                category: "Paragraph",  editor: true  },
  { id: "StyleTextAuthor",      label: "Text author",             category: "Paragraph",  editor: true  },

  // Blocks
  { id: "InsertEmptyLine",      label: "Empty line",              category: "Blocks",     editor: true  },
  { id: "CloneContainer",       label: "Clone container",         category: "Blocks",     editor: true  },
  { id: "RemoveOuterContainer", label: "Unwrap outer container",  category: "Blocks",     editor: true  },
  { id: "AddTitle",             label: "Add title",               category: "Blocks",     editor: true  },
  { id: "AddEpigraph",          label: "Add epigraph",            category: "Blocks",     editor: true  },
  { id: "AddAnnotation",        label: "Add annotation",          category: "Blocks",     editor: true  },
  { id: "AddTextAuthor",        label: "Append text-author",      category: "Blocks",     editor: true  },
  { id: "InsertCite",           label: "Wrap in cite",            category: "Blocks",     editor: true  },
  { id: "InsertPoem",           label: "Wrap in poem",            category: "Blocks",     editor: true  },
  { id: "InsertTable",          label: "Insert table…",           category: "Blocks",     editor: false },
  { id: "MergeContainers",      label: "Merge with next sibling", category: "Blocks",     editor: true  },

  // Dialogs
  { id: "OpenBinaries",         label: "Binary manager…",         category: "Dialogs",    editor: false },
  { id: "OpenSettings",         label: "Settings…",               category: "Dialogs",    editor: false },
  { id: "OpenHelp",             label: "Help…",                   category: "Dialogs",    editor: false },
];

export const HOTKEY_ACTION_IDS: readonly string[] = HOTKEY_ACTIONS.map((a) => a.id);

const ACTION_BY_ID: Record<string, HotkeyAction> = Object.fromEntries(
  HOTKEY_ACTIONS.map((a) => [a.id, a]),
);

export function findAction(id: string): HotkeyAction | undefined {
  return ACTION_BY_ID[id];
}

/**
 * Parsed shape of an accelerator string.
 */
export interface ParsedAccel {
  ctrl: boolean;
  alt: boolean;
  shift: boolean;
  meta: boolean;
  /** Canonicalized key token: uppercase letters, digits, punctuation as-is,
      named keys title-cased (F3, Enter, Escape). */
  key: string;
}

const NAMED_KEYS = new Set([
  "Enter", "Tab", "Escape", "Space", "Backspace", "Delete", "Insert",
  "Home", "End", "PageUp", "PageDown",
  "ArrowUp", "ArrowDown", "ArrowLeft", "ArrowRight",
  "F1", "F2", "F3", "F4", "F5", "F6", "F7", "F8", "F9", "F10", "F11", "F12",
]);

/** Parse a user-typed accelerator ("ctrl+shift+p", "Cmd-Shift-P", "F3"). */
export function parseAccel(raw: string): ParsedAccel | null {
  if (!raw) return null;
  const tokens = raw
    .split(/[+\-\s]+/)
    .map((t) => t.trim())
    .filter(Boolean);
  if (tokens.length === 0) return null;
  const parsed: ParsedAccel = { ctrl: false, alt: false, shift: false, meta: false, key: "" };
  for (const t of tokens) {
    const lower = t.toLowerCase();
    switch (lower) {
      case "ctrl": case "control": case "cmd": case "command": case "mod":
        // Treat Ctrl / Cmd / Mod as synonyms — Go's on-disk form is `Ctrl`,
        // PM's keymap uses `Mod-` which resolves to Cmd on macOS / Ctrl on
        // Linux at runtime, and we carry one binding across platforms.
        parsed.ctrl = true; break;
      case "alt": case "option": case "opt":
        parsed.alt = true; break;
      case "shift":
        parsed.shift = true; break;
      case "meta": case "super": case "win":
        parsed.meta = true; break;
      default:
        if (parsed.key) return null; // more than one non-modifier token
        parsed.key = canonicalKey(t);
    }
  }
  if (!parsed.key) return null;
  return parsed;
}

function canonicalKey(raw: string): string {
  // Single printable character — uppercase letters, others as-is.
  if (raw.length === 1) {
    return raw.toUpperCase() === raw.toLowerCase() ? raw : raw.toUpperCase();
  }
  // Named key — title-case match against the known set.
  const title = raw.charAt(0).toUpperCase() + raw.slice(1).toLowerCase();
  if (NAMED_KEYS.has(title)) return title;
  // Arrow aliases.
  const arrow = title.startsWith("Arrow") ? title : "Arrow" + title;
  if (NAMED_KEYS.has(arrow)) return arrow;
  // Unknown — keep user's form verbatim; lookup at dispatch time will just miss.
  return raw;
}

/** Canonical on-disk form: "Ctrl+Shift+P", "F3", "Ctrl+,". */
export function formatAccel(a: ParsedAccel | null): string {
  if (!a) return "";
  const parts: string[] = [];
  if (a.ctrl)  parts.push("Ctrl");
  if (a.alt)   parts.push("Alt");
  if (a.shift) parts.push("Shift");
  if (a.meta)  parts.push("Meta");
  parts.push(a.key);
  return parts.join("+");
}

/** Pretty form for UI display. macOS uses ⌘⌥⇧⌃; others use full words. */
export function displayAccel(raw: string, platform: "mac" | "other" = detectPlatform()): string {
  const a = parseAccel(raw);
  if (!a) return "";
  if (platform === "mac") {
    const parts: string[] = [];
    if (a.ctrl)  parts.push("⌘");  // Mod → Cmd on mac
    if (a.alt)   parts.push("⌥");
    if (a.shift) parts.push("⇧");
    if (a.meta)  parts.push("⌃");  // literal Meta → Control on mac
    parts.push(a.key);
    return parts.join("");
  }
  return formatAccel(a);
}

function detectPlatform(): "mac" | "other" {
  if (typeof navigator === "undefined") return "other";
  const p = navigator.platform || navigator.userAgent || "";
  return /mac|iphone|ipad/i.test(p) ? "mac" : "other";
}

/** Convert canonical accel to ProseMirror keymap form ("Mod-Shift-p"). */
export function toPMKey(raw: string): string | null {
  const a = parseAccel(raw);
  if (!a) return null;
  const parts: string[] = [];
  // Ctrl maps to PM's Mod (Cmd on mac, Ctrl elsewhere). If the user explicitly
  // asked for literal Meta, pass that through separately.
  if (a.ctrl)  parts.push("Mod");
  if (a.alt)   parts.push("Alt");
  if (a.shift) parts.push("Shift");
  if (a.meta)  parts.push("Meta");
  // Letter keys: PM expects lowercase, except when Shift is present — then
  // convention varies, but lowercase still works because PM inspects both.
  const key = a.key.length === 1 ? a.key.toLowerCase() : a.key;
  parts.push(key);
  return parts.join("-");
}

/**
 * Build a ParsedAccel from a live KeyboardEvent (for UI capture + runtime
 * dispatch). Returns null for pure modifier presses (Shift by itself, etc.).
 */
export function accelFromEvent(e: KeyboardEvent): ParsedAccel | null {
  const key = e.key;
  if (!key || key === "Shift" || key === "Control" || key === "Alt" || key === "Meta") {
    return null;
  }
  // Normalize letter keys to uppercase so casing doesn't depend on Shift.
  let k = key;
  if (k.length === 1) {
    k = k.toUpperCase();
  } else if (NAMED_KEYS.has(k)) {
    // key already matches NAMED_KEYS casing (e.g. "F3", "Enter").
  } else if (k === " ") {
    k = "Space";
  }
  return {
    ctrl: e.ctrlKey || e.metaKey, // unify Cmd↔Ctrl
    alt: e.altKey,
    shift: e.shiftKey,
    meta: false, // explicit Meta is rare in bindings; Ctrl/Cmd already cover it
    key: k,
  };
}

/** True when `event` matches the accelerator `raw`. */
export function matchesEvent(event: KeyboardEvent, raw: string): boolean {
  const want = parseAccel(raw);
  if (!want) return false;
  const got = accelFromEvent(event);
  if (!got) return false;
  if (want.ctrl !== got.ctrl) return false;
  if (want.alt !== got.alt) return false;
  if (want.shift !== got.shift) return false;
  // Explicit Meta in the binding must match, but if the binding has no Meta
  // bit set we accept either (Ctrl unification covers Cmd on mac).
  if (want.meta && !event.metaKey) return false;
  // Case-insensitive key compare for letters.
  if (want.key.length === 1 && got.key.length === 1) {
    return want.key.toUpperCase() === got.key.toUpperCase();
  }
  return want.key === got.key;
}

/** Find duplicate bindings — returns a map of accel → list of action ids. */
export function findConflicts(hotkeys: Record<string, string>): Record<string, string[]> {
  const byAccel: Record<string, string[]> = {};
  for (const [id, raw] of Object.entries(hotkeys)) {
    if (!raw) continue;
    const a = parseAccel(raw);
    if (!a) continue;
    const canon = formatAccel(a);
    (byAccel[canon] ||= []).push(id);
  }
  const conflicts: Record<string, string[]> = {};
  for (const [k, v] of Object.entries(byAccel)) {
    if (v.length > 1) conflicts[k] = v;
  }
  return conflicts;
}
