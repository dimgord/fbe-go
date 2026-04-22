/**
 * TypeScript mirror of internal/fb2/doc/doc.go.
 *
 * When `wails dev` runs, Wails generates bindings at frontend/wailsjs/go/models.ts
 * which supersedes this file. Until then we hand-write the types so the frontend
 * is self-contained and can be developed in plain Vite dev mode.
 */

export interface FictionBook {
  Stylesheets?: Stylesheet[];
  Description: Description;
  Bodies: Body[];
  Binaries?: Binary[];
}

export interface Stylesheet {
  Type: string;
  Content: string;
}

export interface Description {
  TitleInfo?: TitleInfo | null;
  SrcTitleInfo?: TitleInfo | null;
  DocumentInfo: DocumentInfo;
  PublishInfo?: PublishInfo | null;
  CustomInfo?: CustomInfo[] | null;
  Output?: unknown[] | null;
}

export interface TitleInfo {
  Genres: Genre[];
  Authors: Author[];
  BookTitle: string;
  Annotation?: Annotation | null;
  Keywords?: string;
  Date?: DateVal | null;
  Coverpage?: Coverpage | null;
  Lang: string;
  SrcLang?: string;
  Translators?: Author[] | null;
  Sequences?: Sequence[] | null;
}

export interface Genre { Value: string; Match?: string }
export interface Author {
  FirstName?: string;
  MiddleName?: string;
  LastName?: string;
  Nickname?: string;
  HomePage?: string[] | null;
  Email?: string[] | null;
  ID?: string;
}
export interface DateVal { Value?: string; Text?: string }
export interface Coverpage { Images: Image[] }
export interface Sequence { Name: string; Number?: string; Children?: Sequence[] | null }

export interface DocumentInfo {
  Authors: Author[];
  ProgramUsed?: string;
  Date: DateVal;
  SrcURL?: string[] | null;
  SrcOCR?: string;
  ID: string;
  Version: string;
  History?: History | null;
  Publishers?: Author[] | null;
}

export interface PublishInfo {
  BookName?: string;
  Publisher?: string;
  City?: string;
  Year?: string;
  ISBN?: string;
  Sequences?: Sequence[] | null;
}

export interface CustomInfo { InfoType: string; Value: string }

export interface Annotation { ID?: string; Lang?: string; Children?: Block[] }
export interface History { ID?: string; Children?: Block[] }

export interface Body {
  Name?: string;
  Lang?: string;
  Image?: Image | null;
  Title?: Title | null;
  Epigraph?: Epigraph[] | null;
  Sections: Section[];
}

export interface Section {
  ID?: string;
  Title?: Title | null;
  Epigraph?: Epigraph[] | null;
  Image?: Image | null;
  Annotation?: Annotation | null;
  Sections?: Section[] | null;
  Blocks?: Block[] | null;
}

export interface Title { ID?: string; Children?: Block[] }
export interface Epigraph { ID?: string; Children?: Block[]; TextAuthor?: Paragraph[] | null }
export interface Cite { ID?: string; Lang?: string; Children?: Block[]; TextAuthor?: Paragraph[] | null }

export interface Poem {
  ID?: string; Lang?: string;
  Title?: Title | null;
  Epigraph?: Epigraph[] | null;
  Stanzas: Stanza[];
  TextAuthor?: Paragraph[] | null;
  Date?: DateVal | null;
}
export interface Stanza {
  ID?: string;
  Title?: Title | null;
  Subtitle?: Paragraph | null;
  Verses: Paragraph[];
}

/** A Block is a discriminated union — only one of the child fields is non-null. */
export interface Block {
  XMLName?: { Space?: string; Local?: string };
  Paragraph?: Paragraph | null;
  Poem?: Poem | null;
  Subtitle?: Paragraph | null;
  Cite?: Cite | null;
  EmptyLine?: EmptyLine | null;
  Table?: Table | null;
  Image?: Image | null;
}

export interface EmptyLine { ID?: string }
export interface Paragraph {
  ID?: string; Style?: string; Lang?: string;
  Children?: Inline[];
}

export interface Inline {
  XMLName?: { Space?: string; Local?: string };
  Text?: string;
  Strong?: Paragraph | null;
  Emphasis?: Paragraph | null;
  Style?: StyleInline | null;
  A?: Link | null;
  Strikethrough?: Paragraph | null;
  Sub?: Paragraph | null;
  Sup?: Paragraph | null;
  Code?: Paragraph | null;
  Image?: Image | null;
}

export interface StyleInline { Name: string; Children?: Inline[] }
export interface Link { Href: string; Type?: string; Children?: Inline[] }
export interface Image { Href: string; Alt?: string; Title?: string; ID?: string }

export interface Table { ID?: string; Style?: string; Rows: Row[] }
export interface Row { Align?: string; Cells?: Cell[] }
export interface Cell {
  XMLName?: { Space?: string; Local?: string };
  ID?: string; Style?: string;
  ColSpan?: string; RowSpan?: string;
  Align?: string; VAlign?: string;
  Children?: Inline[];
}

export interface Binary { ID: string; ContentType: string; Data: string }
