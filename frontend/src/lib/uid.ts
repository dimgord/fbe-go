/**
 * Per-process counter for stable, unique DOM ids used in label/for pairs.
 *
 * Svelte's a11y lint requires `<label for="…">` + matching `<input id="…">`.
 * Hard-coded ids break when a form component renders multiple times (one
 * instance per author/sequence/genre in a description list), so each instance
 * calls `uid("author")` once and composes the full ids locally.
 */
let counter = 0;

export function uid(prefix = "fbe"): string {
  return `${prefix}-${++counter}`;
}
