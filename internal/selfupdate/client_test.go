package selfupdate

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
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
		case strings.HasPrefix(r.URL.Path, downloadPath), strings.HasPrefix(r.URL.Path, releasePath):
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

// newTestClient builds a production-wired Client (NewClient) pointed at ts.
func newTestClient(t *testing.T, ts *httptest.Server) (*Client, string) {
	t.Helper()
	return NewClient(ts.URL, ""), ""
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
		got, err := c.LatestFresh(context.Background())
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
		_, err := c.LatestFresh(context.Background())
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
		if _, err := c.LatestFresh(context.Background()); err == nil {
			t.Fatal("LatestFresh: want error on malformed JSON body")
		}
	})

	t.Run("missing tag_name is an error", func(t *testing.T) {
		ts := newReleaseServer(t, http.StatusOK, `{}`, nil, nil, nil)
		defer ts.Close()
		c, _ := newTestClient(t, ts)
		if _, err := c.LatestFresh(context.Background()); err == nil {
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
		_, err := c.LatestFresh(context.Background())
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
		got, err := c.LatestFresh(context.Background())
		if err != nil {
			t.Fatalf("LatestFresh with nil HTTP: %v", err)
		}
		if got.Tag != "v0.1.1" {
			t.Errorf("LatestFresh with nil HTTP = %+v, want tag v0.1.1", got)
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
		if _, err := c.LatestFresh(context.Background()); err != nil {
			t.Fatalf("LatestFresh: %v", err)
		}
		archivePath, gotChecksums, err := c.Download(context.Background(), "upp-linux-amd64.tar.gz")
		if err != nil {
			t.Fatalf("Download: %v", err)
		}
		t.Cleanup(func() { _ = os.Remove(archivePath) })
		if archivePath == "" {
			t.Fatal("archivePath is empty")
		}
		if _, err := os.Stat(archivePath); err != nil {
			t.Fatalf("archive file does not exist: %v", err)
		}
		gotAsset, err := os.ReadFile(archivePath)
		if err != nil {
			t.Fatalf("read archive file: %v", err)
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
		if _, err := c.LatestFresh(context.Background()); err != nil {
			t.Fatalf("LatestFresh: %v", err)
		}
		archivePath, _, err := c.Download(context.Background(), "upp-linux-amd64.tar.gz")
		if err == nil {
			t.Fatal("Download: want error on asset 404")
		}
		if archivePath != "" {
			t.Errorf("archivePath = %q on error, want empty", archivePath)
		}
		if !strings.Contains(err.Error(), "404") {
			t.Errorf("error %q does not mention the HTTP status", err)
		}
	})

	t.Run("checksums 404 is an error", func(t *testing.T) {
		ts := newReleaseServer(t, http.StatusOK, `{"tag_name":"v0.1.1"}`, asset, nil, nil)
		defer ts.Close()
		c, _ := newTestClient(t, ts)
		if _, err := c.LatestFresh(context.Background()); err != nil {
			t.Fatalf("LatestFresh: %v", err)
		}
		archivePath, _, err := c.Download(context.Background(), "upp-linux-amd64.tar.gz")
		if err == nil {
			t.Fatal("Download: want error on checksums 404")
		}
		if archivePath != "" {
			t.Errorf("archivePath = %q on error, want empty", archivePath)
		}
	})

	t.Run("no resolved release is an error", func(t *testing.T) {
		ts := newReleaseServer(t, http.StatusOK, `{"tag_name":"v0.1.1"}`, asset, checksums, nil)
		defer ts.Close()
		c, _ := newTestClient(t, ts)
		if archivePath, _, err := c.Download(context.Background(), "upp-linux-amd64.tar.gz"); err == nil {
			t.Fatal("Download before LatestFresh: want error")
		} else if archivePath != "" {
			t.Errorf("archivePath = %q on error, want empty", archivePath)
		}
	})

	t.Run("redirect off https fails closed", func(t *testing.T) {
		var ts *httptest.Server
		ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case latestPath:
				_, _ = io.WriteString(w, `{"tag_name":"v0.1.1"}`)
			default:
				http.Redirect(w, r, ts.URL+"/elsewhere", http.StatusFound)
			}
		}))
		defer ts.Close()
		c := NewClient(ts.URL, "")
		if _, err := c.LatestFresh(context.Background()); err != nil {
			t.Fatalf("LatestFresh: %v", err)
		}
		archivePath, _, err := c.Download(context.Background(), "upp-linux-amd64.tar.gz")
		if err == nil {
			t.Fatal("Download: want error on off-HTTPS redirect")
		}
		if archivePath != "" {
			t.Errorf("archivePath = %q on error, want empty", archivePath)
		}
		if !strings.Contains(err.Error(), "refusing redirect") {
			t.Errorf("error %q does not mention the redirect refusal", err)
		}
	})
}

// TestDownloadDownloadBaseURL pins the U5 BaseURL resolution: the API
// base (https://api.github.com) serves the latest-release lookup but
// does NOT serve /releases/download (it answers 404 — verified against
// api.github.com 2026-08-12), so production routes asset downloads to a
// github.com web base. Two servers prove the split: the latest lookup
// hits only the API base and the asset+checksums hit only the web base.
func TestDownloadDownloadBaseURL(t *testing.T) {
	var apiReqs, webReqs atomic.Int32
	api := newReleaseServer(t, http.StatusOK, `{"tag_name":"v0.1.1"}`, nil, nil, &apiReqs)
	defer api.Close()
	web := newReleaseServer(t, http.StatusOK, `{"tag_name":"ignored"}`, []byte("asset-bytes"), []byte("checksum-bytes"), &webReqs)
	defer web.Close()

	c := &Client{BaseURL: api.URL, DownloadBaseURL: web.URL}
	if _, err := c.LatestFresh(context.Background()); err != nil {
		t.Fatalf("LatestFresh: %v", err)
	}
	archivePath, checksums, err := c.Download(context.Background(), "upp-linux-amd64.tar.gz")
	if err != nil {
		t.Fatalf("Download with DownloadBaseURL: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(archivePath) })
	gotAsset, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}
	if string(gotAsset) != "asset-bytes" {
		t.Errorf("asset = %q, want %q", gotAsset, "asset-bytes")
	}
	if string(checksums) != "checksum-bytes" {
		t.Errorf("checksums = %q, want %q", checksums, "checksum-bytes")
	}
	if got := apiReqs.Load(); got != 1 {
		t.Errorf("latest lookup should hit only the API base, got %d requests", got)
	}
	if got := webReqs.Load(); got != 2 {
		t.Errorf("asset+checksums should hit only the web base, got %d requests", got)
	}
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

	got, err := c.LatestFresh(context.Background())
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
		_, err := c.LatestFresh(context.Background())
		if err == nil {
			t.Fatal("LatestFresh: want timeout error")
		}
		if !strings.Contains(err.Error(), "deadline exceeded") {
			t.Errorf("error %q is not a timeout error", err)
		}
	})
}

func TestClientCancellation(t *testing.T) {
	var reqs atomic.Int32
	ts := newReleaseServer(t, http.StatusOK, `{"tag_name":"v0.1.1"}`, []byte("asset-bytes"), []byte("checksum-bytes"), &reqs)
	defer ts.Close()

	t.Run("LatestFresh honors pre-canceled context", func(t *testing.T) {
		reqs.Store(0)
		c, _ := newTestClient(t, ts)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := c.LatestFresh(ctx)
		if err == nil {
			t.Fatal("LatestFresh: want error on canceled context, got nil")
		}
		if !errors.Is(err, context.Canceled) {
			t.Errorf("LatestFresh: error = %v, want errors.Is context.Canceled", err)
		}
		if n := reqs.Load(); n != 0 {
			t.Errorf("pre-canceled context made %d network requests, want 0", n)
		}
	})

	t.Run("Download honors pre-canceled context", func(t *testing.T) {
		reqs.Store(0)
		c, _ := newTestClient(t, ts)
		// Prime resolved release
		if _, err := c.LatestFresh(context.Background()); err != nil {
			t.Fatalf("setup LatestFresh: %v", err)
		}
		reqs.Store(0)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		archivePath, _, err := c.Download(ctx, "upp-linux-amd64.tar.gz")
		if err == nil {
			t.Fatal("Download: want error on canceled context, got nil")
		}
		if archivePath != "" {
			t.Errorf("archivePath = %q on error, want empty", archivePath)
		}
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Download: error = %v, want errors.Is context.Canceled", err)
		}
		if n := reqs.Load(); n != 0 {
			t.Errorf("pre-canceled Download made %d network requests, want 0", n)
		}
	})
}

func TestClient_DownloadClientTimeout(t *testing.T) {
	t.Run("default client uses downloadTimeout for downloads", func(t *testing.T) {
		c := NewClient("https://api.github.com", "")
		dc := c.downloadHTTPClient()
		if dc.Timeout != 5*time.Minute {
			t.Errorf("downloadHTTPClient timeout = %v, want 5m", dc.Timeout)
		}
	})

	t.Run("zero-value client uses downloadTimeout for downloads", func(t *testing.T) {
		c := &Client{BaseURL: "https://api.github.com"}
		dc := c.downloadHTTPClient()
		if dc.Timeout != 5*time.Minute {
			t.Errorf("downloadHTTPClient timeout = %v, want 5m", dc.Timeout)
		}
	})

	t.Run("injected HTTP client is preserved for downloads", func(t *testing.T) {
		injected := &http.Client{Timeout: 42 * time.Second}
		c := &Client{BaseURL: "https://api.github.com", HTTP: injected}
		dc := c.downloadHTTPClient()
		if dc != injected {
			t.Errorf("downloadHTTPClient = %p, want injected client %p", dc, injected)
		}
	})

	t.Run("Download succeeds and uses download client", func(t *testing.T) {
		asset := []byte("download-test-bytes")
		checksums := []byte("deadbeef  upp-linux-amd64.tar.gz\n")
		ts := newReleaseServer(t, http.StatusOK, `{"tag_name":"v0.1.1"}`, asset, checksums, nil)
		defer ts.Close()

		c := NewClient(ts.URL, "")
		if _, err := c.LatestFresh(context.Background()); err != nil {
			t.Fatalf("LatestFresh: %v", err)
		}

		archivePath, gotChecksums, err := c.Download(context.Background(), "upp-linux-amd64.tar.gz")
		if err != nil {
			t.Fatalf("Download failed: %v", err)
		}
		t.Cleanup(func() { _ = os.Remove(archivePath) })

		if !bytes.Equal(gotChecksums, checksums) {
			t.Errorf("checksums = %q, want %q", gotChecksums, checksums)
		}
		gotAsset, err := os.ReadFile(archivePath)
		if err != nil {
			t.Fatalf("read file: %v", err)
		}
		if !bytes.Equal(gotAsset, asset) {
			t.Errorf("asset = %q, want %q", gotAsset, asset)
		}
	})
}

func TestNewHTTPClient_PreservesHTTP2AndTimeouts(t *testing.T) {
	c := newHTTPClient(10 * time.Second)
	tr, ok := c.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("Transport = %T, want *http.Transport", c.Transport)
	}
	if !tr.ForceAttemptHTTP2 {
		t.Error("ForceAttemptHTTP2 = false, want true")
	}
	if tr.TLSHandshakeTimeout <= 0 {
		t.Errorf("TLSHandshakeTimeout = %v, want > 0", tr.TLSHandshakeTimeout)
	}
	if tr.IdleConnTimeout <= 0 {
		t.Errorf("IdleConnTimeout = %v, want > 0", tr.IdleConnTimeout)
	}
}

func TestClientGet_RejectsOversizedResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(bytes.Repeat([]byte("a"), maxMetaResponseBytes+10))
	}))
	defer srv.Close()

	c := &Client{HTTP: srv.Client()}
	_, err := c.get(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("get() error = nil, want size limit error")
	}
	if !strings.Contains(err.Error(), "exceeds maximum size") {
		t.Errorf("err = %v, want 'exceeds maximum size'", err)
	}
}

func TestCheckRedirect_LimitsHops(t *testing.T) {
	httpsReq := &http.Request{URL: &url.URL{Scheme: "https", Host: "example.com"}}
	if err := checkRedirect(httpsReq, make([]*http.Request, 10)); err == nil {
		t.Fatal("checkRedirect with 10 hops = nil, want error")
	}
}

func TestClientDownload_RejectsOversizedAsset(t *testing.T) {
	origMax := maxDownloadAssetBytes
	maxDownloadAssetBytes = 32
	defer func() { maxDownloadAssetBytes = origMax }()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "latest"):
			_, _ = w.Write([]byte(`{"tag_name":"v1.0.0"}`))
		case strings.HasSuffix(r.URL.Path, "checksums.txt"):
			_, _ = w.Write([]byte("deadbeef  upp.tar.gz\n"))
		default:
			_, _ = w.Write(bytes.Repeat([]byte("A"), 64))
		}
	}))
	defer ts.Close()

	c := &Client{BaseURL: ts.URL, DownloadBaseURL: ts.URL, HTTP: ts.Client()}
	if _, err := c.LatestFresh(context.Background()); err != nil {
		t.Fatalf("LatestFresh error: %v", err)
	}
	if _, _, err := c.Download(context.Background(), "upp.tar.gz"); err == nil {
		t.Fatal("Download() error = nil for oversized asset, want error")
	}
}
