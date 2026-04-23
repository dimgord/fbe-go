// Package updates checks the GitHub Releases API for a newer build of fbe-go
// and exposes the result to the frontend banner.
//
// Scope is intentionally minimal for the pre-1.0 beta: no auto-install, no
// background delta-apply, no code-signing verification. The banner just
// tells the user "newer version N is out → click here to download" and hands
// the release URL off to the OS default browser. This keeps us off the code-
// signing / Sparkle / AppImageUpdate critical path while still closing
// Phase 5's last unchecked box.
package updates

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// DefaultRepo points at the canonical fbe-go upstream. Overridable from the
// caller so forks can check their own release stream.
const DefaultRepo = "dimgord/fbe-go"

// Release is the subset of fields we display and use to decide whether an
// update is available. Anything else the GitHub API returns is ignored.
type Release struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	HTMLURL     string `json:"html_url"`
	Body        string `json:"body"`
	Prerelease  bool   `json:"prerelease"`
	Draft       bool   `json:"draft"`
	PublishedAt string `json:"published_at"`
}

// Info is the payload returned to the frontend — "current vs latest" plus
// everything the banner needs to render without a second API round-trip.
type Info struct {
	Available      bool   `json:"available"`
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	URL            string `json:"url"`
	Notes          string `json:"notes"`
	PublishedAt    string `json:"publishedAt"`
	Prerelease     bool   `json:"prerelease"`
}

// HTTPClient is a narrow interface so tests can inject a fake transport.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// defaultClient times out at 5s — we never want the update check to keep the
// app hanging. The frontend call site is async so a timeout just means "no
// banner this launch", which is the right trade-off for a polite UX.
var defaultClient = &http.Client{Timeout: 5 * time.Second}

// Check asks GitHub for the newest release tagged on `repo` (e.g.
// "owner/name") and compares its tag to `currentVersion`. Prereleases are
// included — fbe-go's entire release history is -beta, and the point of the
// banner is to show *a* newer build regardless of prerelease status.
//
// Returns Info with Available=false when:
//   - no releases exist
//   - the newest release is not strictly newer than currentVersion
//   - the newest tag is unparseable (degrade to "assume up-to-date")
//
// Network / API errors propagate — the caller decides whether to surface
// them or silently skip. The frontend currently swallows.
func Check(ctx context.Context, repo, currentVersion string, client HTTPClient) (*Info, error) {
	if client == nil {
		client = defaultClient
	}
	latest, err := fetchLatestRelease(ctx, repo, client)
	if err != nil {
		return nil, err
	}
	info := &Info{
		CurrentVersion: currentVersion,
	}
	if latest == nil {
		return info, nil
	}
	info.LatestVersion = latest.TagName
	info.URL = latest.HTMLURL
	info.Notes = latest.Body
	info.PublishedAt = latest.PublishedAt
	info.Prerelease = latest.Prerelease
	if IsNewer(latest.TagName, currentVersion) {
		info.Available = true
	}
	return info, nil
}

// fetchLatestRelease asks GitHub for the most recent release on `repo` (per
// published_at). Uses the list endpoint rather than `/releases/latest`
// because the latter excludes prereleases, and fbe-go ships exclusively as
// prerelease during the beta cycle.
//
// `per_page=1` keeps the payload tiny — a single release with its body is
// ~2–10 KiB, which is fine even on a bad connection. Release order on the
// list endpoint is by created_at descending, which matches what we want.
func fetchLatestRelease(ctx context.Context, repo string, client HTTPClient) (*Release, error) {
	if repo == "" {
		return nil, errors.New("updates: empty repo")
	}
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases?per_page=1", repo)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	// User-Agent is required by GitHub's API terms; Accept pins the v3 JSON
	// format so a future default change doesn't surprise us.
	req.Header.Set("User-Agent", "fbe-go/updates")
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		// Read a bit of the body to enrich the error — GitHub rate-limit
		// responses are JSON with a helpful message field.
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("updates: GitHub returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var releases []Release
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("updates: decode: %w", err)
	}
	// Skip drafts (they're visible to repo collaborators on the API even
	// though they're unpublished — the user can't download them).
	for i := range releases {
		if releases[i].Draft {
			continue
		}
		return &releases[i], nil
	}
	return nil, nil
}

// IsNewer reports whether `latest` is strictly greater than `current`.
// Inputs are fbe-go's release tags, shaped like `v0.2.1-beta` or
// `0.2.1-beta`; the leading `v` is optional. Pre-release tails (`-beta`,
// `-rc.1`) are compared as strings *after* the numeric triple matches —
// so `0.2.1` > `0.2.0-beta`, but `0.2.0-rc.2` > `0.2.0-rc.1` and
// `0.2.0-beta` and `0.2.0-beta` compare equal.
//
// Unparseable inputs return false (safe default — never spam "newer
// available" when we can't tell).
func IsNewer(latest, current string) bool {
	lMaj, lMin, lPat, lPre, okL := splitVersion(latest)
	cMaj, cMin, cPat, cPre, okC := splitVersion(current)
	if !okL || !okC {
		return false
	}
	if lMaj != cMaj {
		return lMaj > cMaj
	}
	if lMin != cMin {
		return lMin > cMin
	}
	if lPat != cPat {
		return lPat > cPat
	}
	// Numeric triple is equal. A release with no pre-release tag is
	// canonical — it beats a pre-release of the same triple (`0.2.0` >
	// `0.2.0-beta`).
	switch {
	case lPre == "" && cPre == "":
		return false
	case lPre == "" && cPre != "":
		return true
	case lPre != "" && cPre == "":
		return false
	default:
		// Both have pre-release tails — lexicographic tail compare is
		// good enough for fbe-go's conventions (beta < beta.1 < beta.2 <
		// rc < rc.1 by string order happens to match intent here).
		return lPre > cPre
	}
}

// splitVersion parses `v0.2.1-beta` / `0.2.1-beta` / `0.2.0` into
// (major, minor, patch, preRelease, ok). `ok=false` for anything that
// doesn't start with three dot-separated integers.
func splitVersion(raw string) (int, int, int, string, bool) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "v")
	if raw == "" {
		return 0, 0, 0, "", false
	}
	// Strip pre-release tail at the first `-` so we can parse the triple.
	pre := ""
	if i := strings.Index(raw, "-"); i >= 0 {
		pre = raw[i+1:]
		raw = raw[:i]
	}
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return 0, 0, 0, "", false
	}
	maj, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, "", false
	}
	min, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0, "", false
	}
	pat, err := strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, 0, "", false
	}
	return maj, min, pat, pre, true
}
