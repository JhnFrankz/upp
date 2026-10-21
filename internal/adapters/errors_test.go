package adapters_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/JhnFrankz/upp/internal/adapters"
)

func TestSentinelErrors_Messages(t *testing.T) {
	tests := []struct {
		err  error
		want string
	}{
		{adapters.ErrNotInstalled, "tool is not installed"},
		{adapters.ErrNilAdapter, "adapter is nil"},
		{adapters.ErrNoResolvingOwner, "no resolving owner"},
		{adapters.ErrUnsupportedPackageCheck, "manager does not support per-package checks"},
		{adapters.ErrUnsupportedPackageUpdate, "manager does not support per-package updates"},
		{adapters.ErrNoManagerPackage, "no manager package configured"},
		{adapters.ErrTimeout, "operation timed out"},
		{adapters.ErrCommandFailed, "command execution failed"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("error string = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSentinelErrors_Distinctness(t *testing.T) {
	allSentinels := []struct {
		name string
		err  error
	}{
		{"ErrNotInstalled", adapters.ErrNotInstalled},
		{"ErrNilAdapter", adapters.ErrNilAdapter},
		{"ErrNoResolvingOwner", adapters.ErrNoResolvingOwner},
		{"ErrUnsupportedPackageCheck", adapters.ErrUnsupportedPackageCheck},
		{"ErrUnsupportedPackageUpdate", adapters.ErrUnsupportedPackageUpdate},
		{"ErrNoManagerPackage", adapters.ErrNoManagerPackage},
		{"ErrTimeout", adapters.ErrTimeout},
		{"ErrCommandFailed", adapters.ErrCommandFailed},
	}

	for i, a := range allSentinels {
		for j, b := range allSentinels {
			if i == j {
				if !errors.Is(a.err, b.err) {
					t.Errorf("expected %s to match itself via errors.Is", a.name)
				}
			} else {
				if errors.Is(a.err, b.err) {
					t.Errorf("unexpected match between %s and %s via errors.Is", a.name, b.name)
				}
				if a.err == b.err {
					t.Errorf("expected %s and %s to have distinct pointers", a.name, b.name)
				}
			}
		}
	}
}

func TestSentinelErrors_Wrapping(t *testing.T) {
	allSentinels := []struct {
		name string
		err  error
	}{
		{"ErrNotInstalled", adapters.ErrNotInstalled},
		{"ErrNilAdapter", adapters.ErrNilAdapter},
		{"ErrNoResolvingOwner", adapters.ErrNoResolvingOwner},
		{"ErrUnsupportedPackageCheck", adapters.ErrUnsupportedPackageCheck},
		{"ErrUnsupportedPackageUpdate", adapters.ErrUnsupportedPackageUpdate},
		{"ErrNoManagerPackage", adapters.ErrNoManagerPackage},
		{"ErrTimeout", adapters.ErrTimeout},
		{"ErrCommandFailed", adapters.ErrCommandFailed},
	}

	for _, tt := range allSentinels {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := fmt.Errorf("%w: detailed context for %s", tt.err, tt.name)
			if !errors.Is(wrapped, tt.err) {
				t.Errorf("errors.Is(wrapped, %s) = false, want true", tt.name)
			}

			doubleWrapped := fmt.Errorf("outer: %w", wrapped)
			if !errors.Is(doubleWrapped, tt.err) {
				t.Errorf("errors.Is(doubleWrapped, %s) = false, want true", tt.name)
			}

			unrelated := errors.New("unrelated error")
			if errors.Is(unrelated, tt.err) {
				t.Errorf("errors.Is(unrelated, %s) = true, want false", tt.name)
			}
		})
	}
}
