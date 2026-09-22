// Package platform handles OS/architecture detection.
package platform

import (
	"fmt"
	"runtime"
	"strings"
)

// OS constants for supported operating systems.
const (
	OSLinux   = "linux"
	OSMacOS   = "macos"
	OSWindows = "windows"
)

// Arch constants for supported CPU architectures.
const (
	ArchX86_64  = "x86_64"
	ArchAarch64 = "aarch64"
	ArchArm64   = "arm64"
)

// Platform represents the detected runtime environment.
type Platform struct {
	OS   string
	Arch string
}

// NormalizeOS converts a raw OS string (like runtime.GOOS or "darwin")
// to upp's canonical OS identifier. It is case-insensitive and trims whitespace.
// Returns an error if the OS is unsupported.
func NormalizeOS(rawOS string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(rawOS)) {
	case "linux":
		return OSLinux, nil
	case "darwin", "macos":
		return OSMacOS, nil
	case "windows":
		return OSWindows, nil
	default:
		return "", fmt.Errorf("unsupported platform: %s — upp supports Linux, macOS, and Windows only", rawOS)
	}
}

// NormalizeArch converts a raw architecture string (like runtime.GOARCH or "x86_64")
// to upp's canonical architecture identifier. It is case-insensitive and trims whitespace.
func NormalizeArch(rawArch string) string {
	switch strings.ToLower(strings.TrimSpace(rawArch)) {
	case "amd64", "x86_64":
		return ArchX86_64
	case "arm64", "aarch64":
		return ArchArm64
	default:
		return rawArch
	}
}

// Detect returns the current platform by mapping runtime.GOOS and runtime.GOARCH
// to upp's canonical OS and architecture identifiers.
// Returns an error if the platform is unsupported.
func Detect() (Platform, error) {
	os, err := NormalizeOS(runtime.GOOS)
	if err != nil {
		return Platform{}, err
	}
	return Platform{
		OS:   os,
		Arch: NormalizeArch(runtime.GOARCH),
	}, nil
}

// MustDetect returns the current platform or panics if unsupported.
// Use only in main/entry points where failure is fatal.
func MustDetect() Platform {
	p, err := Detect()
	if err != nil {
		panic(err)
	}
	return p
}
