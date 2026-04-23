<script lang="ts">
  import type { CustomInfo } from "../fb2/types";

  export let items: CustomInfo[] | null | undefined;

  $: if (!items) items = [];

  function add() { items = [...(items ?? []), { InfoType: "", Value: "" }]; }
  function remove(i: number) { items = (items ?? []).filter((_, idx) => idx !== i); }
</script>

<section class="ci">
  <h3>Custom metadata</h3>
  <p class="hint">
    Arbitrary key/value pairs. Useful for library metadata (source, uploader, etc.)
    or tooling-specific annotations.
  </p>
  {#if items}
    {#each items as _, i (i)}
      <div class="entry">
        <div class="row">
          <label for={`ci-${i}-type`}>Type</label>
          <input id={`ci-${i}-type`} bind:value={items[i].InfoType} placeholder="e.g. library-id" />
          <button class="aux" type="button" on:click={() => remove(i)}>×</button>
        </div>
        <div class="row">
          <label for={`ci-${i}-val`}>Value</label>
          <textarea id={`ci-${i}-val`} rows="3" bind:value={items[i].Value} />
        </div>
      </div>
    {/each}
  {/if}
  <button class="link" type="button" on:click={add}>+ add custom info</button>
</section>

<style>
  .ci { display: flex; flex-direction: column; }
  h3 {
    margin: 0 0 0.4rem 0;
    font-size: 0.85rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--fg-muted);
    border-bottom: 1px solid var(--border);
    padding-bottom: 0.2rem;
  }
  .hint { color: var(--fg-muted); font-size: 0.85rem; margin: 0 0 0.8rem 0; }
  .entry {
    border: 1px solid var(--border);
    padding: 0.5rem;
    border-radius: 4px;
    background: var(--bg-card);
    margin-bottom: 0.5rem;
  }
  .row {
    display: flex;
    gap: 0.4rem;
    align-items: start;
    margin-bottom: 0.3rem;
  }
  label { font-size: 0.8rem; color: var(--fg-secondary); min-width: 4rem; padding-top: 0.3rem; }
  input, textarea {
    padding: 0.25rem 0.4rem;
    border: 1px solid var(--border-input);
    border-radius: 3px;
    font: inherit;
    flex: 1;
  }
  textarea { font-family: inherit; resize: vertical; }
  .aux {
    background: var(--bg-surface);
    border: 1px solid var(--border-button);
    border-radius: 3px;
    padding: 0.2rem 0.5rem;
    cursor: pointer;
  }
  .aux:hover { background: var(--bg-hover); }
  .link {
    background: none; border: none; color: var(--fg-link);
    cursor: pointer; padding: 0.15rem 0; font-size: 0.85rem; text-align: left;
    align-self: flex-start;
  }
</style>
