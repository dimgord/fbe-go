<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import { basename, dirname } from "../lib/path";
  import { effectiveTheme, theme, themeIcon } from "./theme";

  export let hasDocument = false;
  export let filename = "";
  export let status = "";
  export let error = "";
  export let recentFiles: string[] = [];
  export let view: "body" | "description" = "body";

  const dispatch = createEventDispatcher<{
    /** A path from the recent list, or undefined for the file picker. */
    open: string | undefined;
    save: boolean; // true = Save As
    validate: void;
    export: void;
    settings: void;
    help: void;
    theme: void;
    view: "body" | "description";
  }>();

  let recentMenuOpen = false;

  function pick(path: string) {
    recentMenuOpen = false;
    dispatch("open", path);
  }
</script>

<header>
  <div class="open-group">
    <button on:click={() => dispatch("open", undefined)}>Open…</button>
    <button
      class="recent-toggle"
      title="Recent files"
      disabled={recentFiles.length === 0}
      on:click={() => (recentMenuOpen = !recentMenuOpen)}
    >▾</button>
    {#if recentMenuOpen}
      <!-- svelte-ignore a11y-click-events-have-key-events -->
      <!-- svelte-ignore a11y-no-static-element-interactions -->
      <div class="recent-backdrop" on:click={() => (recentMenuOpen = false)}></div>
      <ul class="recent-menu" role="menu">
        {#each recentFiles as path}
          <li role="menuitem">
            <button
              type="button"
              class="recent-item"
              on:click={() => pick(path)}
              title={path}
            >
              <span class="basename">{basename(path)}</span>
              <span class="dir">{dirname(path)}</span>
            </button>
          </li>
        {/each}
      </ul>
    {/if}
  </div>
  <button on:click={() => dispatch("save", false)} disabled={!hasDocument}>Save</button>
  <button on:click={() => dispatch("save", true)} disabled={!hasDocument}>Save As…</button>
  <button on:click={() => dispatch("validate")} disabled={!hasDocument}>Validate</button>
  <button on:click={() => dispatch("export")} disabled={!hasDocument}>Export HTML…</button>
  <button on:click={() => dispatch("settings")} title="Settings" aria-label="Open settings">⚙</button>
  <button on:click={() => dispatch("help")} title="Keyboard shortcuts and about">Help</button>
  <button
    class="theme-toggle"
    on:click={() => dispatch("theme")}
    title={`Theme: ${$theme}${$theme === "system" ? ` (effective: ${$effectiveTheme})` : ""} — click to cycle`}
    aria-label="Cycle theme (system → light → dark)"
  >{themeIcon($theme)}</button>
  <div class="view-toggle" role="tablist" aria-label="View">
    <button
      class:active={view === "body"}
      on:click={() => dispatch("view", "body")}
      role="tab"
      aria-selected={view === "body"}>Body</button>
    <button
      class:active={view === "description"}
      on:click={() => dispatch("view", "description")}
      role="tab"
      aria-selected={view === "description"}>Description</button>
  </div>
  <span class="title">FictionBook Editor · <em>{filename}</em></span>
  {#if status}<span class="status">{status}</span>{/if}
  {#if error}<span class="err">{error}</span>{/if}
</header>

<style>
  header {
    display: flex;
    align-items: center;
    padding: 0 0.75rem;
    gap: 0.5rem;
    background: var(--bg-chrome);
    border-bottom: 1px solid var(--border);
  }
  header button {
    padding: 0.25rem 0.7rem;
    border: 1px solid var(--border-button);
    background: var(--bg-surface);
    color: var(--fg);
    border-radius: 4px;
    cursor: pointer;
  }
  header button:hover:not(:disabled) { background: var(--bg-hover); }
  header button:disabled { opacity: 0.5; cursor: default; }

  /* Theme cycle icon-button (lives on the right edge, near Help). */
  header button.theme-toggle {
    padding: 0.25rem 0.5rem;
    font-size: 0.95rem;
    line-height: 1;
  }

  /* Recent-files split-button: Open…<dropdown-caret>. Visually the two
     buttons share a border so they read as one control. */
  .open-group {
    position: relative;
    display: inline-flex;
    gap: 0;
  }
  .open-group > button:first-child {
    border-top-right-radius: 0;
    border-bottom-right-radius: 0;
  }
  .open-group > button.recent-toggle {
    border-top-left-radius: 0;
    border-bottom-left-radius: 0;
    border-left-width: 0;
    padding: 0.25rem 0.45rem;
    font-size: 0.7rem;
    line-height: 1;
  }
  .recent-backdrop {
    position: fixed;
    inset: 0;
    background: transparent;
    z-index: 49;
  }
  .recent-menu {
    position: absolute;
    top: 100%;
    left: 0;
    z-index: 50;
    list-style: none;
    margin: 2px 0 0;
    padding: 0.25rem 0;
    min-width: 22rem;
    max-width: 38rem;
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: 4px;
    box-shadow: 0 6px 18px var(--shadow);
    font-size: 0.85rem;
  }
  .recent-menu li { margin: 0; padding: 0; }
  button.recent-item {
    all: unset;
    display: block;
    width: 100%;
    padding: 0.3rem 0.7rem;
    cursor: pointer;
    box-sizing: border-box;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  button.recent-item:hover { background: var(--bg-hover); }
  button.recent-item .basename { font-weight: 600; color: var(--fg); }
  button.recent-item .dir {
    color: var(--fg-muted);
    font-size: 0.78rem;
    margin-left: 0.4rem;
  }

  .view-toggle {
    display: inline-flex;
    gap: 0;
    margin-left: 0.5rem;
  }
  .view-toggle button {
    border-radius: 0;
    border-right-width: 0;
  }
  .view-toggle button:first-child { border-radius: 4px 0 0 4px; }
  .view-toggle button:last-child { border-radius: 0 4px 4px 0; border-right-width: 1px; }
  .view-toggle button.active {
    background: var(--bg-active);
    font-weight: 600;
  }
  .title { font-size: 0.9rem; color: var(--fg-secondary); margin-left: 0.5rem; }
  .status { color: var(--ok); font-size: 0.8rem; margin-left: auto; }
  .err { color: var(--danger); font-size: 0.8rem; margin-left: auto; }
</style>
