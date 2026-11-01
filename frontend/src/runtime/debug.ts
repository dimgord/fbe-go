import { backend } from "./backend";

/** Record an event in the debug log: counts and states only, never book text. */
export function debugLog(event: string, fields: Record<string, string | number | boolean | null> = {}): void {
  const line = [event, ...Object.entries(fields).map(([k, v]) => `${k}=${v}`)].join(" ");
  void Promise.resolve(backend()?.DebugLog?.(line)).catch(() => {});
}

/** How many lines and sections a piece of XML holds. */
export function xmlShape(xml: string): { bytes: number; lines: number; sections: number } {
  return { bytes: xml.length, lines: xml.split("\n").length, sections: xml.split("<section").length - 1 };
}
