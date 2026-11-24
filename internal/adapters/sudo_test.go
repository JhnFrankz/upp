package adapters

import (
	"context"
	"errors"
	"slices"
	"testing"
)

func TestPrepareSudoArgs(t *testing.T) {
	origGOOS := runtimeGOOSFn
	defer func() { runtimeGOOSFn = origGOOS }()
	runtimeGOOSFn = func() string { return "linux" }

	tests := []struct {
		name string
		cmd  string
		args []string
		want []string
	}{
		{
			name: "injects -n after sudo",
			cmd:  "sudo",
			args: []string{"apt", "install", "-y", "gh"},
			want: []string{"-n", "apt", "install", "-y", "gh"},
		},
		{
			name: "does not duplicate -n",
			cmd:  "sudo",
			args: []string{"-n", "pacman", "-S", "go"},
			want: []string{"-n", "pacman", "-S", "go"},
		},
		{
			name: "does not duplicate --non-interactive",
			cmd:  "sudo",
			args: []string{"--non-interactive", "apt", "update"},
			want: []string{"--non-interactive", "apt", "update"},
		},
		{
			name: "leaves non-sudo commands untouched",
			cmd:  "brew",
			args: []string{"upgrade", "gh"},
			want: []string{"upgrade", "gh"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			in := slices.Clone(tc.args)
			got := PrepareSudoArgs(tc.cmd, in)
			if !slices.Equal(got, tc.want) {
				t.Errorf("PrepareSudoArgs(%q, %v) = %v, want %v", tc.cmd, tc.args, got, tc.want)
			}
		})
	}
}

func TestEnsureSudoAuth(t *testing.T) {
	origGOOS := runtimeGOOSFn
	origTTY := stdinIsTerminalFn
	origAuth := sudoInteractiveAuthFn
	defer func() {
		runtimeGOOSFn = origGOOS
		stdinIsTerminalFn = origTTY
		sudoInteractiveAuthFn = origAuth
	}()

	runtimeGOOSFn = func() string { return "linux" }

	t.Run("TTY runs interactive sudo auth", func(t *testing.T) {
		stdinIsTerminalFn = func() bool { return true }
		called := 0
		sudoInteractiveAuthFn = func(ctx context.Context) error {
			called++
			return nil
		}
		if err := EnsureSudoAuth(context.Background()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if called != 1 {
			t.Errorf("sudoInteractiveAuthFn called %d times, want 1", called)
		}
	})

	t.Run("failed TTY auth returns wrapped error", func(t *testing.T) {
		stdinIsTerminalFn = func() bool { return true }
		authErr := errors.New("exit status 1")
		sudoInteractiveAuthFn = func(ctx context.Context) error {
			return authErr
		}
		err := EnsureSudoAuth(context.Background())
		if !errors.Is(err, authErr) {
			t.Fatalf("err = %v, want errors.Is(authErr)", err)
		}
	})

	t.Run("non-TTY skips interactive sudo auth", func(t *testing.T) {
		stdinIsTerminalFn = func() bool { return false }
		called := 0
		sudoInteractiveAuthFn = func(ctx context.Context) error {
			called++
			return errors.New("should not be called")
		}
		if err := EnsureSudoAuth(context.Background()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if called != 0 {
			t.Errorf("sudoInteractiveAuthFn called %d times, want 0", called)
		}
	})
}

func TestCommandUsesSudo(t *testing.T) {
	tests := []struct {
		cmd  string
		want bool
	}{
		{"sudo apt install -y gh", true},
		{"  sudo -n pacman -S go", true},
		{"echo ok && sudo mv /tmp/go /usr/local/go", true},
		{"pseudocode --check", false},
		{"brew upgrade gh", false},
	}
	for _, tc := range tests {
		if got := CommandUsesSudo(tc.cmd); got != tc.want {
			t.Errorf("CommandUsesSudo(%q) = %v, want %v", tc.cmd, got, tc.want)
		}
	}
}
