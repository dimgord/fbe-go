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
