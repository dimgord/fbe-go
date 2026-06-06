<script lang="ts">
  import type { Coverpage } from "../fb2/types";

  export let cover: Coverpage | null | undefined;
  export let availableIDs: string[] = []; // IDs of binaries (images) available in the document.

  // Do NOT auto-init `cover = { Images: [] }` on mount: with `bind:cover`
  // that would write a non-nil empty Coverpage back into info.Coverpage
  // (e.g. SrcTitleInfo.Coverpage) just because the user clicked the
  // src-title tab once — even if they never touched anything. On save,
  // writer.go would emit `<coverpage></coverpage>` inside <src-title-info>,
  // and libxml2 reports it as "Element coverpage: Missing child element(s).
  // Expected is ( image )." with the line number of an UNRELATED non-empty
  // coverpage (libxml2 doesn't always pinpoint the empty one).
  //
  // Instead, materialize `cover` only when the user explicitly hits
  // "+ add cover image". The {#if cover} guard below renders nothing
  // when cover is nullish — clean fall-through, no ghost element.

  function add() {
    if (!cover) cover = { Images: [] };
    cover.Images = [...cover.Images, { Href: "" }];
  }
  function remove(i: number) {
    if (!cover) return;
    cover.Images = cover.Images.filter((_, idx) => idx !== i);
    // If the last image was removed, collapse back to nullish so the
    // writer omits the element entirely (matches "absent" source semantics).
    if (cover.Images.length === 0) cover = null;
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
