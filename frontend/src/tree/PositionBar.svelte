<script lang="ts">
  import type { BookPosition } from "./position";

  export let position: BookPosition | null = null;

  // The nearest levels matter most: the book's own long title would push the chapter off the bar.
  const SHOWN_LEVELS = 2;
  $: trail = position?.trail ?? [];
  $: where = trail.slice(-SHOWN_LEVELS).join(" › ");
</script>

{#if position}
  <div class="position-bar" role="status" aria-label="Current position in the book">
    <span class="where" title={trail.join(" › ")}>{where}</span>
    <span class="percent">{position.percent}%</span>
  </div>
{/if}

<style>
  .position-bar {
    position: sticky;
    bottom: 0;
    z-index: 1;
    display: flex;
    gap: 1rem;
    justify-content: space-between;
    padding: 0.15rem 0.75rem;
    background: var(--bg-chrome);
    border-top: 1px solid var(--border);
    color: var(--fg-secondary);
    font-size: 0.78rem;
  }
  .where {
    min-width: 0;
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
  }
  .percent { flex: none; }
</style>
