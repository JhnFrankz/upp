package official

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/archive"
	"github.com/JhnFrankz/upp/internal/bufferpool"
)

// GoRelease represents the metadata of an official Go release archive.
type GoRelease struct {
	Version  string
	Filename string
	SHA256   string
	URL      string
}

type goDevFile struct {
	Filename string `json:"filename"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Version  string `json:"version"`
	SHA256   string `json:"sha256"`
	Size     int64  `json:"size"`
	Kind     string `json:"kind"`
}

type goDevRelease struct {
	Version string      `json:"version"`
	Stable  bool        `json:"stable"`
	Files   []goDevFile `json:"files"`
}

// defaultGoCheckClient is the shared HTTP client used for checking Go versions and metadata.
var defaultGoCheckClient = &http.Client{
	Timeout: adapters.CheckTimeout,
}

// defaultGoDownloadClient is the shared HTTP client used for downloading Go release archives.
var defaultGoDownloadClient = &http.Client{
	Timeout: adapters.UpdateTimeout,
}

// goReleaseURL is the endpoint for Go release metadata.
var goReleaseURL = "https://go.dev/dl/?mode=json"

// goDevVersionURL is the endpoint for Go latest version text.
var goDevVersionURL = "https://go.dev/VERSION?m=text"

// goReleaseFn is the seam for fetching Go release metadata.
// Swapped in tests via setExecFakes.
var goReleaseFn = fetchGoRelease

// goDownloadAndVerifyFn is the seam for downloading and verifying a Go archive.
// Swapped in tests via setExecFakes.
var goDownloadAndVerifyFn = downloadAndVerifyGo

// goExtractTarballFn is the seam for extracting a Go archive.
// Swapped in tests via setExecFakes.
var goExtractTarballFn = extractGoTarball

// goTargetExistsFn is the seam for checking if /usr/local/go exists before backing up.
// Swapped in tests via setExecFakes.
var goTargetExistsFn = func() bool {
	_, err := os.Stat("/usr/local/go")
	return err == nil
}

// goBinaryPathFn is the seam for locating the go binary on Linux.
// Swapped in tests via setExecFakes.
var goBinaryPathFn = func() string {
	p, err := exec.LookPath("go")
	if err != nil {
		return ""
	}
	return p
}

// goDevVersionFn is the seam for querying the latest version from go.dev.
// Swapped in tests via setExecFakes.
var goDevVersionFn = fetchGoDevVersion

func fetchGoDevVersion(ctx context.Context) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, goDevVersionURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "upp")
	resp, err := defaultGoCheckClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64*1024))
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d from go.dev", resp.StatusCode)
	}
	buf := make([]byte, 512)
	n, _ := resp.Body.Read(buf)
	firstLine, _, _ := strings.Cut(string(buf[:n]), "\n")
	line := strings.TrimSpace(firstLine)
	return line, nil
}

func fetchGoRelease(ctx context.Context, goos, goarch string) (GoRelease, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, goReleaseURL, nil)
	if err != nil {
		return GoRelease{}, err
	}
	req.Header.Set("User-Agent", "upp")
	resp, err := defaultGoCheckClient.Do(req)
	if err != nil {
		return GoRelease{}, err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64*1024))
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return GoRelease{}, fmt.Errorf("unexpected status %d from go.dev", resp.StatusCode)
	}

	var releases []goDevRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return GoRelease{}, fmt.Errorf("failed to decode go releases: %w", err)
	}

	for _, r := range releases {
		if !r.Stable {
			continue
		}
		for _, f := range r.Files {
			if f.OS == goos && f.Arch == goarch && f.Kind == "archive" {
				return GoRelease{
					Version:  r.Version,
					Filename: f.Filename,
					SHA256:   f.SHA256,
					URL:      "https://go.dev/dl/" + f.Filename,
				}, nil
			}
		}
	}

	return GoRelease{}, fmt.Errorf("no stable Go release found for %s/%s", goos, goarch)
}

func downloadAndVerifyGo(ctx context.Context, rel GoRelease, destPath string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rel.URL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "upp")
	resp, err := defaultGoDownloadClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s failed: HTTP %d", rel.URL, resp.StatusCode)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return err
	}

	hasher := sha256.New()
	mw := io.MultiWriter(f, hasher)

	bufPtr := bufferpool.Get()
	defer bufferpool.Put(bufPtr)

	_, copyErr := io.CopyBuffer(mw, resp.Body, *bufPtr)
	closeErr := f.Close()

	if copyErr != nil {
		_ = os.Remove(destPath)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(destPath)
		return closeErr
	}

	actualHash := fmt.Sprintf("%x", hasher.Sum(nil))
	if !strings.EqualFold(actualHash, rel.SHA256) {
		_ = os.Remove(destPath)
		return fmt.Errorf("checksum mismatch for %s: got %s, want %s", rel.Filename, actualHash, rel.SHA256)
	}

	return nil
}

func extractGoTarball(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer func() { _ = gz.Close() }()

	tr := tar.NewReader(gz)
	cleanDest := filepath.Clean(destDir)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar entry: %w", err)
		}

		targetPath, err := archive.SafeTargetPath(cleanDest, hdr.Name)
		if err != nil {
			if errors.Is(err, archive.ErrPathEscapes) {
				return fmt.Errorf("archive entry escapes destination: %s", hdr.Name)
			}
			return fmt.Errorf("illegal archive entry path: %s", hdr.Name)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			mode := hdr.FileInfo().Mode().Perm()
			if mode == 0 {
				mode = 0644
			}
			if err := archive.ExtractToFile(targetPath, tr, mode); err != nil {
				return err
			}
		case tar.TypeSymlink:
			if err := archive.ValidateSymlinkTarget(hdr.Linkname); err != nil {
				return fmt.Errorf("unsupported or unsafe symlink target: %s", hdr.Linkname)
			}
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return err
			}
			_ = os.Remove(targetPath)
			if err := os.Symlink(hdr.Linkname, targetPath); err != nil {
				return err
			}
		default:
			// Ignore other header types
		}
	}
	return nil
}
