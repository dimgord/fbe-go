/** Small persisted preferences. Best effort: losing one is no reason to bother the user. */
import { backend } from "../runtime/backend";
import type { settings } from "../../wailsjs/go/models";

export async function patchSettings(mutate: (s: settings.Settings) => void): Promise<void> {
  const App = backend();
  if (!App) return;
  try {
    const s = await App.LoadSettings();
    mutate(s);
    await App.SaveSettings(s);
  } catch (e) {
    console.warn("[fbe] settings save failed:", e);
  }
}

export type PaneKey = "outlineWidth" | "validationWidth" | "validationErrorsHeight";

export function persistPane(key: PaneKey, size: number): void {
  void patchSettings((s) => {
    s.panes = { ...s.panes, [key]: Math.round(size) };
  });
}
