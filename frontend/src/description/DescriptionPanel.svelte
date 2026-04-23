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

  function emptyTitleInfo(): TitleInfo {
    return {
      Genres: [],
      Authors: [{}],
      BookTitle: "",
      Lang: "",
    };
  }

  function enableTitleInfo() {
    fb.Description.TitleInfo = emptyTitleInfo();
  }

  function enableSrcTitle() {
    fb.Description.SrcTitleInfo = emptyTitleInfo();
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
      {#if fb.Description.TitleInfo}
        <TitleInfoForm bind:info={fb.Description.TitleInfo} {availableBinaryIDs} />
      {:else}
        <p class="todo">This document has no <code>&lt;title-info&gt;</code>.</p>
        <button class="prompt" on:click={enableTitleInfo}>Add title info</button>
      {/if}
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
    background: var(--bg-card);
    color: var(--fg);
  }
  .tabs {
    display: flex;
    gap: 0.25rem;
    padding: 0.4rem 0.6rem 0 0.6rem;
    background: var(--bg-chrome);
    border-bottom: 1px solid var(--border);
  }
  .tabs button {
    background: var(--bg-sidebar);
    color: var(--fg);
    border: 1px solid var(--border);
    border-bottom: none;
    padding: 0.35rem 0.7rem;
    cursor: pointer;
    border-radius: 4px 4px 0 0;
    font: inherit;
    font-size: 0.88rem;
  }
  .tabs button:hover { background: var(--bg-hover); }
  .tabs button.active {
    background: var(--bg-card);
    border-bottom: 1px solid var(--bg-card);
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
    color: var(--fg-muted);
    font-style: italic;
  }
  code {
    background: var(--bg-chrome);
    padding: 0.15em 0.4em;
    border-radius: 3px;
    font-size: 0.85em;
  }
  .prompt {
    padding: 0.4rem 0.8rem;
    background: var(--bg-surface);
    color: var(--fg);
    border: 1px solid var(--border-button);
    border-radius: 4px;
    cursor: pointer;
    font: inherit;
  }
  .prompt:hover { background: var(--bg-hover); }
</style>
