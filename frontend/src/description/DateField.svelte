<script lang="ts">
  import type { DateVal } from "../fb2/types";
  import { uid } from "../lib/uid";

  export let date: DateVal | null | undefined;
  export let label = "Date";

  const id_ = uid("date");

  /** Ensure the underlying object exists before binding. */
  $: if (!date) date = { Text: "", Value: "" };
</script>

{#if date}
  <div class="date">
    <label for={`${id_}-text`}>{label}</label>
    <input id={`${id_}-text`} placeholder="Text (e.g. 21 April 2026)" bind:value={date.Text} />
    <input
      class="iso"
      aria-label="{label} (ISO value)"
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
    color: var(--fg-secondary);
    min-width: 4rem;
  }
  input {
    padding: 0.25rem 0.4rem;
    border: 1px solid var(--border-input);
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
