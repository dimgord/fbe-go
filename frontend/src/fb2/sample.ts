/**
 * Rich demo sample so the Phase 0/3 work is visible: every block type renders.
 */
import type { FictionBook, Inline } from "./types";

const t = (s: string): Inline => ({ Text: s });
const strong = (...c: Inline[]): Inline => ({ Strong: { Children: c } });
const em = (...c: Inline[]): Inline => ({ Emphasis: { Children: c } });
const code = (...c: Inline[]): Inline => ({ Code: { Children: c } });
const link = (href: string, ...c: Inline[]): Inline => ({ A: { Href: href, Type: "", Children: c } });
const sub = (...c: Inline[]): Inline => ({ Sub: { Children: c } });

export const SAMPLE_BOOK: FictionBook = {
  Description: {
    TitleInfo: {
      Genres: [{ Value: "sf" }],
      Authors: [{ FirstName: "Тарас", LastName: "Шевченко" }],
      BookTitle: "Кобзар (sample)",
      Lang: "uk",
    },
    DocumentInfo: {
      Authors: [{ Nickname: "fbe-go" }],
      ProgramUsed: "FictionBook Editor (Go) 0.0.3",
      Date: { Value: "2026-04-21", Text: "21 April 2026" },
      ID: "00000000-0000-0000-0000-000000000000",
      Version: "1.0",
    },
  },
  Bodies: [
    {
      Title: {
        Children: [{ Paragraph: { Children: [t("Кобзар")] } }],
      },
      Epigraph: [
        {
          Children: [
            { Paragraph: { Children: [em(t("Борітеся — поборете!"))] } },
          ],
          TextAuthor: [{ Children: [t("— Т.Ш.")] }],
        },
      ],
      Sections: [
        {
          Title: { Children: [{ Paragraph: { Children: [t("Заповіт")] } }] },
          Blocks: [
            { Poem: {
              Stanzas: [
                { Verses: [
                  { Children: [t("Як умру, то поховайте")] },
                  { Children: [t("Мене на могилі,")] },
                  { Children: [t("Серед степу широкого,")] },
                  { Children: [t("На Вкраїні милій.")] },
                ] },
                { Verses: [
                  { Children: [t("Щоб лани широкополі,")] },
                  { Children: [t("І Дніпро, і кручі")] },
                  { Children: [t("Було видно, було чути,")] },
                  { Children: [t("Як реве ревучий.")] },
                ] },
              ],
              TextAuthor: [{ Children: [t("25 грудня 1845, Переяслав")] }],
            } },
            { EmptyLine: {} },
            { Paragraph: { Children: [
              t("Це прозовий параграф із різним форматуванням: "),
              strong(t("жирний")),
              t(", "),
              em(t("курсив")),
              t(", "),
              code(t("моноширинний")),
              t(", і "),
              link("https://example.com", t("посилання")),
              t("."),
            ] } },
            { Cite: {
              Children: [
                { Paragraph: { Children: [t("«Учітеся, брати мої, думайте, читайте»")] } },
              ],
              TextAuthor: [{ Children: [t("— I і мертвим, і живим…")] }],
            } },
            { Subtitle: { Children: [t("Таблиця-приклад")] } },
            { Table: {
              Rows: [
                { Cells: [
                  { XMLName: { Local: "th" }, Children: [t("Елемент")] },
                  { XMLName: { Local: "th" }, Children: [t("Опис")] },
                ] },
                { Cells: [
                  { XMLName: { Local: "td" }, Children: [t("H")] },
                  { XMLName: { Local: "td" }, Children: [t("гідроген, H"), sub(t("2")), t("O")] },
                ] },
                { Cells: [
                  { XMLName: { Local: "td" }, Children: [t("O")] },
                  { XMLName: { Local: "td" }, Children: [t("оксиген")] },
                ] },
              ],
            } },
          ],
        },
        {
          Title: { Children: [{ Paragraph: { Children: [t("Вкладена секція")] } }] },
          Sections: [
            {
              Title: { Children: [{ Paragraph: { Children: [t("Підсекція 1")] } }] },
              Annotation: {
                Children: [
                  { Paragraph: { Children: [em(t("Короткий опис підсекції.")), t(" Друге речення.")] } },
                ],
              },
              Blocks: [
                { Paragraph: { Children: [t("Вміст підсекції 1.")] } },
              ],
            },
            {
              Title: { Children: [{ Paragraph: { Children: [t("Підсекція 2")] } }] },
              Blocks: [
                { Paragraph: { Children: [t("Вміст підсекції 2.")] } },
              ],
            },
          ],
        },
      ],
    },
  ],
};
