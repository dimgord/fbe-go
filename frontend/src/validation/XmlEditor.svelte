<script lang="ts">
  /*
   * CodeMirror 6 wrapper for editing raw FB2 XML.
   *
   * The whole XML pane, read-only until the user clicks "Edit XML". It stays
   * one CodeMirror in both modes because only CM6 virtualizes: a plain DOM
   * list of a book's 13,000 lines took 78 s to render in WKWebView.
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
  import { ChangeSet, Compartment, EditorState, type ChangeDesc, type Text } from "@codemirror/state";
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
  export let readOnly = true;

  const dispatch = createEventDispatcher<{
    /** Fired on every edit; payload is the current full document.
     *  Parent uses this to track dirty state without polling. */
    change: string;
    /** The user scrolled; payload is the 1-based line at the top. */
    scrollline: number;
    /** The caret moved; payload is its 1-based line in the XML as last loaded. */
    caretline: number;
  }>();

  let host: HTMLDivElement | undefined;
  let view: EditorView | undefined;
  const themeCompartment = new Compartment();
  const readOnlyCompartment = new Compartment();

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
        if (replacing) return;
        if (u.docChanged) {
          sinceLoad = sinceLoad.composeDesc(u.changes.desc);
          dispatch("change", u.state.doc.toString());
        }
        if (u.docChanged || u.selectionSet) reportCaret(u.state.selection.main.head);
      }),
      themeCompartment.of(theme === "dark" ? oneDark : []),
      readOnlyCompartment.of(EditorState.readOnly.of(readOnly)),
    ];
  }

  // Line numbers elsewhere describe the XML as loaded, so the caret is mapped back through the
  // edits made since: an added line must not shift every mark after it.
  let loaded: Text | undefined;
  let sinceLoad: ChangeDesc = ChangeSet.empty(0).desc;
  let replacing = false;
  let lastCaretLine = 0;

  function loadedDocIs(doc: Text) {
    loaded = doc;
    sinceLoad = ChangeSet.empty(doc.length).desc;
  }

  function reportCaret(head: number) {
    if (!loaded) return;
    const line = loaded.lineAt(sinceLoad.invertedDesc.mapPos(head)).number;
    // CM6 scrolls to a moving caret; that must not read as the user scrolling.
    followingUntil = performance.now() + SCROLL_LEAD_MS;
    leadingUntil = 0;
    if (line !== lastCaretLine) {
      lastCaretLine = line;
      dispatch("caretline", line);
    }
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

  // Text and XML follow each other, so each side must tell a user's scroll from its own, or they
  // would chase one another. Whichever scrolled last leads for SCROLL_LEAD_MS.
  const SCROLL_LEAD_MS = 400;
  let followingUntil = 0;
  let leadingUntil = 0;
  const SCROLL_SETTLE_MS = 120;
  let scrollSettle: ReturnType<typeof setTimeout> | undefined;

  /** Bring a 1-based line to the top without moving the caret or taking focus. */
  export function scrollToLine(n: number) {
    if (!view || n < 1 || n > view.state.doc.lines || performance.now() < leadingUntil) return;
    followingUntil = performance.now() + SCROLL_LEAD_MS;
    view.dispatch({
      effects: EditorView.scrollIntoView(view.state.doc.line(n).from, { y: "start" }),
    });
  }

  // The text follows once the scroll stops: moving it costs far more than moving this pane, and
  // doing it every frame starves CM6's own rendering, so the scrolled lines stayed blank for seconds.
  function onScroll() {
    if (performance.now() < followingUntil) return;
    leadingUntil = performance.now() + SCROLL_LEAD_MS;
    clearTimeout(scrollSettle);
    scrollSettle = setTimeout(() => {
      if (!view) return;
      const top = view.lineBlockAtHeight(view.scrollDOM.scrollTop);
      dispatch("scrollline", view.state.doc.lineAt(top.from).number);
    }, SCROLL_SETTLE_MS);
  }

  export function hasFocus(): boolean {
    return view?.hasFocus ?? false;
  }

  /** Replace the entire doc — used when the panel resets after an
   *  external SerializeCurrent (e.g., after Apply succeeded and the
   *  user did another body-edit + Validate). */
  export function setDoc(next: string) {
    if (!view) {
      initial = next;
      return;
    }
    // Replacing the text moves the scroller; that is not the user scrolling.
    followingUntil = performance.now() + SCROLL_LEAD_MS;
    leadingUntil = 0;
    replacing = true;
    view.dispatch({
      changes: { from: 0, to: view.state.doc.length, insert: next },
    });
    replacing = false;
    loadedDocIs(view.state.doc);
    lastCaretLine = 0;
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
    loadedDocIs(view.state.doc);
    view.scrollDOM.addEventListener("scroll", onScroll, { passive: true });
  });

  // Re-apply theme without rebuilding the whole editor — keeps the
  // doc, history, and selection intact when the user flips light/dark.
  $: if (view) {
    view.dispatch({
      effects: themeCompartment.reconfigure(theme === "dark" ? oneDark : []),
    });
  }

  $: if (view) {
    view.dispatch({ effects: readOnlyCompartment.reconfigure(EditorState.readOnly.of(readOnly)) });
  }

  onDestroy(() => {
    clearTimeout(scrollSettle);
    view?.scrollDOM.removeEventListener("scroll", onScroll);
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
    font-family: var(--font-mono);
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
    background: var(--highlight);
  }
  .xml-editor :global(.cm-activeLineGutter) {
    background: var(--highlight);
    color: var(--warn-fg, var(--fg));
  }
</style>
