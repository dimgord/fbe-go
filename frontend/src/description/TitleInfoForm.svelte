<script lang="ts">
  import type { TitleInfo, Author, Genre, Sequence } from "../fb2/types";
  import AuthorField from "./AuthorField.svelte";
  import GenreField from "./GenreField.svelte";
  import DateField from "./DateField.svelte";
  import SequenceField from "./SequenceField.svelte";
  import CoverpageField from "./CoverpageField.svelte";
  import AnnotationEditor from "./AnnotationEditor.svelte";
  import type { Annotation } from "../fb2/types";
  import { uid } from "../lib/uid";

  export let info: TitleInfo;
  /** For the coverpage dropdown — IDs of binaries available on the book. */
  export let availableBinaryIDs: string[] = [];

  const id_ = uid("ti");

  function onAnnotationChange(e: CustomEvent<Annotation>) {
    info.Annotation = e.detail;
  }

  // Ensure optional arrays exist so two-way binding has a stable target.
  $: if (!info.Translators) info.Translators = [];
  $: if (!info.Sequences)   info.Sequences   = [];

  function addAuthor() { info.Authors = [...info.Authors, {} as Author]; }
  function removeAuthor(i: number) { info.Authors = info.Authors.filter((_, idx) => idx !== i); }
  function cloneAuthor(i: number) {
    info.Authors = [...info.Authors.slice(0, i + 1), JSON.parse(JSON.stringify(info.Authors[i])), ...info.Authors.slice(i + 1)];
  }

  function addGenre() { info.Genres = [...info.Genres, { Value: "" }]; }
  function removeGenre(i: number) { info.Genres = info.Genres.filter((_, idx) => idx !== i); }
  function cloneGenre(i: number) {
    info.Genres = [...info.Genres.slice(0, i + 1), { ...info.Genres[i] }, ...info.Genres.slice(i + 1)];
  }

  function addTranslator() { info.Translators = [...(info.Translators ?? []), {} as Author]; }
  function removeTranslator(i: number) { info.Translators = (info.Translators ?? []).filter((_, idx) => idx !== i); }
  function cloneTranslator(i: number) {
    const list = info.Translators ?? [];
    info.Translators = [...list.slice(0, i + 1), JSON.parse(JSON.stringify(list[i])), ...list.slice(i + 1)];
  }

  function addSequence() { info.Sequences = [...(info.Sequences ?? []), { Name: "" } as Sequence]; }
  function removeSequence(i: number) { info.Sequences = (info.Sequences ?? []).filter((_, idx) => idx !== i); }
  function cloneSequence(i: number) {
    const list = info.Sequences ?? [];
    info.Sequences = [...list.slice(0, i + 1), JSON.parse(JSON.stringify(list[i])), ...list.slice(i + 1)];
  }
</script>

<section class="ti">
  <h3>Genres</h3>
  {#each info.Genres as genre, i (i)}
    <GenreField
      bind:genre={info.Genres[i]}
      on:remove={() => removeGenre(i)}
      on:clone={() => cloneGenre(i)} />
  {/each}
  <button class="link" type="button" on:click={addGenre}>+ add genre</button>

  <h3>Authors</h3>
  {#each info.Authors as _, i (i)}
    <AuthorField
      variant="primary"
      bind:author={info.Authors[i]}
      on:remove={() => removeAuthor(i)}
      on:clone={() => cloneAuthor(i)} />
  {/each}
  <button class="link" type="button" on:click={addAuthor}>+ add author</button>

  <h3>Book</h3>
  <div class="row">
    <label for={`${id_}-title`}>Title</label>
    <input id={`${id_}-title`} class="wide" bind:value={info.BookTitle} />
  </div>
  <div class="row">
    <label for={`${id_}-kw`}>Keywords</label>
    <input id={`${id_}-kw`} class="wide" bind:value={info.Keywords} placeholder="comma, separated, keywords" />
  </div>
  <DateField bind:date={info.Date} label="Date" />
  <div class="row">
    <label for={`${id_}-lang`}>Lang</label>
    <input id={`${id_}-lang`} class="short" bind:value={info.Lang} placeholder="uk" maxlength="10" />
    <label for={`${id_}-src-lang`}>Source lang</label>
    <input id={`${id_}-src-lang`} class="short" bind:value={info.SrcLang} placeholder="ru" maxlength="10" />
  </div>

  <h3>Annotation</h3>
  <AnnotationEditor annotation={info.Annotation ?? { Children: [] }} on:change={onAnnotationChange} />

  <h3>Coverpage</h3>
  <CoverpageField bind:cover={info.Coverpage} {availableBinaryIDs} />

  <h3>Translators</h3>
  {#if info.Translators}
    {#each info.Translators as _, i (i)}
      <AuthorField
        variant="compact"
        bind:author={info.Translators[i]}
        on:remove={() => removeTranslator(i)}
        on:clone={() => cloneTranslator(i)} />
    {/each}
  {/if}
  <button class="link" type="button" on:click={addTranslator}>+ add translator</button>

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
  .ti { display: flex; flex-direction: column; }
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
  label { font-size: 0.8rem; color: #666; min-width: 5.5rem; }
  input {
    padding: 0.25rem 0.4rem;
    border: 1px solid #ccc;
    border-radius: 3px;
    font: inherit;
  }
  .wide { flex: 1; }
  .short { flex: 0 0 6rem; }
  .link {
    background: none; border: none; color: #1a5490;
    cursor: pointer; padding: 0.15rem 0; font-size: 0.85rem; text-align: left;
    align-self: flex-start;
  }
</style>
