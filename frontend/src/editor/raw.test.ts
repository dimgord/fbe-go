/**
 * Round-trip tests for lossless Raw-block / Raw-inline fallback.
 *
 * Verifies that an unknown element parsed into Block.Raw / Inline.Raw on
 * the Go side survives the ProseMirror round-trip:
 *
 *   FictionBook-with-Raw → fb2ToPMDoc → pmDocToFB2 → FictionBook-with-Raw
 *
 * Before Rev 33 the frontend silently dropped any Block / Inline whose
 * only populated field was Raw — this broke the CLAUDE.md "Lossless
 * fallback invariant" at the desktop round-trip boundary.
 */
import { describe, it, expect } from "vitest";
import { fb2ToPMDoc } from "./parse";
import { pmDocToFB2 } from "./serialize";
import type { FictionBook, RawElement } from "../fb2/types";

function minimalBook(blockOrInline: "block" | "inline", raw: RawElement): FictionBook {
  return {
    Description: {
      TitleInfo: { Genres: [], Authors: [], BookTitle: "T", Lang: "en" },
      DocumentInfo: { Authors: [], Date: { Value: "2026-04-22", Text: "x" }, ID: "x", Version: "1.0" },
    },
    Bodies: [{
      Sections: [{
        Blocks: blockOrInline === "block"
          ? [
              { Paragraph: { Children: [{ Text: "before" }] } },
              { Raw: raw },
              { Paragraph: { Children: [{ Text: "after" }] } },
            ]
          : [
              {
                Paragraph: {
                  Children: [
                    { Text: "before " },
                    { Raw: raw },
                    { Text: " after" },
                  ],
                },
              },
            ],
      }],
    }],
  } as unknown as FictionBook;
}

describe("Raw block round-trip", () => {
  const raw: RawElement = {
    XMLName: { Space: "http://www.gribuser.ru/xml/fictionbook/2.0", Local: "empty-lune" },
    Attrs: [],
    Items: [],
  };

  it("preserves Raw block through PM round-trip (no drop)", () => {
    const src = minimalBook("block", raw);
    const out = pmDocToFB2(fb2ToPMDoc(src), src);
    const blocks = out.Bodies[0].Sections![0].Blocks!;
    expect(blocks.length).toBe(3);
    expect(blocks[0].Paragraph).toBeDefined();
    expect(blocks[1].Raw).toBeDefined();
    expect(blocks[1].Raw!.XMLName.Local).toBe("empty-lune");
    expect(blocks[2].Paragraph).toBeDefined();
  });

  it("preserves Raw block attributes and nested items", () => {
    const complex: RawElement = {
      XMLName: { Local: "custom-extension" },
      Attrs: [
        { Name: { Local: "data-source" }, Value: "Flibusta" },
        { Name: { Local: "count" }, Value: "42" },
      ],
      Items: [
        { Text: "extension " },
        { Elem: { XMLName: { Local: "b" }, Attrs: [], Items: [{ Text: "content" }] } },
      ],
    };
    const src = minimalBook("block", complex);
    const out = pmDocToFB2(fb2ToPMDoc(src), src);
    const roundTripped = out.Bodies[0].Sections![0].Blocks![1].Raw!;
    expect(roundTripped.XMLName.Local).toBe("custom-extension");
    expect(roundTripped.Attrs).toHaveLength(2);
    expect(roundTripped.Attrs![0].Value).toBe("Flibusta");
    expect(roundTripped.Items).toHaveLength(2);
    expect(roundTripped.Items![0].Text).toBe("extension ");
    expect(roundTripped.Items![1].Elem?.XMLName.Local).toBe("b");
  });
});

describe("Raw block inside a section mixed with a nested <section>", () => {
  // Regression for Rev 34: FictionBook.xsd strictly requires section content
  // to be (section+ | block+) — not mixed. A real .fb2 that violates this
  // used to round-trip via PM with the block-level raw elements silently
  // dropped (PM schema had the same strict choice). We relaxed PM to
  // (section | block)+ so the round-trip preserves what the file actually
  // contained, and the validator is the one flagging the XSD breach.
  const raw = (name: string): RawElement => ({
    XMLName: { Local: name },
    Attrs: [],
    Items: [],
  });

  it("preserves raw blocks flanking a nested section (was Rev 34 regression)", () => {
    const src: FictionBook = {
      Description: {
        TitleInfo: { Genres: [], Authors: [], BookTitle: "T", Lang: "en" },
        DocumentInfo: { Authors: [], Date: { Value: "2026-04-22", Text: "x" }, ID: "x", Version: "1.0" },
      },
      Bodies: [{
        Sections: [{
          Title: { Children: [{ Paragraph: { Children: [{ Text: "outer" }] } }] },
          Blocks: [
            { Raw: raw("empty-lane") },
          ],
          Sections: [{
            Blocks: [{ Paragraph: { Children: [{ Text: "inner" }] } }],
          }],
        }],
      }],
    } as unknown as FictionBook;
    // The Go side allows storing Blocks AND Sections on the same section
    // (both fields exist on doc.Section) — the XSD choice is enforced at
    // validation time, not at struct level. We lean on that here: feed the
    // frontend a section whose Blocks and Sections are both non-empty, and
    // expect the round-trip not to lose either.
    const out = pmDocToFB2(fb2ToPMDoc(src), src);
    const outer = out.Bodies[0].Sections![0];
    expect(outer.Blocks?.length, "flanking raw block should survive").toBeGreaterThan(0);
    expect(outer.Sections?.length, "nested section should survive").toBeGreaterThan(0);
    const rawBlocks = (outer.Blocks ?? []).filter((b) => b.Raw);
    expect(rawBlocks.map((b) => b.Raw!.XMLName.Local)).toContain("empty-lane");
  });
});

describe("Raw inline round-trip", () => {
  const raw: RawElement = {
    XMLName: { Local: "ruby" },
    Attrs: [
      { Name: { Local: "rb" }, Value: "漢" },
      { Name: { Local: "rt" }, Value: "kan" },
    ],
    Items: [{ Text: "漢" }],
  };

  it("preserves Raw inline inside a paragraph (no drop)", () => {
    const src = minimalBook("inline", raw);
    const out = pmDocToFB2(fb2ToPMDoc(src), src);
    const inlines = out.Bodies[0].Sections![0].Blocks![0].Paragraph!.Children!;

    // Expect at least one Inline with Raw non-null and the text segments around it.
    const rawInline = inlines.find((i) => i.Raw);
    expect(rawInline, "raw inline should survive round-trip").toBeDefined();
    expect(rawInline!.Raw!.XMLName.Local).toBe("ruby");
    expect(rawInline!.Raw!.Attrs).toHaveLength(2);
    expect(rawInline!.Raw!.Items![0].Text).toBe("漢");

    // Surrounding text also present.
    const joinedText = inlines.map((i) => i.Text ?? "").join("");
    expect(joinedText).toContain("before");
    expect(joinedText).toContain("after");
  });
});
