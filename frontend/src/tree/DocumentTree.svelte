<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import OutlineItem from "./OutlineItem.svelte";
  import { buildOutline } from "./outline";
  import type { FictionBook } from "../fb2/types";

  export let fb: FictionBook | null = null;

  const dispatch = createEventDispatcher<{ navigate: { path: number[] } }>();

  $: tree = buildOutline(fb);

  function onNavigate(e: CustomEvent<{ path: number[] }>) {
    dispatch("navigate", e.detail);
  }
</script>

{#if tree.length === 0}
  <div class="empty">No document loaded</div>
{:else}
  <ul class="outline">
    {#each tree as node (node.path.join("."))}
      <OutlineItem {node} on:navigate={onNavigate} />
    {/each}
  </ul>
{/if}

<style>
  .empty {
    padding: 1rem;
    color: #999;
    font-style: italic;
    font-size: 0.85rem;
  }
  .outline {
    list-style: none;
    padding: 0.5rem 0.5rem;
    margin: 0;
  }
</style>
