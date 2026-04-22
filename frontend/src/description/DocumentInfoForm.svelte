<script lang="ts">
  import type { DocumentInfo, Author } from "../fb2/types";
  import AuthorField from "./AuthorField.svelte";
  import DateField from "./DateField.svelte";

  export let info: DocumentInfo;

  $: if (!info.SrcURL) info.SrcURL = [];

  function addAuthor() { info.Authors = [...info.Authors, {} as Author]; }
  function removeAuthor(i: number) { info.Authors = info.Authors.filter((_, idx) => idx !== i); }
  function cloneAuthor(i: number) {
    info.Authors = [
      ...info.Authors.slice(0, i + 1),
      JSON.parse(JSON.stringify(info.Authors[i])),
      ...info.Authors.slice(i + 1),
    ];
  }
  function addURL() { info.SrcURL = [...(info.SrcURL ?? []), ""]; }
  function removeURL(i: number) { info.SrcURL = (info.SrcURL ?? []).filter((_, idx) => idx !== i); }
  function newID() {
    // RFC 4122 v4-ish UUID.
    info.ID = crypto.randomUUID ? crypto.randomUUID() : Math.random().toString(36).slice(2) + Date.now().toString(36);
  }
</script>

<section class="di">
  <h3>Document authors</h3>
  {#each info.Authors as _, i (i)}
    <AuthorField
      bind:author={info.Authors[i]}
      on:remove={() => removeAuthor(i)}
      on:clone={() => cloneAuthor(i)} />
  {/each}
  <button class="link" type="button" on:click={addAuthor}>+ add author</button>

  <h3>Document</h3>
  <div class="row">
    <label for="di-id">ID</label>
    <input id="di-id" class="mono" bind:value={info.ID} />
    <button class="aux" type="button" on:click={newID}>New UUID</button>
  </div>
  <div class="row">
    <label for="di-ver">Version</label>
    <input id="di-ver" class="short" bind:value={info.Version} placeholder="1.0" />
  </div>
  <div class="row">
    <label for="di-prog">Program used</label>
    <input id="di-prog" class="wide" bind:value={info.ProgramUsed} />
  </div>
  <DateField bind:date={info.Date} label="Date" />

  <h3>Source</h3>
  <div class="row">
    <label for="di-ocr">OCR by</label>
    <input id="di-ocr" class="wide" bind:value={info.SrcOCR} />
  </div>
  {#if info.SrcURL}
    {#each info.SrcURL as _, i (i)}
      <div class="row">
        <label for={`di-src-url-${i}`}>Source URL</label>
        <input id={`di-src-url-${i}`} class="wide" bind:value={info.SrcURL[i]} />
        <button class="aux" type="button" on:click={() => removeURL(i)}>×</button>
      </div>
    {/each}
  {/if}
  <button class="link" type="button" on:click={addURL}>+ add source URL</button>

  <h3 class="todo-header">History</h3>
  <p class="hint">Rich-text history block editing isn't supported in the form yet.</p>
</section>

<style>
  .di { display: flex; flex-direction: column; }
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
  }
  .wide { flex: 1; }
  .short { flex: 0 0 6rem; }
  .mono { flex: 1; font-family: "SF Mono", Menlo, monospace; font-size: 0.88rem; }
  .aux {
    background: white;
    border: 1px solid #bbb;
    border-radius: 3px;
    padding: 0.2rem 0.5rem;
    cursor: pointer;
  }
  .aux:hover { background: #fff8e5; }
  .link {
    background: none; border: none; color: #1a5490;
    cursor: pointer; padding: 0.15rem 0; font-size: 0.85rem; text-align: left;
    align-self: flex-start;
  }
  .hint { color: #888; font-size: 0.85rem; margin: 0.2rem 0; }
  .todo-header { color: #aaa; }
</style>
