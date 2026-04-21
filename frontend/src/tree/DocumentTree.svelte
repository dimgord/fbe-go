<script lang="ts">
  // Document outline. Replaces FBE/DocumentTree.cpp.
  // Data source: the current ProseMirror doc — walk the nodes and build a tree
  // (sections, titles, poems, cites, epigraphs, annotations).

  export let tree: TreeNode[] = [];

  type TreeNode = {
    label: string;
    kind: "body" | "section" | "title" | "poem" | "cite" | "epigraph" | "annotation";
    id?: string;
    children?: TreeNode[];
  };
</script>

<ul>
  {#each tree as node}
    <li>
      <span class="kind">{node.kind}</span> {node.label}
      {#if node.children?.length}
        <svelte:self tree={node.children} />
      {/if}
    </li>
  {/each}
</ul>

<style>
  ul {
    list-style: none;
    padding-left: 1rem;
    margin: 0;
  }
  .kind {
    color: #888;
    font-size: 0.75rem;
    text-transform: uppercase;
  }
</style>
