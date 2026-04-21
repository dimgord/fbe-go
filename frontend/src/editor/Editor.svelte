<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { EditorState } from "prosemirror-state";
  import { EditorView } from "prosemirror-view";
  import { history, redo, undo } from "prosemirror-history";
  import { keymap } from "prosemirror-keymap";
  import { baseKeymap } from "prosemirror-commands";
  import { Node as PMNode } from "prosemirror-model";
  import { fb2Schema } from "./schema";
  import { fb2ToPMDoc } from "./parse";
  import type { FictionBook } from "../fb2/types";

  export let fb: FictionBook | null = null;

  let container: HTMLDivElement;
  let view: EditorView | undefined;

  function mount(doc: PMNode) {
    view?.destroy();
    const state = EditorState.create({
      schema: fb2Schema,
      doc,
      plugins: [
        history(),
        keymap({ "Mod-z": undo, "Mod-y": redo, "Mod-Shift-z": redo }),
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

  // Reactively remount when the fb prop changes (after Open).
  $: if (view && fb) {
    const next = fb2ToPMDoc(fb);
    mount(next);
  }
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
</style>
