package selfupdate

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JhnFrankz/upp/internal/platform"
)

// archiveEntry is a single entry for buildArchive. typ defaults to
// tar.TypeReg; link is the target for symlink/hardlink entries.
type archiveEntry struct {
	name    string
	typ     byte
	content string
	link    string
}

// buildArchive returns gzip-compressed tar bytes containing the given
// entries, mirroring the Makefile release layout (tar czf of a staged
// upp-{os}-{arch}/ directory).
func buildArchive(t *testing.T, entries ...archiveEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, e := range entries {
		typ := e.typ
		if typ == 0 {
			typ = tar.TypeReg
		}
		hdr := &tar.Header{
			Name:     e.name,
			Mode:     0o755,
			Typeflag: typ,
			Size:     int64(len(e.content)),
		}
		if typ == tar.TypeSymlink || typ == tar.TypeLink {
			hdr.Linkname = e.link
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("write tar header for %q: %v", e.name, err)
		}
		if typ == tar.TypeReg {
			if _, err := tw.Write([]byte(e.content)); err != nil {
				t.Fatalf("write tar body for %q: %v", e.name, err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}
	return buf.Bytes()
}

// checksumLine renders the sha256sum(1) line the Makefile release target
// produces: "<hex>  <name>" (two spaces).
func checksumLine(t *testing.T, data []byte, name string) string {
	t.Helper()
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x  %s\n", sum, name)
}

// errName returns the sentinel a wrapped error carries, or "".
func errName(err error) string {
	switch {
	case errors.Is(err, ErrChecksumMismatch):
		return "ErrChecksumMismatch"
	case errors.Is(err, ErrUpToDate):
		return "ErrUpToDate"
	case errors.Is(err, ErrDevelopmentBuild):
		return "ErrDevelopmentBuild"
	case errors.Is(err, ErrUnsupportedPlatform):
		return "ErrUnsupportedPlatform"
	case errors.Is(err, ErrNotWritable):
		return "ErrNotWritable"
	}
	return ""
}

func TestVerifyChecksum(t *testing.T) {
	const (
		asset = "upp-linux-amd64.tar.gz"
		body  = "some-archive-bytes"
	)
	sum := sha256.Sum256([]byte(body))
	hex := fmt.Sprintf("%x", sum)

	tests := []struct {
		name      string
		asset     []byte
		checksums string
		wantErr   string // sentinel name or ""
	}{
		{"match", []byte(body), hex + "  " + asset + "\n", ""},
		{"match with other entries present", []byte(body),
			hex + "  " + asset + "\n" + strings.Repeat("0", 64) + "  upp-darwin-arm64.tar.gz\n", ""},
		{"match with binary-mode star prefix", []byte(body), hex + " *" + asset + "\n", ""},
		{"match with uppercase hex", []byte(body), strings.ToUpper(hex) + "  " + asset + "\n", ""},
		{"match with crlf line ending", []byte(body), hex + "  " + asset + "\r\n", ""},
		{"mismatch", []byte(body), strings.Repeat("0", 64) + "  " + asset + "\n", "ErrChecksumMismatch"},
		{"missing entry", []byte(body), hex + "  upp-darwin-arm64.tar.gz\n", "ErrChecksumMismatch"},
		{"empty checksums", []byte(body), "", "ErrChecksumMismatch"},
		{"garbage checksums", []byte(body), "not a checksum file at all\n", "ErrChecksumMismatch"},
		{"truncated hex for asset", []byte(body), "abc123  " + asset + "\n", "ErrChecksumMismatch"},
		{"non-hex for asset", []byte(body), "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz  " + asset + "\n", "ErrChecksumMismatch"},
		{"empty body vs matching empty-line", []byte(body), "  " + asset + "\n", "ErrChecksumMismatch"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := verifyChecksum(tt.asset, []byte(tt.checksums), asset)
			if got := errName(err); got != tt.wantErr {
				t.Errorf("verifyChecksum error = %v (sentinel %q), want %q", err, got, tt.wantErr)
			}
		})
	}
}

func TestExtract(t *testing.T) {
	const assetName = "upp-linux-amd64.tar.gz"

	// realBytes proves extraction preserves exact payload bytes.
	realBytes := []byte("\x7fELF fake binary bytes for upp")

	tests := []struct {
		name    string
		archive []byte
		wantErr bool
	}{
		{"normal archive extracts only the binary",
			buildArchive(t, archiveEntry{name: "upp-linux-amd64/upp", content: string(realBytes)}), false},
		{"extra entries are ignored",
			buildArchive(t,
				archiveEntry{name: "upp-linux-amd64/upp", content: string(realBytes)},
				archiveEntry{name: "README.md", content: "readme"},
				archiveEntry{name: "upp-linux-amd64/LICENSE", content: "mit"},
				archiveEntry{name: "upp-darwin-arm64/upp", content: "other-platform"}), false},
		{"binary entry missing", buildArchive(t,
			archiveEntry{name: "upp-linux-amd64/README", content: "x"}), true},
		{"path traversal rejected", buildArchive(t,
			archiveEntry{name: "../evil", content: "x"}), true},
		{"nested path traversal rejected", buildArchive(t,
			archiveEntry{name: "upp-linux-amd64/../../evil", content: "x"}), true},
		{"absolute path rejected", buildArchive(t,
			archiveEntry{name: "/etc/upp-evil", content: "x"}), true},
		{"symlink entry rejected", buildArchive(t,
			archiveEntry{name: "upp-linux-amd64/upp", content: string(realBytes)},
			archiveEntry{name: "link", typ: tar.TypeSymlink, link: "/etc/passwd"}), true},
		{"binary entry as symlink rejected", buildArchive(t,
			archiveEntry{name: "upp-linux-amd64/upp", typ: tar.TypeSymlink, link: "/bin/sh"}), true},
		{"hardlink entry rejected", buildArchive(t,
			archiveEntry{name: "upp-linux-amd64/upp", typ: tar.TypeLink, link: "/bin/sh"}), true},
		{"binary entry as directory rejected", buildArchive(t,
			archiveEntry{name: "upp-linux-amd64/upp", typ: tar.TypeDir}), true},
		{"not gzip", []byte("this is not gzip data"), true},
		{"gzip but not tar", func() []byte {
			var buf bytes.Buffer
			gz := gzip.NewWriter(&buf)
			_, _ = gz.Write([]byte("hello"))
			_ = gz.Close()
			return buf.Bytes()
		}(), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dest := t.TempDir()
			got, err := extract(tt.archive, assetName, dest)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("extract: want error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("extract: %v", err)
			}
			wantPath := filepath.Join(dest, "upp")
			if got != wantPath {
				t.Errorf("extract returned %q, want %q", got, wantPath)
			}
			data, err := os.ReadFile(wantPath)
			if err != nil {
				t.Fatalf("read extracted binary: %v", err)
			}
			if !bytes.Equal(data, realBytes) {
				t.Errorf("extracted bytes = %q, want %q", data, realBytes)
			}
			fi, err := os.Stat(wantPath)
			if err != nil {
				t.Fatalf("stat extracted binary: %v", err)
			}
			if fi.Mode().Perm() != 0o755 {
				t.Errorf("extracted binary mode = %v, want 0755", fi.Mode().Perm())
			}
			entries, err := os.ReadDir(dest)
			if err != nil {
				t.Fatalf("read dest dir: %v", err)
			}
			if len(entries) != 1 || entries[0].Name() != "upp" {
				t.Errorf("dest dir entries = %v, want exactly [upp]", entries)
			}
		})
	}
}

func TestPrepare(t *testing.T) {
	linuxAmd64 := platform.Platform{OS: platform.OSLinux, Arch: platform.ArchX86_64}
	unsupported := platform.Platform{OS: "freebsd", Arch: "amd64"}

	newArchive := func(t *testing.T) []byte {
		t.Helper()
		return buildArchive(t, archiveEntry{name: "upp-linux-amd64/upp", content: "new-binary-bytes"})
	}

	t.Run("happy path verifies, extracts to temp, returns release", func(t *testing.T) {
		archive := newArchive(t)
		ts := newReleaseServer(t, http.StatusOK, `{"tag_name":"v0.1.1"}`,
			archive, []byte(checksumLine(t, archive, "upp-linux-amd64.tar.gz")), nil)
		defer ts.Close()
		c, _ := newTestClient(t, ts)

		rel, binPath, err := Prepare(c, Version{Tag: [3]int{0, 1, 0}}, linuxAmd64)
		if err != nil {
			t.Fatalf("Prepare: %v", err)
		}
		if rel.Tag != "v0.1.1" {
			t.Errorf("Prepare release = %+v, want tag v0.1.1", rel)
		}
		data, err := os.ReadFile(binPath)
		if err != nil {
			t.Fatalf("read prepared binary: %v", err)
		}
		if string(data) != "new-binary-bytes" {
			t.Errorf("prepared binary = %q, want new-binary-bytes", data)
		}
	})

	t.Run("dev build fails before any network", func(t *testing.T) {
		var reqs atomic.Int32
		ts := newReleaseServer(t, http.StatusOK, `{"tag_name":"v0.1.1"}`, nil, nil, &reqs)
		defer ts.Close()
		c, _ := newTestClient(t, ts)

		_, _, err := Prepare(c, Version{Dev: true}, linuxAmd64)
		if !errors.Is(err, ErrDevelopmentBuild) {
			t.Fatalf("Prepare(dev) error = %v, want ErrDevelopmentBuild", err)
		}
		if n := reqs.Load(); n != 0 {
			t.Errorf("dev build made %d network requests, want 0", n)
		}
	})

	t.Run("dirty build fails before any network", func(t *testing.T) {
		var reqs atomic.Int32
		ts := newReleaseServer(t, http.StatusOK, `{"tag_name":"v0.1.1"}`, nil, nil, &reqs)
		defer ts.Close()
		c, _ := newTestClient(t, ts)

		_, _, err := Prepare(c, Version{Tag: [3]int{0, 1, 0}, Dirty: true}, linuxAmd64)
		if !errors.Is(err, ErrDevelopmentBuild) {
			t.Fatalf("Prepare(dirty) error = %v, want ErrDevelopmentBuild", err)
		}
		if n := reqs.Load(); n != 0 {
			t.Errorf("dirty build made %d network requests, want 0", n)
		}
	})

	t.Run("up to date stops before download", func(t *testing.T) {
		var reqs atomic.Int32
		ts := newReleaseServer(t, http.StatusOK, `{"tag_name":"v0.1.1"}`,
			newArchive(t), []byte("x"), &reqs)
		defer ts.Close()
		c, _ := newTestClient(t, ts)

		_, _, err := Prepare(c, Version{Tag: [3]int{0, 1, 1}}, linuxAmd64)
		if !errors.Is(err, ErrUpToDate) {
			t.Fatalf("Prepare(up-to-date) error = %v, want ErrUpToDate", err)
		}
		if n := reqs.Load(); n != 1 {
			t.Errorf("up-to-date made %d network requests, want 1 (latest lookup only, no download)", n)
		}
	})

	t.Run("current newer than latest is up to date", func(t *testing.T) {
		ts := newReleaseServer(t, http.StatusOK, `{"tag_name":"v0.1.1"}`, nil, nil, nil)
		defer ts.Close()
		c, _ := newTestClient(t, ts)

		_, _, err := Prepare(c, Version{Tag: [3]int{0, 1, 2}}, linuxAmd64)
		if !errors.Is(err, ErrUpToDate) {
			t.Fatalf("Prepare(newer) error = %v, want ErrUpToDate", err)
		}
	})

	t.Run("unsupported platform fails before any network", func(t *testing.T) {
		var reqs atomic.Int32
		ts := newReleaseServer(t, http.StatusOK, `{"tag_name":"v0.1.1"}`, nil, nil, &reqs)
		defer ts.Close()
		c, _ := newTestClient(t, ts)

		_, _, err := Prepare(c, Version{Tag: [3]int{0, 1, 0}}, unsupported)
		if !errors.Is(err, ErrUnsupportedPlatform) {
			t.Fatalf("Prepare(freebsd) error = %v, want ErrUnsupportedPlatform", err)
		}
		if n := reqs.Load(); n != 0 {
			t.Errorf("unsupported platform made %d network requests, want 0", n)
		}
	})

	t.Run("checksum mismatch aborts", func(t *testing.T) {
		archive := newArchive(t)
		wrong := strings.Repeat("0", 64) + "  upp-linux-amd64.tar.gz\n"
		ts := newReleaseServer(t, http.StatusOK, `{"tag_name":"v0.1.1"}`,
			archive, []byte(wrong), nil)
		defer ts.Close()
		c, _ := newTestClient(t, ts)

		_, _, err := Prepare(c, Version{Tag: [3]int{0, 1, 0}}, linuxAmd64)
		if !errors.Is(err, ErrChecksumMismatch) {
			t.Fatalf("Prepare(mismatch) error = %v, want ErrChecksumMismatch", err)
		}
	})

	t.Run("missing checksum entry aborts", func(t *testing.T) {
		archive := newArchive(t)
		ts := newReleaseServer(t, http.StatusOK, `{"tag_name":"v0.1.1"}`,
			archive, []byte(checksumLine(t, archive, "upp-darwin-arm64.tar.gz")), nil)
		defer ts.Close()
		c, _ := newTestClient(t, ts)

		_, _, err := Prepare(c, Version{Tag: [3]int{0, 1, 0}}, linuxAmd64)
		if !errors.Is(err, ErrChecksumMismatch) {
			t.Fatalf("Prepare(missing entry) error = %v, want ErrChecksumMismatch", err)
		}
	})

	t.Run("asset 404 is a visible error", func(t *testing.T) {
		ts := newReleaseServer(t, http.StatusOK, `{"tag_name":"v0.1.1"}`, nil, []byte("x"), nil)
		defer ts.Close()
		c, _ := newTestClient(t, ts)

		_, _, err := Prepare(c, Version{Tag: [3]int{0, 1, 0}}, linuxAmd64)
		if err == nil || !strings.Contains(err.Error(), "404") {
			t.Errorf("Prepare(asset 404) error = %v, want visible 404 error", err)
		}
	})

	t.Run("checksums 404 is a visible error", func(t *testing.T) {
		archive := newArchive(t)
		ts := newReleaseServer(t, http.StatusOK, `{"tag_name":"v0.1.1"}`,
			archive, nil, nil)
		defer ts.Close()
		c, _ := newTestClient(t, ts)

		_, _, err := Prepare(c, Version{Tag: [3]int{0, 1, 0}}, linuxAmd64)
		if err == nil || !strings.Contains(err.Error(), "404") {
			t.Errorf("Prepare(checksums 404) error = %v, want visible 404 error", err)
		}
	})

	t.Run("malformed release tag fails closed", func(t *testing.T) {
		ts := newReleaseServer(t, http.StatusOK, `{"tag_name":"not-a-tag"}`,
			newArchive(t), []byte("x"), nil)
		defer ts.Close()
		c, _ := newTestClient(t, ts)

		_, _, err := Prepare(c, Version{Tag: [3]int{0, 1, 0}}, linuxAmd64)
		if err == nil {
			t.Fatal("Prepare(malformed tag): want error")
		}
	})

	t.Run("archive without the binary path fails", func(t *testing.T) {
		archive := buildArchive(t, archiveEntry{name: "upp-linux-amd64/README", content: "x"})
		ts := newReleaseServer(t, http.StatusOK, `{"tag_name":"v0.1.1"}`,
			archive, []byte(checksumLine(t, archive, "upp-linux-amd64.tar.gz")), nil)
		defer ts.Close()
		c, _ := newTestClient(t, ts)

		_, _, err := Prepare(c, Version{Tag: [3]int{0, 1, 0}}, linuxAmd64)
		if err == nil || !strings.Contains(err.Error(), "upp-linux-amd64/upp") {
			t.Errorf("Prepare(no binary) error = %v, want error naming the missing binary path", err)
		}
	})
}

// TestUpdateSentinels locks the exported error contract the CLI (U5)
// switches on: every sentinel exists, is non-nil, and is distinct.
func TestUpdateSentinels(t *testing.T) {
	sentinels := []error{
		ErrDevelopmentBuild, ErrUpToDate, ErrUnsupportedPlatform,
		ErrChecksumMismatch, ErrNotWritable, ErrDeniedCI, ErrNotTTY,
	}
	for i, e := range sentinels {
		if e == nil {
			t.Fatalf("sentinel %d is nil", i)
		}
		for j, other := range sentinels {
			if i != j && e == other {
				t.Errorf("sentinel %d == sentinel %d (same error value)", i, j)
			}
		}
	}
}

// writeFile writes a fixture file, failing the test on error.
func writeFile(t *testing.T, path string, data []byte, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, data, mode); err != nil {
		t.Fatalf("write fixture %s: %v", path, err)
	}
}

// readFile reads a fixture file, failing the test on error.
func readFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return data
}

// assertBinary asserts path holds want bytes with mode 0755.
func assertBinary(t *testing.T, path string, want []byte) {
	t.Helper()
	if got := readFile(t, path); !bytes.Equal(got, want) {
		t.Errorf("%s bytes = %q, want %q", path, got, want)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if fi.Mode().Perm() != 0o755 {
		t.Errorf("%s mode = %v, want 0755", path, fi.Mode().Perm())
	}
}

// assertNoTempLeftovers asserts dir contains no .upp-* scratch files.
func assertNoTempLeftovers(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir %s: %v", dir, err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".upp-") {
			t.Errorf("leftover temp file %s in %s", e.Name(), dir)
		}
	}
}

// injectRename swaps the package-level rename hook for the duration of
// the test (rename-failure injection, design testing strategy).
func injectRename(t *testing.T, fn func(old, new string) error) {
	t.Helper()
	orig := rename
	rename = fn
	t.Cleanup(func() { rename = orig })
}

// skipIfRoot skips permission-based scenarios: chmod cannot block root.
func skipIfRoot(t *testing.T) {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Skip("permission-based test requires a non-root user")
	}
}

func TestReplace(t *testing.T) {
	oldBytes := []byte("old-binary-bytes")
	newBytes := []byte("brand-new-binary")

	// newFixture lays out dir/upp (old bytes) and dir/staged (new
	// bytes) and returns both paths.
	newFixture := func(t *testing.T) (dir, binPath, newPath string) {
		t.Helper()
		dir = t.TempDir()
		binPath = filepath.Join(dir, "upp")
		newPath = filepath.Join(dir, "staged")
		writeFile(t, binPath, oldBytes, 0o755)
		writeFile(t, newPath, newBytes, 0o755)
		return dir, binPath, newPath
	}

	backups := func(t *testing.T, binPath string) []string {
		t.Helper()
		b, err := filepath.Glob(binPath + ".backup.*")
		if err != nil {
			t.Fatalf("glob backups: %v", err)
		}
		return b
	}

	t.Run("replaces binary and keeps backup with old bytes", func(t *testing.T) {
		dir, binPath, newPath := newFixture(t)
		if err := Replace(binPath, newPath); err != nil {
			t.Fatalf("Replace: %v", err)
		}
		assertBinary(t, binPath, newBytes)
		b := backups(t, binPath)
		if len(b) != 1 {
			t.Fatalf("backup count = %d (%v), want exactly 1", len(b), b)
		}
		assertBinary(t, b[0], oldBytes)
		assertNoTempLeftovers(t, dir)
	})

	t.Run("resolves symlink and replaces the target", func(t *testing.T) {
		root := t.TempDir()
		realDir := filepath.Join(root, "real")
		linkDir := filepath.Join(root, "link")
		if err := os.MkdirAll(realDir, 0o755); err != nil {
			t.Fatalf("mkdir real dir: %v", err)
		}
		if err := os.MkdirAll(linkDir, 0o755); err != nil {
			t.Fatalf("mkdir link dir: %v", err)
		}
		target := filepath.Join(realDir, "upp")
		linkPath := filepath.Join(linkDir, "upp")
		writeFile(t, target, oldBytes, 0o755)
		if err := os.Symlink(target, linkPath); err != nil {
			t.Fatalf("symlink: %v", err)
		}
		newPath := filepath.Join(root, "staged")
		writeFile(t, newPath, newBytes, 0o755)

		if err := Replace(linkPath, newPath); err != nil {
			t.Fatalf("Replace via symlink: %v", err)
		}
		fi, err := os.Lstat(linkPath)
		if err != nil {
			t.Fatalf("lstat symlink: %v", err)
		}
		if fi.Mode()&os.ModeSymlink == 0 {
			t.Fatal("symlink was replaced by a regular file")
		}
		if got, err := os.Readlink(linkPath); err != nil || got != target {
			t.Errorf("symlink target = %q (err %v), want %q intact", got, err, target)
		}
		assertBinary(t, target, newBytes)
	})

	t.Run("unwritable directory fails with ErrNotWritable and changes nothing", func(t *testing.T) {
		skipIfRoot(t)
		dir, binPath, newPath := newFixture(t)
		if err := os.Chmod(dir, 0o555); err != nil {
			t.Fatalf("chmod dir read-only: %v", err)
		}
		t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

		err := Replace(binPath, newPath)
		if !errors.Is(err, ErrNotWritable) {
			t.Fatalf("Replace(unwritable) error = %v, want ErrNotWritable", err)
		}
		if !strings.Contains(err.Error(), dir) {
			t.Errorf("ErrNotWritable message %q does not name the target dir", err)
		}
		assertBinary(t, binPath, oldBytes)
		if b := backups(t, binPath); len(b) != 0 {
			t.Errorf("backup created on unwritable dir: %v", b)
		}
	})

	t.Run("final rename failure restores the backup", func(t *testing.T) {
		dir, binPath, newPath := newFixture(t)
		injectRename(t, func(old, new string) error {
			if new == binPath && !strings.HasPrefix(old, binPath+".backup.") {
				return errors.New("injected: final rename failed")
			}
			return os.Rename(old, new)
		})

		err := Replace(binPath, newPath)
		if err == nil {
			t.Fatal("Replace: want error on injected final-rename failure")
		}
		assertBinary(t, binPath, oldBytes)
		if b := backups(t, binPath); len(b) != 0 {
			t.Errorf("backup not restored: %v", b)
		}
		assertNoTempLeftovers(t, dir)
	})

	t.Run("backup rename failure leaves the binary untouched", func(t *testing.T) {
		dir, binPath, newPath := newFixture(t)
		injectRename(t, func(old, new string) error {
			if old == binPath {
				return errors.New("injected: backup rename failed")
			}
			return os.Rename(old, new)
		})

		err := Replace(binPath, newPath)
		if err == nil {
			t.Fatal("Replace: want error on injected backup-rename failure")
		}
		assertBinary(t, binPath, oldBytes)
		if b := backups(t, binPath); len(b) != 0 {
			t.Errorf("backup created despite backup-rename failure: %v", b)
		}
		assertNoTempLeftovers(t, dir)
	})

	t.Run("restore failure surfaces both errors", func(t *testing.T) {
		dir, binPath, newPath := newFixture(t)
		injectRename(t, func(old, new string) error {
			if new == binPath {
				return errors.New("injected: rename over binary failed")
			}
			return os.Rename(old, new)
		})

		err := Replace(binPath, newPath)
		if err == nil {
			t.Fatal("Replace: want error when final rename and restore both fail")
		}
		if !strings.Contains(err.Error(), "restore") {
			t.Errorf("error %q does not mention the failed restore", err)
		}
		if _, statErr := os.Stat(binPath); !os.IsNotExist(statErr) {
			t.Errorf("binary exists after failed restore, want it gone (renamed to backup, restore failed)")
		}
		assertNoTempLeftovers(t, dir)
	})

	t.Run("missing binary path fails", func(t *testing.T) {
		dir := t.TempDir()
		missing := filepath.Join(dir, "nope")
		newPath := filepath.Join(dir, "staged")
		writeFile(t, newPath, newBytes, 0o755)

		if err := Replace(missing, newPath); err == nil {
			t.Fatal("Replace(missing binary): want error")
		}
	})

	t.Run("missing staged binary fails without touching the binary", func(t *testing.T) {
		dir, binPath, _ := newFixture(t)
		missing := filepath.Join(dir, "missing-staged")

		err := Replace(binPath, missing)
		if err == nil {
			t.Fatal("Replace(missing staged): want error")
		}
		assertBinary(t, binPath, oldBytes)
		if b := backups(t, binPath); len(b) != 0 {
			t.Errorf("backup created despite staging failure: %v", b)
		}
		assertNoTempLeftovers(t, dir)
	})
}
