package official

import "testing"

// TestGoTarballURL verifies the go adapter's Linux download URL matches the
// running process architecture instead of hardcoding amd64.
func TestGoTarballURL(t *testing.T) {
	tests := []struct {
		name   string
		goarch string
		want   string
	}{
		{
			name:   "amd64",
			goarch: "amd64",
			want:   "https://go.dev/dl/$(curl -fsSL https://go.dev/VERSION?m=text | head -1).linux-amd64.tar.gz",
		},
		{
			name:   "arm64",
			goarch: "arm64",
			want:   "https://go.dev/dl/$(curl -fsSL https://go.dev/VERSION?m=text | head -1).linux-arm64.tar.gz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := goTarballURL(tt.goarch); got != tt.want {
				t.Errorf("goTarballURL(%q) = %q, want %q", tt.goarch, got, tt.want)
			}
		})
	}
}
