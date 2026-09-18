package adapters

import (
	"context"
	"testing"
)

// TestRenderPackageCommand drives the pure rendering helper that turns a
// manager's declared per-package command template into the concrete command
// for one owned package (design D6): the FIRST PackagePlaceholder occurrence
// is replaced by pkg; a template without the placeholder appends " "+pkg.
// This replaces UpdateCmdName synthesis — the rendered command is what the
// manager's UpdatePackage() executes and what the plan declares (design D2).
func TestRenderPackageCommand(t *testing.T) {
	tests := []struct {
		name     string
		template string
		pkg      string
		want     string
	}{
		{
			name:     "placeholder-replaced",
			template: "sudo pacman -S --noconfirm <pkg>",
			pkg:      "ripgrep",
			want:     "sudo pacman -S --noconfirm ripgrep",
		},
		{
			name:     "placeholder-absent-appends-pkg",
			template: "brew upgrade",
			pkg:      "gh",
			want:     "brew upgrade gh",
		},
		{
			name:     "only-first-placeholder-replaced",
			template: "mgr --flag <pkg> --other <pkg>",
			pkg:      "gh",
			want:     "mgr --flag gh --other <pkg>",
		},
		{
			name:     "empty-template-appends-pkg",
			template: "",
			pkg:      "gh",
			want:     " gh",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RenderPackageCommand(tt.template, tt.pkg); got != tt.want {
				t.Errorf("RenderPackageCommand(%q, %q) = %q, want %q", tt.template, tt.pkg, got, tt.want)
			}
		})
	}
}

// TestAdapter_ContextContract verifies that Adapter implementations receive and respect context.Context.
func TestAdapter_ContextContract(t *testing.T) {
	// Verify mock adapter satisfies Adapter with context signatures.
	var _ Adapter = (*testCtxAdapter)(nil)
}

type testCtxAdapter struct{}

func (t *testCtxAdapter) Name() string { return "test" }
func (t *testCtxAdapter) Detect() bool { return true }
func (t *testCtxAdapter) Check(ctx context.Context) (UpdateInfo, error) {
	return UpdateInfo{}, ctx.Err()
}
func (t *testCtxAdapter) Update(ctx context.Context, dryRun bool) (Result, error) {
	return Result{}, ctx.Err()
}
func (t *testCtxAdapter) Info() ToolInfo { return ToolInfo{ID: "test"} }
