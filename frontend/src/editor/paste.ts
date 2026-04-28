/**
 * Paste transforms for the FB2 ProseMirror editor.
 *
 * Responsibilities:
 *  - Strip Microsoft Word clutter (mso-* styles, MsoNormal classes,
 *    Office-specific `<o:p>` elements, conditional comments).
 *  - Drop empty paragraphs and normalize whitespace so poetry survives.
 *  - Convert `<br>` sequences into `<p>` breaks where appropriate.
 *  - Replace non-breaking spaces (both `&nbsp;` HTML entity and literal
 *    U+00A0) with the user's configured NBSP character — defaults to a
 *    regular space so tests stay deterministic; `App.svelte` calls
 *    `configurePaste` on settings load / apply so the user's
 *    `settings.nbspChar` drives runtime behavior.
 *
 * Reference: FBEview.cpp::OnPaste / OnRealPaste.
 */

let pasteNbspChar = " "; // default: normalize NBSPs to regular space.

/** App-level hook to apply the user's `settings.nbspChar` to paste
 *  cleanup. No-op for values that aren't a single character. */
export function configurePaste(opts: { nbspChar?: string }): void {
  if (opts.nbspChar && opts.nbspChar.length === 1) {
    pasteNbspChar = opts.nbspChar;
  }
}

/** Test-only: reset module state between unit tests. */
export function resetPasteConfigForTesting(): void {
  pasteNbspChar = " ";
}

/** Run `s.replace(re, "")` repeatedly until the string stops changing.
 *
 *  Single-pass replace on a regex like `<style>[\s\S]*?<\/style>` can be
 *  defeated by interleaved input — `<sty<style></style>le>…</style>`
 *  collapses on the first pass to `<style>…</style>`, which the original
 *  pattern would have matched if it had run again. Looping closes the
 *  CodeQL "Incomplete multi-character sanitization"
 *  (js/incomplete-multi-character-sanitization) finding for unbalanced
 *  open/close tag-pair removals. */
function stripUntilStable(s: string, re: RegExp): string {
  let prev;
  do {
    prev = s;
    s = s.replace(re, "");
  } while (s !== prev);
  return s;
}

/** Clean a pasted HTML fragment before ProseMirror parses it. */
export function cleanPastedHTML(html: string): string {
  let s = html;

  // Strip Word conditional comments: <!--[if …]> … <![endif]-->
  s = stripUntilStable(s, /<!--\s*\[if[^\]]*]>[\s\S]*?<!\[endif]-->/gi);
  // Remove <style>…</style> blocks entirely (Word dumps huge ones).
  s = stripUntilStable(s, /<style[\s\S]*?<\/style>/gi);
  // Remove <meta>, <link>, <xml>, <o:p>, <w:*> etc.
  s = s.replace(/<\/?(?:meta|link|xml|o:[^\s>]+|w:[^\s>]+)\b[^>]*>/gi, "");
  // Strip mso-* and font-family/mso-specific inline styles.
  s = s.replace(/\s*style="[^"]*"/gi, (m) => {
    const cleaned = m
      .replace(/\s*(mso-[^;"']+:[^;"']*;?)/gi, "")
      .replace(/\s*font-family:[^;"']*;?/gi, "")
      .replace(/\s*font-size:[^;"']*;?/gi, "")
      .replace(/\s*color:[^;"']*;?/gi, "")
      .replace(/\s*background[^:]*:[^;"']*;?/gi, "")
      .replace(/\s*line-height:[^;"']*;?/gi, "");
    // If nothing useful left, drop the attribute entirely.
    if (/style="\s*"/.test(cleaned) || /style=""/.test(cleaned)) return "";
    return cleaned;
  });
  // Strip class attributes (we don't want Word/other classes influencing PM parse).
  s = s.replace(/\s*class="[^"]*"/gi, "");
  // Remove <span> wrappers (they usually carried styles/classes we just dropped).
  s = s.replace(/<\/?span\b[^>]*>/gi, "");
  // Collapse multiple <br> into paragraph breaks.
  s = s.replace(/(<br\s*\/?>\s*){2,}/gi, "</p><p>");
  // Non-breaking spaces: Word output pads with them; some users want them
  // normalized to regular space (default), others to keep them as NBSP or
  // swap for narrow-NBSP. Both the HTML entity and literal U+00A0 route
  // through the configured pasteNbspChar.
  s = s.replace(/&nbsp;|\u00A0/g, pasteNbspChar);
  // Empty paragraphs → nothing.
  s = s.replace(/<p[^>]*>\s*<\/p>/gi, "");

  return s.trim();
}

/** Clean a pasted plain-text fragment: normalize newlines, strip
 *  non-printable control characters (keeps \t and \n), and normalize
 *  non-breaking spaces to the configured NBSP char. */
export function cleanPastedText(text: string): string {
  return text
    .replace(/\r\n?/g, "\n")
    .replace(/[\x00-\x08\x0b\x0c\x0e-\x1f\x7f-\x9f]/g, "")
    .replace(/\u00A0/g, pasteNbspChar);
}
