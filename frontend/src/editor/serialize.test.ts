/**
 * Round-trip tests for parse.ts ↔ serialize.ts.
 *
 * Exercises the frontend-side conversion that happens during Save:
 *   FictionBook → ProseMirror doc → FictionBook
 *
 * If any shape is lost, users would see content disappear on Save.
 */
import { describe, it, expect } from "vitest";
import { fb2ToPMDoc } from "./parse";
import { pmDocToFB2 } from "./serialize";
import { SAMPLE_BOOK } from "../fb2/sample";
import type { FictionBook, Inline } from "../fb2/types";

function roundTrip(fb: FictionBook): FictionBook {
  const node = fb2ToPMDoc(fb);
  return pmDocToFB2(node, fb);
}

/** Flatten all leaf-text from an Inline list in document order. */
function flattenText(inlines: Inline[] | undefined): string {
  if (!inlines) return "";
  let out = "";
  for (const i of inlines) {
    if (i.Text) out += i.Text;
    if (i.Strong?.Children)        out += flattenText(i.Strong.Children);
    if (i.Emphasis?.Children)      out += flattenText(i.Emphasis.Children);
    if (i.Strikethrough?.Children) out += flattenText(i.Strikethrough.Children);
    if (i.Sub?.Children)           out += flattenText(i.Sub.Children);
    if (i.Sup?.Children)           out += flattenText(i.Sup.Children);
    if (i.Code?.Children)          out += flattenText(i.Code.Children);
    if (i.Style?.Children)         out += flattenText(i.Style.Children);
    if (i.A?.Children)             out += flattenText(i.A.Children);
  }
  return out;
}

/** Collect inline mark names used in this inline list (recursive). */
function collectMarks(inlines: Inline[] | undefined, acc = new Set<string>()): Set<string> {
  if (!inlines) return acc;
  for (const i of inlines) {
    if (i.Strong)        { acc.add("strong");        collectMarks(i.Strong.Children, acc); }
    if (i.Emphasis)      { acc.add("emphasis");      collectMarks(i.Emphasis.Children, acc); }
    if (i.Strikethrough) { acc.add("strikethrough"); collectMarks(i.Strikethrough.Children, acc); }
    if (i.Sub)           { acc.add("sub");           collectMarks(i.Sub.Children, acc); }
    if (i.Sup)           { acc.add("sup");           collectMarks(i.Sup.Children, acc); }
    if (i.Code)          { acc.add("code");          collectMarks(i.Code.Children, acc); }
    if (i.Style)         { acc.add("style");         collectMarks(i.Style.Children, acc); }
    if (i.A)             { acc.add("a");             collectMarks(i.A.Children, acc); }
  }
  return acc;
}

describe("serialize round-trip on SAMPLE_BOOK", () => {
  const out = roundTrip(SAMPLE_BOOK);

  it("preserves body count", () => {
    expect(out.Bodies.length).toBe(SAMPLE_BOOK.Bodies.length);
  });

  it("preserves top-level section count", () => {
    expect(out.Bodies[0].Sections.length).toBe(SAMPLE_BOOK.Bodies[0].Sections.length);
  });

  it("preserves nested section count", () => {
    const inSrc = SAMPLE_BOOK.Bodies[0].Sections[1].Sections ?? [];
    const inOut = out.Bodies[0].Sections[1].Sections ?? [];
    expect(inOut.length).toBe(inSrc.length);
  });

  it("preserves body title text", () => {
    const srcTitle = SAMPLE_BOOK.Bodies[0].Title?.Children?.[0].Paragraph?.Children?.[0].Text;
    const outTitle = out.Bodies[0].Title?.Children?.[0].Paragraph?.Children?.[0].Text;
    expect(outTitle).toBe(srcTitle);
    expect(outTitle).toBe("Кобзар");
  });

  it("preserves book-title in description", () => {
    expect(out.Description.TitleInfo.BookTitle).toBe("Кобзар (sample)");
  });

  it("preserves body-level epigraph with text-author", () => {
    expect(out.Bodies[0].Epigraph).toBeDefined();
    const epigraph = out.Bodies[0].Epigraph![0];
    expect(epigraph.TextAuthor?.[0]).toBeDefined();
    expect(flattenText(epigraph.TextAuthor![0].Children)).toBe("— Т.Ш.");
  });

  it("preserves poem with two stanzas and text-author", () => {
    const section0 = out.Bodies[0].Sections[0];
    const poem = section0.Blocks?.find((b) => b.Poem)?.Poem;
    expect(poem).toBeDefined();
    expect(poem!.Stanzas.length).toBe(2);
    expect(poem!.Stanzas[0].Verses.length).toBe(4);
    expect(flattenText(poem!.TextAuthor?.[0]?.Children)).toBe("25 грудня 1845, Переяслав");
    // Verse text is preserved verbatim.
    expect(flattenText(poem!.Stanzas[0].Verses[0].Children)).toBe("Як умру, то поховайте");
  });

  it("preserves inline marks in a paragraph", () => {
    // Sample: "жирний", "курсив", "моноширинний", "посилання".
    const section0 = out.Bodies[0].Sections[0];
    const richParagraph = section0.Blocks!
      .filter((b) => b.Paragraph)
      .find((b) => flattenText(b.Paragraph!.Children).includes("жирний"));
    expect(richParagraph).toBeDefined();
    const marks = collectMarks(richParagraph!.Paragraph!.Children);
    expect(marks.has("strong")).toBe(true);
    expect(marks.has("emphasis")).toBe(true);
    expect(marks.has("code")).toBe(true);
    expect(marks.has("a")).toBe(true); // link
  });

  it("preserves empty-line block", () => {
    const section0 = out.Bodies[0].Sections[0];
    const emptyLine = section0.Blocks?.find((b) => b.EmptyLine);
    expect(emptyLine).toBeDefined();
  });

  it("preserves cite with text-author", () => {
    const section0 = out.Bodies[0].Sections[0];
    const cite = section0.Blocks?.find((b) => b.Cite)?.Cite;
    expect(cite).toBeDefined();
    expect(cite!.TextAuthor?.length).toBeGreaterThan(0);
    expect(flattenText(cite!.TextAuthor![0].Children)).toBe("— I і мертвим, і живим…");
  });

  it("preserves subtitle block", () => {
    const section0 = out.Bodies[0].Sections[0];
    const subtitle = section0.Blocks?.find((b) => b.Subtitle);
    expect(subtitle).toBeDefined();
    expect(flattenText(subtitle!.Subtitle!.Children)).toBe("Таблиця-приклад");
  });

  it("preserves table with th + td cells and sub mark in a cell", () => {
    const section0 = out.Bodies[0].Sections[0];
    const table = section0.Blocks?.find((b) => b.Table)?.Table;
    expect(table).toBeDefined();
    expect(table!.Rows.length).toBe(3); // header + 2 data rows
    expect(table!.Rows[0].Cells?.[0].XMLName?.Local).toBe("th");
    expect(table!.Rows[1].Cells?.[0].XMLName?.Local).toBe("td");
    // H2O in the second row has a <sub> mark on "2".
    const h2oCell = table!.Rows[1].Cells![1];
    const marks = collectMarks(h2oCell.Children);
    expect(marks.has("sub")).toBe(true);
    expect(flattenText(h2oCell.Children)).toContain("H2O");
  });

  it("preserves nested section with annotation", () => {
    const nested = out.Bodies[0].Sections[1];
    expect(nested.Sections?.length).toBe(2);
    const withAnnotation = nested.Sections![0];
    expect(withAnnotation.Annotation).toBeDefined();
    expect(flattenText(withAnnotation.Annotation!.Children?.[0]?.Paragraph?.Children))
      .toContain("Короткий опис підсекції.");
  });

  it("preserves description untouched (title-info, document-info)", () => {
    expect(out.Description).toEqual(SAMPLE_BOOK.Description);
  });
});
