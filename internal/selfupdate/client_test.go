package selfupdate

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// newReleaseServer returns an httptest server with the three client
// routes: {latestPath} (latest-release JSON), {downloadPath}/{tag}/{name}
// (release assets), and .../checksums.txt. A nil asset/checksums body
// makes that route 404; a non-200 latestStatus makes the latest route
// return that status. Every request is counted in reqs (may be nil).
func newReleaseServer(t *testing.T, latestStatus int, latestJSON string, asset, checksums []byte, reqs *atomic.Int32) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if reqs != nil {
			reqs.Add(1)
		}
		switch {
		case r.URL.Path == latestPath:
			if latestStatus != http.StatusOK {
				w.WriteHeader(latestStatus)
				return
			}
			_, _ = io.WriteString(w, latestJSON)
		case strings.HasSuffix(r.URL.Path, "/checksums.txt"):
			if checksums == nil {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write(checksums)
		case strings.HasPrefix(r.URL.Path, downloadPath):
			if asset == nil {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write(asset)
		default:
			http.NotFound(w, r)
		}
	}))
}

// newTestClient builds a production-wired Client (NewClient) pointed at
// ts, with a temp-dir cache path and the deterministic fixedNow clock,
// and returns the cache path so tests can seed fixtures.
func newTestClient(t *testing.T, ts *httptest.Server) (*Client, string) {
	t.Helper()
	cachePath := filepath.Join(t.TempDir(), "self-update-cache.json")
	c := NewClient(ts.URL, cachePath)
	c.Now = func() time.Time { return fixedNow }
	return c, cachePath
}

// assertCached verifies the write-through cache at path holds wantTag
// and is fresh at fixedNow.
func assertCached(t *testing.T, path, wantTag string) {
	t.Helper()
	got, ok := LoadDetectionCache(path)
	if !ok {
		t.Fatalf("cache at %s is a miss after write-through", path)
	}
	if got.Tag != wantTag {
		t.Errorf("cached tag = %q, want %q", got.Tag, wantTag)
	}
	if !got.Fresh(fixedNow) {
		t.Errorf("written cache with Fetched %v not fresh at %v", got.Fetched, fixedNow)
	}
}

func TestCheckRedirectPolicy(t *testing.T) {
	tests := []struct {
		name    string
		scheme  string
		wantErr bool
	}{
		{"https target accepted", "https", false},
		{"http target rejected", "http", true},
		{"ftp target rejected", "ftp", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{URL: &url.URL{Scheme: tt.scheme, Host: "example.com"}}
			err := checkRedirect(req, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkRedirect(scheme %q) error = %v, wantErr %v", tt.scheme, err, tt.wantErr)
			}
		})
	}
}

func TestLatestFresh(t *testing.T) {
	t.Run("returns the latest tag", func(t *testing.T) {
		ts := newReleaseServer(t, http.StatusOK, `{"tag_name":"v0.1.1"}`, nil, nil, nil)
		defer ts.Close()
		c, _ := newTestClient(t, ts)
		got, err := c.LatestFresh()
		if err != nil {
			t.Fatalf("LatestFresh: %v", err)
		}
		if got != (Release{Tag: "v0.1.1"}) {
			t.Errorf("LatestFresh = %+v, want Release{Tag: v0.1.1}", got)
		}
	})

	t.Run("api 500 is a visible error", func(t *testing.T) {
		ts := newReleaseServer(t, http.StatusInternalServerError, "", nil, nil, nil)
		defer ts.Close()
		c, _ := newTestClient(t, ts)
		_, err := c.LatestFresh()
		if err == nil {
			t.Fatal("LatestFresh: want error on HTTP 500")
		}
		if !strings.Contains(err.Error(), "500") {
			t.Errorf("error %q does not mention the HTTP status", err)
		}
	})

	t.Run("malformed body is an error", func(t *testing.T) {
		ts := newReleaseServer(t, http.StatusOK, "not json", nil, nil, nil)
		defer ts.Close()
		c, _ := newTestClient(t, ts)
		if _, err := c.LatestFresh(); err == nil {
			t.Fatal("LatestFresh: want error on malformed JSON body")
		}
	})

	t.Run("missing tag_name is an error", func(t *testing.T) {
		ts := newReleaseServer(t, http.StatusOK, `{}`, nil, nil, nil)
		defer ts.Close()
		c, _ := newTestClient(t, ts)
		if _, err := c.LatestFresh(); err == nil {
			t.Fatal("LatestFresh: want error when tag_name is missing")
		}
	})

	t.Run("redirect off https fails closed", func(t *testing.T) {
		var ts *httptest.Server
		ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, ts.URL+"/elsewhere", http.StatusFound)
		}))
		defer ts.Close()
		c := NewClient(ts.URL, "")
		_, err := c.LatestFresh()
		if err == nil {
			t.Fatal("LatestFresh: want error on off-HTTPS redirect")
		}
		if !strings.Contains(err.Error(), "refusing redirect") {
			t.Errorf("error %q does not mention the redirect refusal", err)
		}
	})

	t.Run("nil HTTP falls back to the default client", func(t *testing.T) {
		ts := newReleaseServer(t, http.StatusOK, `{"tag_name":"v0.1.1"}`, nil, nil, nil)
		defer ts.Close()
		c := &Client{BaseURL: ts.URL}
		got, err := c.LatestFresh()
		if err != nil {
			t.Fatalf("LatestFresh with nil HTTP: %v", err)
		}
		if got.Tag != "v0.1.1" {
			t.Errorf("LatestFresh with nil HTTP = %+v, want tag v0.1.1", got)
		}
	})
}

func TestLatestCached(t *testing.T) {
	t.Run("fresh cache is reused without network", func(t *testing.T) {
		var reqs atomic.Int32
		ts := newReleaseServer(t, http.StatusOK, `{"tag_name":"v0.2.0"}`, nil, nil, &reqs)
		defer ts.Close()
		c, cachePath := newTestClient(t, ts)
		// 1h old — fresh at fixedNow; tag deliberately differs from the
		// server's v0.2.0 so a wrong (network) answer is caught.
		if err := os.WriteFile(cachePath, []byte(`{"version":1,"fetched":"2026-08-12T11:00:00Z","tag":"v0.1.1"}`), 0o644); err != nil {
			t.Fatalf("seed cache fixture: %v", err)
		}
		got, ok := c.LatestCached()
		if !ok {
			t.Fatal("fresh cache: LatestCached returned false")
		}
		if got != "v0.1.1" {
			t.Errorf("LatestCached = %q, want cached v0.1.1 (server would serve v0.2.0)", got)
		}
		if n := reqs.Load(); n != 0 {
			t.Errorf("fresh cache: %d network requests, want 0", n)
		}
	})

	t.Run("stale cache refetches and writes through", func(t *testing.T) {
		var reqs atomic.Int32
		ts := newReleaseServer(t, http.StatusOK, `{"tag_name":"v0.1.1"}`, nil, nil, &reqs)
		defer ts.Close()
		c, cachePath := newTestClient(t, ts)
		// 26h old — stale at fixedNow.
		if err := os.WriteFile(cachePath, []byte(`{"version":1,"fetched":"2026-08-11T10:00:00Z","tag":"v0.1.0"}`), 0o644); err != nil {
			t.Fatalf("seed cache fixture: %v", err)
		}
		got, ok := c.LatestCached()
		if !ok {
			t.Fatal("stale cache: LatestCached returned false")
		}
		if got != "v0.1.1" {
			t.Errorf("LatestCached = %q, want fetched v0.1.1", got)
		}
		if n := reqs.Load(); n != 1 {
			t.Errorf("stale cache: %d network requests, want 1", n)
		}
		assertCached(t, cachePath, "v0.1.1")
	})

	t.Run("corrupt cache is a miss and refetches", func(t *testing.T) {
		var reqs atomic.Int32
		ts := newReleaseServer(t, http.StatusOK, `{"tag_name":"v0.1.1"}`, nil, nil, &reqs)
		defer ts.Close()
		c, cachePath := newTestClient(t, ts)
		if err := os.WriteFile(cachePath, []byte("garbage{{{"), 0o644); err != nil {
			t.Fatalf("seed corrupt fixture: %v", err)
		}
		got, ok := c.LatestCached()
		if !ok {
			t.Fatal("corrupt cache: LatestCached returned false")
		}
		if got != "v0.1.1" {
			t.Errorf("LatestCached = %q, want refetched v0.1.1", got)
		}
		if n := reqs.Load(); n != 1 {
			t.Errorf("corrupt cache: %d network requests, want 1", n)
		}
		assertCached(t, cachePath, "v0.1.1")
	})

	t.Run("api 500 is silent", func(t *testing.T) {
		ts := newReleaseServer(t, http.StatusInternalServerError, "", nil, nil, nil)
		defer ts.Close()
		c, _ := newTestClient(t, ts)
		if got, ok := c.LatestCached(); ok {
			t.Errorf("LatestCached = (%q, true) on API 500, want silent false", got)
		}
	})

	t.Run("malformed body is silent", func(t *testing.T) {
		ts := newReleaseServer(t, http.StatusOK, "not json", nil, nil, nil)
		defer ts.Close()
		c, _ := newTestClient(t, ts)
		if got, ok := c.LatestCached(); ok {
			t.Errorf("LatestCached = (%q, true) on malformed body, want silent false", got)
		}
	})

	t.Run("empty cache path fetches without caching", func(t *testing.T) {
		var reqs atomic.Int32
		ts := newReleaseServer(t, http.StatusOK, `{"tag_name":"v0.1.1"}`, nil, nil, &reqs)
		defer ts.Close()
		c := NewClient(ts.URL, "")
		got, ok := c.LatestCached()
		if !ok {
			t.Fatal("empty cache path: LatestCached returned false")
		}
		if got != "v0.1.1" {
			t.Errorf("LatestCached = %q, want v0.1.1", got)
		}
		if n := reqs.Load(); n != 1 {
			t.Errorf("empty cache path: %d network requests, want 1", n)
		}
	})

	t.Run("cache write failure is silent", func(t *testing.T) {
		var reqs atomic.Int32
		ts := newReleaseServer(t, http.StatusOK, `{"tag_name":"v0.1.1"}`, nil, nil, &reqs)
		defer ts.Close()
		// A regular file where the cache's parent directory should be:
		// SaveDetectionCache's MkdirAll fails, so the write-through fails.
		blocker := filepath.Join(t.TempDir(), "blocker")
		if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
			t.Fatalf("create blocker file: %v", err)
		}
		c := NewClient(ts.URL, filepath.Join(blocker, "self-update-cache.json"))
		if got, ok := c.LatestCached(); ok {
			t.Errorf("LatestCached = (%q, true) despite unwritable cache path, want silent false", got)
		}
		if n := reqs.Load(); n != 1 {
			t.Errorf("cache write failure: %d network requests, want 1 (fetch happened, write failed)", n)
		}
	})
}

func TestDownload(t *testing.T) {
	asset := []byte("fake-tarball-bytes")
	checksums := []byte("0123456789abcdef0123456789abcdef  upp-linux-amd64.tar.gz\n")

	t.Run("fetches asset and checksums from the same release", func(t *testing.T) {
		ts := newReleaseServer(t, http.StatusOK, `{"tag_name":"v0.1.1"}`, asset, checksums, nil)
		defer ts.Close()
		c, _ := newTestClient(t, ts)
		if _, err := c.LatestFresh(); err != nil {
			t.Fatalf("LatestFresh: %v", err)
		}
		gotAsset, gotChecksums, err := c.Download("upp-linux-amd64.tar.gz")
		if err != nil {
			t.Fatalf("Download: %v", err)
		}
		if !bytes.Equal(gotAsset, asset) {
			t.Errorf("asset bytes = %q, want %q", gotAsset, asset)
		}
		if !bytes.Equal(gotChecksums, checksums) {
			t.Errorf("checksums bytes = %q, want %q", gotChecksums, checksums)
		}
	})

	t.Run("asset 404 is an error", func(t *testing.T) {
		ts := newReleaseServer(t, http.StatusOK, `{"tag_name":"v0.1.1"}`, nil, checksums, nil)
		defer ts.Close()
		c, _ := newTestClient(t, ts)
		if _, err := c.LatestFresh(); err != nil {
			t.Fatalf("LatestFresh: %v", err)
		}
		_, _, err := c.Download("upp-linux-amd64.tar.gz")
		if err == nil {
			t.Fatal("Download: want error on asset 404")
		}
		if !strings.Contains(err.Error(), "404") {
			t.Errorf("error %q does not mention the HTTP status", err)
		}
	})

	t.Run("checksums 404 is an error", func(t *testing.T) {
		ts := newReleaseServer(t, http.StatusOK, `{"tag_name":"v0.1.1"}`, asset, nil, nil)
		defer ts.Close()
		c, _ := newTestClient(t, ts)
		if _, err := c.LatestFresh(); err != nil {
			t.Fatalf("LatestFresh: %v", err)
		}
		_, _, err := c.Download("upp-linux-amd64.tar.gz")
		if err == nil {
			t.Fatal("Download: want error on checksums 404")
		}
	})

	t.Run("no resolved release is an error", func(t *testing.T) {
		ts := newReleaseServer(t, http.StatusOK, `{"tag_name":"v0.1.1"}`, asset, checksums, nil)
		defer ts.Close()
		c, _ := newTestClient(t, ts)
		if _, _, err := c.Download("upp-linux-amd64.tar.gz"); err == nil {
			t.Fatal("Download before LatestFresh: want error")
		}
	})

	t.Run("redirect off https fails closed", func(t *testing.T) {
		var ts *httptest.Server
		ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.URL.Path == latestPath:
				_, _ = io.WriteString(w, `{"tag_name":"v0.1.1"}`)
			default:
				http.Redirect(w, r, ts.URL+"/elsewhere", http.StatusFound)
			}
		}))
		defer ts.Close()
		c := NewClient(ts.URL, "")
		if _, err := c.LatestFresh(); err != nil {
			t.Fatalf("LatestFresh: %v", err)
		}
		_, _, err := c.Download("upp-linux-amd64.tar.gz")
		if err == nil {
			t.Fatal("Download: want error on off-HTTPS redirect")
		}
		if !strings.Contains(err.Error(), "refusing redirect") {
			t.Errorf("error %q does not mention the redirect refusal", err)
		}
	})
}

// TestRedirectToHTTPSAccepted proves the redirect policy accepts a hop
// from plain HTTP to HTTPS: the policy is consulted with the redirect
// target's scheme, so http→https passes and the target serves the body.
func TestRedirectToHTTPSAccepted(t *testing.T) {
	target := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != latestPath {
			http.NotFound(w, r)
			return
		}
		_, _ = io.WriteString(w, `{"tag_name":"v0.1.1"}`)
	}))
	defer target.Close()

	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+latestPath, http.StatusFound)
	}))
	defer origin.Close()

	hc := target.Client()
	hc.CheckRedirect = checkRedirect
	c := &Client{BaseURL: origin.URL, HTTP: hc}

	got, err := c.LatestFresh()
	if err != nil {
		t.Fatalf("LatestFresh across http→https redirect: %v", err)
	}
	if got != (Release{Tag: "v0.1.1"}) {
		t.Errorf("LatestFresh = %+v, want Release{Tag: v0.1.1}", got)
	}
}

func TestClientTimeouts(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		_, _ = io.WriteString(w, `{"tag_name":"v0.1.1"}`)
	}))
	defer ts.Close()

	t.Run("LatestFresh fails on a slow server", func(t *testing.T) {
		c := &Client{BaseURL: ts.URL, HTTP: &http.Client{Timeout: 50 * time.Millisecond}}
		_, err := c.LatestFresh()
		if err == nil {
			t.Fatal("LatestFresh: want timeout error")
		}
		if !strings.Contains(err.Error(), "deadline exceeded") {
			t.Errorf("error %q is not a timeout error", err)
		}
	})

	t.Run("LatestCached stays silent on timeout", func(t *testing.T) {
		c := &Client{BaseURL: ts.URL, HTTP: &http.Client{Timeout: 50 * time.Millisecond}}
		if got, ok := c.LatestCached(); ok {
			t.Errorf("LatestCached = (%q, true) on timeout, want silent false", got)
		}
	})
}
