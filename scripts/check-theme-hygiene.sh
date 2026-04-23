#!/usr/bin/env bash
# Fails if any Svelte/TS/CSS file under frontend/src/ contains a hardcoded
# color literal outside the palette block (App.svelte). Every color in the
# app should go through a CSS custom property defined once in the palette,
# so theme changes and dark-mode tweaks land in one place.
#
# Detects:
#   - Hex literals:   #RGB / #RRGGBB / #RRGGBBAA
#   - Named CSS colors used as property values (white, black, red, …).
#   - rgb() / rgba() / hsl() / hsla() literals.
#
# Allowed everywhere:
#   - Non-color keywords (transparent, inherit, currentColor, none, auto,
#     initial, unset, revert).
#   - Svelte block directives ({#each, {#if, {#await, {#key).
#
# Allowed only inside PALETTE_FILE:
#   - Literal color values (that's what the palette is FOR).
#
# If the grep patterns get too permissive or miss a color, adjust below —
# the script is intentionally simple enough to read end-to-end.

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
SRC="$ROOT/frontend/src"
PALETTE_FILE="$SRC/App.svelte"

# Named colors commonly used as CSS values. Add to this list if a new one
# sneaks in — the check can only find what it looks for.
NAMED_COLORS='white|black|red|blue|green|yellow|orange|purple|pink|gray|grey|aqua|cyan|magenta|navy|teal|olive|brown|lime|silver|maroon|gold|violet|indigo|tan|khaki|salmon|coral|crimson|orchid|plum|ivory|beige|linen|wheat|snow|azure|mint'

# Skip non-source files. `.test.*` files contain HTML/CSS fixtures fed to
# parsers (not app-rendering styles) — excluded by pattern.
EXCLUDES=(
  --exclude-dir=node_modules
  --exclude-dir=dist
  --exclude-dir=wailsjs
  --exclude='*.test.ts'
  --exclude='*.test.js'
)

fail=0

# --- 1. Hex literals outside the palette file --------------------------------
# Match #RGB / #RRGGBB / #RRGGBBAA; strip false-positive {#each / {#if etc.
if hex_hits=$(grep -rnE '#[0-9a-fA-F]{3,8}\b' "${EXCLUDES[@]}" "$SRC" 2>/dev/null \
              | grep -v "^$PALETTE_FILE:" \
              | grep -vE '\{#(each|if|await|key)\b' || true); then
  if [ -n "$hex_hits" ]; then
    echo "theme-hygiene: hardcoded hex colors outside the palette:" >&2
    echo "$hex_hits" >&2
    echo >&2
    fail=1
  fi
fi

# --- 2. Named CSS colors used as property values -----------------------------
# Anchor on a CSS property name so we don't flag the word "black" in prose
# or in identifiers. Property list covers everywhere a color typically lives.
NAMED_RE="\b(background|background-color|color|border|border-[a-z]+|outline|outline-color|fill|stroke|shadow|box-shadow|text-shadow|caret-color|column-rule)\b[^;}]*:[^;}]*\b($NAMED_COLORS)\b"

if named_hits=$(grep -rnEi "$NAMED_RE" "${EXCLUDES[@]}" "$SRC" 2>/dev/null \
                 | grep -v "^$PALETTE_FILE:" || true); then
  if [ -n "$named_hits" ]; then
    echo "theme-hygiene: named CSS colors used as values outside the palette:" >&2
    echo "$named_hits" >&2
    echo >&2
    fail=1
  fi
fi

# --- 3. rgb() / rgba() / hsl() / hsla() literals -----------------------------
# These are fine in the palette (for shadow alpha), suspicious elsewhere.
if fn_hits=$(grep -rnE '\b(rgb|rgba|hsl|hsla)\s*\(' "${EXCLUDES[@]}" "$SRC" 2>/dev/null \
             | grep -v "^$PALETTE_FILE:" || true); then
  if [ -n "$fn_hits" ]; then
    echo "theme-hygiene: rgb()/rgba()/hsl()/hsla() literals outside the palette:" >&2
    echo "$fn_hits" >&2
    echo >&2
    fail=1
  fi
fi

if [ "$fail" -eq 0 ]; then
  echo "theme-hygiene: clean — all colors reference palette variables."
fi

exit "$fail"
