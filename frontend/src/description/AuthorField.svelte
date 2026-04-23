<script lang="ts">
  import type { Author } from "../fb2/types";
  import { createEventDispatcher } from "svelte";
  import { uid } from "../lib/uid";

  export let author: Author;
  /** "primary" shows all fields; "compact" hides nick/email/home behind a disclosure. */
  export let variant: "primary" | "compact" = "compact";

  let open = variant === "primary" || !!(author.Nickname || author.Email?.length || author.HomePage?.length);

  const id_ = uid("author");
  const dispatch = createEventDispatcher<{ remove: void; clone: void }>();

  // Ensure optional arrays exist for two-way binding, then expose
  // narrowed locals — Svelte's template parser rejects `!` inside
  // `bind:value={…}`, so the assertion has to live in <script>.
  $: if (!author.Email)    author.Email    = [];
  $: if (!author.HomePage) author.HomePage = [];
  $: emails    = author.Email!;
  $: homepages = author.HomePage!;

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
      <label for={`${id_}-nick`}>Nick</label>
      <input id={`${id_}-nick`} bind:value={author.Nickname} />
    </div>
    <div class="row">
      <label for={`${id_}-id`}>ID</label>
      <input id={`${id_}-id`} bind:value={author.ID} />
    </div>
    <div class="multi">
      <label for={`${id_}-email-0`}>Email</label>
      <div class="stack">
        {#each emails as _, i}
          <div class="inline">
            <input id={i === 0 ? `${id_}-email-0` : undefined} bind:value={emails[i]} />
            <button class="aux" type="button" on:click={() => removeEmail(i)}>×</button>
          </div>
        {/each}
        <button class="link" type="button" on:click={addEmail}>+ add email</button>
      </div>
    </div>
    <div class="multi">
      <label for={`${id_}-home-0`}>Home page</label>
      <div class="stack">
        {#each homepages as _, i}
          <div class="inline">
            <input id={i === 0 ? `${id_}-home-0` : undefined} bind:value={homepages[i]} />
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
    border: 1px solid var(--border);
    padding: 0.5rem;
    border-radius: 4px;
    background: var(--bg-card);
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
  label { font-size: 0.8rem; color: var(--fg-secondary); padding-top: 0.3rem; }
  input {
    padding: 0.25rem 0.4rem;
    border: 1px solid var(--border-input);
    border-radius: 3px;
    font: inherit;
    flex: 1;
    min-width: 6rem;
  }
  .aux {
    background: var(--bg-surface);
    border: 1px solid var(--border-button);
    border-radius: 3px;
    padding: 0 0.4rem;
    cursor: pointer;
    font-size: 0.9rem;
    line-height: 1.2;
  }
  .aux:hover { background: var(--bg-hover); }
  .disclosure {
    background: none;
    border: none;
    color: var(--fg-secondary);
    cursor: pointer;
    padding: 0.1rem 0;
    font-size: 0.8rem;
  }
  .link {
    background: none;
    border: none;
    color: var(--fg-link);
    cursor: pointer;
    padding: 0.15rem 0;
    font-size: 0.85rem;
    text-align: left;
  }
</style>
