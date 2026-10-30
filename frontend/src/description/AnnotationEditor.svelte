<script lang="ts">
  import { onMount, onDestroy, createEventDispatcher } from "svelte";
  import { EditorState, Plugin } from "prosemirror-state";
  import { EditorView } from "prosemirror-view";
  import { history, redo, undo } from "prosemirror-history";
  import { keymap } from "prosemirror-keymap";
  import { baseKeymap, toggleMark } from "prosemirror-commands";
  import { fb2Schema } from "../editor/schema";
  import { buildAnnotation } from "../editor/parse";
  import { buildAnnotation as serializeAnnotation } from "../editor/serialize";
  import type { Annotation } from "../fb2/types";
  import { cleanPastedHTML, cleanPastedText } from "../editor/paste";

  /** Annotation editor takes a bindable Annotation and owns a small PM instance. */
  export let annotation: Annotation | null | undefined;
  /** Fired on every change so the parent can persist the updated annotation. */
  const dispatch = createEventDispatcher<{ change: Annotation }>();

  // The annotation node itself is the PM root, so this editor shares fb2Schema and the main
  // parse/serialize code: everything an annotation can hold round-trips, Raw included.
  const M = fb2Schema.marks;

  let container: HTMLDivElement;
  let view: EditorView | undefined;
  // Tracks the exact object we last dispatched up to the parent. Parent
  // typically writes that same object back into our `annotation` prop
  // (`info.Annotation = e.detail`), which would otherwise trigger
  // setupView() and destroy the PM state mid-typing. Skipping the rebuild
  // when prop === lastDispatched is the only correct way to break the
  // self-dispatch feedback loop while still handling true external
  // replacement (e.g. opening a different document).
  let lastDispatched: Annotation | null = null;

  function setupView() {
    view?.destroy();
    const state = EditorState.create({
      doc: buildAnnotation(annotation),
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
              const next = serializeAnnotation(v.state.doc);
              lastDispatched = next;
              dispatch("change", next);
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

  // Reactively rebuild ONLY when the parent replaces the annotation prop
  // wholesale (e.g., opening a different document). When the new prop is
  // the exact object we just dispatched, that's our own update bouncing
  // back through the parent's `info.Annotation = e.detail` — silently
  // adopt the reference and skip setupView, otherwise every keystroke
  // would destroy and rebuild the PM view (cursor lost, mid-typing
  // content lost when an unrelated control fires reactivity later).
  let lastAnnotationRef: Annotation | null | undefined = annotation;
  $: if (view && annotation !== lastAnnotationRef) {
    if (annotation === lastDispatched) {
      lastAnnotationRef = annotation;
    } else {
      lastAnnotationRef = annotation;
      setupView();
    }
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
