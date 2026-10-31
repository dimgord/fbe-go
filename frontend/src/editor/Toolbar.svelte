<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import type Editor from "./Editor.svelte";
  import type { CommandId } from "./commandRegistry";
  import { displayAccel } from "../settings/hotkeys";

  export let editor: Editor | undefined = undefined;
  /** The user's bindings, so tooltips show the keys that actually work. */
  export let hotkeys: Record<string, string> = {};

  const dispatch = createEventDispatcher<{ openBinaries: void }>();

  const run = (id: CommandId) => editor?.run(id);
  $: tip = (label: string, id: string) => (hotkeys[id] ? `${label} (${displayAccel(hotkeys[id])})` : label);
</script>

<div class="toolbar" role="toolbar" aria-label="Editor formatting">
  <button title="Undo (⌘Z)" on:click={() => run("Undo")}>↶</button>
  <button title="Redo (⌘⇧Z)" on:click={() => run("Redo")}>↷</button>
  <span class="sep" />
  <button title={tip("Bold", "ToggleStrong")} on:click={() => run("ToggleStrong")}><b>B</b></button>
  <button title={tip("Italic", "ToggleEmphasis")} on:click={() => run("ToggleEmphasis")}><i>I</i></button>
  <button title={tip("Strikethrough", "ToggleStrikethrough")} on:click={() => run("ToggleStrikethrough")}><s>S</s></button>
  <button title={tip("Subscript", "ToggleSub")} on:click={() => run("ToggleSub")}>X<sub>2</sub></button>
  <button title={tip("Superscript", "ToggleSup")} on:click={() => run("ToggleSup")}>X<sup>2</sup></button>
  <button title={tip("Code", "ToggleCode")} on:click={() => run("ToggleCode")}><code>&lt;/&gt;</code></button>
  <button title="Link" on:click={() => editor?.execLink()}>🔗</button>
  <span class="sep" />
  <button title={tip("Normal paragraph", "StyleNormal")} on:click={() => run("StyleNormal")}>¶</button>
  <button title={tip("Subtitle", "StyleSubtitle")} on:click={() => run("StyleSubtitle")}>Sub</button>
  <button title={tip("Text author (end of poem/cite/epigraph)", "StyleTextAuthor")} on:click={() => run("StyleTextAuthor")}>T-A</button>
  <button title={tip("Empty line", "InsertEmptyLine")} on:click={() => run("InsertEmptyLine")}>␣</button>
  <span class="sep" />
  <button title={tip("Clone section / poem / stanza / cite / epigraph", "CloneContainer")} on:click={() => run("CloneContainer")}>Clone</button>
  <button title={tip("Remove outer section (promote children up)", "RemoveOuterContainer")} on:click={() => run("RemoveOuterContainer")}>Unwrap</button>
  <button title={tip("Add title to enclosing section / body / poem / stanza", "AddTitle")} on:click={() => run("AddTitle")}>+ Title</button>
  <button title={tip("Add epigraph to enclosing body / section / poem", "AddEpigraph")} on:click={() => run("AddEpigraph")}>+ Epigraph</button>
  <button title={tip("Add annotation to enclosing section", "AddAnnotation")} on:click={() => run("AddAnnotation")}>+ Annot.</button>
  <button title={tip("Append text-author to enclosing poem / cite / epigraph", "AddTextAuthor")} on:click={() => run("AddTextAuthor")}>+ T-A</button>
  <span class="sep" />
  <button title={tip("Wrap selection in a &lt;cite&gt;", "InsertCite")} on:click={() => run("InsertCite")}>❝ Cite</button>
  <button title={tip("Wrap selection in a &lt;poem&gt; (paragraphs → verses; empty-line splits stanzas)", "InsertPoem")} on:click={() => run("InsertPoem")}>♪ Poem</button>
  <button title="Insert table…" on:click={() => editor?.openTableDialog()}>▦ Table…</button>
  <button title={tip("Merge with next sibling section / stanza / cite", "MergeContainers")} on:click={() => run("MergeContainers")}>⟛ Merge</button>
  <span class="sep" />
  <button title="Manage binaries (upload / rename / delete embedded images)" on:click={() => dispatch("openBinaries")}>📎 Binaries…</button>
</div>

<style>
  .toolbar {
    display: flex;
    align-items: center;
    gap: 0.25rem;
    padding: 0.3rem 0.5rem;
    background: var(--bg-chrome);
    border-bottom: 1px solid var(--border);
  }
  .toolbar button {
    min-width: 2rem;
    height: 1.8rem;
    background: var(--bg-surface);
    color: var(--fg);
    border: 1px solid var(--border);
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
  .toolbar button:hover { background: var(--bg-hover); }
  .toolbar button:active { background: var(--bg-active); }
  .sep {
    width: 1px;
    height: 1.2rem;
    background: var(--border);
    margin: 0 0.25rem;
  }
  code {
    font-family: var(--font-mono);
    font-size: 0.8rem;
  }
</style>
