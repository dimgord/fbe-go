<script lang="ts">
  import type Editor from "./Editor.svelte";

  export let editor: Editor | undefined = undefined;

  function cmd(name: keyof Editor): void {
    if (!editor) return;
    const c = (editor as any)[name];
    if (typeof c === "function") {
      editor.exec(c);
    }
  }

  function link(): void {
    editor?.execLink();
  }
</script>

<div class="toolbar" role="toolbar" aria-label="Editor formatting">
  <button title="Undo (⌘Z)" on:click={() => cmd("undo")}>↶</button>
  <button title="Redo (⌘⇧Z)" on:click={() => cmd("redo")}>↷</button>
  <span class="sep" />
  <button title="Bold (⌘B)" on:click={() => cmd("toggleStrong")}><b>B</b></button>
  <button title="Italic (⌘I)" on:click={() => cmd("toggleEmphasis")}><i>I</i></button>
  <button title="Strikethrough (⌘⇧S)" on:click={() => cmd("toggleStrikethrough")}><s>S</s></button>
  <button title="Subscript (⌘,)" on:click={() => cmd("toggleSub")}>X<sub>2</sub></button>
  <button title="Superscript (⌘.)" on:click={() => cmd("toggleSup")}>X<sup>2</sup></button>
  <button title="Code (⌘⇧C)" on:click={() => cmd("toggleCode")}><code>&lt;/&gt;</code></button>
  <button title="Link" on:click={link}>🔗</button>
  <span class="sep" />
  <button title="Normal paragraph" on:click={() => cmd("styleNormal")}>¶</button>
  <button title="Subtitle" on:click={() => cmd("styleSubtitle")}>Sub</button>
  <button title="Text author (end of poem/cite/epigraph)" on:click={() => cmd("styleTextAuthor")}>T-A</button>
  <button title="Empty line" on:click={() => cmd("insertEmptyLine")}>␣</button>
  <span class="sep" />
  <button title="Clone section / poem / stanza / cite / epigraph" on:click={() => cmd("cloneContainer")}>Clone</button>
  <button title="Remove outer section (promote children up)" on:click={() => cmd("removeOuterContainer")}>Unwrap</button>
  <button title="Add title to enclosing section / body / poem / stanza" on:click={() => cmd("addTitle")}>+ Title</button>
  <button title="Add epigraph to enclosing body / section / poem" on:click={() => cmd("addEpigraph")}>+ Epigraph</button>
  <button title="Add annotation to enclosing section" on:click={() => cmd("addAnnotation")}>+ Annot.</button>
  <button title="Append text-author to enclosing poem / cite / epigraph" on:click={() => cmd("addTextAuthor")}>+ T-A</button>
  <span class="sep" />
  <button title="Wrap selection in a &lt;cite&gt;" on:click={() => cmd("insertCite")}>❝ Cite</button>
  <button title="Wrap selection in a &lt;poem&gt; (paragraphs → verses; empty-line splits stanzas)" on:click={() => cmd("insertPoem")}>♪ Poem</button>
  <button title="Insert table…" on:click={() => editor?.openTableDialog()}>▦ Table…</button>
  <button title="Merge with next sibling section / stanza / cite" on:click={() => cmd("mergeContainers")}>⟛ Merge</button>
</div>

<style>
  .toolbar {
    display: flex;
    align-items: center;
    gap: 0.25rem;
    padding: 0.3rem 0.5rem;
    background: #eceae0;
    border-bottom: 1px solid #d5d5cb;
  }
  .toolbar button {
    min-width: 2rem;
    height: 1.8rem;
    background: white;
    border: 1px solid #c5c5bb;
    border-radius: 3px;
    cursor: pointer;
    font-size: 0.85rem;
    padding: 0 0.4rem;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    white-space: nowrap;
    line-height: 1;
  }
  .toolbar button:hover { background: #fff8e5; }
  .toolbar button:active { background: #fce6a0; }
  .sep {
    width: 1px;
    height: 1.2rem;
    background: #c5c5bb;
    margin: 0 0.25rem;
  }
  code {
    font-family: "SF Mono", Menlo, monospace;
    font-size: 0.8rem;
  }
</style>
