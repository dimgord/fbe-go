# Post-1.0 backlog

Deferred items found during 1.0 RC soak. None block the 1.0 cut. Ordered
roughly by user-impact, not chronology.

## Auto-fix invalid xs:ID and duplicate binaries

**Found:** Rev 83 RC3 soak — `~/Documents/books/The Long Watch.fb2`.

**Symptom in editor:** Validation panel shows `1 XSD ERROR` —
`Element 'binary', attribute 'id': 'cover.jpg' is not a valid value of the
atomic type 'xs:ID'.` Two `<binary id="cover.jpg">` elements appear back-to-
back at the bottom of the round-tripped XML pane.

**Root cause:** the source file *itself* contains two `<binary>` elements
with the same id, and the id is not a valid `xs:ID` (no dots allowed —
must be an XML NCName). Verified directly with `xmllint --xpath
'count(//binary)'` against the source. Our parser/writer round-trips both
faithfully, so the round-trip output mirrors the source — invalid `id`,
duplicate `id`. Same XSD error appears on the source as on the round-trip.

**Why we did not fix it for 1.0:** the cardinal CLAUDE.md invariant is
fidelity — an XSD-valid source must round-trip as XSD-valid output, and
that holds (`fidelityBroken=0` on the 166-file corpus). Silently rewriting
a user's invalid source would cross the line from "faithful editor" to
"opinionated linter". That is a separate feature, not a bug fix.

**Shape of the eventual fix:**

- A non-destructive *Validate → Fix* flow in `validation/ValidationPanel.svelte`,
  invoked by an explicit user click, never automatic on save.
- Auto-fix candidates the validator can suggest:
  - **Invalid `xs:ID`:** offer a sanitized id (`cover.jpg` → `cover_jpg`)
    and rewrite every `l:href="#cover.jpg"` reference in the same pass.
    The rename is one transaction so refs and target stay consistent.
  - **Duplicate `<binary>` ids:** show both content sizes / hashes side
    by side and let the user pick which to keep, or keep both and
    rename the second.
- A `--fix` flag on `cmd/fbe validate` for batch use on a corpus.
- Tests: corpus assertion that fix preserves XSD validity AND keeps every
  reachable `l:href` resolvable.

**Out of scope for fix:** anything beyond what the validator can confidently
prove safe. We do not normalize element order, strip whitespace, "improve"
markup. The validator surfaces problems; the user clicks fix.

**Estimated work:** half a day for the xs:ID + dedupe paths, including UI
button and the multi-fix transaction. Bundle with v1.1 alongside other
linter-style features (genre enum normalization, etc).

## Preserve PM undo history across body ↔ description tab switch

**Found:** Rev 88 v1.0.2 development.

**Symptom:** While editing the body, switching to the Description tab and
back resets the PM editor's undo stack. The body content itself is
preserved (Rev 88 added `editor.currentFB()` → `fb` sync on tab leave),
but typing in the body, switching tabs, switching back, and pressing
Cmd-Z no longer undoes the earlier edits — the new mount starts from a
fresh history baseline.

**Root cause:** `App.svelte` mounts Editor inside `{#if view === "body"}`,
so a tab switch unmounts the component entirely. PM's history plugin
state lives inside the component instance; there's no way to carry it
across an unmount/mount cycle short of rehydrating from a serialized
form (which PM doesn't natively support).

**Why we did not fix it for 1.0.2:** the immediate blocker was silent
data loss on quit; that's now closed. Undo-across-tab-switch is a
quality-of-life issue, not a correctness one. Users can still undo
within a single body session.

**Shape of the eventual fix:** drop the `{#if}` and use `display: none`
to hide the body editor when the description tab is active. The
component stays mounted; PM history persists. Need to verify search /
spellcheck attributes still behave on a hidden but mounted editor, and
that the description tab's layout doesn't fight an always-rendered
sibling.

**Estimated work:** ~1 hour, mostly testing the rendering swap doesn't
break the existing layout.

## Body editor undo edge case: image-deletion + text edits interleave

**Found:** Rev 88 v1.0.2 manual testing on macOS.

**Symptom (not reliably reproducible):** Delete every inline `<image>` in
a section one by one, then edit some text, then position the cursor at
the section start, then press Cmd-Z. The text undo works, but the next
Cmd-Z is supposed to start undoing the image deletions and instead
either does nothing OR — after one more text edit + undo — runs all the
image undos in a single pass.

Likely a `prosemirror-history` interaction with our split
`image_block` / `image_inline` schema nodes (image is intentionally not a
single PM type because FB2 allows `<image>` both as a block sibling of
`<section>` and as an inline in `<p>` — see `editor/schema.ts` and
`CLAUDE.md`'s "Why ProseMirror" note).

**Why we did not fix it for 1.0.2:** can't reliably reproduce without a
specific corpus and click sequence; not data-loss; PM history remains
useful for the common case.

**Shape of the eventual fix:** narrow a reproducer (corpus file +
keystroke log), then trace through `prosemirror-history`'s transaction
batching to see whether image-block deletion is interacting badly with
addToHistory or being grouped into the wrong undo bucket. Likely a
custom history-merging tweak rather than a schema change.

**Estimated work:** unknown until a reliable repro exists.
