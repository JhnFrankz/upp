package selfupdate

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    Version
		wantErr bool
	}{
		// Spec R1 shapes from git describe output.
		{"clean tag", "v0.1.0", Version{Tag: [3]int{0, 1, 0}}, false},
		{"untagged build", "v0.1.0-19-gd40e428", Version{Tag: [3]int{0, 1, 0}}, false},
		{"untagged dirty build", "v0.1.0-19-gd40e428-dirty", Version{Tag: [3]int{0, 1, 0}, Dirty: true}, false},
		{"clean tag dirty", "v0.1.0-dirty", Version{Tag: [3]int{0, 1, 0}, Dirty: true}, false},
		{"dev build", "dev", Version{Dev: true}, false},
		// Unparseable inputs must fail closed.
		{"empty", "", Version{}, true},
		{"missing v prefix", "1.2.3", Version{}, true},
		{"two part tag", "v1.2", Version{}, true},
		{"four part tag", "v1.2.3.4", Version{}, true},
		{"non-numeric tag", "v1.2.x", Version{}, true},
		{"negative tag component", "v-1.2.3", Version{}, true},
		{"missing commit count", "v0.1.0-gd40e428", Version{}, true},
		{"missing commit hash", "v0.1.0-19-g", Version{}, true},
		{"non-hex commit hash", "v0.1.0-19-gd40e42z", Version{}, true},
		{"junk", "not-a-version", Version{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.in)
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
			a, err := Parse(tt.a)
			if err != nil {
				t.Fatalf("Parse(%q): %v", tt.a, err)
			}
			b, err := Parse(tt.b)
			if err != nil {
				t.Fatalf("Parse(%q): %v", tt.b, err)
			}
			if got := a.Compare(b); got != tt.want {
				t.Errorf("%q.Compare(%q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}
