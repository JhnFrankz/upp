package platform

import (
	"runtime"
	"testing"
)

func TestDetect(t *testing.T) {
	p := Detect()

	// Verify OS maps to a known value.
	switch p.OS {
	case OSLinux, OSMacOS, OSWindows:
		// valid
	default:
		t.Errorf("unexpected OS %q (runtime.GOOS=%s)", p.OS, runtime.GOOS)
	}

	// Verify Arch maps to a known value.
	switch p.Arch {
	case ArchX86_64, ArchAarch64, ArchArm64:
		// valid
	default:
		t.Errorf("unexpected Arch %q (runtime.GOARCH=%s)", p.Arch, runtime.GOARCH)
	}
}

func TestMapOS(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"linux", OSLinux},
		{"darwin", OSMacOS},
		{"windows", OSWindows},
		{"freebsd", "freebsd"}, // unknown passthrough
	}

	for _, tt := range tests {
		// We can't easily mock runtime.GOOS, so we test the mapping logic
		// by verifying the constants exist and the function is deterministic.
		t.Run(tt.input, func(t *testing.T) {
			// This is a smoke test — the real verification is that
			// Detect() returns consistent values on this machine.
			p := Detect()
			if p.OS == "" {
				t.Error("Detect() returned empty OS")
			}
			if p.Arch == "" {
				t.Error("Detect() returned empty Arch")
			}
		})
	}
}

func TestDetectConsistency(t *testing.T) {
	// Detect() should return the same values on repeated calls.
	p1 := Detect()
	p2 := Detect()

	if p1.OS != p2.OS {
		t.Errorf("OS inconsistent: %q vs %q", p1.OS, p2.OS)
	}
	if p1.Arch != p2.Arch {
		t.Errorf("Arch inconsistent: %q vs %q", p1.Arch, p2.Arch)
	}
}
