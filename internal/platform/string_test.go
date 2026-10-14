package platform

import "testing"

func TestContainsFold(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   bool
	}{
		{"sudo apt update", "sudo", true},
		{"SUDO apt update", "sudo", true},
		{"Sudo apt update", "SUDO", true},
		{"prefix-sudo-suffix", "sudo", true},
		{"apt update", "sudo", false},
		{"su", "sudo", false},
		{"", "sudo", false},
		{"sudo", "", true},
		{"", "", true},
		{"pkexec doas runas", "DOAS", true},
		{"hello world", "WORLD", true},
		{"hello", "world", false},
		{"external package manager", "PACKAGE", true},
	}

	for _, tt := range tests {
		t.Run(tt.s+"_"+tt.substr, func(t *testing.T) {
			got := ContainsFold(tt.s, tt.substr)
			if got != tt.want {
				t.Errorf("ContainsFold(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}

func TestContainsFold_NoAllocs(t *testing.T) {
	allocs := testing.AllocsPerRun(100, func() {
		_ = ContainsFold("SUDO apt-get install -y mypackage", "sudo")
	})
	if allocs > 0 {
		t.Errorf("ContainsFold allocated %f heap objects, want 0", allocs)
	}
}
