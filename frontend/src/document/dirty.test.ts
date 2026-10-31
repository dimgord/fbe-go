import { describe, expect, it } from "vitest";
import { canonicalize, descriptionSnapshot, stableStringify } from "./dirty";
import type { FictionBook } from "../fb2/types";

describe("stableStringify", () => {
  it("ignores key order", () => {
    expect(stableStringify({ a: 1, b: { c: 2, d: 3 } })).toBe(stableStringify({ b: { d: 3, c: 2 }, a: 1 }));
  });

  it("treats null, empty string, array and object as absent", () => {
    expect(stableStringify({ a: "x", b: null, c: "", d: [], e: {}, f: { g: [null] } })).toBe('{"a":"x"}');
  });

  it("keeps zero and false, which are content", () => {
    expect(canonicalize({ n: 0, b: false })).toEqual({ b: false, n: 0 });
  });

  it("sees a real change", () => {
    expect(stableStringify({ a: "x" })).not.toBe(stableStringify({ a: "y" }));
  });
});

describe("descriptionSnapshot", () => {
  const book = (d: object) => ({ Description: d }) as unknown as FictionBook;

  it("does not flag the empty defaults the forms inject on first mount", () => {
    const opened = book({ TitleInfo: { BookTitle: "T", Annotation: null } });
    const visited = book({
      TitleInfo: { BookTitle: "T", Annotation: { Children: [] }, Translators: [], Sequences: [] },
      PublishInfo: { Sequences: [] },
      CustomInfo: [],
    });
    expect(descriptionSnapshot(visited)).toBe(descriptionSnapshot(opened));
  });

  it("is empty for no document", () => {
    expect(descriptionSnapshot(null)).toBe("");
  });
});
