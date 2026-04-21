<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import type { OutlineNode } from "./outline";

  export let node: OutlineNode;

  const dispatch = createEventDispatcher<{ navigate: { path: number[] } }>();

  function handleClick(e: MouseEvent) {
    e.stopPropagation();
    dispatch("navigate", { path: node.path });
  }

  function onChildNavigate(e: CustomEvent<{ path: number[] }>) {
    dispatch("navigate", e.detail);
  }
</script>

<li>
  <button
    type="button"
    class="item kind-{node.kind}"
    on:click={handleClick}
    title={node.label}>
    {node.label}
  </button>
  {#if node.children.length > 0}
    <ul>
      {#each node.children as child (child.path.join("."))}
        <svelte:self node={child} on:navigate={onChildNavigate} />
      {/each}
    </ul>
  {/if}
</li>

<style>
  li { list-style: none; }
  ul {
    list-style: none;
    padding-left: 0.9rem;
    margin: 0;
  }
  .item {
    display: block;
    width: 100%;
    text-align: left;
    background: transparent;
    border: none;
    padding: 0.2rem 0.4rem;
    cursor: pointer;
    border-radius: 3px;
    color: #333;
    font-family: inherit;
    font-size: 0.88rem;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .item:hover { background: #e5e5da; }
  .item.kind-body {
    font-weight: 600;
    color: #1a5490;
  }
  .item.kind-section { color: #444; }
</style>
