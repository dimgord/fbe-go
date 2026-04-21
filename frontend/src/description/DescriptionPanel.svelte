<script lang="ts">
  import type { FictionBook, TitleInfo } from "../fb2/types";
  import TitleInfoForm from "./TitleInfoForm.svelte";
  import DocumentInfoForm from "./DocumentInfoForm.svelte";
  import PublishInfoForm from "./PublishInfoForm.svelte";
  import CustomInfoForm from "./CustomInfoForm.svelte";

  export let fb: FictionBook;

  type Tab = "title" | "src-title" | "document" | "publish" | "custom";
  let activeTab: Tab = "title";

  $: availableBinaryIDs = (fb.Binaries ?? []).map((b) => b.ID);

  // Ensure optional containers exist so two-way binding has a target.
  $: if (!fb.Description.PublishInfo) fb.Description.PublishInfo = {};
  $: if (!fb.Description.CustomInfo)  fb.Description.CustomInfo  = [];

  function enableSrcTitle() {
    const empty: TitleInfo = {
      Genres: [],
      Authors: [{}],
      BookTitle: "",
      Lang: "",
    };
    fb.Description.SrcTitleInfo = empty;
  }
</script>

<div class="description">
  <div class="tabs" role="tablist">
    <button
      role="tab"
      aria-selected={activeTab === "title"}
      class:active={activeTab === "title"}
      on:click={() => (activeTab = "title")}>
      Title info
    </button>
    <button
      role="tab"
      aria-selected={activeTab === "src-title"}
      class:active={activeTab === "src-title"}
      on:click={() => (activeTab = "src-title")}>
      Source title
    </button>
    <button
      role="tab"
      aria-selected={activeTab === "document"}
      class:active={activeTab === "document"}
      on:click={() => (activeTab = "document")}>
      Document
    </button>
    <button
      role="tab"
      aria-selected={activeTab === "publish"}
      class:active={activeTab === "publish"}
      on:click={() => (activeTab = "publish")}>
      Publish
    </button>
    <button
      role="tab"
      aria-selected={activeTab === "custom"}
      class:active={activeTab === "custom"}
      on:click={() => (activeTab = "custom")}>
      Custom
    </button>
  </div>

  <div class="form-area">
    {#if activeTab === "title"}
      <TitleInfoForm bind:info={fb.Description.TitleInfo} {availableBinaryIDs} />
    {:else if activeTab === "src-title"}
      {#if fb.Description.SrcTitleInfo}
        <TitleInfoForm bind:info={fb.Description.SrcTitleInfo} {availableBinaryIDs} />
      {:else}
        <p class="todo">This document has no <code>&lt;src-title-info&gt;</code>.</p>
        <button class="prompt" on:click={enableSrcTitle}>Add source title info</button>
      {/if}
    {:else if activeTab === "document"}
      <DocumentInfoForm bind:info={fb.Description.DocumentInfo} />
    {:else if activeTab === "publish"}
      {#if fb.Description.PublishInfo}
        <PublishInfoForm bind:info={fb.Description.PublishInfo} />
      {/if}
    {:else if activeTab === "custom"}
      <CustomInfoForm bind:items={fb.Description.CustomInfo} />
    {/if}
  </div>
</div>

<style>
  .description {
    display: grid;
    grid-template-rows: auto 1fr;
    height: 100%;
    background: #fffdf8;
  }
  .tabs {
    display: flex;
    gap: 0.25rem;
    padding: 0.4rem 0.6rem 0 0.6rem;
    background: #eceae0;
    border-bottom: 1px solid #d5d5cb;
  }
  .tabs button {
    background: #f5f5f0;
    border: 1px solid #d5d5cb;
    border-bottom: none;
    padding: 0.35rem 0.7rem;
    cursor: pointer;
    border-radius: 4px 4px 0 0;
    font: inherit;
    font-size: 0.88rem;
  }
  .tabs button:hover { background: #fff8e5; }
  .tabs button.active {
    background: #fffdf8;
    border-bottom: 1px solid #fffdf8;
    margin-bottom: -1px;
    font-weight: 600;
  }
  .form-area {
    overflow: auto;
    padding: 1rem 1.5rem;
    max-width: 820px;
    width: 100%;
  }
  .todo {
    color: #888;
    font-style: italic;
  }
  code {
    background: #f5f5ef;
    padding: 0.15em 0.4em;
    border-radius: 3px;
    font-size: 0.85em;
  }
  .prompt {
    padding: 0.4rem 0.8rem;
    background: white;
    border: 1px solid #bbb;
    border-radius: 4px;
    cursor: pointer;
    font: inherit;
  }
  .prompt:hover { background: #fff8e5; }
</style>
