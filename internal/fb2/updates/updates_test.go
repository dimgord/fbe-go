package updates

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIsNewer(t *testing.T) {
	cases := []struct {
		name          string
		latest, cur   string
		wantIsNewer   bool
	}{
		{"patch bump", "v0.2.1-beta", "v0.2.0-beta", true},
		{"minor bump", "v0.3.0-beta", "v0.2.5-beta", true},
		{"major bump", "v1.0.0", "v0.9.9", true},
		{"equal tags", "v0.2.0-beta", "v0.2.0-beta", false},
		{"older patch", "v0.2.0-beta", "v0.2.1-beta", false},
		{"missing v prefix", "0.2.1-beta", "v0.2.0-beta", true},
		{"stable beats prerelease", "v0.2.0", "v0.2.0-beta", true},
		{"prerelease beats nothing (equal numeric)", "v0.2.0-beta", "v0.2.0", false},
		{"beta to rc", "v0.2.0-rc", "v0.2.0-beta", true},
		{"rc.2 beats rc.1", "v0.2.0-rc.2", "v0.2.0-rc.1", true},
		{"unparseable latest", "latest-main", "v0.2.0-beta", false},
		{"unparseable current", "v0.2.1-beta", "nightly", false},
		{"empty", "", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsNewer(tc.latest, tc.cur); got != tc.wantIsNewer {
				t.Errorf("IsNewer(%q, %q) = %v, want %v", tc.latest, tc.cur, got, tc.wantIsNewer)
			}
		})
	}
}

func TestCheck_NewerAvailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Errorf("missing User-Agent header")
		}
		if !strings.Contains(r.URL.RawQuery, "per_page=1") {
			t.Errorf("expected per_page=1 in query, got %q", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `[{
			"tag_name": "v0.3.0-beta",
			"name": "fbe-go 0.3.0-beta",
			"html_url": "https://github.com/dimgord/fbe-go/releases/tag/v0.3.0-beta",
			"body": "Release notes\n\n- cool feature",
			"prerelease": true,
			"draft": false,
			"published_at": "2026-05-01T00:00:00Z"
		}]`)
	}))
	defer srv.Close()

	info, err := Check(context.Background(), "owner/name", "v0.2.0-beta", newTestClient(srv))
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !info.Available {
		t.Fatalf("expected Available=true, got %+v", info)
	}
	if info.LatestVersion != "v0.3.0-beta" {
		t.Errorf("LatestVersion = %q, want v0.3.0-beta", info.LatestVersion)
	}
	if info.URL == "" {
		t.Errorf("missing URL")
	}
}

func TestCheck_UpToDate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `[{
			"tag_name": "v0.2.0-beta",
			"html_url": "https://example.com",
			"draft": false
		}]`)
	}))
	defer srv.Close()
	info, err := Check(context.Background(), "owner/name", "v0.2.0-beta", newTestClient(srv))
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if info.Available {
		t.Errorf("expected Available=false for equal tags, got %+v", info)
	}
	if info.LatestVersion != "v0.2.0-beta" {
		t.Errorf("LatestVersion = %q", info.LatestVersion)
	}
}

func TestCheck_SkipsDrafts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `[
			{"tag_name":"v0.9.0","draft":true,"html_url":""},
			{"tag_name":"v0.2.1-beta","draft":false,"html_url":"https://x"}
		]`)
	}))
	defer srv.Close()
	// fetchLatestRelease returns the FIRST non-draft — but per_page=1, so in
	// practice drafts would shadow a published release. Cover the skip path
	// by giving the server two items (simulating per_page=5).
	info, err := Check(context.Background(), "owner/name", "v0.2.0-beta", newTestClient(srv))
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if info.LatestVersion != "v0.2.1-beta" {
		t.Errorf("LatestVersion = %q, want v0.2.1-beta", info.LatestVersion)
	}
}

func TestCheck_404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"message":"Not Found"}`)
	}))
	defer srv.Close()
	_, err := Check(context.Background(), "owner/name", "v0.2.0-beta", newTestClient(srv))
	if err == nil {
		t.Fatalf("expected error on 404, got nil")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("expected error to mention 404, got %v", err)
	}
}

func TestCheck_EmptyList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `[]`)
	}))
	defer srv.Close()
	info, err := Check(context.Background(), "owner/name", "v0.2.0-beta", newTestClient(srv))
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if info.Available {
		t.Errorf("expected Available=false on empty list, got %+v", info)
	}
	if info.CurrentVersion != "v0.2.0-beta" {
		t.Errorf("CurrentVersion = %q", info.CurrentVersion)
	}
}

// newTestClient rewrites all outbound URLs to hit the test server, so the
// httptest server can observe the exact path + headers our code produces.
func newTestClient(srv *httptest.Server) *http.Client {
	return &http.Client{
		Transport: rewriteTransport{srvURL: srv.URL},
	}
}

type rewriteTransport struct {
	srvURL string
}

func (r rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Rewrite scheme+host to the test server, keep path + query.
	u := *req.URL
	// Parse srvURL for host.
	// Accept anything like http://127.0.0.1:PORT — strip the leading http://.
	host := strings.TrimPrefix(r.srvURL, "http://")
	u.Scheme = "http"
	u.Host = host
	req.URL = &u
	return http.DefaultTransport.RoundTrip(req)
}
