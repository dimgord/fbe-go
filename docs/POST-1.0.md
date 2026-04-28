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
