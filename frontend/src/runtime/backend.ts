/**
 * The one gateway to the Go side. Components call `backend()` instead of importing Wails
 * bindings, which gives a single "no bridge" answer (null in a plain vite tab) and a single
 * place for the cast between Wails' generated models and our FictionBook type.
 */
import * as go from "../../wailsjs/go/main/App";
import { BrowserOpenURL, EventsEmit, EventsOn } from "../../wailsjs/runtime/runtime";
import type { FictionBook } from "../fb2/types";

type GoApp = typeof go;
type Retyped = "OpenFile" | "ParseXMLString" | "CurrentDocument" | "UpdateDocument";

export interface Backend extends Omit<GoApp, Retyped> {
  OpenFile(path: string): Promise<FictionBook>;
  ParseXMLString(xml: string): Promise<FictionBook>;
  CurrentDocument(): Promise<FictionBook | null>;
  UpdateDocument(fb: FictionBook): Promise<void>;
  /** Subscribe to a Go event; returns the unsubscriber. */
  on(event: string, handler: (...args: never[]) => void): () => void;
  emit(event: string, ...args: unknown[]): void;
  openUrl(url: string): void;
}

const wails = {
  ...go,
  on: (event: string, handler: (...args: never[]) => void) =>
    EventsOn(event, handler as (...args: unknown[]) => void),
  emit: EventsEmit,
  openUrl: BrowserOpenURL,
} as unknown as Backend;

let override: Backend | null = null;

/** The Go side, or null when running without the Wails bridge. */
export function backend(): Backend | null {
  if (override) return override;
  const bridge = (globalThis as { go?: unknown }).go;
  return bridge ? wails : null;
}

/** Substitute a fake in tests. */
export function setBackend(fake: Backend | null): void {
  override = fake;
}
