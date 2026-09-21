package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRun_ContextCanceled(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
	}{
		{"list", "list"},
		{"self-update", "self-update"},
		{"init", "init"},
		{"uninstall", "uninstall"},
		{"update", "update"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			var stderr bytes.Buffer
			code := run(ctx, &stderr, tt.cmd)
			if code != 130 {
				t.Errorf("run(%s) canceled exit code = %d, want 130", tt.cmd, code)
			}
			if !strings.Contains(stderr.String(), "operation canceled") {
				t.Errorf("stderr = %q, want 'operation canceled'", stderr.String())
			}
		})
	}
}

func TestRun_Error(t *testing.T) {
	var stderr bytes.Buffer
	code := run(context.Background(), &stderr, "nonexistent-command")
	if code != 1 {
		t.Errorf("run(unknown cmd) exit code = %d, want 1", code)
	}
	if stderr.Len() == 0 {
		t.Errorf("expected stderr output for error, got empty")
	}
}

func TestRun_Success(t *testing.T) {
	var stderr bytes.Buffer
	code := run(context.Background(), &stderr, "--help")
	if code != 0 {
		t.Errorf("run(--help) exit code = %d, want 0", code)
	}
}
