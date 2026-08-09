package platform

import (
	"runtime"
	"testing"
)

func TestDetect(t *testing.T) {
	p, err := Detect()
	if err != nil {
		t.Fatalf("Detect() returned error: %v", err)
	}

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
		wantErr  bool
	}{
		{"linux", OSLinux, false},
		{"darwin", OSMacOS, false},
		{"windows", OSWindows, false},
		{"freebsd", "", true}, // unknown → error
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := mapOS()
			// We can't mock runtime.GOOS, so we test the current platform
			if runtime.GOOS == tt.input {
				if tt.wantErr {
					t.Skip("cannot test error case on supported platform")
				}
				if err != nil {
					t.Errorf("mapOS() unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("mapOS() = %q, want %q", result, tt.expected)
				}
			}
		})
	}
}

func TestDetectConsistency(t *testing.T) {
	// Detect() should return the same values on repeated calls.
	p1, err1 := Detect()
	p2, err2 := Detect()

	if err1 != err2 {
		t.Errorf("Detect() error inconsistent: %v vs %v", err1, err2)
	}
	if err1 != nil {
		t.Skipf("unsupported platform: %v", err1)
	}
	if p1.OS != p2.OS {
		t.Errorf("OS inconsistent: %q vs %q", p1.OS, p2.OS)
	}
	if p1.Arch != p2.Arch {
		t.Errorf("Arch inconsistent: %q vs %q", p1.Arch, p2.Arch)
	}
}

func TestMustDetectPanics(t *testing.T) {
	// On supported platforms, MustDetect should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MustDetect() panicked on supported platform: %v", r)
		}
	}()
	_ = MustDetect()
}
