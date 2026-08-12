package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JhnFrankz/upp/internal/platform"
	"github.com/JhnFrankz/upp/internal/selfupdate"
)

// The hint tests drive runCheck directly — the exact function both
// `upp check` and bare `upp` funnel into (parser.go) — with an injected
// client factory, mirroring the selfUpdateDeps seam. Zero real network:
// every hint client points at an httptest server; the "client never
// constructed" assertions prove zero-network-by-default structurally.

// writeCheckConfig writes a config.toml with the given [settings] body
// into a fresh HOME, so config.Load() inside runCheck sees it. Every
// official catalog tool is written as disabled so the check loop is
// hermetic and fast — the hint path is what is under test, not the tool
// adapters.
func writeCheckConfig(t *testing.T, settingsBody string) string {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfgDir := filepath.Join(tmpDir, ".config", "upp")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}

	var tools strings.Builder
	if p, err := platform.Detect(); err == nil {
		for _, tool := range platform.CatalogFor(p.OS) {
			fmt.Fprintf(&tools, "\n[tools.%s]\nenabled = false\n", tool.ID)
		}
	}

	tomlContent := "version = 1\n\n[settings]\n" + settingsBody + "\n" + tools.String()
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(tomlContent), 0o644); err != nil {
		t.Fatal(err)
	}
	return tmpDir
}

// writeDetectionCache pre-writes a self-update detection cache at the
// config dir with the given tag and age, so the hint path's freshness
// branch can be exercised without any network.
func writeDetectionCache(t *testing.T, configDir, tag string, age time.Duration) {
	t.Helper()
	cache := fmt.Sprintf(
		`{"version":1,"fetched":%q,"tag":%q}`,
		time.Now().Add(-age).Format(time.RFC3339), tag)
	path := filepath.Join(configDir, ".config", "upp", "self-update-cache.json")
	if err := os.WriteFile(path, []byte(cache), 0o644); err != nil {
		t.Fatal(err)
	}
}

// readCachedTag returns the tag stored in the detection cache at the
// config dir, or "" when the cache file does not exist.
func readCachedTag(t *testing.T, configDir string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(configDir, ".config", "upp", "self-update-cache.json"))
	if os.IsNotExist(err) {
		return ""
	}
	if err != nil {
		t.Fatal(err)
	}
	var c selfupdate.DetectionCache
	if err := json.Unmarshal(data, &c); err != nil {
		t.Fatalf("read cache: %v", err)
	}
	return c.Tag
}

// hintFactory returns a client factory wired to the test server plus a
// construction counter. The returned client keeps the cachePath the hook
// passes in, so the real write-through cache lands at {config-dir}/
// self-update-cache.json (spec cache-location scenario).
func hintFactory(ts *httptest.Server, constructions *int) (func(string) *selfupdate.Client, *string) {
	var lastCachePath string
	return func(cachePath string) *selfupdate.Client {
		*constructions++
		lastCachePath = cachePath
		return &selfupdate.Client{BaseURL: ts.URL, CachePath: cachePath}
	}, &lastCachePath
}

const enHintLine = "⬆️ upp v0.1.1 available (current v0.1.0) — run \"upp self-update\"\n"

// TestCheckHint_DefaultOff_ZeroNetwork locks the spec config-system
// "Zero network default" scenario: with no setting present the hint path
// must not even construct a client — the factory failing on any call
// proves it, and runCheck still exits 0.
func TestCheckHint_DefaultOff_ZeroNetwork(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // no config file at all → defaults

	ts := httptest.NewServer(httpNotFoundHandler())
	defer ts.Close()

	constructions := 0
	factory, _ := hintFactory(ts, &constructions)

	output := withCapturedStdout(func() {
		if err := runCheck(&GlobalFlags{}, "v0.1.0", checkDeps{clientFactory: factory}); err != nil {
			t.Errorf("runCheck error: %v", err)
		}
	})

	if constructions != 0 {
		t.Errorf("client constructed %d times with hint disabled, want 0 (zero network)", constructions)
	}
	if strings.Contains(output, "self-update") {
		t.Errorf("no hint expected with default config, got:\n%s", output)
	}
}

// TestCheckHint_On_NewerRelease covers the spec ux-patterns "Newer
// release" scenario end to end: one hint line after the summary, exit
// unchanged, exactly one network call, and the detection cache created
// at {config-dir}/self-update-cache.json.
func TestCheckHint_On_NewerRelease(t *testing.T) {
	tmpDir := writeCheckConfig(t, "check_self_update = true")

	var reqs atomic.Int32
	ts := selfUpdateServer(t, "v0.1.1", nil, nil, &reqs)
	defer ts.Close()

	constructions := 0
	factory, lastCachePath := hintFactory(ts, &constructions)

	output := withCapturedStdout(func() {
		if err := runCheck(&GlobalFlags{}, "v0.1.0", checkDeps{clientFactory: factory}); err != nil {
			t.Errorf("runCheck error: %v", err)
		}
	})

	wantCachePath := filepath.Join(tmpDir, ".config", "upp", "self-update-cache.json")
	if !strings.Contains(output, enHintLine) {
		t.Errorf("expected hint line %q in output, got:\n%s", enHintLine, output)
	}
	if constructions != 1 {
		t.Errorf("client constructed %d times, want 1", constructions)
	}
	if *lastCachePath != wantCachePath {
		t.Errorf("factory cache path = %q, want %q", *lastCachePath, wantCachePath)
	}
	if got := readCachedTag(t, tmpDir); got != "v0.1.1" {
		t.Errorf("cache tag = %q, want v0.1.1 (cache must be created on first hint check)", got)
	}
	if reqs.Load() != 1 {
		t.Errorf("network requests = %d, want 1", reqs.Load())
	}
}

// TestCheckHint_FreshCache_ZeroNetwork covers the "fresh cache" branch:
// a <24h cache with a newer tag serves the hint with ZERO network calls.
func TestCheckHint_FreshCache_ZeroNetwork(t *testing.T) {
	tmpDir := writeCheckConfig(t, "check_self_update = true")
	writeDetectionCache(t, tmpDir, "v0.1.1", time.Hour)

	var reqs atomic.Int32
	ts := selfUpdateServer(t, "v9.9.9", nil, nil, &reqs) // server would disagree
	defer ts.Close()

	constructions := 0
	factory, _ := hintFactory(ts, &constructions)

	output := withCapturedStdout(func() {
		if err := runCheck(&GlobalFlags{}, "v0.1.0", checkDeps{clientFactory: factory}); err != nil {
			t.Errorf("runCheck error: %v", err)
		}
	})

	if !strings.Contains(output, enHintLine) {
		t.Errorf("expected hint from fresh cache, got:\n%s", output)
	}
	if reqs.Load() != 0 {
		t.Errorf("network requests = %d, want 0 (fresh cache)", reqs.Load())
	}
}

// TestCheckHint_StaleCache_Refetch covers the stale-cache branch: a
// >=24h cache triggers a refetch, the hint is served from the fresh
// response, and the cache is rewritten (write-through).
func TestCheckHint_StaleCache_Refetch(t *testing.T) {
	tmpDir := writeCheckConfig(t, "check_self_update = true")
	writeDetectionCache(t, tmpDir, "v0.1.0", 25*time.Hour) // stale: older tag too

	var reqs atomic.Int32
	ts := selfUpdateServer(t, "v0.1.1", nil, nil, &reqs)
	defer ts.Close()

	constructions := 0
	factory, _ := hintFactory(ts, &constructions)

	output := withCapturedStdout(func() {
		if err := runCheck(&GlobalFlags{}, "v0.1.0", checkDeps{clientFactory: factory}); err != nil {
			t.Errorf("runCheck error: %v", err)
		}
	})

	if !strings.Contains(output, enHintLine) {
		t.Errorf("expected hint after refetch, got:\n%s", output)
	}
	if reqs.Load() != 1 {
		t.Errorf("network requests = %d, want 1 (stale refetch)", reqs.Load())
	}
	if got := readCachedTag(t, tmpDir); got != "v0.1.1" {
		t.Errorf("cache tag after refetch = %q, want v0.1.1 (write-through)", got)
	}
}

// TestCheckHint_Offline_Silent covers the spec ux-patterns "Offline"
// scenario: API failure → no hint, no error, exit unchanged.
func TestCheckHint_Offline_Silent(t *testing.T) {
	writeCheckConfig(t, "check_self_update = true")

	ts := httptest.NewServer(httpInternalErrorHandler())
	defer ts.Close()

	constructions := 0
	factory, _ := hintFactory(ts, &constructions)

	output := withCapturedStdout(func() {
		if err := runCheck(&GlobalFlags{}, "v0.1.0", checkDeps{clientFactory: factory}); err != nil {
			t.Errorf("runCheck must stay silent on API failure, got error: %v", err)
		}
	})

	if strings.Contains(output, "self-update") {
		t.Errorf("offline must not print a hint, got:\n%s", output)
	}
}

// TestCheckHint_Quiet_NoHint covers the spec ux-patterns "Quiet"
// scenario: --quiet suppresses the hint (and skips the whole hint path,
// so no client is constructed — zero network in quiet mode).
func TestCheckHint_Quiet_NoHint(t *testing.T) {
	writeCheckConfig(t, "check_self_update = true")

	ts := httptest.NewServer(httpNotFoundHandler())
	defer ts.Close()

	constructions := 0
	factory, _ := hintFactory(ts, &constructions)

	output := withCapturedStdout(func() {
		if err := runCheck(&GlobalFlags{Quiet: true}, "v0.1.0", checkDeps{clientFactory: factory}); err != nil {
			t.Errorf("runCheck error: %v", err)
		}
	})

	if constructions != 0 {
		t.Errorf("client constructed %d times in quiet mode, want 0", constructions)
	}
	if strings.Contains(output, "self-update") {
		t.Errorf("quiet mode must not print a hint, got:\n%s", output)
	}
}

// TestCheckHint_DevBuild_NoHint covers dev and dirty builds: no hint and
// no client construction — development builds never claim updates and
// never hit the network (spec R1 / design D9 gate).
func TestCheckHint_DevBuild_NoHint(t *testing.T) {
	for _, version := range []string{"dev", "v0.1.0-dirty"} {
		t.Run(version, func(t *testing.T) {
			writeCheckConfig(t, "check_self_update = true")

			ts := httptest.NewServer(httpNotFoundHandler())
			defer ts.Close()

			constructions := 0
			factory, _ := hintFactory(ts, &constructions)

			output := withCapturedStdout(func() {
				if err := runCheck(&GlobalFlags{}, version, checkDeps{clientFactory: factory}); err != nil {
					t.Errorf("runCheck error: %v", err)
				}
			})

			if constructions != 0 {
				t.Errorf("client constructed %d times for %q, want 0", constructions, version)
			}
			if strings.Contains(output, "self-update") {
				t.Errorf("dev build must not print a hint for %q, got:\n%s", version, output)
			}
		})
	}
}

// TestCheckHint_UpToDate_NoHint covers the spec ux-patterns "Up to
// date" scenario: latest == current → no hint (the lookup happens, the
// comparison decides).
func TestCheckHint_UpToDate_NoHint(t *testing.T) {
	writeCheckConfig(t, "check_self_update = true")

	var reqs atomic.Int32
	ts := selfUpdateServer(t, "v0.1.0", nil, nil, &reqs)
	defer ts.Close()

	constructions := 0
	factory, _ := hintFactory(ts, &constructions)

	output := withCapturedStdout(func() {
		if err := runCheck(&GlobalFlags{}, "v0.1.0", checkDeps{clientFactory: factory}); err != nil {
			t.Errorf("runCheck error: %v", err)
		}
	})

	if strings.Contains(output, "self-update") {
		t.Errorf("up-to-date must not print a hint, got:\n%s", output)
	}
	if reqs.Load() != 1 {
		t.Errorf("network requests = %d, want 1 (lookup still happens)", reqs.Load())
	}
}

// httpNotFoundHandler is a server that answers 404 to everything — any
// request through it would be a test failure signal.
func httpNotFoundHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
}

// httpInternalErrorHandler answers 500 to everything — simulates an
// unreachable/offline API for the silent-failure scenario.
func httpInternalErrorHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", 500)
	})
}
