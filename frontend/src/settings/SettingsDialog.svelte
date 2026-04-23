<script lang="ts">
  import { createEventDispatcher, onMount } from "svelte";
  import type { settings as SettingsNS } from "../../wailsjs/go/models";
  import {
    HOTKEY_ACTIONS,
    displayAccel,
    accelFromEvent,
    formatAccel,
    findConflicts,
    type HotkeyCategory,
  } from "./hotkeys";

  export let open = false;

  const dispatch = createEventDispatcher<{
    close: void;
    apply: { theme: "system" | "light" | "dark"; settings: SettingsNS.Settings };
  }>();

  /** When truthy, the next keydown on window captures a new accel for this
      action id. Cleared by Escape, Enter, or successful capture. */
  let capturingId: string | null = null;
  /** Live preview of what the user just pressed while capturing. */
  let captureLive = "";

  // Grouped action list — stable order per category.
  const CATEGORIES: HotkeyCategory[] = ["File", "Edit", "Format", "Paragraph", "Blocks", "Dialogs"];
  const ACTIONS_BY_CATEGORY = CATEGORIES.reduce((acc, c) => {
    acc[c] = HOTKEY_ACTIONS.filter((a) => a.category === c);
    return acc;
  }, {} as Record<HotkeyCategory, typeof HOTKEY_ACTIONS>);

  // Draft state. Loaded on open (watched via $:) and mutated by inputs;
  // committed to disk only on Apply. Cancel discards without saving.
  let draft: SettingsNS.Settings | null = null;

  // System font list. Seeded with a handful of generic CSS fallbacks so the
  // dialog always has something to show on first open (if `ListSystemFonts`
  // hasn't finished enumerating yet, see app.go::populateSystemFonts).
  // Real list arrives asynchronously; we merge + dedupe.
  let fontFamilies: string[] = [
    "system-ui", "serif", "sans-serif", "monospace",
  ];

  // Custom combobox state (WebKit's built-in <datalist> dropdown is invisible
  // in WebKitGTK — no arrow affordance and the popup sometimes doesn't
  // appear — so we render our own list under the input).
  //
  // `fontFilter` is separate from `draft.font.family`: the input's value is
  // the stored/editing font, the filter is *only* populated while the user
  // is actively typing to narrow the visible list. Clicking the ▾ caret
  // opens the menu with the filter cleared so users can browse the full
  // list without having to first erase their current selection (the
  // behavior Dmitry reported in Rev 62 beta).
  let fontMenuOpen = false;
  let fontFilter = "";
  $: filteredFonts = fontFilter
    ? fontFamilies.filter((f) =>
        f.toLowerCase().includes(fontFilter.toLowerCase()))
    : fontFamilies;

  function toggleFontMenu() {
    if (fontMenuOpen) {
      fontMenuOpen = false;
      return;
    }
    fontFilter = "";
    fontMenuOpen = true;
  }
  function onFontInput(e: Event) {
    const val = (e.target as HTMLInputElement).value;
    fontFilter = val;
    fontMenuOpen = true;
  }
  function onFontFocus() {
    // Open menu on focus but DON'T auto-filter by the stored value —
    // users expect to browse the full list from this point.
    fontFilter = "";
    fontMenuOpen = true;
  }
  function selectFont(name: string) {
    if (draft) draft.font.family = name;
    fontFilter = "";
    fontMenuOpen = false;
  }
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
    // Fetch real font list from the OS. Don't block the main load path on
    // it — settings-load needs to unblock the dialog first, fonts can
    // arrive a tick later and merge into the datalist.
    try {
      const fonts = (await App.ListSystemFonts()) ?? [];
      if (fonts.length > 0) {
        const set = new Set<string>(fontFamilies);
        for (const f of fonts) set.add(f);
        fontFamilies = Array.from(set).sort((a, b) => a.localeCompare(b));
      }
    } catch (e) {
      console.warn("[fbe] list fonts failed:", e);
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
    // Capture mode: next real key becomes the accelerator. Escape cancels
    // capture without clearing the previous binding; Backspace clears.
    if (capturingId) {
      if (e.key === "Escape") {
        e.preventDefault();
        e.stopPropagation();
        capturingId = null;
        captureLive = "";
        return;
      }
      if (e.key === "Backspace" || e.key === "Delete") {
        e.preventDefault();
        e.stopPropagation();
        if (draft) {
          draft.hotkeys = { ...draft.hotkeys, [capturingId]: "" };
        }
        capturingId = null;
        captureLive = "";
        return;
      }
      const a = accelFromEvent(e);
      if (!a) return; // pure modifier press
      e.preventDefault();
      e.stopPropagation();
      const canon = formatAccel(a);
      captureLive = canon;
      if (draft) {
        draft.hotkeys = { ...draft.hotkeys, [capturingId]: canon };
      }
      capturingId = null;
      captureLive = "";
      return;
    }
    if (e.key === "Escape") {
      e.preventDefault();
      cancel();
    } else if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
      e.preventDefault();
      void apply();
    }
  }

  function startCapture(id: string) {
    capturingId = id;
    captureLive = "";
  }

  function clearHotkey(id: string) {
    if (!draft) return;
    draft.hotkeys = { ...draft.hotkeys, [id]: "" };
  }

  async function resetAllHotkeys() {
    const App = await wailsApp();
    if (!App || !draft) return;
    try {
      // Trip defaults-merge on the Go side: save an empty map, then reload —
      // settings.Load will backfill every action from DefaultHotkeys().
      await App.SaveSettings({ ...draft, hotkeys: {} } as SettingsNS.Settings);
      const s = await App.LoadSettings();
      if (s && draft) {
        draft.hotkeys = { ...s.hotkeys };
      }
    } catch (e) {
      console.warn("[fbe] reset hotkeys failed:", e);
    }
  }

  // Conflict map — canonical-accel → list of colliding action ids.
  $: hotkeyConflicts = draft ? findConflicts(draft.hotkeys ?? {}) : {};
  $: conflictsByAction = (() => {
    const out: Record<string, string> = {};
    for (const [accel, ids] of Object.entries(hotkeyConflicts)) {
      for (const id of ids) out[id] = accel;
    }
    return out;
  })();

  // Platform hint for the display form.
  const platform: "mac" | "other" =
    typeof navigator !== "undefined" && /mac|iphone|ipad/i.test(navigator.platform || navigator.userAgent || "")
      ? "mac"
      : "other";

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
            <div class="combobox">
              <input
                id="sd-font-family"
                type="text"
                bind:value={draft.font.family}
                placeholder="Trebuchet MS"
                autocomplete="off"
                style={`font-family: ${draft.font.family || "inherit"}`}
                on:focus={onFontFocus}
                on:input={onFontInput}
              />
              <button
                type="button"
                class="combobox-toggle"
                title="Show installed fonts"
                aria-label="Toggle font list"
                on:click={toggleFontMenu}
              >▾</button>
              {#if fontMenuOpen}
                <!-- svelte-ignore a11y-click-events-have-key-events -->
                <!-- svelte-ignore a11y-no-static-element-interactions -->
                <div class="combobox-backdrop" on:click={() => (fontMenuOpen = false)}></div>
                <ul class="combobox-menu" role="listbox">
                  {#each filteredFonts as f}
                    <li>
                      <button
                        type="button"
                        class="combobox-item"
                        style={`font-family: ${f}`}
                        on:click={() => selectFont(f)}
                      >{f}</button>
                    </li>
                  {:else}
                    <li class="combobox-empty">No match — your typed value will be saved as-is.</li>
                  {/each}
                </ul>
              {/if}
            </div>
            <span class="help">Type to filter; click a font to pick it. Custom family names are saved even if not in the list.</span>
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
          <h4>Keyboard shortcuts</h4>
          <p class="hotkey-help">
            Click a shortcut cell then press the keys to record.
            <strong>Esc</strong> cancels, <strong>Backspace</strong> clears.
            Duplicate bindings are highlighted but allowed — the topmost match
            wins at dispatch time.
          </p>
          {#each CATEGORIES as cat}
            {#if ACTIONS_BY_CATEGORY[cat].length > 0}
              <div class="hotkey-group">
                <h5>{cat}</h5>
                <table class="hotkey-table">
                  <tbody>
                    {#each ACTIONS_BY_CATEGORY[cat] as action}
                      {@const raw = draft.hotkeys?.[action.id] ?? ""}
                      {@const conflict = conflictsByAction[action.id]}
                      <tr>
                        <td class="hk-label">{action.label}</td>
                        <td class="hk-accel">
                          <button
                            type="button"
                            class="hk-btn"
                            class:capturing={capturingId === action.id}
                            class:conflict={!!conflict}
                            class:empty={!raw && capturingId !== action.id}
                            on:click={() => startCapture(action.id)}
                            title={conflict ? `Also used by: ${hotkeyConflicts[conflict].filter((i) => i !== action.id).join(", ")}` : ""}
                          >
                            {#if capturingId === action.id}
                              {captureLive || "Press any key…"}
                            {:else if raw}
                              {displayAccel(raw, platform)}
                            {:else}
                              Unbound
                            {/if}
                          </button>
                          {#if raw && capturingId !== action.id}
                            <button type="button" class="hk-clear" title="Clear binding" aria-label="Clear binding" on:click={() => clearHotkey(action.id)}>×</button>
                          {/if}
                        </td>
                      </tr>
                    {/each}
                  </tbody>
                </table>
              </div>
            {/if}
          {/each}
          <div class="row">
            <span class="label">All shortcuts</span>
            <button type="button" class="secondary" on:click={resetAllHotkeys}>Reset to defaults</button>
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
  /* Font-family field is wider so names like "Helvetica Neue" fit. */
  .row input#sd-font-family { min-width: 14rem; max-width: 22rem; }

  /* Explicit combobox (WebKit's <datalist> dropdown is invisible there). */
  .combobox {
    position: relative;
    display: inline-flex;
    align-items: stretch;
    max-width: 24rem;
  }
  .combobox input#sd-font-family {
    border-top-right-radius: 0;
    border-bottom-right-radius: 0;
    border-right: none;
  }
  button.combobox-toggle {
    padding: 0 0.55rem;
    border: 1px solid var(--border-input);
    background: var(--bg-chrome);
    color: var(--fg-secondary);
    border-top-right-radius: 3px;
    border-bottom-right-radius: 3px;
    cursor: pointer;
    font-size: 0.85rem;
    line-height: 1;
  }
  button.combobox-toggle:hover { background: var(--bg-hover); }
  .combobox-backdrop {
    position: fixed;
    inset: 0;
    background: transparent;
    z-index: 101;
  }
  ul.combobox-menu {
    position: absolute;
    top: calc(100% + 2px);
    left: 0;
    right: 0;
    z-index: 102;
    list-style: none;
    margin: 0;
    padding: 0.25rem 0;
    max-height: 18rem;
    overflow-y: auto;
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: 4px;
    box-shadow: 0 6px 18px var(--shadow);
    font-size: 0.88rem;
  }
  ul.combobox-menu li { margin: 0; padding: 0; }
  button.combobox-item {
    all: unset;
    display: block;
    width: 100%;
    padding: 0.25rem 0.7rem;
    cursor: pointer;
    box-sizing: border-box;
    color: var(--fg);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  button.combobox-item:hover { background: var(--bg-hover); }
  .combobox-empty {
    padding: 0.4rem 0.7rem;
    color: var(--fg-muted);
    font-size: 0.82rem;
    font-style: italic;
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

  /* Keyboard-shortcuts section. Per-category group with a tight two-column
     table: [action label | clickable key-capture button]. The button is
     the entire interactive surface — clicking it enters capture mode,
     pressing a key writes the binding. ✕ clears the binding separately. */
  .hotkey-help {
    margin: 0.2rem 0 0.6rem;
    color: var(--fg-muted);
    font-size: 0.8rem;
    line-height: 1.45;
  }
  .hotkey-group {
    margin: 0.2rem 0 0.8rem;
  }
  .hotkey-group h5 {
    margin: 0.6rem 0 0.3rem;
    font-size: 0.72rem;
    color: var(--fg-muted);
    text-transform: uppercase;
    letter-spacing: 0.5px;
    font-weight: 600;
  }
  table.hotkey-table {
    width: 100%;
    border-collapse: collapse;
  }
  table.hotkey-table tr { border-bottom: 1px solid var(--border); }
  table.hotkey-table tr:last-child { border-bottom: none; }
  td.hk-label {
    padding: 0.35rem 0.5rem 0.35rem 0;
    color: var(--fg);
    width: 55%;
  }
  td.hk-accel {
    padding: 0.25rem 0;
    text-align: right;
    white-space: nowrap;
  }
  button.hk-btn {
    min-width: 9rem;
    padding: 0.22rem 0.55rem;
    border: 1px solid var(--border-input);
    background: var(--bg-surface);
    color: var(--fg);
    border-radius: 3px;
    cursor: pointer;
    font: inherit;
    font-family: "SF Mono", Menlo, Consolas, monospace;
    font-size: 0.85rem;
    text-align: center;
  }
  button.hk-btn:hover { background: var(--bg-hover); }
  button.hk-btn.empty {
    color: var(--fg-muted);
    font-style: italic;
    font-family: inherit;
  }
  button.hk-btn.capturing {
    background: var(--bg-active);
    border-color: var(--warn);
    color: var(--fg-strong);
    font-style: italic;
  }
  button.hk-btn.conflict {
    border-color: var(--warn);
    background: var(--warn-bg-a);
    color: var(--warn-fg);
  }
  button.hk-clear {
    margin-left: 0.25rem;
    padding: 0 0.4rem;
    border: 1px solid var(--border);
    background: transparent;
    color: var(--fg-muted);
    border-radius: 3px;
    cursor: pointer;
    font-size: 0.85rem;
    line-height: 1;
    height: 1.4rem;
  }
  button.hk-clear:hover {
    background: var(--bg-hover);
    color: var(--fg-strong);
  }
</style>
