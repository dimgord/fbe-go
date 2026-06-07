<script lang="ts">
  /*
   * CodeMirror 6 wrapper for editing raw FB2 XML.
   *
   * Mounted by ValidationPanel when the user clicks "Edit XML". The
   * editor takes the current XML source as its initial doc, lets the
   * user freely edit it, and exposes getXml() + a "dirty" event for
   * the parent panel.
   *
   * Theme tracks the app's effective theme (light/dark) via the `theme`
   * prop; the CM6 instance is rebuilt when theme flips because CM6's
   * theme is part of EditorState, not a CSS-toggle. Cheap to do — the
   * panel is rarely visible during theme switches.
   *
   * Why raw CM6 (not a Svelte wrapper lib): matches this codebase's
   * raw-ProseMirror style and keeps deps minimal — only the @codemirror
   * scoped packages we actually use.
   */
  import { onMount, onDestroy, createEventDispatcher } from "svelte";
  import { EditorView, keymap, lineNumbers, highlightActiveLine } from "@codemirror/view";
  import { EditorState, Compartment } from "@codemirror/state";
  import { xml } from "@codemirror/lang-xml";
  import { defaultKeymap, history, historyKeymap, indentWithTab } from "@codemirror/commands";
  import {
    bracketMatching,
    foldGutter,
    foldKeymap,
    indentOnInput,
    syntaxHighlighting,
    defaultHighlightStyle,
  } from "@codemirror/language";
  import { oneDark } from "@codemirror/theme-one-dark";

  /** Initial XML text. Captured at mount; later prop changes are
   *  ignored on purpose so the user's in-progress edits aren't blown
   *  away by every SerializeCurrent re-run upstream. */
  export let initial: string = "";

  /** "light" | "dark" — drives CM6 theme selection. */
  export let theme: "light" | "dark" = "light";

  const dispatch = createEventDispatcher<{
    /** Fired on every edit; payload is the current full document.
     *  Parent uses this to track dirty state without polling. */
    change: string;
  }>();

  let host: HTMLDivElement | undefined;
  let view: EditorView | undefined;
  const themeCompartment = new Compartment();

  function buildExtensions() {
    return [
      lineNumbers(),
      highlightActiveLine(),
      foldGutter(),
      history(),
      indentOnInput(),
      bracketMatching(),
      syntaxHighlighting(defaultHighlightStyle, { fallback: true }),
      xml(),
      keymap.of([...defaultKeymap, ...historyKeymap, ...foldKeymap, indentWithTab]),
      EditorView.lineWrapping,
      EditorView.updateListener.of((u) => {
        if (u.docChanged) dispatch("change", u.state.doc.toString());
      }),
      themeCompartment.of(theme === "dark" ? oneDark : []),
    ];
  }

  /** Read the current document out of CM6. Called by the parent on Apply. */
  export function getXml(): string {
    return view?.state.doc.toString() ?? initial;
  }

  /** Move the caret to a 1-based line number (used when user clicks an
   *  XSD error). Best-effort — no-op if the line is out of range or
   *  the editor isn't ready yet. */
  export function gotoLine(n: number) {
    if (!view) return;
    const doc = view.state.doc;
    if (n < 1 || n > doc.lines) return;
    const line = doc.line(n);
    view.dispatch({
      selection: { anchor: line.from },
      scrollIntoView: true,
    });
    view.focus();
  }

  /** Replace the entire doc — used when the panel resets after an
   *  external SerializeCurrent (e.g., after Apply succeeded and the
   *  user did another body-edit + Validate). */
  export function setDoc(next: string) {
    if (!view) {
      initial = next;
      return;
    }
    view.dispatch({
      changes: { from: 0, to: view.state.doc.length, insert: next },
    });
  }

  onMount(() => {
    if (!host) return;
    view = new EditorView({
      state: EditorState.create({
        doc: initial,
        extensions: buildExtensions(),
      }),
      parent: host,
    });
  });

  // Re-apply theme without rebuilding the whole editor — keeps the
  // doc, history, and selection intact when the user flips light/dark.
  $: if (view) {
    view.dispatch({
      effects: themeCompartment.reconfigure(theme === "dark" ? oneDark : []),
    });
  }

  onDestroy(() => {
    view?.destroy();
    view = undefined;
  });
</script>

<div class="xml-editor" bind:this={host} />

<style>
  .xml-editor {
    height: 100%;
    overflow: hidden;
    display: flex;
    flex-direction: column;
    /* Containment so CM6's internal scrollbar geometry isn't recomputed
       by parent layout passes on resize. Pairs with the
       body.resizing display:none rule in App.svelte. */
    contain: layout style;
  }
  .xml-editor :global(.cm-editor) {
    height: 100%;
    font-family: "SF Mono", "JetBrains Mono", Menlo, Consolas, monospace;
    font-size: 0.82rem;
  }
  .xml-editor :global(.cm-scroller) {
    overflow: auto;
  }
  .xml-editor :global(.cm-gutters) {
    background: var(--bg-chrome);
    border-right: 1px solid var(--border);
    color: var(--fg-muted-soft);
  }
  .xml-editor :global(.cm-activeLine) {
    background: var(--highlight, rgba(255, 230, 120, 0.18));
  }
  .xml-editor :global(.cm-activeLineGutter) {
    background: var(--highlight, rgba(255, 230, 120, 0.18));
    color: var(--warn-fg, var(--fg));
  }
</style>
