<script lang="ts">
  import { createEventDispatcher, onMount } from "svelte";

  export let open = false;

  let rows = 3;
  let cols = 3;
  let header = true;
  let rowsInput: HTMLInputElement | undefined;

  const dispatch = createEventDispatcher<{
    insert: { rows: number; cols: number; header: boolean };
    close: void;
  }>();

  $: if (open && rowsInput) {
    // Focus rows input when the dialog opens.
    rowsInput.focus();
    rowsInput.select();
  }

  function submit() {
    if (rows < 1 || cols < 1) return;
    dispatch("insert", { rows: Math.max(1, Math.min(50, rows)), cols: Math.max(1, Math.min(20, cols)), header });
    open = false;
  }

  function cancel() {
    open = false;
    dispatch("close");
  }

  function onKey(e: KeyboardEvent) {
    if (!open) return;
    if (e.key === "Escape") {
      e.preventDefault();
      cancel();
    } else if (e.key === "Enter") {
      e.preventDefault();
      submit();
    }
  }

  onMount(() => {
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  });
</script>

{#if open}
  <div
    class="backdrop"
    role="button"
    tabindex="-1"
    aria-label="Dismiss dialog"
    on:click={cancel}
    on:keydown={(e) => e.key === "Escape" && cancel()}>
    <div
      class="dialog"
      role="dialog"
      aria-modal="true"
      aria-labelledby="td-title"
      on:click|stopPropagation
      on:keydown|stopPropagation>
      <h3 id="td-title">Insert table</h3>
      <div class="row">
        <label for="td-rows">Rows</label>
        <input
          id="td-rows"
          type="number"
          min="1"
          max="50"
          bind:value={rows}
          bind:this={rowsInput} />
      </div>
      <div class="row">
        <label for="td-cols">Columns</label>
        <input id="td-cols" type="number" min="1" max="20" bind:value={cols} />
      </div>
      <div class="row check">
        <input id="td-header" type="checkbox" bind:checked={header} />
        <label for="td-header">First row is a header (&lt;th&gt;)</label>
      </div>
      <div class="actions">
        <button type="button" on:click={cancel}>Cancel</button>
        <button type="button" class="primary" on:click={submit}>Insert</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.35);
    display: grid;
    place-items: center;
    z-index: 100;
  }
  .dialog {
    background: #fffdf8;
    border: 1px solid #d5d5cb;
    border-radius: 6px;
    padding: 1.2rem 1.5rem;
    min-width: 20rem;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.25);
    font-family: inherit;
  }
  h3 {
    margin: 0 0 0.8rem 0;
    font-size: 1rem;
  }
  .row {
    display: grid;
    grid-template-columns: 7rem 1fr;
    align-items: center;
    gap: 0.5rem;
    margin-bottom: 0.5rem;
  }
  .row.check {
    grid-template-columns: auto 1fr;
  }
  label {
    font-size: 0.9rem;
    color: #444;
  }
  input[type="number"] {
    padding: 0.25rem 0.4rem;
    border: 1px solid #ccc;
    border-radius: 3px;
    font: inherit;
    width: 5rem;
  }
  .actions {
    display: flex;
    justify-content: flex-end;
    gap: 0.5rem;
    margin-top: 1rem;
  }
  button {
    padding: 0.35rem 0.9rem;
    border: 1px solid #bbb;
    background: white;
    border-radius: 4px;
    cursor: pointer;
    font: inherit;
  }
  button:hover { background: #fff8e5; }
  button.primary {
    background: #fce6a0;
    font-weight: 600;
    border-color: #b89a3e;
  }
  button.primary:hover { background: #f5da7c; }
</style>
