package selfupdate

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	// dialTimeout bounds TCP connection setup and requestTimeout bounds
	// the whole request including the response read (spec R2: ~10s
	// dial/read timeouts).
	dialTimeout    = 10 * time.Second
	requestTimeout = 10 * time.Second

	// latestPath is the GitHub "latest release" API endpoint, relative to
	// Client.BaseURL. The owner/repo is fixed: upp is published from
	// github.com/JhnFrankz/upp.
	latestPath = "/repos/JhnFrankz/upp/releases/latest"
	// downloadPath is the GitHub release-asset download path, relative to
	// Client.BaseURL: .../releases/download/{tag}/{asset}.
	downloadPath = "/repos/JhnFrankz/upp/releases/download"
)

// Release is a resolved upstream release. Tag is the release tag — the
// tag_name field of the GitHub releases/latest response (e.g. v0.1.1).
type Release struct {
	Tag string
}

// Client talks to the GitHub release endpoints for upp. BaseURL is the
// API base (https://api.github.com in production); every path is
// BaseURL-relative. HTTP is injectable for tests — NewClient installs
// the production client (10s timeouts, HTTPS-only redirect policy), and
// a nil HTTP falls back to it. CachePath is the detection-cache file
// (empty disables caching); Now is the clock for cache freshness (nil
// falls back to time.Now).
//
// The client remembers the most recently resolved release so Download
// fetches from that same release (spec R4). A Client is not safe for
// concurrent use.
type Client struct {
	BaseURL   string
	HTTP      *http.Client
	CachePath string
	Now       func() time.Time

	release Release
}

// NewClient returns a Client with the production HTTP client: ~10s dial
// and request timeouts, environment proxies, and the HTTPS-only redirect
// policy (design D6). Tests construct Client directly with an injected
// HTTP client instead.
func NewClient(baseURL, cachePath string) *Client {
	return &Client{
		BaseURL:   baseURL,
		CachePath: cachePath,
		HTTP:      defaultHTTPClient(),
		Now:       time.Now,
	}
}

// defaultHTTPClient is the production HTTP client: ~10s dial and request
// timeouts (spec R2), environment proxies, and the HTTPS-only redirect
// policy (security-model delta: off-HTTPS redirect → fail closed).
func defaultHTTPClient() *http.Client {
	return &http.Client{
		Timeout: requestTimeout,
		Transport: &http.Transport{
			Proxy:       http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{Timeout: dialTimeout}).DialContext,
		},
		CheckRedirect: checkRedirect,
	}
}

// checkRedirect is the client-wide redirect policy: any hop whose target
// leaves HTTPS fails closed. GitHub's real redirect chains (api.github.com
// → api.github.com, github.com → objects.githubusercontent.com) stay on
// HTTPS and pass.
func checkRedirect(req *http.Request, via []*http.Request) error {
	if req.URL.Scheme != "https" {
		return fmt.Errorf("selfupdate: refusing redirect to non-HTTPS URL %q", req.URL)
	}
	return nil
}

// httpClient returns the injected client, falling back to the production
// one for zero-value Clients.
func (c *Client) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return defaultHTTPClient()
}

// fetchLatest queries the latest-release endpoint and parses the tag. It
// never consults the cache (design D4: explicit self-update is always
// fresh; the hint path applies the TTL itself in LatestCached).
func (c *Client) fetchLatest() (Release, error) {
	url := strings.TrimRight(c.BaseURL, "/") + latestPath
	resp, err := c.httpClient().Get(url)
	if err != nil {
		return Release{}, fmt.Errorf("selfupdate: latest release lookup failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Release{}, fmt.Errorf("selfupdate: latest release lookup failed: HTTP %d", resp.StatusCode)
	}
	var body struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return Release{}, fmt.Errorf("selfupdate: latest release lookup failed: decoding response: %w", err)
	}
	if body.TagName == "" {
		return Release{}, fmt.Errorf("selfupdate: latest release lookup failed: response has no tag_name")
	}
	return Release{Tag: body.TagName}, nil
}

// LatestFresh returns the latest release, always over the network
// (design D4). Errors propagate: on `upp self-update` an API failure
// must be visible and exit non-zero (spec R2).
func (c *Client) LatestFresh() (Release, error) {
	r, err := c.fetchLatest()
	if err != nil {
		return Release{}, err
	}
	c.release = r
	return r, nil
}

// LatestCached returns the latest release tag for the hint path: a fresh
// cache (< 24h per CacheTTL, decided with the injected Now clock) is
// reused with no network; otherwise the release is fetched and written
// through to CachePath. ANY failure — network, parse, or cache write —
// is silent: it returns false, never an error, so the hint can never
// fail the run (spec R2: offline silent). An empty CachePath disables
// caching entirely.
func (c *Client) LatestCached() (string, bool) {
	now := c.now()
	if c.CachePath != "" {
		if cached, ok := LoadDetectionCache(c.CachePath); ok && cached.Fresh(now) {
			return cached.Tag, true
		}
	}
	r, err := c.fetchLatest()
	if err != nil {
		return "", false
	}
	c.release = r
	if c.CachePath != "" {
		if err := SaveDetectionCache(c.CachePath, DetectionCache{
			Version: CacheVersion,
			Fetched: now,
			Tag:     r.Tag,
		}); err != nil {
			return "", false
		}
	}
	return r.Tag, true
}

// Download fetches the release asset name and checksums.txt from the
// same release that LatestFresh (or LatestCached) most recently
// resolved, both over HTTPS. It returns the asset bytes and the
// checksums bytes; sha256 verification happens in the update pipeline,
// not here (spec R4). A non-200 response for either file fails the whole
// download — nothing is returned partially.
func (c *Client) Download(name string) ([]byte, []byte, error) {
	if c.release.Tag == "" {
		return nil, nil, fmt.Errorf("selfupdate: no release resolved: call LatestFresh before Download")
	}
	base := strings.TrimRight(c.BaseURL, "/") + downloadPath + "/" + c.release.Tag + "/"
	asset, err := c.get(base + name)
	if err != nil {
		return nil, nil, err
	}
	checksums, err := c.get(base + "checksums.txt")
	if err != nil {
		return nil, nil, err
	}
	return asset, checksums, nil
}

// get performs a single GET and returns the body on HTTP 200.
func (c *Client) get(url string) ([]byte, error) {
	resp, err := c.httpClient().Get(url)
	if err != nil {
		return nil, fmt.Errorf("selfupdate: download failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("selfupdate: download %s failed: HTTP %d", url, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// now returns the injected clock, falling back to the wall clock.
func (c *Client) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}
