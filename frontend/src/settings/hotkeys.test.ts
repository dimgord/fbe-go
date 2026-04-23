import { describe, it, expect } from "vitest";
import {
  parseAccel,
  formatAccel,
  toPMKey,
  matchesEvent,
  findConflicts,
  displayAccel,
  HOTKEY_ACTION_IDS,
} from "./hotkeys";

describe("parseAccel", () => {
  it("parses Ctrl+Shift+P", () => {
    const a = parseAccel("Ctrl+Shift+P")!;
    expect(a.ctrl).toBe(true);
    expect(a.shift).toBe(true);
    expect(a.alt).toBe(false);
    expect(a.meta).toBe(false);
    expect(a.key).toBe("P");
  });
  it("treats Ctrl / Cmd / Mod as synonyms", () => {
    expect(formatAccel(parseAccel("cmd+b"))).toBe("Ctrl+B");
    expect(formatAccel(parseAccel("MOD-B"))).toBe("Ctrl+B");
  });
  it("parses function keys", () => {
    expect(formatAccel(parseAccel("F3"))).toBe("F3");
    expect(formatAccel(parseAccel("shift+f3"))).toBe("Shift+F3");
  });
  it("parses punctuation keys", () => {
    expect(formatAccel(parseAccel("Ctrl+,"))).toBe("Ctrl+,");
    expect(formatAccel(parseAccel("ctrl+."))).toBe("Ctrl+.");
  });
  it("rejects multi-token non-modifiers", () => {
    expect(parseAccel("Ctrl+A+B")).toBeNull();
  });
  it("rejects empty / modifier-only strings", () => {
    expect(parseAccel("")).toBeNull();
    expect(parseAccel("Ctrl+Shift")).toBeNull();
  });
});

describe("toPMKey", () => {
  it("maps Ctrl → Mod, letter → lowercase", () => {
    expect(toPMKey("Ctrl+B")).toBe("Mod-b");
    expect(toPMKey("Ctrl+Shift+P")).toBe("Mod-Shift-p");
  });
  it("keeps function keys as-is", () => {
    expect(toPMKey("F3")).toBe("F3");
    expect(toPMKey("Shift+F3")).toBe("Shift-F3");
  });
});

describe("matchesEvent", () => {
  // Vitest runs in node; no DOM KeyboardEvent. We only read .key /
  // .ctrlKey / .metaKey / .altKey / .shiftKey, so a plain object cast works.
  function ev(init: KeyboardEventInit): KeyboardEvent {
    return {
      key: init.key ?? "",
      ctrlKey: init.ctrlKey ?? false,
      metaKey: init.metaKey ?? false,
      altKey: init.altKey ?? false,
      shiftKey: init.shiftKey ?? false,
    } as KeyboardEvent;
  }
  it("matches Ctrl+B against a ctrl-b event", () => {
    expect(matchesEvent(ev({ key: "b", ctrlKey: true }), "Ctrl+B")).toBe(true);
  });
  it("matches Ctrl+B against a cmd-b event (Mod unification)", () => {
    expect(matchesEvent(ev({ key: "b", metaKey: true }), "Ctrl+B")).toBe(true);
  });
  it("rejects when shift state mismatches", () => {
    expect(matchesEvent(ev({ key: "b", ctrlKey: true, shiftKey: true }), "Ctrl+B")).toBe(false);
    expect(matchesEvent(ev({ key: "b", ctrlKey: true }), "Ctrl+Shift+B")).toBe(false);
  });
  it("matches comma / period punctuation", () => {
    expect(matchesEvent(ev({ key: ",", ctrlKey: true }), "Ctrl+,")).toBe(true);
    expect(matchesEvent(ev({ key: ".", ctrlKey: true }), "Ctrl+.")).toBe(true);
  });
  it("ignores pure modifier events", () => {
    expect(matchesEvent(ev({ key: "Shift" }), "Shift+F3")).toBe(false);
  });
});

describe("findConflicts", () => {
  it("reports accels used by 2+ actions", () => {
    const conflicts = findConflicts({
      ToggleStrong: "Ctrl+B",
      Save: "Ctrl+S",
      ToggleEmphasis: "Ctrl+B", // conflict
    });
    expect(conflicts["Ctrl+B"]).toEqual(expect.arrayContaining(["ToggleStrong", "ToggleEmphasis"]));
    expect(conflicts["Ctrl+S"]).toBeUndefined();
  });
  it("ignores empty bindings", () => {
    const conflicts = findConflicts({
      A: "",
      B: "",
    });
    expect(conflicts).toEqual({});
  });
  it("normalizes before comparing (cmd ≡ ctrl)", () => {
    const conflicts = findConflicts({
      A: "Cmd+B",
      B: "Ctrl+B",
    });
    expect(conflicts["Ctrl+B"]).toEqual(["A", "B"]);
  });
});

describe("displayAccel", () => {
  it("renders mac glyphs", () => {
    expect(displayAccel("Ctrl+Shift+P", "mac")).toBe("⌘⇧P");
  });
  it("keeps plain form on non-mac", () => {
    expect(displayAccel("Ctrl+Shift+P", "other")).toBe("Ctrl+Shift+P");
  });
});

describe("action catalog", () => {
  it("ids are unique", () => {
    expect(new Set(HOTKEY_ACTION_IDS).size).toBe(HOTKEY_ACTION_IDS.length);
  });
});
