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

  // Search history, kept in the settings file by Go.
  let recent: string[] = [];
  let historyOpen = false;
  let historyIdx = -1;

  const backend = () => import("../../../wailsjs/go/main/App").catch(() => null);

  /** Only a query that found something is worth keeping; called on Enter and on close. */
  async function remember() {
    const q = pattern.trim();
    if (!q || matchCount === 0 || q === recent[0]) return;
    const App = await backend();
    recent = (await App?.AddRecentSearch(q).catch(() => null)) ?? recent;
  }

  async function clearHistory() {
    const App = await backend();
    await App?.ClearRecentSearches().catch(() => null);
    recent = [];
    closeHistory();
  }

  function openHistory() {
    if (recent.length === 0) return;
    historyOpen = true;
    historyIdx = -1;
  }

  function closeHistory() {
    historyOpen = false;
    historyIdx = -1;
  }

  function pickHistory(q: string) {
    pattern = q;
    closeHistory();
    findInput?.focus();
  }

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
    const App = await backend();
    recent = (await App?.RecentSearches().catch(() => null)) ?? [];
  });

  async function switchMode(next: "find" | "replace") {
    mode = next;
    await tick();
    if (next === "replace") replaceInput?.focus();
    else findInput?.focus();
  }

  function close() {
    void remember();
    if (view) clearSearch(view);
    dispatch("close");
  }

  function onFindKeydown(e: KeyboardEvent) {
    if (historyOpen) {
      if (e.key === "ArrowDown" || e.key === "ArrowUp") {
        e.preventDefault();
        const step = e.key === "ArrowDown" ? 1 : -1;
        historyIdx = (historyIdx + step + recent.length + (historyIdx < 0 && step < 0 ? 1 : 0)) % recent.length;
        return;
      }
      if (e.key === "Escape") {
        e.preventDefault();
        closeHistory();
        return;
      }
      if (e.key === "Enter" && historyIdx >= 0) {
        e.preventDefault();
        pickHistory(recent[historyIdx]);
        return;
      }
      closeHistory();
    } else if (e.key === "ArrowDown") {
      e.preventDefault();
      openHistory();
      return;
    }
    // F3 too: the editor's keymap never sees keys typed in this input.
    if (e.key === "Enter" || e.key === "F3") {
      e.preventDefault();
      if (!view) return;
      void remember();
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
    <div class="find-field">
      <input
        bind:this={findInput}
        type="text"
        class="query"
        class:invalid={pattern && !valid}
        placeholder="Find"
        bind:value={pattern}
        on:keydown={onFindKeydown}
        on:blur={closeHistory}
        spellcheck="false"
        role="combobox"
        aria-controls="find-history"
        aria-expanded={historyOpen}
      />
      {#if recent.length > 0}
        <button
          type="button"
          class="history-toggle"
          title="Recent searches (↓)"
          aria-label="Recent searches"
          on:mousedown|preventDefault={() => (historyOpen ? closeHistory() : openHistory())}
        >▾</button>
      {/if}
      {#if historyOpen}
        <ul id="find-history" class="history" role="listbox" aria-label="Recent searches">
          {#each recent as q, i (q)}
            <li role="option" aria-selected={i === historyIdx}>
              <button
                type="button"
                class:selected={i === historyIdx}
                title={q}
                on:mousedown|preventDefault={() => pickHistory(q)}
              >{q}</button>
            </li>
          {/each}
          <li class="clear">
            <button type="button" on:mousedown|preventDefault={clearHistory}>Clear history</button>
          </li>
        </ul>
      {/if}
    </div>
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
  .find-field {
    position: relative;
    display: flex;
    flex: 1 1 auto;
    min-width: 0;
  }
  .find-field .query {
    flex: 1 1 auto;
    min-width: 0;
    padding-right: 1.5rem;
  }
  .history-toggle {
    position: absolute;
    top: 0;
    right: 0;
    bottom: 0;
    width: 1.4rem;
    padding: 0;
    border: none;
    background: transparent;
    color: var(--fg-secondary);
    cursor: pointer;
  }
  .history {
    position: absolute;
    top: 100%;
    left: 0;
    right: 0;
    z-index: 20;
    max-height: 16rem;
    margin: 2px 0 0;
    padding: 0.2rem 0;
    overflow-y: auto;
    list-style: none;
    background: var(--bg);
    border: 1px solid var(--border-input);
    border-radius: 3px;
  }
  .history button {
    display: block;
    width: 100%;
    padding: 0.2rem 0.5rem;
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
    text-align: left;
    border: none;
    background: transparent;
    color: var(--fg);
    font: inherit;
    cursor: pointer;
  }
  .history button:hover,
  .history button.selected {
    background: var(--bg-hover);
  }
  .history .clear {
    border-top: 1px solid var(--border);
  }
  .history .clear button {
    color: var(--fg-secondary);
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
