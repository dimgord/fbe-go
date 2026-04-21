<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { EditorState, type Transaction } from "prosemirror-state";
  import { EditorView } from "prosemirror-view";
  import { history, redo, undo } from "prosemirror-history";
  import { keymap } from "prosemirror-keymap";
  import { baseKeymap } from "prosemirror-commands";
  import { Node as PMNode } from "prosemirror-model";
  import { fb2Schema } from "./schema";
  import { fb2ToPMDoc } from "./parse";
  import {
    toggleStrong, toggleEmphasis, toggleStrikethrough,
    toggleSub, toggleSup, toggleCode, toggleLink,
    styleNormal, styleSubtitle, styleTextAuthor,
    insertEmptyLine,
  } from "./commands";
  import type { FictionBook } from "../fb2/types";

  export let fb: FictionBook | null = null;

  /** Export the EditorView so the toolbar in App.svelte can dispatch commands. */
  export let view: EditorView | undefined = undefined;

  let container: HTMLDivElement;

  function mount(doc: PMNode) {
    view?.destroy();
    const state = EditorState.create({
      schema: fb2Schema,
      doc,
      plugins: [
        history(),
        keymap({
          "Mod-z": undo,
          "Mod-y": redo,
          "Mod-Shift-z": redo,
          "Mod-b": toggleStrong,
          "Mod-i": toggleEmphasis,
          "Mod-Shift-s": toggleStrikethrough,
          "Mod-,": toggleSub,
          "Mod-.": toggleSup,
          "Mod-Shift-c": toggleCode,
        }),
        keymap(baseKeymap),
      ],
    });
    view = new EditorView(container, { state });
  }

  onMount(() => {
    const initial = fb ? fb2ToPMDoc(fb) : fb2Schema.topNodeType.createAndFill()!;
    mount(initial);
  });

  onDestroy(() => view?.destroy());

  $: if (view && fb) {
    const next = fb2ToPMDoc(fb);
    mount(next);
  }

  export function exec(cmd: (state: EditorState, dispatch?: (tr: Transaction) => void) => boolean): void {
    if (!view) return;
    cmd(view.state, view.dispatch);
    view.focus();
  }

  export function execLink(): void {
    if (!view) return;
    const href = prompt("Link URL (leave empty to remove):") ?? "";
    toggleLink(href)(view.state, view.dispatch);
    view.focus();
  }

  // Re-export commands so App.svelte can import-and-bind from the same place.
  export {
    toggleStrong, toggleEmphasis, toggleStrikethrough,
    toggleSub, toggleSup, toggleCode,
    styleNormal, styleSubtitle, styleTextAuthor,
    insertEmptyLine,
    undo, redo,
  };
</script>

<div bind:this={container} class="editor" />

<style>
  .editor {
    padding: 1rem 2rem;
    max-width: 820px;
    margin: 0 auto;
    min-height: 100%;
  }
  :global(.ProseMirror) {
    outline: none;
    line-height: 1.65;
    font-family: "Trebuchet MS", -apple-system, sans-serif;
    font-size: 16px;
  }
  :global(.ProseMirror p) {
    margin: 0 0 0.6em 0;
  }
  :global(.ProseMirror p.empty-line) {
    height: 1em;
  }
  :global(.ProseMirror p.subtitle) {
    font-weight: 600;
    font-size: 1.15em;
    margin-top: 1em;
  }
  :global(.ProseMirror p.text-author) {
    text-align: right;
    font-style: italic;
  }
  :global(.ProseMirror div.title) {
    margin: 1.5em 0 1em 0;
    text-align: center;
  }
  :global(.ProseMirror div.title p) {
    font-size: 1.6em;
    font-weight: 600;
  }
  :global(.ProseMirror div.section) {
    margin-bottom: 1em;
  }
  :global(.ProseMirror div.epigraph) {
    margin: 1em 0 1em 3em;
    font-style: italic;
    color: #444;
  }
  :global(.ProseMirror div.cite) {
    margin: 1em 0 1em 2em;
    padding-left: 1em;
    border-left: 3px solid #ccc;
  }
  :global(.ProseMirror div.annotation) {
    margin: 1em 2em;
    font-size: 0.95em;
    color: #555;
  }
  :global(.ProseMirror div.poem) {
    margin: 1.5em 0 1.5em 2em;
  }
  :global(.ProseMirror div.stanza) {
    margin-bottom: 1em;
  }
  :global(.ProseMirror p.v) {
    margin: 0;
  }
  :global(.ProseMirror div.table) {
    display: table;
    border-collapse: collapse;
    margin: 1em 0;
    border: 1px solid #d0d0c0;
  }
  :global(.ProseMirror div.tr) {
    display: table-row;
  }
  :global(.ProseMirror p.td),
  :global(.ProseMirror p.th) {
    display: table-cell;
    padding: 0.3em 0.7em;
    border: 1px solid #d0d0c0;
    margin: 0;
  }
  :global(.ProseMirror p.th) {
    background: #f0f0ea;
    font-weight: 600;
  }
  :global(.ProseMirror span.code),
  :global(.ProseMirror code) {
    font-family: "SF Mono", Menlo, monospace;
    font-size: 0.92em;
    background: #f5f5ef;
    padding: 0.1em 0.3em;
    border-radius: 3px;
  }
  :global(.ProseMirror a) {
    color: #1a5490;
  }
  :global(.ProseMirror div.image) {
    text-align: center;
    margin: 1em 0;
  }
  :global(.ProseMirror img) {
    max-width: 100%;
    height: auto;
  }
</style>
