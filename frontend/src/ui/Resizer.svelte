<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import { clamp } from "../lib/math";

  /** "vertical" is a bar between columns, "horizontal" one between rows. */
  export let orientation: "vertical" | "horizontal" = "vertical";
  export let label: string;
  /** The size now; arrow keys step from it. */
  export let current: () => number;
  export let bounds: () => { min: number; max: number };
  /** The size a pointer position asks for. */
  export let fromPointer: (e: PointerEvent) => number;
  export let dragging = false;

  const SMALL_STEP = 10;
  const LARGE_STEP = 40;

  /** `change` fires on every step of a drag; `commit` once it is final, for persisting. */
  const dispatch = createEventDispatcher<{ change: number; commit: number }>();

  let last: number | null = null;

  function resize(size: number) {
    const { min, max } = bounds();
    last = clamp(size, min, max);
    dispatch("change", last);
  }

  function start(e: PointerEvent) {
    e.preventDefault();
    dragging = true;
    last = null;
    (e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
    document.body.style.cursor = orientation === "vertical" ? "ew-resize" : "ns-resize";
    document.body.style.userSelect = "none";
  }

  function move(e: PointerEvent) {
    if (dragging) resize(fromPointer(e));
  }

  function end(e: PointerEvent) {
    if (!dragging) return;
    dragging = false;
    const el = e.currentTarget as HTMLElement;
    if (el.hasPointerCapture(e.pointerId)) el.releasePointerCapture(e.pointerId);
    document.body.style.cursor = "";
    document.body.style.userSelect = "";
    if (last !== null) dispatch("commit", last);
  }

  function key(e: KeyboardEvent) {
    const [less, more] = orientation === "vertical" ? ["ArrowLeft", "ArrowRight"] : ["ArrowDown", "ArrowUp"];
    if (e.key !== less && e.key !== more) return;
    e.preventDefault();
    const step = e.shiftKey ? LARGE_STEP : SMALL_STEP;
    resize(current() + (e.key === more ? step : -step));
    if (last !== null) dispatch("commit", last);
  }
</script>

<!-- svelte-ignore a11y-no-noninteractive-tabindex -->
<!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
<div
  class="resizer {orientation}"
  class:dragging
  role="separator"
  aria-orientation={orientation}
  aria-label={label}
  tabindex="0"
  on:pointerdown={start}
  on:pointermove={move}
  on:pointerup={end}
  on:pointercancel={end}
  on:keydown={key}
></div>

<style>
  .resizer {
    position: relative;
    background: var(--border);
    touch-action: none;
  }
  .resizer.vertical {
    cursor: ew-resize;
    border-left: 1px solid var(--border-strong);
    border-right: 1px solid var(--border-strong);
  }
  .resizer.horizontal {
    height: 6px;
    cursor: ns-resize;
    border-top: 1px solid var(--border-strong);
    border-bottom: 1px solid var(--border-strong);
  }
  .resizer::before {
    content: "";
    position: absolute;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    background: var(--fg-muted);
    border-radius: 1px;
  }
  .vertical::before {
    width: 2px;
    height: 32px;
    box-shadow: 3px 0 0 var(--fg-muted);
  }
  .horizontal::before {
    width: 32px;
    height: 2px;
    box-shadow: 0 3px 0 var(--fg-muted);
  }
  .resizer:hover,
  .resizer.dragging,
  .resizer:focus-visible {
    background: var(--border-strong);
    outline: none;
  }
  .resizer:focus-visible::before {
    background: var(--fg-secondary);
  }
  .vertical:focus-visible::before { box-shadow: 3px 0 0 var(--fg-secondary); }
  .horizontal:focus-visible::before { box-shadow: 0 3px 0 var(--fg-secondary); }
</style>
