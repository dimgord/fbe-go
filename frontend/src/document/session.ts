/**
 * The open document and what can happen to it: open, save, export, validate, apply XML, and
 * whether there are unsaved edits. No DOM, so every flow can run against a fake backend.
 *
 * The live body sits in ProseMirror inside Editor, which exists only in the body view. `doc` is
 * therefore a snapshot, and `liveBody` is how the session asks for the current text.
 */
import { get, writable } from "svelte/store";
import { backend } from "../runtime/backend";
import { basename } from "../lib/path";
import { SAMPLE_BOOK } from "../fb2/sample";
import type { FictionBook } from "../fb2/types";
import { descriptionSnapshot } from "./dirty";

export interface SessionHooks {
  /** The body as edited, or null when no editor is mounted. */
  liveBody: () => FictionBook | null;
  /** Called after a save, so the editor can treat this state as clean. */
  onSaved: () => void;
}

export interface XmlProblem {
  line: number;
  column: number;
  message: string;
}

const STATUS_MS = 3000;

export function createDocumentSession(hooks: SessionHooks) {
  const doc = writable<FictionBook | null>(null);
  const filename = writable("(untitled)");
  const status = writable("");
  const error = writable("");
  const recentFiles = writable<string[]>([]);

  const panelOpen = writable(false);
  const xmlSource = writable("");
  const xmlErrors = writable<XmlProblem[]>([]);
  const applyError = writable("");
  const canUndoApply = writable(false);

  let path = "";
  let statusTimer: ReturnType<typeof setTimeout> | undefined;

  // Body edits are tracked by the editor's undo stack, description edits by comparing snapshots:
  // the parser and the PM serializer build different trees for the same body, so only the
  // description can be compared. The XML panel reports its own unapplied edits.
  let bodyDirty = false;
  let xmlDirty = false;
  let cleanDescription: string | null = null;

  let undoDoc: FictionBook | null = null;
  let undoXml = "";

  /** A transient message. A newer one cancels the older one's timer, so it is not cut short. */
  function showStatus(message: string, ms = STATUS_MS): void {
    clearTimeout(statusTimer);
    status.set(message);
    statusTimer = setTimeout(() => status.set(""), ms);
  }

  function clearMessages(): void {
    error.set("");
    status.set("");
  }

  function isDirty(): boolean {
    if (bodyDirty || xmlDirty) return true;
    return cleanDescription !== null && descriptionSnapshot(get(doc)) !== cleanDescription;
  }

  /** The current state becomes the clean baseline. */
  function markClean(): void {
    cleanDescription = descriptionSnapshot(get(doc));
    bodyDirty = false;
  }

  /** The document as it stands now. */
  function live(): FictionBook | null {
    return hooks.liveBody() ?? get(doc);
  }

  function showSample(): void {
    doc.set(SAMPLE_BOOK);
    path = "";
    filename.set("blank.fb2 (sample)");
    markClean();
  }

  async function refreshRecent(): Promise<void> {
    try {
      recentFiles.set((await backend()?.RecentFiles()) ?? []);
    } catch { /* keep the old list */ }
  }

  /** Open `preset`, or ask the user. Unsaved edits are offered a save first. */
  async function open(preset?: string): Promise<void> {
    clearMessages();
    const App = backend();
    try {
      if (!App) throw new Error("Wails bindings not available — running in plain vite dev. Loaded bundled sample.");
      if (isDirty()) {
        const choice = await App.ConfirmUnsavedChanges();
        if (choice === "cancel") return;
        if (choice === "save") {
          await save(false);
          if (isDirty()) return; // the save failed, or the picker was cancelled
        }
      }
      const target = preset ?? (await App.PickFB2ToOpen());
      if (!target) return;
      status.set(`Opening ${basename(target)}…`);
      const parsed = await App.OpenFile(target);
      // Let the status paint before converting a huge document blocks the UI thread.
      await new Promise((r) => setTimeout(r, 0));
      doc.set(parsed);
      path = target;
      filename.set(basename(target));
      showStatus(`Opened ${basename(target)}`);
      markClean();
      void refreshRecent();
    } catch (e) {
      console.error("[fbe] open failed:", e);
      error.set(`Open failed: ${(e as Error).message || String(e)}`);
      if (preset) {
        // A recent entry that no longer opens should not keep being offered.
        try { await App?.RemoveFromRecent(preset); } catch { /* ignore */ }
        void refreshRecent();
      } else {
        showSample();
      }
    }
  }

  async function save(saveAs: boolean): Promise<void> {
    clearMessages();
    try {
      const App = backend();
      if (!App) throw new Error("Wails bindings not available — save works only in the desktop app.");
      const current = live();
      if (!current) throw new Error("No document loaded.");
      let target = path;
      if (saveAs || !target) {
        const name = get(filename);
        target = await App.PickFB2ToSave(name.endsWith(".fb2") ? name : "untitled.fb2");
        if (!target) return;
      }
      await App.UpdateDocument(current);
      await App.SaveFile(target);
      path = target;
      filename.set(basename(target));
      showStatus(`Saved ${basename(target)}`);
      markClean();
      hooks.onSaved();
      void refreshRecent();
    } catch (e) {
      error.set((e as Error).message);
    }
  }

  async function exportHtml(): Promise<void> {
    clearMessages();
    try {
      const App = backend();
      if (!App) throw new Error("Wails bindings not available.");
      const current = live();
      if (current) await App.UpdateDocument(current);
      const stem = (get(filename) || "untitled").replace(/\.fb2(\.zip)?$/i, "");
      const target = await App.PickHTMLToSave(`${stem}.html`);
      if (!target) return;
      await App.ExportHTML(target);
      showStatus(`Exported ${basename(target)}`);
    } catch (e) {
      error.set((e as Error).message);
    }
  }

  /** Serialize and validate the live document, and show the XML panel. */
  async function validate(): Promise<void> {
    clearMessages();
    try {
      const App = backend();
      if (!App) throw new Error("Wails bindings not available.");
      const current = live();
      if (!current) throw new Error("No document loaded.");
      await App.UpdateDocument(current);
      const [xml, errs] = await Promise.all([App.SerializeCurrent(), App.ValidateCurrent()]);
      xmlSource.set(xml ?? "");
      xmlErrors.set(errs ?? []);
      panelOpen.set(true);
      showStatus(errs?.length ? `XSD: ${errs.length} error(s)` : "XSD valid ✓", 4000);
    } catch (e) {
      error.set((e as Error).message);
    }
  }

  /** Replace the document from edited XML. On failure the error is shown and the text is kept. */
  async function applyXml(xml: string): Promise<void> {
    applyError.set("");
    const App = backend();
    if (!App) {
      applyError.set("Wails bindings unavailable");
      return;
    }
    try {
      const parsed = await App.ParseXMLString(xml);
      if (!parsed) throw new Error("parser returned null");
      const before = get(doc);
      undoDoc = before ? structuredClone(before) : null;
      undoXml = get(xmlSource);
      canUndoApply.set(undoDoc !== null);
      doc.set(parsed);
      await App.UpdateDocument(parsed);
      try { xmlSource.set(await App.SerializeCurrent()); } catch { /* keep the old text */ }
      try { xmlErrors.set((await App.ValidateCurrent()) ?? []); } catch { /* keep the old list */ }
      markClean();
      showStatus("XML applied — Undo Apply is available in the panel", 5000);
    } catch (e) {
      applyError.set((e as Error).message || String(e));
    }
  }

  /** One level of undo for Apply: restore what the last Apply replaced. */
  async function undoApply(): Promise<void> {
    if (!undoDoc) return;
    const restored = undoDoc;
    doc.set(restored);
    xmlSource.set(undoXml);
    forgetUndo();
    const App = backend();
    if (App) {
      try { await App.UpdateDocument(restored); } catch { /* ignore */ }
      try { xmlErrors.set((await App.ValidateCurrent()) ?? []); } catch { /* ignore */ }
    }
    markClean();
    showStatus("Apply undone");
  }

  function forgetUndo(): void {
    undoDoc = null;
    undoXml = "";
    canUndoApply.set(false);
  }

  /** Close the XML panel. Only its working buffer goes; the document is untouched. */
  function closePanel(): void {
    panelOpen.set(false);
    xmlDirty = false;
    applyError.set("");
    forgetUndo();
  }

  /** First document of a launch: the file the OS passed, what Go has open, or the sample. */
  async function loadInitial(launched: string): Promise<void> {
    if (launched) {
      await open(launched);
      if (get(doc)) return;
    }
    const App = backend();
    if (App) {
      try {
        const current = await App.CurrentDocument();
        if (current?.Bodies?.length) {
          doc.set(current);
          filename.set("(opened in native window)");
          markClean();
          return;
        }
      } catch { /* fall through to the sample */ }
    }
    showSample();
  }

  return {
    doc, filename, status, error, recentFiles,
    panelOpen, xmlSource, xmlErrors, applyError, canUndoApply,
    isDirty,
    setBodyDirty: (dirty: boolean) => { bodyDirty = dirty; },
    setXmlDirty: (dirty: boolean) => { xmlDirty = dirty; },
    open, save, exportHtml, validate, applyXml, undoApply, closePanel,
    refreshRecent, loadInitial,
  };
}
