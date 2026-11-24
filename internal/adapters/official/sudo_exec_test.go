package official

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/JhnFrankz/upp/internal/adapters"
)

func TestPrepareSudoUpdateInvocation(t *testing.T) {
	t.Run("TTY pre-authenticates and injects -n", func(t *testing.T) {
		origAuth := adapters.SetSudoTestSeams(
			func() bool { return true },
			func(context.Context) error { return nil },
		)
		defer origAuth()

		name, args, err := prepareSudoUpdate(context.Background(), "linux", "sudo", []string{"apt", "install", "-y", "--only-upgrade", "gh"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if name != "sudo" {
			t.Fatalf("name = %q, want sudo", name)
		}
		want := []string{"-n", "apt", "install", "-y", "--only-upgrade", "gh"}
		if !slices.Equal(args, want) {
			t.Errorf("args = %v, want %v", args, want)
		}
	})

	t.Run("failed TTY auth aborts before spawning child", func(t *testing.T) {
		authErr := errors.New("auth failed")
		origAuth := adapters.SetSudoTestSeams(
			func() bool { return true },
			func(context.Context) error { return authErr },
		)
		defer origAuth()

		_, _, err := prepareSudoUpdate(context.Background(), "linux", "sudo", []string{"apt", "install", "-y", "--only-upgrade", "gh"})
		if !errors.Is(err, authErr) {
			t.Fatalf("err = %v, want errors.Is(authErr)", err)
		}
	})

	t.Run("non-TTY skips auth and injects -n", func(t *testing.T) {
		authCalled := false
		origAuth := adapters.SetSudoTestSeams(
			func() bool { return false },
			func(context.Context) error {
				authCalled = true
				return errors.New("unexpected")
			},
		)
		defer origAuth()

		_, args, err := prepareSudoUpdate(context.Background(), "linux", "sudo", []string{"pacman", "-S", "--noconfirm", "ripgrep"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if authCalled {
			t.Error("interactive sudo auth must not run when stdin is not a TTY")
		}
		want := []string{"-n", "pacman", "-S", "--noconfirm", "ripgrep"}
		if !slices.Equal(args, want) {
			t.Errorf("args = %v, want %v", args, want)
		}
	})
}
