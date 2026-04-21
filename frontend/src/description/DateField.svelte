<script lang="ts">
  import type { DateVal } from "../fb2/types";
  export let date: DateVal | null | undefined;
  export let label = "Date";

  /** Ensure the underlying object exists before binding. */
  $: if (!date) date = { Text: "", Value: "" };
</script>

{#if date}
  <div class="date">
    <label>{label}</label>
    <input placeholder="Text (e.g. 21 April 2026)" bind:value={date.Text} />
    <input
      class="iso"
      placeholder="ISO yyyy-mm-dd"
      bind:value={date.Value}
    />
  </div>
{/if}

<style>
  .date {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    margin-bottom: 0.3rem;
  }
  label {
    font-size: 0.8rem;
    color: #666;
    min-width: 4rem;
  }
  input {
    padding: 0.25rem 0.4rem;
    border: 1px solid #ccc;
    border-radius: 3px;
    font: inherit;
    flex: 1;
  }
  .iso {
    flex: 0 0 11rem;
    font-family: "SF Mono", Menlo, monospace;
    font-size: 0.88rem;
  }
</style>
