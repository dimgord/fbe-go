/**
 * Tests for the structural commands added in Rev 10.
 *
 * Each test builds a minimal PM doc via fb2ToPMDoc, places the cursor using
 * TextSelection.atStart / near a block, invokes the command with a dispatch
 * that captures the new state, and asserts the resulting doc shape.
 */
import { describe, it, expect } from "vitest";
import { EditorState, TextSelection } from "prosemirror-state";
import { Node as PMNode } from "prosemirror-model";
import { fb2Schema } from "./schema";
import { fb2ToPMDoc } from "./parse";
import {
  cloneContainer,
  removeOuterContainer,
  addTitle,
  addEpigraph,
  addAnnotation,
  addTextAuthor,
  insertCite,
  insertPoem,
  insertTableCmd,
  mergeContainers,
} from "./commands";
import { SAMPLE_BOOK } from "../fb2/sample";
import type { FictionBook } from "../fb2/types";

/** Build an EditorState with the cursor at a position satisfying `predicate` (on a resolved pos's parent node name). */
function buildStateWithCursor(
  fb: FictionBook,
  predicate: (ancestors: string[]) => boolean,
): EditorState {
  const doc = fb2ToPMDoc(fb);
  const state = EditorState.create({ schema: fb2Schema, doc });
  let target = -1;
  state.doc.descendants((n, pos, parent) => {
    if (target !== -1) return false;
    if (n.type.name !== "paragraph" && n.type.name !== "verse") return true;
    // Walk up from this paragraph position and collect ancestor names.
    const $pos = state.doc.resolve(pos + 1);
    const ancestors: string[] = [];
    for (let d = $pos.depth; d >= 0; d--) ancestors.push($pos.node(d).type.name);
    if (predicate(ancestors)) {
      target = pos + 1;
      return false;
    }
    return true;
  });
  if (target < 0) throw new Error(`no paragraph matched predicate`);
  return state.apply(state.tr.setSelection(TextSelection.create(state.doc, target)));
}

/** Cursor inside the first <section>'s paragraph content. */
function inFirstSection(fb: FictionBook = SAMPLE_BOOK): EditorState {
  return buildStateWithCursor(fb, (a) => a.includes("section"));
}

/** Apply a command that mutates state and return the resulting doc. */
function apply(state: EditorState, cmd: (s: EditorState, d?: (tr: any) => void) => boolean): PMNode {
  let result: PMNode = state.doc;
  const ok = cmd(state, (tr) => {
    result = state.apply(tr).doc;
  });
  if (!ok) throw new Error("command reported not applicable");
  return result;
}

function countChildrenByName(parent: PMNode, name: string): number {
  let n = 0;
  parent.forEach((c) => {
    if (c.type.name === name) n++;
  });
  return n;
}

/** Find the first section in the doc and return it along with its index in body. */
function findFirstSection(doc: PMNode): { section: PMNode; idxInBody: number } {
  const body = doc.firstChild!;
  let idx = -1;
  body.forEach((c, _, i) => {
    if (idx !== -1) return;
    if (c.type.name === "section") idx = i;
  });
  if (idx < 0) throw new Error("no section in body");
  return { section: body.child(idx), idxInBody: idx };
}

describe("structural commands on SAMPLE_BOOK", () => {
  it("cloneContainer duplicates the enclosing section", () => {
    const state = inFirstSection();
    const body = state.doc.firstChild!;
    const sectionsBefore = countChildrenByName(body, "section");
    const doc = apply(state, cloneContainer);
    const sectionsAfter = countChildrenByName(doc.firstChild!, "section");
    expect(sectionsAfter).toBe(sectionsBefore + 1);
  });

  it("addTitle does nothing if the section already has a title", () => {
    const state = inFirstSection();
    // SAMPLE's first section "Заповіт" already has a title.
    expect(addTitle(state)).toBe(false);
  });

  it("addTitle adds a title to a section that lacks one", () => {
    const fb: FictionBook = {
      Description: SAMPLE_BOOK.Description,
      Bodies: [
        {
          Sections: [{ Blocks: [{ Paragraph: { Children: [{ Text: "body text" }] } }] }],
        },
      ],
    };
    const state = inFirstSection(fb);
    const doc = apply(state, addTitle);
    const { section } = findFirstSection(doc);
    expect(countChildrenByName(section, "title")).toBe(1);
  });

  it("addEpigraph inserts an epigraph inside the current section (after title)", () => {
    const state = inFirstSection();
    const { idxInBody } = findFirstSection(state.doc);
    const doc = apply(state, addEpigraph);
    const section = doc.firstChild!.child(idxInBody);
    expect(section.type.name).toBe("section");
    // After addEpigraph, section children start with title, then the new epigraph.
    let names: string[] = [];
    section.forEach((c) => names.push(c.type.name));
    expect(names[0]).toBe("title");
    expect(names[1]).toBe("epigraph");
  });

  it("addAnnotation inserts an annotation after title/epigraph in the section", () => {
    const state = inFirstSection();
    const { idxInBody } = findFirstSection(state.doc);
    const doc = apply(state, addAnnotation);
    const section = doc.firstChild!.child(idxInBody);
    expect(countChildrenByName(section, "annotation")).toBe(1);
  });

  it("addAnnotation is not applicable if section already has one", () => {
    // Move cursor into the already-annotated Підсекція 1 (nested inside second section).
    const doc = fb2ToPMDoc(SAMPLE_BOOK);
    const state = EditorState.create({ schema: fb2Schema, doc });
    // Find the first annotation, then put cursor inside its parent section's paragraph.
    let sectionWithAnnPos = -1;
    doc.descendants((n, pos) => {
      if (sectionWithAnnPos !== -1) return false;
      if (n.type.name === "annotation") {
        sectionWithAnnPos = pos;
        return false;
      }
    });
    expect(sectionWithAnnPos).toBeGreaterThan(-1);
    // Place the cursor at the annotation content (inside its first paragraph).
    const ann = doc.resolve(sectionWithAnnPos + 1).parent;
    const sel = TextSelection.create(doc, sectionWithAnnPos + 2);
    const stateInAnn = state.apply(state.tr.setSelection(sel));
    expect(addAnnotation(stateInAnn)).toBe(false);
  });

  it("addTextAuthor appends a text-author to the enclosing poem", () => {
    const fb = fb2ToPMDoc(SAMPLE_BOOK);
    const state = EditorState.create({ schema: fb2Schema, doc: fb });
    // Find first poem, position cursor inside its first verse.
    let versePos = -1;
    state.doc.descendants((n, pos) => {
      if (versePos !== -1) return false;
      if (n.type.name === "verse") {
        versePos = pos + 1;
        return false;
      }
    });
    expect(versePos).toBeGreaterThan(0);
    const sel = TextSelection.create(state.doc, versePos);
    const stateAtVerse = state.apply(state.tr.setSelection(sel));

    const before = countTextAuthorInPoem(stateAtVerse.doc);
    const doc = apply(stateAtVerse, addTextAuthor);
    const after = countTextAuthorInPoem(doc);
    expect(after).toBe(before + 1);
  });

  it("removeOuterContainer is NOT applicable to a section with flat block content", () => {
    const state = inFirstSection();
    // "Заповіт" contains paragraphs/poems/etc., not pure nested sections.
    expect(removeOuterContainer(state)).toBe(false);
  });

  it("removeOuterContainer promotes nested sections to parent when outer is wrapper-only", () => {
    // Place cursor inside the second section ("Вкладена секція") which has only sub-sections.
    const fb = SAMPLE_BOOK;
    const doc = fb2ToPMDoc(fb);
    const state = EditorState.create({ schema: fb2Schema, doc });
    // Find the outermost wrapper section (the one whose children are all sections).
    let wrapperStart = -1;
    state.doc.descendants((n, pos) => {
      if (wrapperStart !== -1) return false;
      if (n.type.name === "section") {
        let allSections = true;
        n.forEach((c) => {
          if (c.type.name !== "section" && c.type.name !== "title") allSections = false;
        });
        let hasNestedSection = false;
        n.forEach((c) => { if (c.type.name === "section") hasNestedSection = true; });
        if (allSections && hasNestedSection) {
          wrapperStart = pos;
          return false;
        }
      }
    });
    expect(wrapperStart).toBeGreaterThan(-1);
    // Place the cursor inside the wrapper's title paragraph (inside the wrapper).
    const sel = TextSelection.create(state.doc, wrapperStart + 3);
    const stateAtWrapper = state.apply(state.tr.setSelection(sel));
    const bodyBefore = stateAtWrapper.doc.firstChild!;
    const sectionsBefore = countChildrenByName(bodyBefore, "section");
    const after = apply(stateAtWrapper, removeOuterContainer);
    const sectionsAfter = countChildrenByName(after.firstChild!, "section");
    // The wrapper is removed; its two sub-sections are promoted into the body.
    // Net: -1 wrapper + 2 promoted = +1 section at body level.
    expect(sectionsAfter).toBe(sectionsBefore - 1 + 2);
  });
});

describe("insertCite / insertPoem", () => {
  /** Build a FictionBook with a section containing N simple paragraphs. */
  function sectionOfParagraphs(texts: (string | "empty")[]): FictionBook {
    return {
      Description: SAMPLE_BOOK.Description,
      Bodies: [
        {
          Sections: [
            {
              Title: { Children: [{ Paragraph: { Children: [{ Text: "Test" }] } }] },
              Blocks: texts.map((t) =>
                t === "empty"
                  ? { EmptyLine: {} }
                  : { Paragraph: { Children: [{ Text: t }] } },
              ),
            },
          ],
        },
      ],
    };
  }

  /** Find positions of paragraphs whose direct parent is the first section. */
  function bodyParaPositions(state: EditorState): number[] {
    const out: number[] = [];
    state.doc.descendants((n, pos, parent) => {
      if (n.type.name === "paragraph" && parent?.type.name === "section") out.push(pos);
    });
    return out;
  }

  it("insertCite wraps the selected paragraphs in a <cite>", () => {
    const fb = sectionOfParagraphs(["alpha", "beta", "gamma"]);
    const doc = fb2ToPMDoc(fb);
    const state = EditorState.create({ schema: fb2Schema, doc });
    const p = bodyParaPositions(state); // [alpha, beta, gamma] positions

    // Select beta + gamma.
    const gamma = state.doc.nodeAt(p[2])!;
    const selStart = p[1] + 1; // inside beta
    const selEnd = p[2] + gamma.nodeSize - 1;
    const stateSel = state.apply(
      state.tr.setSelection(TextSelection.create(state.doc, selStart, selEnd)),
    );

    let resultDoc: PMNode = stateSel.doc;
    const ok = insertCite(stateSel, (tr) => { resultDoc = stateSel.apply(tr).doc; });
    expect(ok).toBe(true);

    const section = resultDoc.firstChild!.firstChild!;
    const names: string[] = [];
    section.forEach((c) => names.push(c.type.name));
    expect(names).toEqual(["title", "paragraph", "cite"]);

    let cite: PMNode | null = null;
    section.forEach((c) => { if (c.type.name === "cite") cite = c; });
    expect((cite as any).childCount).toBe(2);
  });

  it("insertPoem converts selected paragraphs to a stanza of verses", () => {
    const fb = sectionOfParagraphs(["line one", "line two", "line three"]);
    const doc = fb2ToPMDoc(fb);
    const state = EditorState.create({ schema: fb2Schema, doc });
    const p = bodyParaPositions(state); // 3 body paragraphs

    const last = state.doc.nodeAt(p[2])!;
    const selStart = p[0] + 1;
    const selEnd = p[2] + last.nodeSize - 1;
    const stateSel = state.apply(
      state.tr.setSelection(TextSelection.create(state.doc, selStart, selEnd)),
    );

    let resultDoc: PMNode = stateSel.doc;
    const ok = insertPoem(stateSel, (tr) => { resultDoc = stateSel.apply(tr).doc; });
    expect(ok).toBe(true);

    const section = resultDoc.firstChild!.firstChild!;
    let poem: PMNode | null = null;
    section.forEach((c) => { if (c.type.name === "poem") poem = c; });
    expect(poem).not.toBeNull();
    expect((poem as any).childCount).toBe(1); // one stanza
    const stanza = (poem as any).firstChild;
    expect(stanza.type.name).toBe("stanza");
    expect(stanza.childCount).toBe(3);
    stanza.forEach((v: PMNode) => { expect(v.type.name).toBe("verse"); });
  });

  it("insertPoem splits stanzas at empty-line blocks", () => {
    const fb = sectionOfParagraphs([
      "stanza1 line1", "stanza1 line2",
      "empty",
      "stanza2 line1", "stanza2 line2",
    ]);
    const doc = fb2ToPMDoc(fb);
    const state = EditorState.create({ schema: fb2Schema, doc });
    const p = bodyParaPositions(state); // 4 body paragraphs (empty-line not included)

    const last = state.doc.nodeAt(p[3])!;
    const selStart = p[0] + 1;
    const selEnd = p[3] + last.nodeSize - 1;
    const stateSel = state.apply(
      state.tr.setSelection(TextSelection.create(state.doc, selStart, selEnd)),
    );

    let resultDoc: PMNode = stateSel.doc;
    const ok = insertPoem(stateSel, (tr) => { resultDoc = stateSel.apply(tr).doc; });
    expect(ok).toBe(true);

    let poem: PMNode | null = null;
    resultDoc.descendants((n) => {
      if (poem) return false;
      if (n.type.name === "poem") { poem = n; return false; }
    });
    expect(poem).not.toBeNull();
    expect((poem as any).childCount).toBe(2); // two stanzas
    // First stanza has 2 verses, second stanza has 2 verses.
    expect((poem as any).child(0).childCount).toBe(2);
    expect((poem as any).child(1).childCount).toBe(2);
  });
});

describe("insertTable", () => {
  function sectionOfParagraphs(texts: string[]): FictionBook {
    return {
      Description: SAMPLE_BOOK.Description,
      Bodies: [
        {
          Sections: [
            {
              Title: { Children: [{ Paragraph: { Children: [{ Text: "Test" }] } }] },
              Blocks: texts.map((t) => ({ Paragraph: { Children: [{ Text: t }] } })),
            },
          ],
        },
      ],
    };
  }

  it("inserts a 3×3 header table at the end of the containing section", () => {
    const fb = sectionOfParagraphs(["one"]);
    const doc = fb2ToPMDoc(fb);
    const state = EditorState.create({ schema: fb2Schema, doc });

    // Place cursor inside the body paragraph.
    let paraPos = -1;
    state.doc.descendants((n, pos, parent) => {
      if (paraPos === -1 && n.type.name === "paragraph" && parent?.type.name === "section") {
        paraPos = pos + 1;
        return false;
      }
    });
    const stateAtPara = state.apply(
      state.tr.setSelection(TextSelection.create(state.doc, paraPos)),
    );

    let result: PMNode = stateAtPara.doc;
    const ok = insertTableCmd(3, 3, true)(stateAtPara, (tr) => {
      result = stateAtPara.apply(tr).doc;
    });
    expect(ok).toBe(true);

    // Locate the inserted table in the first section.
    const section = result.firstChild!.firstChild!;
    let table: PMNode | null = null;
    section.forEach((c) => { if (c.type.name === "table") table = c; });
    expect(table).not.toBeNull();
    const t = table as unknown as PMNode;
    expect(t.childCount).toBe(3);
    // First row is all <th>.
    const firstRow = t.firstChild!;
    expect(firstRow.type.name).toBe("table_row");
    expect(firstRow.childCount).toBe(3);
    firstRow.forEach((cell: PMNode) => {
      expect(cell.type.name).toBe("table_cell");
      expect(cell.attrs.header).toBe(true);
    });
    // Subsequent rows are <td>.
    for (let r = 1; r < t.childCount; r++) {
      const row = t.child(r);
      row.forEach((cell: PMNode) => {
        expect(cell.attrs.header).toBe(false);
      });
    }
  });

  it("refuses to insert inside a <body> (no valid parent)", () => {
    const fb: FictionBook = {
      Description: SAMPLE_BOOK.Description,
      Bodies: [{ Title: { Children: [{ Paragraph: { Children: [{ Text: "t" }] } }] }, Sections: [] }],
    };
    // Force fb2ToPMDoc to produce a body without sections; put cursor in the title.
    const doc = fb2ToPMDoc({
      ...fb,
      Bodies: [{ ...fb.Bodies[0], Sections: [{ Blocks: [{ Paragraph: { Children: [{ Text: "x" }] } }] }] }],
    });
    const state = EditorState.create({ schema: fb2Schema, doc });

    // Place cursor in the body's title paragraph — no section ancestor → command refuses.
    let titlePos = -1;
    state.doc.descendants((n, pos, parent) => {
      if (titlePos === -1 && n.type.name === "paragraph" && parent?.type.name === "title") {
        titlePos = pos + 1;
        return false;
      }
    });
    const stateInTitle = state.apply(
      state.tr.setSelection(TextSelection.create(state.doc, titlePos)),
    );
    expect(insertTableCmd(2, 2, false)(stateInTitle)).toBe(false);
  });

  it("rejects invalid dimensions", () => {
    const fb = sectionOfParagraphs(["x"]);
    const doc = fb2ToPMDoc(fb);
    const state = EditorState.create({ schema: fb2Schema, doc });
    expect(insertTableCmd(0, 3, true)(state)).toBe(false);
    expect(insertTableCmd(3, -1, true)(state)).toBe(false);
  });
});

describe("mergeContainers", () => {
  function twoSections(cp: string[], nx: string[]): FictionBook {
    return {
      Description: SAMPLE_BOOK.Description,
      Bodies: [
        {
          Sections: [
            { Blocks: cp.map((t) => ({ Paragraph: { Children: [{ Text: t }] } })) },
            { Blocks: nx.map((t) => ({ Paragraph: { Children: [{ Text: t }] } })) },
          ],
        },
      ],
    };
  }

  /**
   * Place cursor inside the FIRST top-level section of body[0]. Prefers the
   * section's own title paragraph; falls back to the first flat block; as a
   * last resort descends into a nested section's paragraph.
   */
  function cursorInFirstSection(fb: FictionBook): EditorState {
    const doc = fb2ToPMDoc(fb);
    const state = EditorState.create({ schema: fb2Schema, doc });

    // Find the first top-level section (body's first `section` child).
    const body = state.doc.firstChild!;
    let sectionStart = -1;
    let sectionNode: PMNode | null = null;
    let acc = 1; // enter body
    for (let i = 0; i < body.childCount; i++) {
      const c = body.child(i);
      if (c.type.name === "section") {
        sectionStart = acc;
        sectionNode = c;
        break;
      }
      acc += c.nodeSize;
    }
    if (sectionStart < 0 || !sectionNode) throw new Error("no section found");

    // Walk the section's children, preferring title > flat paragraph > deeper.
    let inner = -1;
    let offset = sectionStart + 1; // enter the section
    for (let i = 0; i < sectionNode.childCount; i++) {
      const c = sectionNode.child(i);
      if (c.type.name === "title") {
        // Title → paragraph.
        let innerOffset = offset + 1;
        c.forEach((pCandidate) => {
          if (inner === -1 && pCandidate.type.name === "paragraph") {
            inner = innerOffset + 1;
          }
        });
        if (inner !== -1) break;
      } else if (c.type.name === "paragraph") {
        inner = offset + 1;
        break;
      }
      offset += c.nodeSize;
    }
    if (inner < 0) {
      // Fall back to the first paragraph anywhere in the section (e.g., nested subsection).
      sectionNode.descendants((n, p) => {
        if (inner === -1 && n.type.name === "paragraph") {
          inner = sectionStart + p + 2; // +2 accounts for the section wrapper entry
          return false;
        }
      });
    }
    if (inner < 0) throw new Error("no cursor target in first section");
    return state.apply(state.tr.setSelection(TextSelection.create(state.doc, inner)));
  }

  function paragraphTexts(node: PMNode): string[] {
    const out: string[] = [];
    node.forEach((c) => {
      if (c.type.name === "paragraph" || c.type.name === "subtitle") out.push(c.textContent);
    });
    return out;
  }

  it("merges two flat sections: concatenated paragraphs, one section", () => {
    const state = cursorInFirstSection(twoSections(["a1", "a2"], ["b1", "b2"]));
    let result: PMNode = state.doc;
    expect(mergeContainers(state, (tr) => { result = state.apply(tr).doc; })).toBe(true);
    const body = result.firstChild!;
    expect(countChildrenByName(body, "section")).toBe(1);
    expect(paragraphTexts(body.firstChild!)).toEqual(["a1", "a2", "b1", "b2"]);
  });

  it("flat+flat: nx's <title> becomes subtitles, <epigraph>/<annotation> unwrap", () => {
    const fb: FictionBook = {
      Description: SAMPLE_BOOK.Description,
      Bodies: [
        {
          Sections: [
            { Blocks: [{ Paragraph: { Children: [{ Text: "cp1" }] } }] },
            {
              Title: { Children: [{ Paragraph: { Children: [{ Text: "Next Title" }] } }] },
              Annotation: { Children: [{ Paragraph: { Children: [{ Text: "Ann text" }] } }] },
              Blocks: [{ Paragraph: { Children: [{ Text: "nx1" }] } }],
            },
          ],
        },
      ],
    };
    const state = cursorInFirstSection(fb);
    let result: PMNode = state.doc;
    expect(mergeContainers(state, (tr) => { result = state.apply(tr).doc; })).toBe(true);
    const section = result.firstChild!.firstChild!;
    // cp1 + subtitle("Next Title") + Ann text + nx1 (order: cp content, then nx's unwrapped content).
    const kinds: string[] = [];
    section.forEach((c) => kinds.push(c.type.name));
    expect(kinds.includes("subtitle")).toBe(true);
    // Find the subtitle and check text.
    let sub: PMNode | null = null;
    section.forEach((c) => { if (!sub && c.type.name === "subtitle") sub = c; });
    expect((sub as any).textContent).toBe("Next Title");
  });

  it("nested+flat: nx flat content is wrapped in a new subsection", () => {
    const fb: FictionBook = {
      Description: SAMPLE_BOOK.Description,
      Bodies: [
        {
          Sections: [
            {
              // cp has only nested sections (nested) plus a title to host the cursor.
              Title: { Children: [{ Paragraph: { Children: [{ Text: "cp-title" }] } }] },
              Sections: [
                { Blocks: [{ Paragraph: { Children: [{ Text: "inner" }] } }] },
              ],
            },
            { Blocks: [{ Paragraph: { Children: [{ Text: "flat nx" }] } }] },
          ],
        },
      ],
    };
    const state = cursorInFirstSection(fb);
    let result: PMNode = state.doc;
    expect(mergeContainers(state, (tr) => { result = state.apply(tr).doc; })).toBe(true);
    const body = result.firstChild!;
    expect(countChildrenByName(body, "section")).toBe(1);
    const merged = body.firstChild!;
    // After merge, merged section should have 2 sub-sections.
    let subSections = 0;
    merged.forEach((c) => { if (c.type.name === "section") subSections++; });
    expect(subSections).toBe(2);
  });

  it("flat+nested: nx's nested sections are flattened into cp's block content", () => {
    const fb: FictionBook = {
      Description: SAMPLE_BOOK.Description,
      Bodies: [
        {
          Sections: [
            { Blocks: [{ Paragraph: { Children: [{ Text: "cp-flat" }] } }] },
            {
              Sections: [
                {
                  Title: { Children: [{ Paragraph: { Children: [{ Text: "InnerTitle" }] } }] },
                  Blocks: [{ Paragraph: { Children: [{ Text: "inner-p" }] } }],
                },
              ],
            },
          ],
        },
      ],
    };
    const state = cursorInFirstSection(fb);
    let result: PMNode = state.doc;
    expect(mergeContainers(state, (tr) => { result = state.apply(tr).doc; })).toBe(true);
    const body = result.firstChild!;
    expect(countChildrenByName(body, "section")).toBe(1);
    const merged = body.firstChild!;
    // After merge: cp-flat, subtitle(InnerTitle), inner-p.
    const texts = paragraphTexts(merged);
    expect(texts).toEqual(["cp-flat", "InnerTitle", "inner-p"]);
  });

  it("nested+nested: concat section children, drop nx's headers", () => {
    const fb: FictionBook = {
      Description: SAMPLE_BOOK.Description,
      Bodies: [
        {
          Sections: [
            {
              Title: { Children: [{ Paragraph: { Children: [{ Text: "cp-title" }] } }] },
              Sections: [{ Blocks: [{ Paragraph: { Children: [{ Text: "cp-sub1" }] } }] }],
            },
            {
              Title: { Children: [{ Paragraph: { Children: [{ Text: "dropped title" }] } }] },
              Sections: [{ Blocks: [{ Paragraph: { Children: [{ Text: "nx-sub1" }] } }] }],
            },
          ],
        },
      ],
    };
    const state = cursorInFirstSection(fb);
    let result: PMNode = state.doc;
    expect(mergeContainers(state, (tr) => { result = state.apply(tr).doc; })).toBe(true);
    const merged = result.firstChild!.firstChild!;
    let subs = 0;
    merged.forEach((c) => { if (c.type.name === "section") subs++; });
    expect(subs).toBe(2);
    // No title with "dropped title" content should survive.
    let foundDropped = false;
    merged.descendants((n) => {
      if (n.type.name === "paragraph" && n.textContent === "dropped title") foundDropped = true;
    });
    expect(foundDropped).toBe(false);
  });

  it("stanzas concatenate their verses", () => {
    const fb: FictionBook = {
      Description: SAMPLE_BOOK.Description,
      Bodies: [
        {
          Sections: [
            {
              Blocks: [
                {
                  Poem: {
                    Stanzas: [
                      { Verses: [{ Children: [{ Text: "v1" }] }, { Children: [{ Text: "v2" }] }] },
                      { Verses: [{ Children: [{ Text: "v3" }] }, { Children: [{ Text: "v4" }] }] },
                    ],
                  },
                },
              ],
            },
          ],
        },
      ],
    };
    const doc = fb2ToPMDoc(fb);
    const state = EditorState.create({ schema: fb2Schema, doc });
    // Position cursor inside the first stanza's first verse.
    let versePos = -1;
    state.doc.descendants((n, pos) => {
      if (versePos === -1 && n.type.name === "verse") { versePos = pos + 1; return false; }
    });
    const stateAtVerse = state.apply(state.tr.setSelection(TextSelection.create(state.doc, versePos)));
    let result: PMNode = stateAtVerse.doc;
    expect(mergeContainers(stateAtVerse, (tr) => { result = stateAtVerse.apply(tr).doc; })).toBe(true);
    // Find poem and count stanzas.
    let poem: PMNode | null = null;
    result.descendants((n) => { if (!poem && n.type.name === "poem") { poem = n; return false; } });
    expect((poem as any).childCount).toBe(1);
    const stanza = (poem as any).firstChild;
    expect(stanza.childCount).toBe(4); // v1..v4
  });

  it("cites concatenate, cp's text-author demotes to paragraph", () => {
    const fb: FictionBook = {
      Description: SAMPLE_BOOK.Description,
      Bodies: [
        {
          Sections: [
            {
              Blocks: [
                {
                  Cite: {
                    Children: [{ Paragraph: { Children: [{ Text: "quote1" }] } }],
                    TextAuthor: [{ Children: [{ Text: "— author1" }] }],
                  },
                },
                {
                  Cite: {
                    Children: [{ Paragraph: { Children: [{ Text: "quote2" }] } }],
                  },
                },
              ],
            },
          ],
        },
      ],
    };
    const doc = fb2ToPMDoc(fb);
    const state = EditorState.create({ schema: fb2Schema, doc });
    let pos = -1;
    state.doc.descendants((n, p) => {
      if (pos === -1 && n.type.name === "cite") { pos = p + 3; return false; }
    });
    const stateInCite = state.apply(state.tr.setSelection(TextSelection.create(state.doc, pos)));
    let result: PMNode = stateInCite.doc;
    expect(mergeContainers(stateInCite, (tr) => { result = stateInCite.apply(tr).doc; })).toBe(true);
    // Only one cite should remain.
    let cites = 0;
    result.descendants((n) => { if (n.type.name === "cite") cites++; });
    expect(cites).toBe(1);
    // The text_author must have been demoted to a paragraph.
    let cite: PMNode | null = null;
    result.descendants((n) => { if (!cite && n.type.name === "cite") { cite = n; return false; } });
    let hasTextAuthor = false;
    (cite as any).forEach((c: PMNode) => {
      if (c.type.name === "text_author") hasTextAuthor = true;
    });
    expect(hasTextAuthor).toBe(false);
    // "— author1" text should now be in a plain paragraph.
    const texts = paragraphTexts(cite as any);
    expect(texts).toContain("— author1");
  });

  it("refuses when no same-type sibling follows", () => {
    const fb = twoSections(["only"], []); // nx has no paragraphs but still a section
    const doc = fb2ToPMDoc(fb);
    const state = EditorState.create({ schema: fb2Schema, doc });
    let pos = -1;
    state.doc.descendants((n, p, parent) => {
      if (pos === -1 && n.type.name === "paragraph" && parent?.type.name === "section") {
        pos = p + 1;
        return false;
      }
    });
    const stateAtFirst = state.apply(state.tr.setSelection(TextSelection.create(state.doc, pos)));
    // A second section DOES follow here (with empty blocks → one filler paragraph).
    expect(mergeContainers(stateAtFirst)).toBe(true);
    // But if cursor is in the SECOND section, no sibling follows → refuse.
    let lastSectionPara = -1;
    state.doc.descendants((n, p, parent) => {
      if (n.type.name === "paragraph" && parent?.type.name === "section") {
        lastSectionPara = p + 1;
      }
    });
    const stateAtLast = state.apply(state.tr.setSelection(TextSelection.create(state.doc, lastSectionPara)));
    expect(mergeContainers(stateAtLast)).toBe(false);
  });
});

/** Count all text-author descendants of poems in the doc. */
function countTextAuthorInPoem(doc: PMNode): number {
  let count = 0;
  doc.descendants((n) => {
    if (n.type.name === "poem") {
      n.forEach((child) => {
        if (child.type.name === "text_author") count++;
      });
    }
  });
  return count;
}
