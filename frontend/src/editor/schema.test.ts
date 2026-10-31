import { describe, expect, it } from "vitest";
import { estimateHeight, fb2Schema } from "./schema";

const N = fb2Schema.nodes;
const para = (chars: number) => N.paragraph.create(null, chars ? fb2Schema.text("x".repeat(chars)) : []);
const section = (...kids: Parameters<typeof N.section.create>[1][]) => N.section.create(null, kids as never);

/** The toDOM spec's attributes: ["div", attrs, 0]. */
const attrsOf = (n: ReturnType<typeof N.section.create>) =>
  (n.type.spec.toDOM!(n) as unknown as [string, Record<string, string>])[1];

describe("estimateHeight", () => {
  it("grows with the text", () => {
    expect(estimateHeight(section(para(10)))).toBeLessThan(estimateHeight(section(para(1000))));
  });

  it("counts an empty paragraph as a line", () => {
    expect(estimateHeight(section(para(0)))).toBeGreaterThan(0);
  });

  it("gives images room", () => {
    const img = N.image_block.create({ href: "#a" });
    expect(estimateHeight(section(para(10), img))).toBeGreaterThan(estimateHeight(section(para(10))));
  });
});

describe("section DOM", () => {
  it("makes a leaf lazy, with its own height estimate", () => {
    const a = attrsOf(section(para(500)));
    expect(a.class).toBe("section leaf");
    expect(a.style).toMatch(/^contain-intrinsic-size: auto \d+px$/);
  });

  // A wrapper skipped by content-visibility keeps a size remembered before its children were
  // laid out, which collapsed a whole book to a few screens.
  it("leaves a wrapper section out of lazy rendering", () => {
    const a = attrsOf(section(section(para(500))));
    expect(a.class).toBe("section");
    expect(a.style).toBeUndefined();
  });
});
