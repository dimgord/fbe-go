<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import OutlineItem from "./OutlineItem.svelte";
  import type { OutlineNode } from "./outline";

  export let tree: OutlineNode[] = [];
  /** Outline path of the section being read; highlighted in the tree. */
  export let activePath: number[] | null = null;

  const dispatch = createEventDispatcher<{ navigate: { path: number[] } }>();

  $: activeKey = activePath?.join(".") ?? "";

  function onNavigate(e: CustomEvent<{ path: number[] }>) {
    dispatch("navigate", e.detail);
  }
</script>

{#if tree.length === 0}
  <div class="empty">No document loaded</div>
{:else}
  <ul class="outline">
    {#each tree as node (node.path.join("."))}
      <OutlineItem {node} {activeKey} on:navigate={onNavigate} />
    {/each}
  </ul>
{/if}

<style>
  .empty {
    padding: 1rem;
    color: var(--fg-muted);
    font-style: italic;
    font-size: 0.85rem;
  }
  .outline {
    list-style: none;
    padding: 0.5rem 0.5rem;
    margin: 0;
  }
</style>
