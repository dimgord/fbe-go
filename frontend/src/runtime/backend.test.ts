import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("../../wailsjs/go/main/App", () => ({ OpenFile: vi.fn() }));
vi.mock("../../wailsjs/runtime/runtime", () => ({
  BrowserOpenURL: vi.fn(),
  EventsEmit: vi.fn(),
  EventsOn: vi.fn(),
}));

import { backend, setBackend, type Backend } from "./backend";

const global = globalThis as { go?: unknown };

afterEach(() => {
  setBackend(null);
  delete global.go;
});

describe("backend()", () => {
  it("is null without the Wails bridge", () => {
    expect(backend()).toBeNull();
  });

  it("is the gateway once the bridge exists", () => {
    global.go = {};
    expect(backend()?.OpenFile).toBeTypeOf("function");
    expect(backend()?.on).toBeTypeOf("function");
  });

  it("prefers a fake, so flows can be tested without Wails", () => {
    const fake = {} as Backend;
    setBackend(fake);
    expect(backend()).toBe(fake);
  });
});
