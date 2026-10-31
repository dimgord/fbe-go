import { describe, expect, it } from "vitest";
import { fb2ToPMDoc } from "../editor/parse";
import { buildOutline, findNodePos } from "./outline";
import { blockAtLine, blockPos, positionAt, xmlBlockLines } from "./position";
import type { Block, FictionBook, Section } from "../fb2/types";

const para = (text: string): Block => ({ Paragraph: { Children: [{ Text: text }] } });
const title = (text: string) => ({ Children: [para(text)] });
const twoLineTitle = { Children: [para("Author"), para("Book")] };
const section = (name: string, body: Block[]): Section => ({ Title: title(name), Body: body });

const book = {
  Description: {},
  Bodies: [
    {
      Title: twoLineTitle,
      Sections: [
        section("One", [para("in one")]),
        section("Two", [
          para("lead"),
          { Section: section("Two A", [para("in two a")]) },
          { Section: section("Two B", [para("in two b")]) },
        ]),
      ],
    },
    { Name: "notes", Title: title("Notes"), Sections: [section("n1", [para("a note")])] },
  ],
} as unknown as FictionBook;

/** Position just inside the paragraph holding `text`. */
function posOf(doc: ReturnType<typeof fb2ToPMDoc>, text: string): number {
  let found = -1;
  doc.descendants((n, pos) => {
    if (found < 0 && n.isText && n.text === text) found = pos;
  });
  expect(found).toBeGreaterThan(0);
  return found;
}

function outlinePaths(nodes: ReturnType<typeof buildOutline>): string[] {
  return nodes.flatMap((n) => [n.path.join("."), ...outlinePaths(n.children)]);
}

describe("positionAt", () => {
  const doc = fb2ToPMDoc(book);

  it("gives the section's outline path and its titles", () => {
    expect(positionAt(doc, posOf(doc, "in one"))).toMatchObject({ path: [0, 0], trail: ["Author Book", "One"] });
    expect(positionAt(doc, posOf(doc, "in two b"))).toMatchObject({
      path: [0, 1, 1],
      trail: ["Author Book", "Two", "Two B"],
    });
  });

  it("counts bodies, and shows a titled body in the trail", () => {
    expect(positionAt(doc, posOf(doc, "a note"))).toMatchObject({
      path: [1, 0],
      trail: ["Notes", "n1"],
    });
  });

  it("stays on the parent in text before the first subsection", () => {
    expect(positionAt(doc, posOf(doc, "lead")).path).toEqual([0, 1]);
  });

  it("only produces paths the outline knows", () => {
    const known = outlinePaths(buildOutline(book));
    for (const text of ["in one", "lead", "in two a", "in two b", "a note"]) {
      expect(known).toContain(positionAt(doc, posOf(doc, text)).path.join("."));
    }
  });

  it("round-trips through findNodePos for every outline node", () => {
    for (const path of outlinePaths(buildOutline(book))) {
      const nums = path.split(".").map(Number);
      const pos = findNodePos(doc, nums);
      expect(pos).not.toBeNull();
      expect(positionAt(doc, pos! + 1).path).toEqual(nums);
    }
    expect(findNodePos(doc, [9])).toBeNull();
    expect(findNodePos(doc, [])).toBeNull();
  });

  it("grows from start to end and stays within 0–100", () => {
    const early = positionAt(doc, posOf(doc, "in one")).percent;
    const late = positionAt(doc, posOf(doc, "a note")).percent;
    expect(early).toBeLessThan(late);
    expect(positionAt(doc, doc.content.size).percent).toBe(100);
    expect(positionAt(doc, 0).percent).toBe(0);
  });
});

describe("xmlBlockLines", () => {
  const doc = fb2ToPMDoc({
    Description: {},
    Bodies: [{
      Sections: [{
        Title: title("Heading"),
        Body: [para("it's first"), para("second"), para("third")],
      }],
    }],
  } as unknown as FictionBook);

  const xml = (extra: string[]) => [
    "<FictionBook>",                       // 1
    "<description><annotation><p>skip</p></annotation></description>", // 2
    "<body>",                              // 3
    "<section>",                           // 4
    "<title><p>Heading</p></title>",       // 5
    "<p>it&#39;s first</p>",               // 6
    ...extra,
    "<p>second</p>",
    "<p>third</p>",
    "</section>",
    "</body>",
    '<binary id="a"><p>nope</p></binary>',
    "</FictionBook>",
  ].join("\n");

  it("pairs blocks with their lines, past the description", () => {
    expect(xmlBlockLines(doc, xml([]))).toEqual([5, 6, 7, 8]);
  });

  // The book that exposed this: one <text-author> inside a preserved element shifted 2,703 blocks.
  it("realigns on text after tags the document does not have as blocks", () => {
    const extra = ["<unknown><p>inside raw</p><text-author>x</text-author></unknown>"];
    expect(xmlBlockLines(doc, xml(extra))).toEqual([5, 6, 8, 9]);
  });

  it("copes with no body", () => {
    expect(xmlBlockLines(doc, "<x/>")).toEqual([]);
  });
});

describe("from XML back to the text", () => {
  const lines = [5, 6, 9, 12];

  it("finds the block at or above a line", () => {
    expect([1, 5, 7, 9, 11, 12, 99].map((l) => blockAtLine(lines, l))).toEqual([0, 0, 1, 2, 2, 3, 3]);
  });

  it("gives a position inside the block, and null past the end", () => {
    const doc = fb2ToPMDoc(book);
    const p = blockPos(doc, 1)!;
    expect(doc.resolve(p).parent.isTextblock).toBe(true);
    expect(blockPos(doc, 9999)).toBeNull();
  });
});
