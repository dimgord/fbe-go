<script lang="ts">
  import type { PublishInfo, Sequence } from "../fb2/types";
  import SequenceField from "./SequenceField.svelte";

  export let info: PublishInfo;

  $: if (!info.Sequences) info.Sequences = [];

  function addSequence() { info.Sequences = [...(info.Sequences ?? []), { Name: "" } as Sequence]; }
  function removeSequence(i: number) { info.Sequences = (info.Sequences ?? []).filter((_, idx) => idx !== i); }
  function cloneSequence(i: number) {
    const list = info.Sequences ?? [];
    info.Sequences = [...list.slice(0, i + 1), JSON.parse(JSON.stringify(list[i])), ...list.slice(i + 1)];
  }
</script>

<section class="pi">
  <h3>Paper edition</h3>
  <div class="row">
    <label for="pi-name">Book name</label>
    <input id="pi-name" class="wide" bind:value={info.BookName} />
  </div>
  <div class="row">
    <label for="pi-pub">Publisher</label>
    <input id="pi-pub" class="wide" bind:value={info.Publisher} />
  </div>
  <div class="row">
    <label for="pi-city">City</label>
    <input id="pi-city" bind:value={info.City} />
    <label for="pi-year">Year</label>
    <input id="pi-year" class="short" bind:value={info.Year} />
  </div>
  <div class="row">
    <label for="pi-isbn">ISBN</label>
    <input id="pi-isbn" class="wide" bind:value={info.ISBN} />
  </div>

  <h3>Sequence</h3>
  {#if info.Sequences}
    {#each info.Sequences as _, i (i)}
      <SequenceField
        bind:seq={info.Sequences[i]}
        on:remove={() => removeSequence(i)}
        on:clone={() => cloneSequence(i)} />
    {/each}
  {/if}
  <button class="link" type="button" on:click={addSequence}>+ add sequence</button>
</section>

<style>
  .pi { display: flex; flex-direction: column; }
  h3 {
    margin: 1.2rem 0 0.4rem 0;
    font-size: 0.85rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: #888;
    border-bottom: 1px solid #e5e5da;
    padding-bottom: 0.2rem;
  }
  h3:first-child { margin-top: 0; }
  .row {
    display: flex;
    gap: 0.4rem;
    align-items: center;
    margin-bottom: 0.3rem;
  }
  label { font-size: 0.8rem; color: #666; min-width: 6rem; }
  input {
    padding: 0.25rem 0.4rem;
    border: 1px solid #ccc;
    border-radius: 3px;
    font: inherit;
    flex: 1;
  }
  .wide { flex: 1; }
  .short { flex: 0 0 6rem; }
  .link {
    background: none; border: none; color: #1a5490;
    cursor: pointer; padding: 0.15rem 0; font-size: 0.85rem; text-align: left;
    align-self: flex-start;
  }
</style>
