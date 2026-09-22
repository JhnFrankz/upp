package platform

import (
	"fmt"
	"runtime"
	"testing"
)

func TestNormalizeOS(t *testing.T) {
	tests := []struct {
		name    string
		rawOS   string
		want    string
		wantErr bool
	}{
		{name: "linux lowercase", rawOS: "linux", want: OSLinux, wantErr: false},
		{name: "darwin lowercase", rawOS: "darwin", want: OSMacOS, wantErr: false},
		{name: "macos lowercase", rawOS: "macos", want: OSMacOS, wantErr: false},
		{name: "windows lowercase", rawOS: "windows", want: OSWindows, wantErr: false},
		{name: "linux uppercase", rawOS: "LINUX", want: OSLinux, wantErr: false},
		{name: "darwin titlecase", rawOS: "Darwin", want: OSMacOS, wantErr: false},
		{name: "windows with whitespace", rawOS: "  windows  ", want: OSWindows, wantErr: false},
		{name: "freebsd unsupported", rawOS: "freebsd", want: "", wantErr: true},
		{name: "openbsd unsupported", rawOS: "openbsd", want: "", wantErr: true},
		{name: "empty unsupported", rawOS: "", want: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeOS(tt.rawOS)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NormalizeOS(%q) error = %v, wantErr %v", tt.rawOS, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("NormalizeOS(%q) = %q, want %q", tt.rawOS, got, tt.want)
			}
			if tt.wantErr && err != nil {
				expectedErr := fmt.Sprintf("unsupported platform: %s — upp supports Linux, macOS, and Windows only", tt.rawOS)
				if err.Error() != expectedErr {
					t.Errorf("NormalizeOS(%q) error = %q, want %q", tt.rawOS, err.Error(), expectedErr)
				}
			}
		})
	}
}

func TestNormalizeArch(t *testing.T) {
	tests := []struct {
		name    string
		rawArch string
		want    string
	}{
		{name: "amd64", rawArch: "amd64", want: ArchX86_64},
		{name: "x86_64", rawArch: "x86_64", want: ArchX86_64},
		{name: "arm64", rawArch: "arm64", want: ArchArm64},
		{name: "aarch64", rawArch: "aarch64", want: ArchArm64},
		{name: "386", rawArch: "386", want: "386"},
		{name: "case insensitive", rawArch: "AMD64", want: ArchX86_64},
		{name: "whitespace trimmed", rawArch: "  arm64  ", want: ArchArm64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeArch(tt.rawArch)
			if got != tt.want {
				t.Errorf("NormalizeArch(%q) = %q, want %q", tt.rawArch, got, tt.want)
			}
		})
	}
}

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

func TestDetectConsistency(t *testing.T) {
	p1, err1 := Detect()
	p2, err2 := Detect()

	if err1 != err2 {
		t.Errorf("Detect() error inconsistent: %v vs %v", err1, err2)
	}
	if err1 != nil {
		t.Fatalf("Detect() unexpected error: %v", err1)
	}
	if p1.OS != p2.OS {
		t.Errorf("OS inconsistent: %q vs %q", p1.OS, p2.OS)
	}
	if p1.Arch != p2.Arch {
		t.Errorf("Arch inconsistent: %q vs %q", p1.Arch, p2.Arch)
	}
}

func TestMustDetectPanics(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MustDetect() panicked on supported platform: %v", r)
		}
	}()
	_ = MustDetect()
}
