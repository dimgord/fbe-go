import { describe, it, expect } from "vitest";
import { buildOutline } from "./outline";
import { SAMPLE_BOOK } from "../fb2/sample";

describe("buildOutline(SAMPLE_BOOK)", () => {
  const tree = buildOutline(SAMPLE_BOOK);

  it("has one body", () => {
    expect(tree).toHaveLength(1);
    expect(tree[0].kind).toBe("body");
    expect(tree[0].label).toBe("Кобзар");
  });

  it("has two top-level sections under the body", () => {
    expect(tree[0].children).toHaveLength(2);
    expect(tree[0].children[0].label).toBe("Заповіт");
    expect(tree[0].children[1].label).toBe("Вкладена секція");
  });

  it("has two nested sections inside the second section", () => {
    const nested = tree[0].children[1].children;
    expect(nested).toHaveLength(2);
    expect(nested[0].label).toBe("Підсекція 1");
    expect(nested[1].label).toBe("Підсекція 2");
  });

  it("path uniquely identifies each section", () => {
    expect(tree[0].path).toEqual([0]);
    expect(tree[0].children[0].path).toEqual([0, 0]);
    expect(tree[0].children[1].children[0].path).toEqual([0, 1, 0]);
  });

  it("returns empty array for null input", () => {
    expect(buildOutline(null)).toEqual([]);
  });
});
