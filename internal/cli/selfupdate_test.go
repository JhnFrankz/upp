package cli

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JhnFrankz/upp/internal/platform"
	"github.com/JhnFrankz/upp/internal/selfupdate"
)

// selfUpdateServer routes the production release endpoints on one
// httptest server, mirroring internal/selfupdate's newReleaseServer
// (the path constants are duplicated here on purpose: the package does
// not export them). latestTag serves the latest-release JSON; a nil
// asset or checksums body makes that route 404. Every request is
// counted in reqs (may be nil). No real network is ever used.
func selfUpdateServer(t *testing.T, latestTag string, asset, checksums []byte, reqs *atomic.Int32) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if reqs != nil {
			reqs.Add(1)
		}
		switch {
		case r.URL.Path == "/repos/JhnFrankz/upp/releases/latest":
			_, _ = io.WriteString(w, fmt.Sprintf(`{"tag_name":%q}`, latestTag))
		case strings.HasSuffix(r.URL.Path, "/checksums.txt"):
			if checksums == nil {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write(checksums)
		case strings.HasPrefix(r.URL.Path, "/repos/JhnFrankz/upp/releases/download"):
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

// cliTestClient returns a selfupdate.Client wired to the test server.
func cliTestClient(ts *httptest.Server) *selfupdate.Client {
	return &selfupdate.Client{BaseURL: ts.URL}
}

// fakeBinary writes a fake installed upp binary with the given content
// and returns its path (the execPath seam target).
func fakeBinary(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "upp")
	if err := os.WriteFile(p, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

// cliArchive returns gzip-compressed tar bytes with the Makefile release
// layout entry "{assetDir}/upp" (assetDir = asset name minus .tar.gz).
func cliArchive(t *testing.T, assetName, content string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	hdr := &tar.Header{
		Name: strings.TrimSuffix(assetName, ".tar.gz") + "/upp",
		Mode: 0o755,
		Size: int64(len(content)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("write tar header: %v", err)
	}
	if _, err := tw.Write([]byte(content)); err != nil {
		t.Fatalf("write tar body: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}
	return buf.Bytes()
}

// cliChecksumLine renders the sha256sum(1) line the Makefile release
// target produces: "<hex>  <name>" (two spaces).
func cliChecksumLine(t *testing.T, data []byte, name string) string {
	t.Helper()
	return fmt.Sprintf("%x  %s\n", sha256.Sum256(data), name)
}

// linuxAMD64 is the canonical linux/amd64 platform used by the tests.
func linuxAMD64() (platform.Platform, error) {
	return platform.Platform{OS: "linux", Arch: "x86_64"}, nil
}

// backupFiles lists {binPath}.backup.* files next to the fake binary.
func backupFiles(t *testing.T, binPath string) []string {
	t.Helper()
	matches, err := filepath.Glob(binPath + ".backup.*")
	if err != nil {
		t.Fatal(err)
	}
	return matches
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// newSelfUpdateDeps builds the standard test deps: TTY stdin wired to
// reader, linux/amd64, the fake binary at binPath, and a client pointed
// at the test server.
func newSelfUpdateDeps(ts *httptest.Server, reader string, binPath string) selfUpdateDeps {
	return selfUpdateDeps{
		stdin:    strings.NewReader(reader),
		isTTY:    func() bool { return true },
		detect:   linuxAMD64,
		execPath: func() (string, error) { return binPath, nil },
		client:   cliTestClient(ts),
	}
}

func TestSelfUpdate_DevBuild(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	var reqs atomic.Int32
	ts := selfUpdateServer(t, "v0.1.1", nil, nil, &reqs)
	defer ts.Close()
	bin := fakeBinary(t, "OLD-BINARY")

	output := withCapturedStdout(func() {
		if err := runSelfUpdate(&GlobalFlags{}, "dev", newSelfUpdateDeps(ts, "y\n", bin)); err != nil {
			t.Errorf("dev build should exit 0, got: %v", err)
		}
	})

	if !strings.Contains(output, "development build") {
		t.Errorf("output should contain the development-build message, got: %q", output)
	}
	if got := reqs.Load(); got != 0 {
		t.Errorf("dev build must not touch the network, got %d requests", got)
	}
	if got := readFile(t, bin); got != "OLD-BINARY" {
		t.Errorf("binary must stay untouched, got %q", got)
	}
}

func TestSelfUpdate_DirtyBuild(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	var reqs atomic.Int32
	ts := selfUpdateServer(t, "v0.1.1", nil, nil, &reqs)
	defer ts.Close()

	output := withCapturedStdout(func() {
		if err := runSelfUpdate(&GlobalFlags{}, "v0.1.0-19-gd40e428-dirty", newSelfUpdateDeps(ts, "y\n", fakeBinary(t, "OLD"))); err != nil {
			t.Errorf("dirty build should exit 0, got: %v", err)
		}
	})

	if !strings.Contains(output, "development build") {
		t.Errorf("output should contain the development-build message, got: %q", output)
	}
	if got := reqs.Load(); got != 0 {
		t.Errorf("dirty build must not touch the network, got %d requests", got)
	}
}

func TestSelfUpdate_InvalidVersion(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	var reqs atomic.Int32
	ts := selfUpdateServer(t, "v0.1.1", nil, nil, &reqs)
	defer ts.Close()

	err := runSelfUpdate(&GlobalFlags{}, "banana", newSelfUpdateDeps(ts, "y\n", fakeBinary(t, "OLD")))
	if err == nil {
		t.Fatal("unparseable version should error")
	}
	if !strings.Contains(err.Error(), "banana") {
		t.Errorf("error should mention the version, got: %v", err)
	}
	if got := reqs.Load(); got != 0 {
		t.Errorf("invalid version must not touch the network, got %d requests", got)
	}
}

func TestSelfUpdate_UpToDate(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	var reqs atomic.Int32
	ts := selfUpdateServer(t, "v0.1.1", nil, nil, &reqs)
	defer ts.Close()

	output := withCapturedStdout(func() {
		if err := runSelfUpdate(&GlobalFlags{}, "v0.1.1", newSelfUpdateDeps(ts, "y\n", fakeBinary(t, "OLD"))); err != nil {
			t.Errorf("up-to-date should exit 0, got: %v", err)
		}
	})

	if !strings.Contains(output, "already up to date") {
		t.Errorf("output should contain the up-to-date message, got: %q", output)
	}
	if !strings.Contains(output, "v0.1.1") {
		t.Errorf("up-to-date message should show the version, got: %q", output)
	}
	if got := reqs.Load(); got != 1 {
		t.Errorf("up-to-date should make exactly one network call (latest lookup), got %d", got)
	}
}

func TestSelfUpdate_Confirmed(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	const assetName = "upp-linux-amd64.tar.gz"
	asset := cliArchive(t, assetName, "NEW-BINARY")
	checksums := []byte(cliChecksumLine(t, asset, assetName))
	var reqs atomic.Int32
	ts := selfUpdateServer(t, "v0.1.1", asset, checksums, &reqs)
	defer ts.Close()
	bin := fakeBinary(t, "OLD-BINARY")

	output := withCapturedStdout(func() {
		if err := runSelfUpdate(&GlobalFlags{}, "v0.1.0", newSelfUpdateDeps(ts, "y\n", bin)); err != nil {
			t.Fatalf("confirmed update should exit 0, got: %v", err)
		}
	})

	if !strings.Contains(output, "Update upp from v0.1.0 to v0.1.1?") {
		t.Errorf("prompt should show current → latest, got: %q", output)
	}
	if !strings.Contains(output, bin) {
		t.Errorf("prompt should show the target binary path, got: %q", output)
	}
	if !strings.Contains(output, "upp updated: v0.1.0 → v0.1.1") {
		t.Errorf("output should contain the success line, got: %q", output)
	}
	if got := reqs.Load(); got != 3 {
		t.Errorf("confirmed flow should make exactly 3 requests (latest+asset+checksums), got %d", got)
	}
	if got := readFile(t, bin); got != "NEW-BINARY" {
		t.Errorf("binary should be replaced, got %q", got)
	}
	backups := backupFiles(t, bin)
	if len(backups) != 1 {
		t.Fatalf("expected exactly one backup file, got %v", backups)
	}
	if got := readFile(t, backups[0]); got != "OLD-BINARY" {
		t.Errorf("backup should hold the old binary, got %q", got)
	}
}

func TestSelfUpdate_Declined(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	const assetName = "upp-linux-amd64.tar.gz"
	asset := cliArchive(t, assetName, "NEW-BINARY")
	checksums := []byte(cliChecksumLine(t, asset, assetName))
	var reqs atomic.Int32
	ts := selfUpdateServer(t, "v0.1.1", asset, checksums, &reqs)
	defer ts.Close()
	bin := fakeBinary(t, "OLD-BINARY")

	output := withCapturedStdout(func() {
		if err := runSelfUpdate(&GlobalFlags{}, "v0.1.0", newSelfUpdateDeps(ts, "n\n", bin)); err != nil {
			t.Fatalf("declining should exit 0, got: %v", err)
		}
	})

	if !strings.Contains(output, "Update upp from v0.1.0 to v0.1.1?") {
		t.Errorf("prompt should still be shown, got: %q", output)
	}
	if got := readFile(t, bin); got != "OLD-BINARY" {
		t.Errorf("declining must not replace the binary, got %q", got)
	}
	if backups := backupFiles(t, bin); len(backups) != 0 {
		t.Errorf("declining must not create backups, got %v", backups)
	}
	if got := reqs.Load(); got != 3 {
		t.Errorf("declined flow downloads before the gate (design D8), got %d requests", got)
	}
}

func TestSelfUpdate_NonTTY(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	const assetName = "upp-linux-amd64.tar.gz"
	asset := cliArchive(t, assetName, "NEW-BINARY")
	checksums := []byte(cliChecksumLine(t, asset, assetName))
	var reqs atomic.Int32
	ts := selfUpdateServer(t, "v0.1.1", asset, checksums, &reqs)
	defer ts.Close()
	bin := fakeBinary(t, "OLD-BINARY")
	deps := newSelfUpdateDeps(ts, "y\n", bin)
	deps.isTTY = func() bool { return false } // stdin piped

	err := runSelfUpdate(&GlobalFlags{}, "v0.1.0", deps)
	if err == nil {
		t.Fatal("non-TTY stdin must deny the update")
	}
	if !errors.Is(err, selfupdate.ErrNotTTY) {
		t.Errorf("error should carry ErrNotTTY, got: %v", err)
	}
	if !strings.Contains(err.Error(), "interactive terminal") {
		t.Errorf("deny message should be clear, got: %v", err)
	}
	if got := readFile(t, bin); got != "OLD-BINARY" {
		t.Errorf("non-TTY denial must not replace the binary, got %q", got)
	}
	if backups := backupFiles(t, bin); len(backups) != 0 {
		t.Errorf("non-TTY denial must not create backups, got %v", backups)
	}
	if got := reqs.Load(); got != 3 {
		t.Errorf("non-TTY denial happens at the gate after download (design D8), got %d requests", got)
	}
}

func TestSelfUpdate_CIDeny(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	var reqs atomic.Int32
	ts := selfUpdateServer(t, "v0.1.1", nil, nil, &reqs)
	defer ts.Close()

	err := runSelfUpdate(&GlobalFlags{CI: true}, "v0.1.0", newSelfUpdateDeps(ts, "y\n", fakeBinary(t, "OLD")))
	if err == nil {
		t.Fatal("--ci must deny the update")
	}
	if !errors.Is(err, selfupdate.ErrDeniedCI) {
		t.Errorf("error should carry ErrDeniedCI, got: %v", err)
	}
	if !strings.Contains(err.Error(), "denied in --ci mode") {
		t.Errorf("deny message should be clear, got: %v", err)
	}
	if got := reqs.Load(); got != 0 {
		t.Errorf("--ci deny must happen before any network call, got %d requests", got)
	}
}

func TestSelfUpdate_CIDenyThroughRoot(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	root, gf := BuildRoot()
	AddCommands(root, gf)
	root.Version = "v0.1.0"
	root.SetArgs([]string{"self-update", "--ci"})

	// Safe to execute: the --ci gate runs before the client is
	// constructed, so no real network can be hit.
	err := root.Execute()
	if err == nil {
		t.Fatal("self-update --ci through the root must fail")
	}
	if !errors.Is(err, selfupdate.ErrDeniedCI) {
		t.Errorf("error should carry ErrDeniedCI, got: %v", err)
	}
}

func TestSelfUpdate_QuietKeepsPrompt(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	const assetName = "upp-linux-amd64.tar.gz"
	asset := cliArchive(t, assetName, "NEW-BINARY")
	checksums := []byte(cliChecksumLine(t, asset, assetName))
	ts := selfUpdateServer(t, "v0.1.1", asset, checksums, nil)
	defer ts.Close()
	bin := fakeBinary(t, "OLD-BINARY")

	output := withCapturedStdout(func() {
		if err := runSelfUpdate(&GlobalFlags{Quiet: true}, "v0.1.0", newSelfUpdateDeps(ts, "n\n", bin)); err != nil {
			t.Fatalf("quiet flow should exit 0, got: %v", err)
		}
	})

	if !strings.Contains(output, "Update upp from v0.1.0 to v0.1.1?") {
		t.Errorf("--quiet must never suppress the confirm prompt, got: %q", output)
	}
	if !strings.Contains(output, "Proceed?") {
		t.Errorf("--quiet must never suppress the prompt question, got: %q", output)
	}
}

func TestSelfUpdate_OnlySkipIgnored(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	var reqs atomic.Int32
	ts := selfUpdateServer(t, "v0.1.1", nil, nil, &reqs)
	defer ts.Close()

	output := withCapturedStdout(func() {
		err := runSelfUpdate(&GlobalFlags{Only: "brew", Skip: "apt"}, "v0.1.1", newSelfUpdateDeps(ts, "y\n", fakeBinary(t, "OLD")))
		if err != nil {
			t.Fatalf("--only/--skip must be ignored (normal flow), got: %v", err)
		}
	})

	if !strings.Contains(output, "already up to date") {
		t.Errorf("flow should proceed normally ignoring --only/--skip, got: %q", output)
	}
	if got := reqs.Load(); got != 1 {
		t.Errorf("normal flow should make exactly one network call, got %d", got)
	}
}

// TestSelfUpdate_UnknownFlagRejected pins the command-interface
// Self-Update Flag Semantics scenario: self-update accepts no flags in v1,
// so any unknown flag gets cobra's default rejection (error + usage,
// non-zero exit) instead of being silently ignored.
func TestSelfUpdate_UnknownFlagRejected(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	root, gf := BuildRoot()
	AddCommands(root, gf)
	root.Version = "v0.1.0"
	root.SetArgs([]string{"self-update", "--yes"})

	err := root.Execute()
	if err == nil {
		t.Fatal("self-update --yes must be rejected with an error (non-zero exit)")
	}
	if !strings.Contains(err.Error(), "unknown flag") {
		t.Errorf("error should be cobra's unknown-flag rejection, got: %v", err)
	}
}

func TestSelfUpdate_WindowsUnsupported(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	var reqs atomic.Int32
	ts := selfUpdateServer(t, "v0.1.1", nil, nil, &reqs)
	defer ts.Close()
	deps := newSelfUpdateDeps(ts, "y\n", fakeBinary(t, "OLD"))
	deps.detect = func() (platform.Platform, error) {
		return platform.Platform{OS: "windows", Arch: "x86_64"}, nil
	}

	err := runSelfUpdate(&GlobalFlags{}, "v0.1.0", deps)
	if err == nil {
		t.Fatal("windows must be refused")
	}
	if !errors.Is(err, selfupdate.ErrUnsupportedPlatform) {
		t.Errorf("error should carry ErrUnsupportedPlatform, got: %v", err)
	}
	if !strings.Contains(err.Error(), "not supported") {
		t.Errorf("error should say not supported yet, got: %v", err)
	}
	if got := reqs.Load(); got != 0 {
		t.Errorf("windows refusal must not touch the network, got %d requests", got)
	}
}

func TestSelfUpdate_ChecksumMismatch(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	const assetName = "upp-linux-amd64.tar.gz"
	asset := cliArchive(t, assetName, "NEW-BINARY")
	checksums := []byte(fmt.Sprintf("%064x  %s\n", 0xdead, assetName)) // wrong sum
	var reqs atomic.Int32
	ts := selfUpdateServer(t, "v0.1.1", asset, checksums, &reqs)
	defer ts.Close()
	bin := fakeBinary(t, "OLD-BINARY")

	err := runSelfUpdate(&GlobalFlags{}, "v0.1.0", newSelfUpdateDeps(ts, "y\n", bin))
	if err == nil {
		t.Fatal("checksum mismatch must abort")
	}
	if !errors.Is(err, selfupdate.ErrChecksumMismatch) {
		t.Errorf("error should carry ErrChecksumMismatch, got: %v", err)
	}
	if got := readFile(t, bin); got != "OLD-BINARY" {
		t.Errorf("mismatch must leave the binary untouched, got %q", got)
	}
	if backups := backupFiles(t, bin); len(backups) != 0 {
		t.Errorf("mismatch must not create backups, got %v", backups)
	}
	if got := reqs.Load(); got != 3 {
		t.Errorf("mismatch flow should make exactly 3 requests, got %d", got)
	}
}
