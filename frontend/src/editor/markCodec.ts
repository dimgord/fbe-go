/**
 * The inline marks, in one table: PM mark name, FB2 Inline field, and how attributes cross
 * between them. Array order is the nesting order, outermost first, so that a saved document is
 * stable however the marks were applied.
 */
import type { Mark } from "prosemirror-model";
import type { Inline } from "../fb2/types";

type Wrapper = { Children?: Inline[]; Name?: string; Href?: string; Type?: string };

interface MarkCodec {
  mark: string;
  field: keyof Inline;
  /** Mark attrs from the FB2 element; omitted for plain marks. */
  attrs?: (w: Wrapper) => Record<string, unknown>;
  /** FB2 element fields from the mark; omitted for plain marks. */
  fields?: (m: Mark) => Wrapper;
}

export const MARK_CODECS: readonly MarkCodec[] = [
  { mark: "strong", field: "Strong" },
  { mark: "emphasis", field: "Emphasis" },
  { mark: "strikethrough", field: "Strikethrough" },
  { mark: "sub", field: "Sub" },
  { mark: "sup", field: "Sup" },
  { mark: "code", field: "Code" },
  {
    mark: "style",
    field: "Style",
    attrs: (w) => ({ name: w.Name }),
    fields: (m) => ({ Name: m.attrs.name ?? "" }),
  },
  {
    mark: "link",
    field: "A",
    attrs: (w) => ({ href: w.Href, type: w.Type ?? "" }),
    fields: (m) => ({ Href: m.attrs.href ?? "", Type: m.attrs.type ?? "" }),
  },
];

/** The wrappers set on an Inline, in table order, for walking its nested text. */
export function wrappersOf(i: Inline): { codec: MarkCodec; wrapper: Wrapper }[] {
  return MARK_CODECS.flatMap((codec) => {
    const wrapper = i[codec.field] as Wrapper | null | undefined;
    return wrapper ? [{ codec, wrapper }] : [];
  });
}

const orderOf = (m: Mark) => {
  const i = MARK_CODECS.findIndex((c) => c.mark === m.type.name);
  return i < 0 ? MARK_CODECS.length : i;
};

/** A text run with its marks as nested Inlines; the first mark in table order is outermost. */
export function textToInline(text: string, marks: readonly Mark[]): Inline {
  let inner: Inline = { Text: text };
  for (const m of [...marks].sort((a, b) => orderOf(b) - orderOf(a))) {
    const codec = MARK_CODECS.find((c) => c.mark === m.type.name);
    if (codec) inner = { [codec.field]: { ...codec.fields?.(m), Children: [inner] } };
  }
  return inner;
}

/** Plain text of inlines, through every mark. */
export function plainText(items: Inline[]): string {
  let out = "";
  for (const i of items) {
    if (i.Text) out += i.Text;
    for (const { wrapper } of wrappersOf(i)) out += plainText(wrapper.Children ?? []);
  }
  return out;
}
