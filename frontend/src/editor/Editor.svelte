<script lang="ts">
  import { onMount } from "svelte";
  import { EditorState } from "prosemirror-state";
  import { EditorView } from "prosemirror-view";
  import { history, redo, undo } from "prosemirror-history";
  import { keymap } from "prosemirror-keymap";
  import { baseKeymap } from "prosemirror-commands";
  import { fb2Schema } from "./schema";

  let container: HTMLDivElement;
  let view: EditorView | undefined;

  onMount(() => {
    const state = EditorState.create({
      schema: fb2Schema,
      plugins: [
        history(),
        keymap({ "Mod-z": undo, "Mod-y": redo, "Mod-Shift-z": redo }),
        keymap(baseKeymap),
      ],
    });
    view = new EditorView(container, { state });
    return () => view?.destroy();
  });
</script>

<div bind:this={container} class="editor" />

<style>
  .editor {
    padding: 1rem 2rem;
    max-width: 800px;
    margin: 0 auto;
    min-height: 100%;
  }
  :global(.ProseMirror) {
    outline: none;
    line-height: 1.6;
  }
  :global(.ProseMirror p) {
    margin: 0 0 0.5em 0;
  }
  :global(.ProseMirror p.subtitle) {
    font-weight: 600;
    font-size: 1.2em;
  }
  :global(.ProseMirror p.text-author) {
    text-align: right;
    font-style: italic;
  }
</style>
