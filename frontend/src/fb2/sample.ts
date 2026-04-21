/**
 * Bundled sample of testdata/blank.fb2 so the frontend renders something in
 * plain `vite dev` mode (when Wails bindings aren't available).
 */
import type { FictionBook } from "./types";

export const SAMPLE_BOOK: FictionBook = {
  Description: {
    TitleInfo: {
      Genres: [{ Value: "sf", Match: "" }],
      Authors: [{ FirstName: "Unknown", LastName: "Author" }],
      BookTitle: "Blank",
      Lang: "en",
    },
    DocumentInfo: {
      Authors: [{ Nickname: "fbe-go" }],
      ProgramUsed: "FictionBook Editor (Go) 0.0.1",
      Date: { Value: "2026-04-21", Text: "21 April 2026" },
      ID: "00000000-0000-0000-0000-000000000000",
      Version: "1.0",
    },
  },
  Bodies: [
    {
      Title: {
        Children: [
          { Paragraph: { Children: [{ Text: "Blank" }] } },
        ],
      },
      Sections: [
        {
          Blocks: [
            { Paragraph: { Children: [
              { Text: "Start writing here… This is a " },
              { Strong: { Children: [{ Text: "bold" }] } },
              { Text: " word and this is " },
              { Emphasis: { Children: [{ Text: "italic" }] } },
              { Text: "." },
            ] } },
            { EmptyLine: {} },
            { Paragraph: { Children: [
              { Text: "Second paragraph with a " },
              { A: { Href: "https://example.com", Type: "", Children: [{ Text: "link" }] } },
              { Text: ", subscript H" },
              { Sub: { Children: [{ Text: "2" }] } },
              { Text: "O, and code: " },
              { Code: { Children: [{ Text: "hello()" }] } },
              { Text: "." },
            ] } },
          ],
        },
      ],
    },
  ],
};
