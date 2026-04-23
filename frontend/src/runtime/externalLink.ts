/**
 * External-link routing for the Wails webview.
 *
 * The webview treats a plain `<a href="https://…">` click as window
 * navigation — the editor page unloads, replaced by whatever target URL
 * responds, and there's no back button to return. Every link therefore
 * needs to be routed through `runtime.BrowserOpenURL`, which hands the URL
 * to the OS (xdg-open / open / etc.).
 *
 * `installExternalLinkHandler` attaches a document-level capture-phase
 * listener that catches every protocol-URL anchor click — editor content,
 * Help modal, future UI — and routes it. One install covers the whole
 * app. Fragment / relative / javascript: links are left alone so internal
 * navigation still works if we ever want it.
 */

/** Returns true when `href` is a URL we should open in the system browser. */
export function isExternalUrl(href: string | null | undefined): href is string {
  if (!href) return false;
  if (/^(https?|ftp|mailto|file):/i.test(href)) return true;
  // Protocol-relative URL.
  if (href.startsWith("//")) return true;
  return false;
}

/** Routes a URL to the OS default handler via Wails runtime. Falls back to
 *  `window.open` in plain browser contexts (vite dev without the Wails
 *  bridge, or the dev-server tab opened directly). */
export async function openExternalUrl(url: string): Promise<void> {
  try {
    const rt = await import("../../wailsjs/runtime/runtime");
    if (typeof rt.BrowserOpenURL === "function") {
      rt.BrowserOpenURL(url);
      return;
    }
  } catch {
    /* not running under Wails */
  }
  window.open(url, "_blank", "noopener,noreferrer");
}

/**
 * Install a document-level capture-phase click listener that intercepts
 * clicks on any `<a>` with an external href and routes it through Wails.
 * Returns a disposer for the caller to call on unmount.
 *
 * Implementation notes:
 * - Capture phase so we fire before any component-level `on:click` handler
 *   (removing the need for per-link wrappers).
 * - Only `preventDefault`, not `stopPropagation` — other handlers (e.g.
 *   ProseMirror's cursor placement inside a link's text) must still run.
 * - `closest("a")` so a click on an inline `<img>` / `<span>` inside a
 *   link still routes to the link's href.
 */
export function installExternalLinkHandler(): () => void {
  const handler = (e: MouseEvent) => {
    const target = e.target as HTMLElement | null;
    if (!target || typeof target.closest !== "function") return;
    const anchor = target.closest("a") as HTMLAnchorElement | null;
    if (!anchor) return;
    const href = anchor.getAttribute("href");
    if (!isExternalUrl(href)) return;
    e.preventDefault();
    void openExternalUrl(href);
  };
  document.addEventListener("click", handler, true);
  return () => document.removeEventListener("click", handler, true);
}
