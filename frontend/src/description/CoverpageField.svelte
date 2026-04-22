<script lang="ts">
  import type { Coverpage } from "../fb2/types";

  export let cover: Coverpage | null | undefined;
  export let availableIDs: string[] = []; // IDs of binaries (images) available in the document.

  $: if (!cover) cover = { Images: [] };

  function add() {
    cover!.Images = [...cover!.Images, { Href: "" }];
  }
  function remove(i: number) {
    cover!.Images = cover!.Images.filter((_, idx) => idx !== i);
  }
</script>

<div class="cover">
  <span class="title">Coverpage images</span>
  {#if cover}
    {#each cover.Images as image, i (i)}
      <div class="row">
        <select bind:value={image.Href}>
          <option value="">— none —</option>
          {#each availableIDs as id}
            <option value={`#${id}`}>{id}</option>
          {/each}
        </select>
        <input class="custom" placeholder="or custom href" bind:value={image.Href} />
        <button class="aux" type="button" on:click={() => remove(i)} title="Remove">×</button>
      </div>
    {/each}
  {/if}
  <button class="link" type="button" on:click={add}>+ add cover image</button>
</div>

<style>
  .cover {
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
    margin-bottom: 0.5rem;
  }
  .title {
    font-size: 0.8rem;
    color: var(--fg-secondary);
  }
  .row {
    display: flex;
    gap: 0.4rem;
    align-items: center;
  }
  select {
    flex: 0 0 14rem;
    padding: 0.25rem;
    border: 1px solid var(--border-input);
    border-radius: 3px;
  }
  .custom { flex: 1; padding: 0.25rem 0.4rem; border: 1px solid var(--border-input); border-radius: 3px; font: inherit; }
  .aux { background: var(--bg-surface); border: 1px solid var(--border-button); border-radius: 3px; padding: 0 0.4rem; cursor: pointer; }
  .aux:hover { background: var(--bg-hover); }
  .link {
    background: none; border: none; color: var(--fg-link);
    cursor: pointer; padding: 0.15rem 0; font-size: 0.85rem; text-align: left;
    align-self: flex-start;
  }
</style>
