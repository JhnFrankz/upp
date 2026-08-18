package selfupdate

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/JhnFrankz/upp/internal/platform"
)

// Sentinel errors for the self-update pipeline (design Interfaces /
// Contracts). The CLI (U5) switches on these with errors.Is to produce
// localized messages and exit codes.
var (
	// ErrDevelopmentBuild marks a dev or -dirty build: no update claim,
	// no network (design D2).
	ErrDevelopmentBuild = errors.New("selfupdate: development build; self-update requires a release build")
	// ErrUpToDate marks the current version at or above the latest
	// release: nothing to download (spec R1 "already up to date").
	ErrUpToDate = errors.New("selfupdate: already up to date")
	// ErrUnsupportedPlatform marks an OS/arch with no release asset
	// (design D5 fails closed).
	ErrUnsupportedPlatform = errors.New("selfupdate: platform is not supported for self-update")
	// ErrChecksumMismatch marks a sha256 mismatch or a missing checksum
	// entry: abort with the current binary untouched (spec R4, security
	// model — stricter than install.sh's warn-and-skip).
	ErrChecksumMismatch = errors.New("selfupdate: checksum mismatch; current binary left untouched")
	// ErrNotWritable marks a binary directory the preflight cannot write
	// to: actionable error, never sudo (spec R6).
	ErrNotWritable = errors.New("selfupdate: binary directory is not writable")
	// ErrDeniedCI marks a self-update attempt under --ci (U5 prompt
	// gate): never auto-proceed.
	ErrDeniedCI = errors.New("selfupdate: self-update denied in --ci mode")
	// ErrNotTTY marks a self-update attempt with non-TTY stdin (U5
	// prompt gate): never hang, never silently skip.
	ErrNotTTY = errors.New("selfupdate: self-update requires an interactive terminal")
)

// verifyChecksum verifies asset against the checksums.txt bytes fetched
// from the SAME release (spec R4 + security-model delta). The file uses
// the sha256sum(1) format produced by `make release`: "<hex>  <name>",
// or "<hex> *<name>" in binary mode. The entry for name must exist and
// equal sha256(asset); a missing, malformed, or mismatched entry fails
// closed with ErrChecksumMismatch — the archive is never extracted.
// Lines that cannot name a file are ignored: they cannot weaken the
// check, because the asset's own entry must still parse and match.
func verifyChecksum(asset, checksums []byte, name string) error {
	want := fmt.Sprintf("%x", sha256.Sum256(asset))
	for _, line := range strings.Split(string(checksums), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		sum, entry := fields[0], strings.TrimPrefix(fields[1], "*")
		if entry != name {
			continue
		}
		if len(sum) != sha256.Size*2 {
			return fmt.Errorf("%w: malformed checksum entry for %s", ErrChecksumMismatch, name)
		}
		if !strings.EqualFold(sum, want) {
			return fmt.Errorf("%w: %s: checksum %s, want %s", ErrChecksumMismatch, name, sum, want)
		}
		return nil
	}
	return fmt.Errorf("%w: no checksum entry for %s", ErrChecksumMismatch, name)
}

// binarySuffix is the archive path of the upp binary inside a release
// asset: the Makefile release target stages assets as
// upp-{os}-{arch}/upp before tarring, so the entry is
// "upp-{os}-{arch}/upp".
const binarySuffix = "/upp"

// extract writes ONLY the known binary entry (upp-{os}-{arch}/upp,
// derived from assetName) from a tar.gz archive into destDir as "upp",
// mode 0755, and returns its path. Every other entry is ignored (spec
// R5: extra paths are not written), but dangerous entries — absolute
// paths, path traversal (".." components), symlinks, hardlinks — abort
// the whole extraction. The downloaded bytes are read only by
// archive/tar; nothing is ever executed.
func extract(asset []byte, assetName, destDir string) (string, error) {
	dir := strings.TrimSuffix(assetName, ".tar.gz")
	if dir == assetName {
		return "", fmt.Errorf("selfupdate: %s is not a .tar.gz asset name", assetName)
	}
	binaryPath := dir + binarySuffix

	gz, err := gzip.NewReader(bytes.NewReader(asset))
	if err != nil {
		return "", fmt.Errorf("selfupdate: release archive is not gzip: %w", err)
	}
	defer func() { _ = gz.Close() }()

	out := filepath.Join(destDir, "upp")
	found := false
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("selfupdate: reading release archive: %w", err)
		}
		// Every entry is checked — a dangerous entry anywhere in the
		// archive aborts the extraction, even one that appears after
		// the binary (fail closed).
		if err := checkEntry(hdr); err != nil {
			return "", err
		}
		if hdr.Name != binaryPath {
			continue
		}
		if hdr.Typeflag != tar.TypeReg {
			return "", fmt.Errorf("selfupdate: archive entry %s is not a regular file", binaryPath)
		}
		if !found {
			if err := writeBinary(out, tr); err != nil {
				return "", err
			}
			found = true
		}
	}
	if !found {
		return "", fmt.Errorf("selfupdate: release archive does not contain %s", binaryPath)
	}
	return out, nil
}

// checkEntry rejects archive entries that could escape destDir or
// redirect writes: absolute paths, path-traversal names, and link
// entries. Everything else passes; only the known binary path is ever
// written, so non-binary regular entries are harmless.
func checkEntry(hdr *tar.Header) error {
	name := hdr.Name
	if name == "" || strings.HasPrefix(name, "/") || filepath.IsAbs(name) {
		return fmt.Errorf("selfupdate: release archive contains absolute path %q; refusing", name)
	}
	for _, comp := range strings.Split(name, "/") {
		if comp == ".." {
			return fmt.Errorf("selfupdate: release archive contains path traversal entry %q; refusing", name)
		}
	}
	if hdr.Typeflag == tar.TypeSymlink || hdr.Typeflag == tar.TypeLink {
		return fmt.Errorf("selfupdate: release archive contains link entry %q; refusing", name)
	}
	return nil
}

// writeBinary streams an archive entry into out with mode 0755.
func writeBinary(out string, r io.Reader) error {
	f, err := os.OpenFile(out, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return fmt.Errorf("selfupdate: cannot write extracted binary: %w", err)
	}
	if _, err := io.Copy(f, r); err != nil {
		_ = f.Close()
		return fmt.Errorf("selfupdate: cannot write extracted binary: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("selfupdate: cannot write extracted binary: %w", err)
	}
	return nil
}

// Prepare runs the first half of the self-update pipeline (design D6):
// gate development builds, map the platform to its release asset,
// resolve the latest release, compare versions, download the asset and
// checksums.txt from that SAME release, verify the sha256, and extract
// only the known binary to a fresh temp dir outside the install path.
// It returns the resolved release and the absolute path of the
// extracted binary. On any failure nothing outside the temp dir is
// modified; the temp dir is removed on extract errors.
//
// Sentinel errors: ErrDevelopmentBuild (dev/dirty — no network),
// ErrUpToDate (current >= latest — no download), ErrUnsupportedPlatform
// (no asset for this platform — no network), ErrChecksumMismatch
// (mismatch or missing entry — binary untouched).
func Prepare(c *Client, current Version, p platform.Platform) (Release, string, error) {
	if current.Dev || current.Dirty {
		return Release{}, "", ErrDevelopmentBuild
	}
	assetName, err := AssetName(p)
	if err != nil {
		return Release{}, "", fmt.Errorf("%w: %v", ErrUnsupportedPlatform, err)
	}
	rel, err := c.LatestFresh()
	if err != nil {
		return Release{}, "", err
	}
	latest, err := Parse(rel.Tag)
	if err != nil {
		return Release{}, "", fmt.Errorf("selfupdate: cannot parse latest release tag %q: %w", rel.Tag, err)
	}
	if current.Compare(latest) >= 0 {
		return Release{}, "", ErrUpToDate
	}
	asset, checksums, err := c.Download(assetName)
	if err != nil {
		return Release{}, "", err
	}
	if err := verifyChecksum(asset, checksums, assetName); err != nil {
		return Release{}, "", err
	}
	tmpDir, err := os.MkdirTemp("", "upp-selfupdate-*")
	if err != nil {
		return Release{}, "", fmt.Errorf("selfupdate: cannot create temp dir: %w", err)
	}
	binPath, err := extract(asset, assetName, tmpDir)
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return Release{}, "", err
	}
	return rel, binPath, nil
}

// rename is the atomic replace primitive (os.Rename). It is a package
// var so tests can inject failures at exactly the backup or the final
// rename and assert the backup is restored (design testing strategy).
var rename = os.Rename

// Replace atomically swaps the running binary at execPath for the
// verified binary at newPath (design D7): resolve symlinks, preflight
// target-dir writability (never sudo), stage a temp copy with mode
// 0755 in the target dir, back up the current binary to
// {binary}.backup.<ts>, rename the temp file over the binary, and on
// ANY failure restore the backup and return a non-zero error. On
// success the backup remains in place. An unwritable target directory
// yields ErrNotWritable with an actionable message before anything is
// modified.
func Replace(execPath, newPath string) error {
	resolved, err := filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("selfupdate: cannot resolve binary path %s: %w", execPath, err)
	}
	dir := filepath.Dir(resolved)

	tmp, err := os.CreateTemp(dir, ".upp-*")
	if err != nil {
		return fmt.Errorf("%w: %s: %v (make %s writable or install upp under your home, e.g. ~/.local/bin; upp never uses sudo)",
			ErrNotWritable, dir, err, dir)
	}
	tmpPath := tmp.Name()
	removeTmp := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}

	if err := stageBinary(tmp, newPath); err != nil {
		removeTmp()
		return err
	}
	if err := tmp.Chmod(0o755); err != nil {
		removeTmp()
		return fmt.Errorf("selfupdate: cannot set mode on staged binary: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("selfupdate: cannot close staged binary: %w", err)
	}

	backup := fmt.Sprintf("%s.backup.%s", resolved, time.Now().Format("20060102.150405"))
	if err := rename(resolved, backup); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("selfupdate: cannot back up current binary: %w", err)
	}
	if err := rename(tmpPath, resolved); err != nil {
		_ = os.Remove(tmpPath)
		if rerr := rename(backup, resolved); rerr != nil {
			return fmt.Errorf("selfupdate: replacing binary failed (%v) and restore of backup failed: %v", err, rerr)
		}
		return fmt.Errorf("selfupdate: replacing binary failed; backup restored: %w", err)
	}
	return nil
}

// stageBinary copies the verified binary bytes from newPath into tmp.
func stageBinary(tmp *os.File, newPath string) error {
	f, err := os.Open(newPath)
	if err != nil {
		return fmt.Errorf("selfupdate: cannot open staged binary %s: %w", newPath, err)
	}
	defer func() { _ = f.Close() }()
	if _, err := io.Copy(tmp, f); err != nil {
		return fmt.Errorf("selfupdate: cannot stage binary: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("selfupdate: cannot sync staged binary: %w", err)
	}
	return nil
}
