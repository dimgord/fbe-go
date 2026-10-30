import { describe, expect, it } from "vitest";
import { clamp } from "./math";
import { basename, dirname } from "./path";

describe("path", () => {
  it("splits POSIX and Windows paths", () => {
    expect(basename("/a/b/c.fb2")).toBe("c.fb2");
    expect(dirname("/a/b/c.fb2")).toBe("/a/b");
    expect(basename("C:\\a\\c.fb2")).toBe("c.fb2");
    expect(dirname("C:\\a\\c.fb2")).toBe("C:\\a");
  });
  it("copes with a bare name", () => {
    expect(basename("c.fb2")).toBe("c.fb2");
    expect(dirname("c.fb2")).toBe("");
  });
});

describe("clamp", () => {
  it("bounds a value", () => {
    expect([clamp(5, 0, 3), clamp(-1, 0, 3), clamp(2, 0, 3)]).toEqual([3, 0, 2]);
  });
});
