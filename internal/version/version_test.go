package version_test

import (
	"testing"

	"github.com/JhnFrankz/upp/internal/version"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    version.Version
		wantErr bool
	}{
		// Spec R1 shapes from git describe output.
		{"clean tag", "v0.1.0", version.Version{Tag: [3]int{0, 1, 0}}, false},
		{"untagged build", "v0.1.0-19-gd40e428", version.Version{Tag: [3]int{0, 1, 0}}, false},
		{"untagged dirty build", "v0.1.0-19-gd40e428-dirty", version.Version{Tag: [3]int{0, 1, 0}, Dirty: true}, false},
		{"clean tag dirty", "v0.1.0-dirty", version.Version{Tag: [3]int{0, 1, 0}, Dirty: true}, false},
		{"dev build", "dev", version.Version{Dev: true}, false},
		// Unparseable inputs must fail closed.
		{"empty", "", version.Version{}, true},
		{"missing v prefix", "1.2.3", version.Version{}, true},
		{"two part tag", "v1.2", version.Version{}, true},
		{"four part tag", "v1.2.3.4", version.Version{}, true},
		{"non-numeric tag", "v1.2.x", version.Version{}, true},
		{"negative tag component", "v-1.2.3", version.Version{}, true},
		{"missing commit count", "v0.1.0-gd40e428", version.Version{}, true},
		{"missing commit hash", "v0.1.0-19-g", version.Version{}, true},
		{"non-hex commit hash", "v0.1.0-19-gd40e42z", version.Version{}, true},
		{"junk", "not-a-version", version.Version{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := version.Parse(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Parse(%q) = %+v, want error", tt.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%q) unexpected error: %v", tt.in, err)
			}
			if got != tt.want {
				t.Errorf("Parse(%q) = %+v, want %+v", tt.in, got, tt.want)
			}
		})
	}
}

func TestCompare(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want int
	}{
		{"equal clean tags", "v0.1.0", "v0.1.0", 0},
		{"patch bump", "v0.1.0", "v0.1.1", -1},
		{"patch bump reverse", "v0.1.1", "v0.1.0", 1},
		{"minor bump", "v0.1.9", "v0.2.0", -1},
		{"major bump", "v1.2.3", "v2.0.0", -1},
		{"numeric not lexical patch", "v0.1.9", "v0.1.10", -1},
		{"numeric not lexical minor", "v0.9.0", "v0.10.0", -1},
		{"untagged compares tag prefix", "v0.1.0-19-gd40e428", "v0.1.1", -1},
		{"untagged equal to tag", "v0.1.0-19-gd40e428", "v0.1.0", 0},
		{"dirty compares tag prefix", "v0.1.0-19-gd40e428-dirty", "v0.1.1", -1},
		{"dirty equal to tag", "v0.1.0-dirty", "v0.1.0", 0},
		{"dev is below any release", "dev", "v0.1.0", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, err := version.Parse(tt.a)
			if err != nil {
				t.Fatalf("Parse(%q): %v", tt.a, err)
			}
			b, err := version.Parse(tt.b)
			if err != nil {
				t.Fatalf("Parse(%q): %v", tt.b, err)
			}
			if got := a.Compare(b); got != tt.want {
				t.Errorf("%q.Compare(%q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestExtractVersionFromString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty string", "", ""},
		{"no version", "command not found", ""},
		{"semver", "1.2.3", "1.2.3"},
		{"v-prefix", "v1.2.3", "v1.2.3"},
		{"go version", "go version go1.22.0 linux/amd64", "go1.22.0"},
		{"node version", "v20.11.0", "v20.11.0"},
		{"brew version", "Homebrew 4.1.0", "4.1.0"},
		{"pnpm version", "pnpm: 8.14.0", "8.14.0"},
		{"multiline", "first line\n1.2.3\nthird line", "1.2.3"},
		{"pre-release", "1.2.3-rc1", "1.2.3-rc1"},
		{"build metadata", "1.2.3+build.123", "1.2.3+build.123"},
		{"four-part", "1.2.3.4", "1.2.3.4"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := version.ExtractVersionFromString(tt.input)
			if got != tt.want {
				t.Errorf("ExtractVersionFromString(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExtractVersionFromOutput(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"finds version in output", "tool version 2.5.1 (stable)", "2.5.1"},
		{"falls back to raw string when no version", "unrecognized text", "unrecognized text"},
		{"empty string fallback", "", ""},
		{"multiline with version", "Header info\nRelease 3.0.0-beta\nFooter", "3.0.0-beta"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := version.ExtractVersionFromOutput(tt.input)
			if got != tt.want {
				t.Errorf("ExtractVersionFromOutput(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsVersionLike(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"", false},
		{"1", false},
		{"1.2", true},
		{"1.2.3", true},
		{"1.2.3.4", true},
		{"v1.2.3", true},
		{"go1.22.0", true},
		{"abc", false},
		{"abc.def", false},
		{"1.2.3-rc1", true},
		{"1.2.3-beta.1", true},
		{"1.2.3+build", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := version.IsVersionLike(tt.input)
			if got != tt.want {
				t.Errorf("IsVersionLike(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
