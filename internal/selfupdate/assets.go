package selfupdate

import (
	"fmt"

	"github.com/JhnFrankz/upp/internal/platform"
)

// releaseOS maps platform.Detect() canonical OS names to the OS names
// used in published release assets (see Makefile release target:
// upp-{os}-{arch}.tar.gz / upp-{os}-{arch}.zip).
var releaseOS = map[string]string{
	"macos":   "darwin",
	"darwin":  "darwin", // identity: already a release name
	"linux":   "linux",  // identity
	"windows": "windows",
}

// releaseArch maps platform.Detect() canonical architecture names to
// release asset architecture names.
var releaseArch = map[string]string{
	"x86_64":  "amd64",
	"aarch64": "arm64",
	"amd64":   "amd64", // identity: already a release name
	"arm64":   "arm64", // identity
}

// AssetName maps a detected platform to its release asset name,
// "upp-{os}-{arch}.zip" on Windows, "upp-{os}-{arch}.tar.gz" on other OSes,
// per design D5. Unknown OS or architecture fails closed with a clear error.
func AssetName(p platform.Platform) (string, error) {
	osName, ok := releaseOS[p.OS]
	if !ok {
		return "", fmt.Errorf("selfupdate: unsupported OS %q (supported: linux, macos, windows)", p.OS)
	}
	archName, ok := releaseArch[p.Arch]
	if !ok {
		return "", fmt.Errorf("selfupdate: unsupported architecture %q (supported: amd64, arm64)", p.Arch)
	}
	if osName == "windows" {
		return fmt.Sprintf("upp-%s-%s.zip", osName, archName), nil
	}
	return fmt.Sprintf("upp-%s-%s.tar.gz", osName, archName), nil
}
