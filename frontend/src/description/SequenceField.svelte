<script lang="ts">
  import type { Sequence } from "../fb2/types";
  import { createEventDispatcher } from "svelte";
  import { uid } from "../lib/uid";

  export let seq: Sequence;
  const id_ = uid("seq");
  const dispatch = createEventDispatcher<{ remove: void; clone: void; addChild: void }>();

  function addChild() {
    seq.Children = [...(seq.Children ?? []), { Name: "", Number: "" }];
  }
  function removeChild(i: number) {
    seq.Children = (seq.Children ?? []).filter((_, idx) => idx !== i);
  }
  function cloneChild(i: number) {
    const list = seq.Children ?? [];
    seq.Children = [...list.slice(0, i + 1), JSON.parse(JSON.stringify(list[i])), ...list.slice(i + 1)];
  }
</script>

<div class="seq">
  <div class="row">
    <label for={`${id_}-name`}>Name</label>
    <input id={`${id_}-name`} placeholder="Series name" bind:value={seq.Name} />
    <label for={`${id_}-num`}>№</label>
    <input id={`${id_}-num`} class="num" placeholder="3" bind:value={seq.Number} />
    <button class="aux" type="button" on:click={addChild} title="Nested series">↳</button>
    <button class="aux" type="button" on:click={() => dispatch("clone")} title="Clone">＋</button>
    <button class="aux" type="button" on:click={() => dispatch("remove")} title="Remove">×</button>
  </div>
  {#if seq.Children?.length}
    <div class="nested">
      {#each seq.Children as child, i (i)}
        <svelte:self
          seq={child}
          on:remove={() => removeChild(i)}
          on:clone={() => cloneChild(i)} />
      {/each}
    </div>
  {/if}
</div>

<style>
  .seq { margin-bottom: 0.3rem; }
  .row {
    display: flex;
    gap: 0.4rem;
    align-items: center;
  }
  .nested {
    margin-left: 1.5rem;
    margin-top: 0.3rem;
    border-left: 2px solid #d5d5cb;
    padding-left: 0.5rem;
  }
  label { font-size: 0.8rem; color: #666; }
  input {
    padding: 0.25rem 0.4rem;
    border: 1px solid #ccc;
    border-radius: 3px;
    font: inherit;
    flex: 1;
  }
  .num { flex: 0 0 4rem; text-align: center; }
  .aux {
    background: white;
    border: 1px solid #bbb;
    border-radius: 3px;
    padding: 0 0.4rem;
    cursor: pointer;
  }
  .aux:hover { background: #fff8e5; }
</style>
