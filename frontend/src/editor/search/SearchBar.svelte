<script lang="ts">
  import { createEventDispatcher, onMount, tick } from "svelte";
  import type { EditorView } from "prosemirror-view";
  import {
    setSearch, clearSearch, findNext, findPrev,
    replaceActive, replaceAll, searchStore,
    type SearchFlags,
  } from "./plugin";

  export let view: EditorView | undefined;
  /** Which mode the bar opens in. `replace` reveals the second input row. */
  export let mode: "find" | "replace" = "find";

  const dispatch = createEventDispatcher<{ close: void }>();

  let pattern = "";
  let replacement = "";
  let caseSensitive = false;
  let wholeWord = false;
  let regex = false;

  let findInput: HTMLInputElement;
  let replaceInput: HTMLInputElement;

  // Push every relevant change into the plugin. Separate reactive from the
  // input binding so we don't rescan on every keystroke inside replacement.
  $: if (view) {
    const flags: SearchFlags = { caseSensitive, wholeWord, regex };
    setSearch(view, pattern, flags);
  }

  $: matchCount = $searchStore.matches.length;
  $: activeIdx = $searchStore.active;
  $: valid = $searchStore.valid;
  $: counter = pattern
    ? valid
      ? matchCount === 0
        ? "No matches"
        : `${activeIdx + 1} / ${matchCount}`
      : "Invalid regex"
    : "";

  onMount(async () => {
    await tick();
    findInput?.focus();
    findInput?.select();
  });

  async function switchMode(next: "find" | "replace") {
    mode = next;
    await tick();
    if (next === "replace") replaceInput?.focus();
    else findInput?.focus();
  }

  function close() {
    if (view) clearSearch(view);
    dispatch("close");
  }

  function onFindKeydown(e: KeyboardEvent) {
    if (e.key === "Enter") {
      e.preventDefault();
      if (!view) return;
      if (e.shiftKey) findPrev(view);
      else findNext(view);
    } else if (e.key === "Escape") {
      e.preventDefault();
      close();
    }
  }

  function onReplaceKeydown(e: KeyboardEvent) {
    if (e.key === "Enter") {
      e.preventDefault();
      if (!view) return;
      if (e.shiftKey) {
        // Shift+Enter in replace field → replace-all, matches VS Code.
        replaceAll(view, replacement);
      } else {
        replaceActive(view, replacement);
      }
    } else if (e.key === "Escape") {
      e.preventDefault();
      close();
    }
  }

  function onReplaceClick() {
    if (!view) return;
    replaceActive(view, replacement);
  }

  function onReplaceAllClick() {
    if (!view) return;
    const n = replaceAll(view, replacement);
    // Tiny feedback so the user sees the count; re-uses the counter slot.
    if (n > 0) pattern = pattern; // force reactive refresh of counter via $searchStore
  }
</script>

<div class="search-bar" role="toolbar" aria-label="Find and replace">
  <div class="row">
    <button
      type="button"
      class="twist"
      class:open={mode === "replace"}
      aria-label={mode === "replace" ? "Collapse replace" : "Expand replace"}
      on:click={() => switchMode(mode === "replace" ? "find" : "replace")}
    >
      ▸
    </button>
    <input
      bind:this={findInput}
      type="text"
      class="query"
      class:invalid={pattern && !valid}
      placeholder="Find"
      bind:value={pattern}
      on:keydown={onFindKeydown}
      spellcheck="false"
    />
    <span class="counter" class:empty={!pattern}>{counter}</span>

    <div class="toggles">
      <button
        type="button"
        class="toggle"
        class:on={caseSensitive}
        title="Match case (Aa)"
        aria-pressed={caseSensitive}
        on:click={() => (caseSensitive = !caseSensitive)}
      >Aa</button>
      <button
        type="button"
        class="toggle"
        class:on={wholeWord}
        title="Whole word"
        aria-pressed={wholeWord}
        on:click={() => (wholeWord = !wholeWord)}
      >\b</button>
      <button
        type="button"
        class="toggle"
        class:on={regex}
        title="Regular expression"
        aria-pressed={regex}
        on:click={() => (regex = !regex)}
      >.*</button>
    </div>

    <div class="nav">
      <button type="button" title="Previous (Shift+Enter)" on:click={() => view && findPrev(view)}>◀</button>
      <button type="button" title="Next (Enter)" on:click={() => view && findNext(view)}>▶</button>
    </div>

    <button type="button" class="close" title="Close (Esc)" on:click={close}>✕</button>
  </div>

  {#if mode === "replace"}
    <div class="row">
      <span class="twist-placeholder"></span>
      <input
        bind:this={replaceInput}
        type="text"
        class="query"
        placeholder="Replace"
        bind:value={replacement}
        on:keydown={onReplaceKeydown}
        spellcheck="false"
      />
      <div class="nav">
        <button
          type="button"
          title="Replace one (Enter)"
          disabled={activeIdx < 0}
          on:click={onReplaceClick}
        >Replace</button>
        <button
          type="button"
          title="Replace all (Shift+Enter)"
          disabled={matchCount === 0}
          on:click={onReplaceAllClick}
        >Replace all</button>
      </div>
    </div>
  {/if}
</div>

<style>
  .search-bar {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
    padding: 0.5rem 0.75rem;
    background: var(--bg-chrome);
    border-bottom: 1px solid var(--border);
    font-size: 0.9em;
  }
  .row {
    display: grid;
    grid-template-columns: auto 1fr auto auto auto auto;
    align-items: center;
    gap: 0.4rem;
  }
  .twist,
  .twist-placeholder {
    width: 1.2rem;
    height: 1.2rem;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    background: transparent;
    border: none;
    color: var(--fg-secondary);
    cursor: pointer;
    padding: 0;
    font-size: 0.9em;
    transition: transform 0.1s ease;
  }
  .twist.open {
    transform: rotate(90deg);
  }
  .query {
    min-width: 180px;
    padding: 0.25rem 0.5rem;
    background: var(--bg);
    color: var(--fg);
    border: 1px solid var(--border-input);
    border-radius: 3px;
    font-family: inherit;
    font-size: inherit;
  }
  .query.invalid {
    border-color: var(--danger);
  }
  .query:focus {
    outline: 2px solid var(--fg-link);
    outline-offset: -1px;
  }
  .counter {
    min-width: 5.5em;
    text-align: right;
    color: var(--fg-secondary);
    font-size: 0.85em;
    font-variant-numeric: tabular-nums;
  }
  .counter.empty {
    visibility: hidden;
  }
  .toggles,
  .nav {
    display: inline-flex;
    gap: 0.15rem;
  }
  button.toggle {
    min-width: 1.8rem;
    height: 1.7rem;
    padding: 0 0.4rem;
    background: var(--bg);
    color: var(--fg-secondary);
    border: 1px solid var(--border-input);
    border-radius: 3px;
    cursor: pointer;
    font-family: "SF Mono", Menlo, Consolas, monospace;
    font-size: 0.85em;
  }
  button.toggle.on {
    background: var(--fg-link);
    color: var(--bg);
    border-color: var(--fg-link);
  }
  .nav button,
  .close {
    height: 1.7rem;
    padding: 0 0.6rem;
    background: var(--bg);
    color: var(--fg);
    border: 1px solid var(--border-input);
    border-radius: 3px;
    cursor: pointer;
    font-size: 0.9em;
  }
  .nav button:disabled {
    opacity: 0.5;
    cursor: default;
  }
  .nav button:hover:not(:disabled),
  .close:hover {
    background: var(--bg-hover);
  }
  .close {
    margin-left: 0.25rem;
    color: var(--fg-secondary);
  }
</style>
