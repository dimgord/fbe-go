<script lang="ts">
  import { createEventDispatcher } from "svelte";

  export let open = false;
  /** Id of the heading inside the slot that names the dialog. */
  export let labelledBy: string;
  /** Sizing and layout, which differ per dialog. */
  export let style = "";

  const dispatch = createEventDispatcher<{ close: void }>();
  const close = () => dispatch("close");
</script>

<!-- Keys are not handled here: each dialog listens on window, so Escape must be free to bubble. -->
{#if open}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <div class="backdrop" role="button" tabindex="-1" aria-label="Dismiss dialog" on:click={close}>
    <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
    <!-- svelte-ignore a11y-click-events-have-key-events -->
    <div
      class="dialog"
      role="dialog"
      aria-modal="true"
      aria-labelledby={labelledBy}
      {style}
      on:click|stopPropagation>
      <slot />
    </div>
  </div>
{/if}

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    background: var(--backdrop);
    display: grid;
    place-items: center;
    z-index: 100;
  }
  .dialog {
    position: relative;
    background: var(--bg-card);
    color: var(--fg);
    border: 1px solid var(--border);
    border-radius: 6px;
    box-shadow: 0 8px 24px var(--shadow);
  }
</style>
