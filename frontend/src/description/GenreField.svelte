<script lang="ts">
  import type { Genre } from "../fb2/types";
  import { createEventDispatcher } from "svelte";

  export let genre: Genre;
  const dispatch = createEventDispatcher<{ remove: void; clone: void }>();
</script>

<div class="genre">
  <input class="name" placeholder="Genre (e.g. sf, prose_contemporary)" bind:value={genre.Value} />
  <label>match</label>
  <input class="match" type="text" maxlength="3" placeholder="100" bind:value={genre.Match} />
  <button class="aux" type="button" on:click={() => dispatch("clone")} title="Clone">＋</button>
  <button class="aux" type="button" on:click={() => dispatch("remove")} title="Remove">×</button>
</div>

<style>
  .genre {
    display: flex;
    gap: 0.4rem;
    align-items: center;
    margin-bottom: 0.3rem;
  }
  input {
    padding: 0.25rem 0.4rem;
    border: 1px solid #ccc;
    border-radius: 3px;
    font: inherit;
  }
  .name { flex: 1; }
  .match { width: 3.5rem; text-align: center; }
  label { font-size: 0.8rem; color: #666; }
  .aux {
    background: white;
    border: 1px solid #bbb;
    border-radius: 3px;
    padding: 0 0.4rem;
    cursor: pointer;
  }
  .aux:hover { background: #fff8e5; }
</style>
