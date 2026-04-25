package main

// Version is the fbe-go product version, compiled into the binary and reported
// back to the frontend via App.CheckForUpdate so the update banner can compare
// against the latest GitHub release.
//
// Bump this in lockstep with `wails.json::info.productVersion` and
// `frontend/package.json::version` — the three values must stay in sync.
// CI does not currently enforce the triple-sync; see CLAUDE.md for the
// revision-bump checklist.
const Version = "1.0.0-rc2"
