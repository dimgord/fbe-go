<script lang="ts">
  import type { Author } from "../fb2/types";
  import { createEventDispatcher } from "svelte";

  export let author: Author;
  /** "primary" shows all fields; "compact" hides nick/email/home behind a disclosure. */
  export let variant: "primary" | "compact" = "compact";

  let open = variant === "primary" || !!(author.Nickname || author.Email?.length || author.HomePage?.length);

  const dispatch = createEventDispatcher<{ remove: void; clone: void }>();

  // Ensure optional arrays exist for two-way binding.
  $: if (!author.Email)    author.Email    = [];
  $: if (!author.HomePage) author.HomePage = [];

  function addEmail() {
    author.Email = [...(author.Email ?? []), ""];
  }
  function removeEmail(i: number) {
    author.Email = (author.Email ?? []).filter((_, idx) => idx !== i);
  }
  function addHomePage() {
    author.HomePage = [...(author.HomePage ?? []), ""];
  }
  function removeHomePage(i: number) {
    author.HomePage = (author.HomePage ?? []).filter((_, idx) => idx !== i);
  }
</script>

<div class="author">
  <div class="row">
    <input placeholder="First name" bind:value={author.FirstName} />
    <input placeholder="Middle" bind:value={author.MiddleName} />
    <input placeholder="Last name" bind:value={author.LastName} />
    <button class="aux" type="button" on:click={() => dispatch("clone")} title="Clone">＋</button>
    <button class="aux" type="button" on:click={() => dispatch("remove")} title="Remove">×</button>
  </div>
  <button class="disclosure" type="button" on:click={() => (open = !open)}>
    {open ? "▾" : "▸"} more
  </button>
  {#if open}
    <div class="row">
      <label>Nick</label>
      <input bind:value={author.Nickname} />
    </div>
    <div class="row">
      <label>ID</label>
      <input bind:value={author.ID} />
    </div>
    <div class="multi">
      <label>Email</label>
      <div class="stack">
        {#each author.Email ?? [] as _, i}
          <div class="inline">
            <input bind:value={author.Email[i]} />
            <button class="aux" type="button" on:click={() => removeEmail(i)}>×</button>
          </div>
        {/each}
        <button class="link" type="button" on:click={addEmail}>+ add email</button>
      </div>
    </div>
    <div class="multi">
      <label>Home page</label>
      <div class="stack">
        {#each author.HomePage ?? [] as _, i}
          <div class="inline">
            <input bind:value={author.HomePage[i]} />
            <button class="aux" type="button" on:click={() => removeHomePage(i)}>×</button>
          </div>
        {/each}
        <button class="link" type="button" on:click={addHomePage}>+ add URL</button>
      </div>
    </div>
  {/if}
</div>

<style>
  .author {
    border: 1px solid #dcdcd0;
    padding: 0.5rem;
    border-radius: 4px;
    background: #fcfbf6;
    margin-bottom: 0.5rem;
  }
  .row, .inline {
    display: flex;
    gap: 0.4rem;
    align-items: center;
    margin-bottom: 0.35rem;
  }
  .multi {
    display: grid;
    grid-template-columns: 6rem 1fr;
    align-items: start;
    gap: 0.4rem;
    margin-bottom: 0.35rem;
  }
  .stack { display: flex; flex-direction: column; gap: 0.2rem; }
  label { font-size: 0.8rem; color: #666; padding-top: 0.3rem; }
  input {
    padding: 0.25rem 0.4rem;
    border: 1px solid #ccc;
    border-radius: 3px;
    font: inherit;
    flex: 1;
    min-width: 6rem;
  }
  .aux {
    background: white;
    border: 1px solid #bbb;
    border-radius: 3px;
    padding: 0 0.4rem;
    cursor: pointer;
    font-size: 0.9rem;
    line-height: 1.2;
  }
  .aux:hover { background: #fff8e5; }
  .disclosure {
    background: none;
    border: none;
    color: #666;
    cursor: pointer;
    padding: 0.1rem 0;
    font-size: 0.8rem;
  }
  .link {
    background: none;
    border: none;
    color: #1a5490;
    cursor: pointer;
    padding: 0.15rem 0;
    font-size: 0.85rem;
    text-align: left;
  }
</style>
