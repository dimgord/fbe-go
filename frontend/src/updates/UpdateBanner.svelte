<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import type { updates as UpdatesNS } from "../../wailsjs/go/models";
  import { openExternalUrl } from "../runtime/externalLink";

  /** Null when no newer release has been found (or the check is still in
      flight / failed). Truthy when the banner should render. */
  export let info: UpdatesNS.Info | null = null;

  const dispatch = createEventDispatcher<{ dismiss: void }>();

  async function onDownload() {
    if (!info?.url) return;
    await openExternalUrl(info.url);
    // Don't auto-dismiss — user may want the banner to linger until they
    // actually finish the download. One click to download, one click to ×.
  }
</script>

{#if info && info.available}
  <div class="banner" role="status">
    <span class="pill">Update</span>
    <span class="msg">
      <strong>fbe-go {info.latestVersion}</strong> is available
      <span class="sub">(you're on {info.currentVersion})</span>
    </span>
    <div class="actions">
      <button type="button" class="primary" on:click={onDownload}>Download…</button>
      <button
        type="button"
        class="close"
        title="Dismiss this banner for the session"
        aria-label="Dismiss"
        on:click={() => dispatch("dismiss")}>×</button>
    </div>
  </div>
{/if}

<style>
  .banner {
    display: flex;
    align-items: center;
    gap: 0.6rem;
    padding: 0.35rem 0.8rem;
    background: var(--warn-bg-a);
    border-bottom: 1px solid var(--warn);
    color: var(--warn-fg);
    font-size: 0.85rem;
    font-family: -apple-system, "Segoe UI", sans-serif;
  }
  .pill {
    padding: 0.12rem 0.45rem;
    background: var(--warn);
    color: var(--bg-card);
    border-radius: 10px;
    font-size: 0.7rem;
    font-weight: 700;
    letter-spacing: 0.5px;
    text-transform: uppercase;
    flex-shrink: 0;
  }
  .msg { flex: 1; line-height: 1.3; }
  .msg strong { color: var(--fg-strong); font-weight: 600; }
  .msg .sub { color: var(--fg-muted); font-size: 0.8rem; margin-left: 0.35rem; }
  .actions { display: inline-flex; gap: 0.35rem; align-items: center; }
  button {
    padding: 0.22rem 0.7rem;
    border: 1px solid var(--border-button);
    background: var(--bg-surface);
    color: var(--fg);
    border-radius: 3px;
    cursor: pointer;
    font: inherit;
    font-size: 0.82rem;
  }
  button:hover { background: var(--bg-hover); }
  button.primary {
    background: var(--warn);
    color: var(--bg-card);
    border-color: var(--warn);
    font-weight: 600;
  }
  button.primary:hover { filter: brightness(1.05); }
  button.close {
    padding: 0 0.5rem;
    line-height: 1;
    height: 1.5rem;
    font-size: 1rem;
    color: var(--warn-fg);
    border: none;
    background: transparent;
  }
  button.close:hover { background: var(--warn-bg-b); }
</style>
