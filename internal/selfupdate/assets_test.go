package selfupdate

import (
	"testing"

	"github.com/JhnFrankz/upp/internal/platform"
)

func TestAssetName(t *testing.T) {
	tests := []struct {
		name    string
		p       platform.Platform
		want    string
		wantErr bool
	}{
		// Every real detect() combination over the Makefile PLATFORMS
		// set (linux/darwin × amd64/arm64, canonical platform names).
		{"linux amd64", platform.Platform{OS: platform.OSLinux, Arch: platform.ArchX86_64}, "upp-linux-amd64.tar.gz", false},
		{"linux arm64", platform.Platform{OS: platform.OSLinux, Arch: platform.ArchArm64}, "upp-linux-arm64.tar.gz", false},
		{"linux aarch64", platform.Platform{OS: platform.OSLinux, Arch: platform.ArchAarch64}, "upp-linux-arm64.tar.gz", false},
		{"macos amd64", platform.Platform{OS: platform.OSMacOS, Arch: platform.ArchX86_64}, "upp-darwin-amd64.tar.gz", false},
		{"macos arm64", platform.Platform{OS: platform.OSMacOS, Arch: platform.ArchArm64}, "upp-darwin-arm64.tar.gz", false},
		// Identity entries accept release-style names unchanged.
		{"identity release names", platform.Platform{OS: "darwin", Arch: "amd64"}, "upp-darwin-amd64.tar.gz", false},
		// Unknown OS/arch must fail closed.
		{"unknown os", platform.Platform{OS: "freebsd", Arch: platform.ArchX86_64}, "", true},
		{"unknown arch", platform.Platform{OS: platform.OSLinux, Arch: "mips"}, "", true},
		{"windows fails closed", platform.Platform{OS: platform.OSWindows, Arch: platform.ArchX86_64}, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AssetName(tt.p)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("AssetName(%+v) = %q, want error", tt.p, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("AssetName(%+v) unexpected error: %v", tt.p, err)
			}
			if got != tt.want {
				t.Errorf("AssetName(%+v) = %q, want %q", tt.p, got, tt.want)
			}
		})
	}
}
