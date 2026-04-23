<script lang="ts">
  import { createEventDispatcher, onMount } from "svelte";
  import type { settings as SettingsNS } from "../../wailsjs/go/models";

  export let open = false;

  const dispatch = createEventDispatcher<{
    close: void;
    apply: { theme: "system" | "light" | "dark"; settings: SettingsNS.Settings };
  }>();

  // Draft state. Loaded on open (watched via $:) and mutated by inputs;
  // committed to disk only on Apply. Cancel discards without saving.
  let draft: SettingsNS.Settings | null = null;
  // Snapshot taken at open time — used as the Cancel baseline for fields
  // we've applied live (reset panes / clear recent); Apply writes `draft`.
  let loaded = false;

  async function wailsApp() {
    return await import("../../wailsjs/go/main/App").catch(() => null);
  }

  async function load() {
    const App = await wailsApp();
    if (!App) return;
    try {
      const s = await App.LoadSettings();
      draft = s;
      loaded = true;
    } catch (e) {
      console.warn("[fbe] load settings failed:", e);
    }
  }

  // Fire `load` whenever the dialog transitions from closed to open.
  //
  // Sequenced inside a single `$:` block intentionally — two separate
  // blocks (`$: if (open && !wasOpen) { load… }` and `$: wasOpen = open`)
  // get topologically sorted by Svelte, and the wasOpen-writer runs
  // FIRST, so the transition check always sees the new value and the
  // if-branch never fires. Inside one block, statements run top-to-
  // bottom deterministically.
  let wasOpen = false;
  $: {
    if (open && !wasOpen) {
      loaded = false;
      draft = null;
      void load();
    }
    wasOpen = open;
  }

  function cancel() {
    open = false;
    dispatch("close");
  }

  async function apply() {
    if (!draft) {
      cancel();
      return;
    }
    const App = await wailsApp();
    if (App) {
      try {
        await App.SaveSettings(draft);
      } catch (e) {
        console.warn("[fbe] save settings failed:", e);
      }
    }
    // Normalize theme to the union the parent expects.
    const t = draft.theme;
    const theme: "system" | "light" | "dark" =
      t === "light" || t === "dark" ? t : "system";
    dispatch("apply", { theme, settings: draft });
    open = false;
  }

  // --- In-dialog actions: these mutate disk immediately (not via draft)
  //     because "reset panes" / "clear recent" are one-shot commands, not
  //     editable fields. If the user then clicks Cancel, the mutation
  //     sticks — documented inline below.

  async function resetPanes() {
    const App = await wailsApp();
    if (!App) return;
    try {
      const s = await App.LoadSettings();
      s.panes = { outlineWidth: 0, validationWidth: 0, validationErrorsHeight: 0 };
      await App.SaveSettings(s);
      // Reload draft so the dialog keeps showing fresh state.
      await load();
      // Also reflect in the parent's live panes — need page reload for now,
      // but a hint in the help text tells the user that.
    } catch (e) {
      console.warn("[fbe] reset panes failed:", e);
    }
  }

  async function clearRecent() {
    const App = await wailsApp();
    if (!App) return;
    try {
      const s = await App.LoadSettings();
      s.recentFiles = [];
      await App.SaveSettings(s);
      await load();
    } catch (e) {
      console.warn("[fbe] clear recent failed:", e);
    }
  }

  function onKey(e: KeyboardEvent) {
    if (!open) return;
    if (e.key === "Escape") {
      e.preventDefault();
      cancel();
    } else if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
      e.preventDefault();
      void apply();
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
    aria-label="Dismiss settings"
    on:click={cancel}
    on:keydown={(e) => e.key === "Escape" && cancel()}>
    <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
    <div
      class="dialog"
      role="dialog"
      aria-modal="true"
      aria-labelledby="sd-title"
      on:click|stopPropagation
      on:keydown|stopPropagation>
      <header>
        <h3 id="sd-title">Settings</h3>
        <button type="button" class="close" on:click={cancel} title="Close without saving">×</button>
      </header>

      {#if !loaded || !draft}
        <p class="loading">Loading…</p>
      {:else}
        <section>
          <h4>Appearance</h4>
          <div class="row">
            <span class="label">Theme</span>
            <div class="radios">
              <label><input type="radio" bind:group={draft.theme} value="system" /> System</label>
              <label><input type="radio" bind:group={draft.theme} value="light" /> Light</label>
              <label><input type="radio" bind:group={draft.theme} value="dark" /> Dark</label>
            </div>
          </div>
        </section>

        <section>
          <h4>Editor</h4>
          <div class="row">
            <label class="label" for="sd-font-family">Font family</label>
            <input id="sd-font-family" type="text" bind:value={draft.font.family} placeholder="Trebuchet MS" />
          </div>
          <div class="row">
            <label class="label" for="sd-font-size">Font size</label>
            <input id="sd-font-size" type="number" min="8" max="32" bind:value={draft.font.size} />
          </div>
          <div class="row">
            <label class="label" for="sd-nbsp">NBSP char</label>
            <input id="sd-nbsp" type="text" maxlength="1" bind:value={draft.nbspChar} class="mono" />
            <span class="help">Inserted in place of runs of whitespace at paste time.</span>
          </div>
        </section>

        <section>
          <h4>Interface</h4>
          <div class="row">
            <label class="label" for="sd-lang">Language</label>
            <select id="sd-lang" bind:value={draft.interfaceLanguage} disabled>
              <option value="english">English</option>
            </select>
            <span class="help">Translations not yet available.</span>
          </div>
        </section>

        <section>
          <h4>Layout</h4>
          <div class="row">
            <span class="label">Pane sizes</span>
            <button type="button" class="secondary" on:click={resetPanes}>Reset to defaults</button>
            <span class="help">Outline / validation-panel widths, errors-pane height. Applies after app restart.</span>
          </div>
        </section>

        <section>
          <h4>Privacy</h4>
          <div class="row">
            <span class="label">Recent files</span>
            <button type="button" class="secondary" on:click={clearRecent}>
              Clear list{draft.recentFiles && draft.recentFiles.length > 0 ? ` (${draft.recentFiles.length})` : ""}
            </button>
          </div>
        </section>

        <div class="actions">
          <button type="button" on:click={cancel}>Cancel</button>
          <button type="button" class="primary" on:click={apply}>Apply</button>
        </div>
      {/if}
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
    background: var(--bg-card);
    color: var(--fg);
    border: 1px solid var(--border);
    border-radius: 6px;
    padding: 1rem 1.4rem 1.2rem;
    min-width: 32rem;
    max-width: 40rem;
    max-height: 85vh;
    overflow: auto;
    box-shadow: 0 8px 24px var(--shadow);
    font-family: -apple-system, "Segoe UI", sans-serif;
    font-size: 0.9rem;
  }
  header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 0.4rem;
  }
  h3 {
    margin: 0;
    font-size: 1.05rem;
  }
  h4 {
    margin: 1rem 0 0.4rem;
    font-size: 0.78rem;
    color: var(--fg-secondary);
    text-transform: uppercase;
    letter-spacing: 0.6px;
  }
  .close {
    border: none;
    background: transparent;
    font-size: 1.2rem;
    line-height: 1;
    padding: 0.1rem 0.45rem;
    color: var(--fg-muted);
    cursor: pointer;
    border-radius: 3px;
  }
  .close:hover { background: var(--bg-hover); color: var(--fg-strong); }

  .loading { color: var(--fg-muted); margin: 1rem 0; }

  .row {
    display: grid;
    grid-template-columns: 10rem 1fr;
    align-items: baseline;
    gap: 0.6rem;
    padding: 0.3rem 0;
  }
  .row .label { color: var(--fg-secondary); }
  .row .help { grid-column: 2; color: var(--fg-muted); font-size: 0.78rem; }
  .row input[type="text"],
  .row input[type="number"],
  .row select {
    padding: 0.28rem 0.45rem;
    border: 1px solid var(--border-input);
    background: var(--bg-surface);
    color: var(--fg);
    border-radius: 3px;
    font: inherit;
    max-width: 18rem;
  }
  .row input.mono {
    font-family: "SF Mono", Menlo, Consolas, monospace;
    width: 3rem;
    text-align: center;
  }
  .row .radios {
    display: inline-flex;
    gap: 0.8rem;
  }
  .row .radios label {
    display: inline-flex;
    align-items: center;
    gap: 0.3rem;
  }

  .actions {
    display: flex;
    justify-content: flex-end;
    gap: 0.5rem;
    margin-top: 1.2rem;
    padding-top: 0.6rem;
    border-top: 1px solid var(--border);
  }
  .actions button,
  button.secondary {
    padding: 0.35rem 0.9rem;
    border: 1px solid var(--border-button);
    background: var(--bg-surface);
    color: var(--fg);
    border-radius: 4px;
    cursor: pointer;
    font: inherit;
  }
  .actions button:hover,
  button.secondary:hover { background: var(--bg-hover); }
  .actions button.primary {
    background: var(--bg-active);
    font-weight: 600;
    border-color: var(--warn);
  }
  .actions button.primary:hover { background: var(--bg-active-hover); }
  button.secondary {
    padding: 0.25rem 0.7rem;
    font-size: 0.85rem;
  }
</style>
