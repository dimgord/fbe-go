<script lang="ts">
  import { createEventDispatcher, tick } from "svelte";

  export let xmlSource: string = "";
  export let errors: { line: number; column: number; message: string }[] = [];

  const dispatch = createEventDispatcher<{ close: void }>();

  let xmlPane: HTMLDivElement | undefined;
  let highlightedLine: number | null = null;

  $: xmlLines = xmlSource.split("\n");

  async function gotoLine(n: number) {
    highlightedLine = n;
    await tick();
    const el = xmlPane?.querySelector(`#xml-line-${n}`);
    el?.scrollIntoView({ block: "center", behavior: "smooth" });
    setTimeout(() => {
      if (highlightedLine === n) highlightedLine = null;
    }, 2500);
  }

</script>

<div class="panel">
  <div class="panel-title">
    <span>XML source{xmlLines.length > 0 ? ` · ${xmlLines.length} lines` : ""}</span>
    <button on:click={() => dispatch("close")} title="Close panel">×</button>
  </div>

  <div class="xml" bind:this={xmlPane}>
    {#each xmlLines as line, i}
      <div
        id={`xml-line-${i + 1}`}
        class="xml-line"
        class:hl={highlightedLine === i + 1}
      >
        <span class="ln">{i + 1}</span>
        <span class="content">{line}</span>
      </div>
    {/each}
  </div>

  {#if errors.length > 0}
    <div class="errors">
      <div class="errors-title">
        <span class="dot">●</span>
        {errors.length} XSD error{errors.length === 1 ? "" : "s"}
      </div>
      <ul>
        {#each errors as e}
          <li>
            <button
              type="button"
              on:click={() => gotoLine(e.line)}
              title="Jump to line {e.line}"
            >
              <span class="pos">L{e.line}:{e.column}</span>
              <span class="msg">{e.message}</span>
            </button>
          </li>
        {/each}
      </ul>
    </div>
  {:else if xmlSource}
    <div class="errors ok">
      <span class="dot">✓</span> XSD valid
    </div>
  {/if}
</div>

<style>
  .panel {
    display: grid;
    grid-template-rows: 2rem 1fr auto;
    height: 100%;
    min-height: 0;
    border-left: 1px solid #d5d5cb;
    background: #fdfdfa;
    font-family: "SF Mono", Menlo, Consolas, monospace;
    font-size: 0.78rem;
  }

  .panel-title {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 0.25rem 0 0.6rem;
    background: #f1f1ec;
    border-bottom: 1px solid #d5d5cb;
    font-family: -apple-system, "Segoe UI", sans-serif;
    font-size: 0.8rem;
    font-weight: 600;
    color: #333;
  }
  .panel-title button {
    border: none;
    background: transparent;
    font-size: 1.1rem;
    line-height: 1;
    padding: 0.15rem 0.5rem;
    color: #666;
    cursor: pointer;
    border-radius: 3px;
  }
  .panel-title button:hover {
    background: #e8e4d8;
    color: #111;
  }

  .xml {
    overflow: auto;
    padding: 0.25rem 0;
    min-height: 0;
  }
  .xml-line {
    display: grid;
    grid-template-columns: 3.25rem 1fr;
    gap: 0.35rem;
    padding: 0 0.25rem;
    white-space: pre;
    line-height: 1.35;
  }
  .xml-line .ln {
    color: #aaa;
    text-align: right;
    user-select: none;
    border-right: 1px solid #eee6d4;
    padding-right: 0.35rem;
  }
  .xml-line.hl {
    background: #fce6a0;
    transition: background 0.4s ease-out;
  }
  .xml-line.hl .ln {
    color: #7a5a10;
    font-weight: 600;
  }

  .errors {
    max-height: 42%;
    overflow: auto;
    border-top: 1px solid #d5d5cb;
    background: #fffaf0;
    font-family: -apple-system, "Segoe UI", sans-serif;
    font-size: 0.8rem;
  }
  .errors-title {
    position: sticky;
    top: 0;
    z-index: 1;
    background: #fdecec;
    color: #a33;
    padding: 0.3rem 0.6rem;
    border-bottom: 1px solid #edc7c7;
    font-size: 0.72rem;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    font-weight: 700;
  }
  .errors .dot { margin-right: 0.25rem; }

  .errors.ok {
    color: #2a7;
    padding: 0.5rem 0.75rem;
    background: #f2faf4;
    border-top: 1px solid #cfe7d6;
    font-weight: 600;
  }

  .errors ul {
    list-style: none;
    margin: 0;
    padding: 0;
  }
  .errors li {
    border-bottom: 1px solid #f3e8d0;
  }
  .errors li button {
    all: unset;
    display: grid;
    grid-template-columns: 4.5rem 1fr;
    gap: 0.5rem;
    align-items: start;
    width: 100%;
    padding: 0.3rem 0.6rem;
    box-sizing: border-box;
    cursor: pointer;
    text-align: left;
  }
  .errors li button:hover { background: #fff3cc; }
  .errors li button:focus-visible {
    outline: 2px solid #c99;
    outline-offset: -2px;
    background: #fff3cc;
  }
  .errors li .pos {
    color: #888;
    font-family: "SF Mono", Menlo, Consolas, monospace;
    font-size: 0.72rem;
  }
  .errors li .msg {
    color: #a33;
    word-break: break-word;
    white-space: normal;
  }
</style>
