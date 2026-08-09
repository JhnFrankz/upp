// Package platform handles OS/architecture detection and tool catalog management.
package platform

import (
	"fmt"
	"runtime"
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

// Detect returns the current platform by mapping runtime.GOOS and runtime.GOARCH
// to upp's canonical OS and architecture identifiers.
// Returns an error if the platform is unsupported.
func Detect() (Platform, error) {
	os, err := mapOS()
	if err != nil {
		return Platform{}, err
	}
	return Platform{
		OS:   os,
		Arch: mapArch(),
	}, nil
}

// mapOS converts runtime.GOOS to a canonical OS name.
func mapOS() (string, error) {
	switch runtime.GOOS {
	case "linux":
		return OSLinux, nil
	case "darwin":
		return OSMacOS, nil
	case "windows":
		return OSWindows, nil
	default:
		return "", fmt.Errorf("unsupported platform: %s/%s — upp supports Linux, macOS, and Windows only", runtime.GOOS, runtime.GOARCH)
	}
}

// mapArch converts runtime.GOARCH to a canonical architecture name.
func mapArch() string {
	switch runtime.GOARCH {
	case "amd64":
		return ArchX86_64
	case "arm64":
		return ArchArm64 // also covers Apple Silicon
	case "arm":
		return ArchAarch64
	default:
		return runtime.GOARCH // pass through unknown values
	}
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
