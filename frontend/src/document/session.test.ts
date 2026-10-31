import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { get } from "svelte/store";

vi.mock("../../wailsjs/go/main/App", () => ({}));
vi.mock("../../wailsjs/runtime/runtime", () => ({
  BrowserOpenURL: vi.fn(),
  EventsEmit: vi.fn(),
  EventsOn: vi.fn(),
}));

import { setBackend, type Backend } from "../runtime/backend";
import { createDocumentSession, type SessionHooks } from "./session";
import type { FictionBook } from "../fb2/types";

const book = (title: string) =>
  ({ Description: { TitleInfo: { BookTitle: title } }, Bodies: [{ Sections: [] }] }) as unknown as FictionBook;

function fake(overrides: Partial<Record<keyof Backend, unknown>> = {}) {
  const calls: string[] = [];
  const impl = {
    RecentFiles: async () => [],
    RemoveFromRecent: async (p: string) => { calls.push(`remove ${p}`); },
    ConfirmUnsavedChanges: async () => { calls.push("confirm"); return "discard"; },
    PickFB2ToOpen: async () => "/books/picked.fb2",
    PickFB2ToSave: async () => "/books/saved.fb2",
    PickHTMLToSave: async () => "/books/out.html",
    OpenFile: async () => book("opened"),
    UpdateDocument: async () => { calls.push("update"); },
    SaveFile: async (p: string) => { calls.push(`save ${p}`); },
    ExportHTML: async (p: string) => { calls.push(`export ${p}`); },
    SerializeCurrent: async () => "<xml/>",
    ValidateCurrent: async () => [],
    ParseXMLString: async () => book("from xml"),
    CurrentDocument: async () => null,
    ...overrides,
  };
  setBackend(impl as unknown as Backend);
  return calls;
}

function session(hooks: Partial<SessionHooks> = {}) {
  return createDocumentSession({ liveBody: () => null, onSaved: () => {}, ...hooks });
}

const title = (s: ReturnType<typeof session>) => get(s.doc)?.Description.TitleInfo?.BookTitle;

beforeEach(() => vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout"], shouldAdvanceTime: true }));
afterEach(() => {
  setBackend(null);
  vi.useRealTimers();
});

describe("open", () => {
  it("shows the file and starts clean", async () => {
    fake();
    const s = session();
    await s.open("/books/a.fb2");
    expect(title(s)).toBe("opened");
    expect(get(s.filename)).toBe("a.fb2");
    expect(s.isDirty()).toBe(false);
  });

  it("does not ask when clean, and stops on Cancel when dirty", async () => {
    const calls = fake({ ConfirmUnsavedChanges: async () => { calls.push("confirm"); return "cancel"; } });
    const s = session();
    await s.open("/books/a.fb2");
    expect(calls).not.toContain("confirm");
    s.setBodyDirty(true);
    await s.open("/books/b.fb2");
    expect(calls).toContain("confirm");
    expect(get(s.filename)).toBe("a.fb2");
  });

  it("saves first on Save, and stays if the save did not happen", async () => {
    const calls = fake({ ConfirmUnsavedChanges: async () => "save", PickFB2ToSave: async () => "" });
    const s = session();
    await s.open("/books/a.fb2");
    s.setBodyDirty(true);
    await s.open("/books/b.fb2");
    expect(calls).toContain("save /books/a.fb2");
    expect(get(s.filename)).toBe("b.fb2");

    const unsaved = session();
    await unsaved.loadInitial("");
    unsaved.setBodyDirty(true);
    await unsaved.open("/books/c.fb2");
    expect(get(unsaved.filename)).not.toBe("c.fb2");
  });

  it("drops a recent file that no longer opens, keeping the document", async () => {
    const calls = fake();
    const s = session();
    await s.open("/books/a.fb2");
    setBackend(null);
    const again = fake({ OpenFile: async () => { throw new Error("gone"); } });
    await s.open("/books/dead.fb2");
    expect(again).toContain("remove /books/dead.fb2");
    expect(get(s.error)).toContain("gone");
    expect(title(s)).toBe("opened");
    expect(calls).toEqual([]);
  });

  it("falls back to the sample when a picked file fails", async () => {
    fake({ OpenFile: async () => { throw new Error("bad"); } });
    const s = session();
    await s.open();
    expect(get(s.filename)).toContain("sample");
  });
});

describe("save", () => {
  it("writes the live body, marks clean and tells the editor", async () => {
    const calls = fake();
    const onSaved = vi.fn();
    const live = book("edited");
    const update = vi.fn(async () => {});
    setBackend(null);
    const c2 = fake({ UpdateDocument: update });
    const s = session({ liveBody: () => live, onSaved });
    await s.open("/books/a.fb2");
    s.setBodyDirty(true);
    await s.save(false);
    expect(update).toHaveBeenCalledWith(live);
    expect(c2).toContain("save /books/a.fb2");
    expect(onSaved).toHaveBeenCalled();
    expect(s.isDirty()).toBe(false);
    expect(calls).toEqual([]);
  });

  it("works with no editor mounted, from the description view", async () => {
    const calls = fake();
    const s = session();
    await s.open("/books/a.fb2");
    await s.save(false);
    expect(calls).toContain("save /books/a.fb2");
  });

  it("asks where for Save As and an untitled book, and a cancelled picker changes nothing", async () => {
    const calls = fake();
    const s = session();
    await s.loadInitial("");
    await s.save(false);
    expect(calls).toContain("save /books/saved.fb2");
    expect(get(s.filename)).toBe("saved.fb2");

    const none = fake({ PickFB2ToSave: async () => "" });
    await s.save(true);
    expect(none).toEqual([]);
  });

  it("stays dirty and reports when writing fails", async () => {
    fake({ SaveFile: async () => { throw new Error("disk full"); } });
    const s = session();
    await s.open("/books/a.fb2");
    s.setBodyDirty(true);
    await s.save(false);
    expect(get(s.error)).toBe("disk full");
    expect(s.isDirty()).toBe(true);
  });
});

describe("dirty", () => {
  it("sees a description edit and its undo", async () => {
    fake();
    const s = session();
    await s.open("/books/a.fb2");
    const d = get(s.doc)!;
    d.Description.TitleInfo!.BookTitle = "changed";
    expect(s.isDirty()).toBe(true);
    d.Description.TitleInfo!.BookTitle = "opened";
    expect(s.isDirty()).toBe(false);
  });

  it("counts unapplied XML edits", async () => {
    fake();
    const s = session();
    await s.open("/books/a.fb2");
    s.setXmlDirty(true);
    expect(s.isDirty()).toBe(true);
    s.closePanel();
    expect(s.isDirty()).toBe(false);
  });
});

describe("xml", () => {
  it("validate opens the panel with the XML and errors", async () => {
    const problem = { line: 3, column: 1, message: "bad" };
    fake({ ValidateCurrent: async () => [problem] });
    const s = session();
    await s.open("/books/a.fb2");
    await s.validate();
    expect(get(s.panelOpen)).toBe(true);
    expect(get(s.xmlSource)).toBe("<xml/>");
    expect(get(s.xmlErrors)).toEqual([problem]);
    expect(get(s.status)).toContain("1 error");
  });

  it("applies, then undoes once", async () => {
    fake();
    const s = session();
    await s.open("/books/a.fb2");
    await s.validate();
    await s.applyXml("<new/>");
    expect(title(s)).toBe("from xml");
    expect(get(s.canUndoApply)).toBe(true);
    expect(s.isDirty()).toBe(false);
    await s.undoApply();
    expect(title(s)).toBe("opened");
    expect(get(s.canUndoApply)).toBe(false);
  });

  it("keeps the document when Apply is refused", async () => {
    fake({ ParseXMLString: async () => { throw new Error("Unknown element <autho>"); } });
    const s = session();
    await s.open("/books/a.fb2");
    await s.applyXml("<bad/>");
    expect(get(s.applyError)).toContain("autho");
    expect(title(s)).toBe("opened");
    expect(get(s.canUndoApply)).toBe(false);
  });
});

describe("export", () => {
  it("uses the book's name and sends the live body", async () => {
    let suggested = "";
    const calls = fake({ PickHTMLToSave: async (n: string) => { suggested = n; return "/x.html"; } });
    const s = session();
    await s.open("/books/a.fb2");
    await s.exportHtml();
    expect(suggested).toBe("a.html");
    expect(calls).toEqual(["update", "export /x.html"]);
  });
});

describe("loadInitial", () => {
  it("prefers the launch file, then Go's document, then the sample", async () => {
    fake();
    const launched = session();
    await launched.loadInitial("/books/a.fb2");
    expect(get(launched.filename)).toBe("a.fb2");

    fake({ CurrentDocument: async () => book("in go") });
    const reused = session();
    await reused.loadInitial("");
    expect(title(reused)).toBe("in go");

    fake();
    const blank = session();
    await blank.loadInitial("");
    expect(get(blank.filename)).toContain("sample");
  });
});
