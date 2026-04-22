<script lang="ts">
  import { onMount, onDestroy, createEventDispatcher } from "svelte";
  import { Schema, Node as PMNode, type Mark } from "prosemirror-model";
  import { EditorState, Plugin } from "prosemirror-state";
  import { EditorView } from "prosemirror-view";
  import { history, redo, undo } from "prosemirror-history";
  import { keymap } from "prosemirror-keymap";
  import { baseKeymap, toggleMark } from "prosemirror-commands";
  import { fb2Schema } from "../editor/schema";
  import type { Annotation, Block, Inline } from "../fb2/types";
  import { cleanPastedHTML, cleanPastedText } from "../editor/paste";

  /** Annotation editor takes a bindable Annotation and owns a small PM instance. */
  export let annotation: Annotation | null | undefined;
  /** Fired on every change so the parent can persist the updated annotation. */
  const dispatch = createEventDispatcher<{ change: Annotation }>();

  // Schema: root accepts annotation-level blocks. Cite/poem/table stay as the
  // full node types from fb2Schema so marks round-trip correctly.
  const nodes = fb2Schema.spec.nodes.update("doc", {
    content: "(paragraph | subtitle | empty_line | cite | poem | table)+",
  });
  const annotationSchema = new Schema({ nodes, marks: fb2Schema.spec.marks });
  const N = annotationSchema.nodes;
  const M = annotationSchema.marks;

  let container: HTMLDivElement;
  let view: EditorView | undefined;
  let ignoreNextUpdate = false;

  function annotationToDoc(a: Annotation | null | undefined): PMNode {
    const kids: PMNode[] = [];
    for (const b of a?.Children ?? []) {
      const n = blockToNode(b);
      if (n) kids.push(n);
    }
    if (kids.length === 0) kids.push(N.paragraph.createAndFill()!);
    return N.doc.create(null, kids);
  }

  function blockToNode(b: Block): PMNode | null {
    if (b.Paragraph) {
      return N.paragraph.create(
        { id: b.Paragraph.ID ?? null, style: b.Paragraph.Style ?? null },
        inlinesToNodes(b.Paragraph.Children ?? []),
      );
    }
    if (b.Subtitle) {
      return N.subtitle.create(null, inlinesToNodes(b.Subtitle.Children ?? []));
    }
    if (b.EmptyLine) {
      return N.empty_line.create();
    }
    // cite/poem/table — rarely used in annotations; skip for MVP (preserved via parent's Raw if needed).
    return null;
  }

  function inlinesToNodes(items: Inline[]): PMNode[] {
    const out: PMNode[] = [];
    for (const i of items) pushInline(i, [], out);
    return out;
  }

  function pushInline(i: Inline, marks: Mark[], out: PMNode[]): void {
    if (i.Text !== undefined && i.Text !== "") {
      out.push(annotationSchema.text(i.Text, marks));
    }
    const wrap = (name: keyof typeof M, attrs: Record<string, unknown>, kids: Inline[]) => {
      const m = M[name].create(attrs);
      for (const c of kids) pushInline(c, marks.concat([m]), out);
    };
    if (i.Strong)        wrap("strong", {}, i.Strong.Children ?? []);
    if (i.Emphasis)      wrap("emphasis", {}, i.Emphasis.Children ?? []);
    if (i.Strikethrough) wrap("strikethrough", {}, i.Strikethrough.Children ?? []);
    if (i.Sub)           wrap("sub", {}, i.Sub.Children ?? []);
    if (i.Sup)           wrap("sup", {}, i.Sup.Children ?? []);
    if (i.Code)          wrap("code", {}, i.Code.Children ?? []);
    if (i.A)             wrap("link", { href: i.A.Href, type: i.A.Type ?? "" }, i.A.Children ?? []);
  }

  function docToAnnotation(doc: PMNode): Annotation {
    const blocks: Block[] = [];
    doc.forEach((n) => {
      if (n.type.name === "paragraph") {
        blocks.push({ Paragraph: { Children: inlinesFromNode(n) } });
      } else if (n.type.name === "subtitle") {
        blocks.push({ Subtitle: { Children: inlinesFromNode(n) } });
      } else if (n.type.name === "empty_line") {
        blocks.push({ EmptyLine: {} });
      }
    });
    return { Children: blocks };
  }

  function inlinesFromNode(node: PMNode): Inline[] {
    const out: Inline[] = [];
    node.forEach((child) => {
      if (child.isText) {
        pushTextWithMarks(child.text ?? "", child.marks, out);
      }
    });
    return out;
  }

  function pushTextWithMarks(text: string, marks: readonly Mark[], out: Inline[]): void {
    if (marks.length === 0) { out.push({ Text: text }); return; }
    const ordered = [...marks].sort((a, b) => markOrder(a) - markOrder(b));
    let inner: Inline = { Text: text };
    for (let i = ordered.length - 1; i >= 0; i--) inner = wrapMark(ordered[i], inner);
    out.push(inner);
  }
  function markOrder(m: Mark): number {
    return ({ strong: 1, emphasis: 2, strikethrough: 3, sub: 4, sup: 5, code: 6, link: 8 } as Record<string, number>)[m.type.name] ?? 99;
  }
  function wrapMark(m: Mark, child: Inline): Inline {
    const c = [child];
    switch (m.type.name) {
      case "strong":        return { Strong: { Children: c } };
      case "emphasis":      return { Emphasis: { Children: c } };
      case "strikethrough": return { Strikethrough: { Children: c } };
      case "sub":           return { Sub: { Children: c } };
      case "sup":           return { Sup: { Children: c } };
      case "code":          return { Code: { Children: c } };
      case "link":          return { A: { Href: m.attrs.href ?? "", Type: m.attrs.type ?? "", Children: c } };
    }
    return child;
  }

  function setupView() {
    view?.destroy();
    const state = EditorState.create({
      schema: annotationSchema,
      doc: annotationToDoc(annotation),
      plugins: [
        history(),
        keymap({
          "Mod-z": undo, "Mod-y": redo, "Mod-Shift-z": redo,
          "Mod-b": toggleMark(M.strong),
          "Mod-i": toggleMark(M.emphasis),
        }),
        keymap(baseKeymap),
        new Plugin({
          view: () => ({
            update(v, prev) {
              if (v.state.doc === prev.doc) return;
              if (ignoreNextUpdate) { ignoreNextUpdate = false; return; }
              dispatch("change", docToAnnotation(v.state.doc));
            },
          }),
        }),
      ],
    });
    view = new EditorView(container, {
      state,
      transformPastedHTML: cleanPastedHTML,
      transformPastedText: cleanPastedText,
    });
  }

  onMount(setupView);
  onDestroy(() => view?.destroy());

  // Reactively rebuild when the parent replaces the annotation prop wholesale
  // (rare — usually they just mutate it through our events).
  let lastAnnotationRef: Annotation | null | undefined = annotation;
  $: if (view && annotation !== lastAnnotationRef) {
    lastAnnotationRef = annotation;
    ignoreNextUpdate = true;
    setupView();
  }
</script>

<div class="annotation-editor" bind:this={container} />

<style>
  .annotation-editor {
    min-height: 5rem;
    padding: 0.6rem 0.8rem;
    border: 1px solid var(--border-input);
    border-radius: 4px;
    background: var(--bg-surface);
    font-family: Georgia, serif;
    line-height: 1.5;
  }
  :global(.annotation-editor .ProseMirror) {
    outline: none;
    min-height: 4rem;
  }
  :global(.annotation-editor p) { margin: 0 0 0.4em 0; }
  :global(.annotation-editor p.subtitle) { font-weight: 600; margin-top: 0.6em; }
  :global(.annotation-editor p.empty-line) { height: 0.8em; }
</style>
