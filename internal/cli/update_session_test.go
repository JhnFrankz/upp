package cli

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/engine"
	"github.com/JhnFrankz/upp/internal/output"
	"github.com/JhnFrankz/upp/internal/platform"
	"github.com/JhnFrankz/upp/internal/security"
)

func TestNewUpdateSession_Defaults(t *testing.T) {
	var nilCtx context.Context
	s := NewUpdateSession(nilCtx, nil, nil, nil, nil, nil, platform.OSLinux, nil, nil)
	if s == nil {
		t.Fatal("NewUpdateSession returned nil")
	}
	if s.Ctx == nil {
		t.Error("expected non-nil default Context")
	}
	if s.GlobalFlags == nil {
		t.Error("expected non-nil default GlobalFlags")
	}
	if s.UpdateFlags == nil {
		t.Error("expected non-nil default UpdateFlags")
	}
	if s.Renderer == nil {
		t.Error("expected non-nil default Renderer")
	}
	if s.Engine == nil {
		t.Error("expected non-nil default Engine")
	}
	if s.In == nil {
		t.Error("expected non-nil default In")
	}
	if s.Out == nil {
		t.Error("expected non-nil default Out")
	}
	if s.ErrOut == nil {
		t.Error("expected non-nil default ErrOut")
	}
	if s.OSName != platform.OSLinux {
		t.Errorf("expected OSName %q, got %q", platform.OSLinux, s.OSName)
	}
}

func TestNewUpdateSession_Custom(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gf := &GlobalFlags{Verbose: true}
	uf := &UpdateFlags{DryRun: true}
	var inBuf, outBuf, errBuf bytes.Buffer
	r := output.NewRendererVerbose(&outBuf, false, true)
	eng := engine.New(nil, platform.OSMacOS)
	all := []adapters.Adapter{&fakeUpdateAdapter{name: "tool-a"}}

	s := NewUpdateSession(ctx, gf, uf, r, eng, all, platform.OSMacOS, &inBuf, &outBuf, &errBuf)
	if s.Ctx != ctx {
		t.Error("Ctx not preserved")
	}
	if s.GlobalFlags != gf {
		t.Error("GlobalFlags not preserved")
	}
	if s.UpdateFlags != uf {
		t.Error("UpdateFlags not preserved")
	}
	if s.Renderer != r {
		t.Error("Renderer not preserved")
	}
	if s.Engine != eng {
		t.Error("Engine not preserved")
	}
	if len(s.AllAdapters) != 1 || s.AllAdapters[0].Name() != "tool-a" {
		t.Error("AllAdapters not preserved")
	}
	if s.OSName != platform.OSMacOS {
		t.Errorf("expected OSName %q, got %q", platform.OSMacOS, s.OSName)
	}
	if s.In != &inBuf || s.Out != &outBuf || s.ErrOut != &errBuf {
		t.Error("IO streams not preserved")
	}
}

func TestUpdateSession_ExecutePlannedUpdate(t *testing.T) {
	fake := &fakeUpdateAdapter{
		name:  "tool-x",
		trust: security.TrustOfficial,
		result: adapters.Result{
			Success: true,
			Before:  "1.0.0",
			After:   "2.0.0",
		},
	}
	planned := engine.PlannedUpdate{
		ToolName:       "tool-x",
		ToolID:         "tool-x",
		CurrentVersion: "1.0.0",
		LatestVersion:  "2.0.0",
		Trust:          security.TrustOfficial,
		RiskCommand:    "tool-x upgrade",
	}

	var buf bytes.Buffer
	r := output.NewRenderer(&buf, false)
	sess := NewUpdateSession(context.Background(), &GlobalFlags{}, &UpdateFlags{}, r, nil, []adapters.Adapter{fake}, platform.OSLinux, nil, &buf)

	res := sess.ExecutePlannedUpdate(planned, fake, 1, 1)
	if res.Status != output.StatusUpdated {
		t.Fatalf("expected StatusUpdated, got %v", res.Status)
	}
	if res.Version != "1.0.0 → 2.0.0" {
		t.Errorf("expected version %q, got %q", "1.0.0 → 2.0.0", res.Version)
	}
}

func TestUpdateSession_RunSequential_Empty(t *testing.T) {
	var buf bytes.Buffer
	r := output.NewRenderer(&buf, false)
	sess := NewUpdateSession(context.Background(), &GlobalFlags{}, &UpdateFlags{}, r, nil, nil, platform.OSLinux, nil, &buf)

	err := sess.RunSequential(nil)
	if err != nil {
		t.Fatalf("unexpected error for empty adapters: %v", err)
	}
}

func TestUpdateSession_RunSequential_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var buf bytes.Buffer
	r := output.NewRenderer(&buf, false)
	sess := NewUpdateSession(ctx, &GlobalFlags{}, &UpdateFlags{}, r, nil, nil, platform.OSLinux, nil, &buf)

	err := sess.RunSequential([]adapters.Adapter{&fakeUpdateAdapter{name: "tool"}})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestUpdateSession_AuthorizeChecks(t *testing.T) {
	fake := &fakeUpdateAdapter{
		name:  "official-tool",
		trust: security.TrustOfficial,
	}
	var buf bytes.Buffer
	r := output.NewRenderer(&buf, false)
	sess := NewUpdateSession(context.Background(), &GlobalFlags{}, &UpdateFlags{}, r, nil, nil, platform.OSLinux, nil, &buf)

	auth, err := sess.AuthorizeChecks([]adapters.Adapter{fake})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(auth) != 1 {
		t.Fatalf("expected 1 adapter, got %d", len(auth))
	}
}

func TestNewUpdateSession_EmptyAdapters(t *testing.T) {
	s := NewUpdateSession(context.Background(), nil, nil, nil, nil, []adapters.Adapter{}, platform.OSLinux, nil, nil)
	resolved, err := s.Engine.Resolve(engine.Filter{})
	if err != nil {
		t.Fatalf("unexpected resolve error: %v", err)
	}
	if len(resolved) != 0 {
		t.Errorf("expected 0 resolved adapters for explicit empty adapter list, got %d", len(resolved))
	}
}

func TestUpdateSession_NilReceiver(t *testing.T) {
	var s *UpdateSession
	if err := s.RunSequential(nil); err == nil {
		t.Error("expected error on nil receiver RunSequential")
	}
	if err := s.RunInteractive(nil); err == nil {
		t.Error("expected error on nil receiver RunInteractive")
	}
	if _, err := s.AuthorizeChecks(nil); err == nil {
		t.Error("expected error on nil receiver AuthorizeChecks")
	}
	res := s.ExecutePlannedUpdate(engine.PlannedUpdate{ToolName: "test"}, nil, 1, 1)
	if res.Status != output.StatusFailed {
		t.Errorf("expected StatusFailed on nil receiver ExecutePlannedUpdate, got %v", res.Status)
	}
}

func TestUpdateSession_DirectStructLiteral_EnsureDefaults(t *testing.T) {
	var buf bytes.Buffer
	s := &UpdateSession{
		Out: &buf,
	}
	if err := s.RunSequential(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Ctx == nil || s.GlobalFlags == nil || s.Renderer == nil || s.Engine == nil {
		t.Errorf("expected ensureDefaults to populate nil fields, got %+v", s)
	}
}

func TestUpdateSession_RunInteractive_Empty(t *testing.T) {
	var buf bytes.Buffer
	r := output.NewRenderer(&buf, false)
	sess := NewUpdateSession(context.Background(), &GlobalFlags{}, &UpdateFlags{}, r, nil, nil, platform.OSLinux, nil, &buf)
	err := sess.RunInteractive(nil)
	if err != nil {
		t.Fatalf("unexpected error for empty interactive run: %v", err)
	}
}

func TestUpdateSession_RunInteractive_Canceled(t *testing.T) {
	var buf bytes.Buffer
	r := output.NewRenderer(&buf, false)
	fake := &fakeUpdateAdapter{
		name:   "tool-interactive",
		policy: adapters.PolicyAlwaysUpdate,
		trust:  security.TrustOfficial,
	}
	sess := NewUpdateSession(context.Background(), &GlobalFlags{}, &UpdateFlags{}, r, nil, []adapters.Adapter{fake}, platform.OSLinux, nil, &buf)
	sess.Selector = func(opts []output.SelectOption) ([]string, bool) {
		return nil, true
	}
	err := sess.RunInteractive([]adapters.Adapter{fake})
	if err != nil {
		t.Fatalf("unexpected error on canceled interactive run: %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("Update canceled")) {
		t.Errorf("expected output to contain 'Update canceled', got %q", buf.String())
	}
}

func TestUpdateSession_RunInteractive_Selected(t *testing.T) {
	var buf bytes.Buffer
	r := output.NewRenderer(&buf, false)
	fake := &fakeUpdateAdapter{
		name:   "tool-interactive",
		policy: adapters.PolicyAlwaysUpdate,
		trust:  security.TrustOfficial,
		result: adapters.Result{Success: true, Before: "1.0", After: "2.0"},
	}
	sess := NewUpdateSession(context.Background(), &GlobalFlags{}, &UpdateFlags{}, r, nil, []adapters.Adapter{fake}, platform.OSLinux, nil, &buf)
	sess.Selector = func(opts []output.SelectOption) ([]string, bool) {
		return []string{"tool-interactive"}, false
	}
	err := sess.RunInteractive([]adapters.Adapter{fake})
	if err != nil {
		t.Fatalf("unexpected error on selected interactive run: %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("1 updated")) {
		t.Errorf("expected summary to contain '1 updated', got %q", buf.String())
	}
}

func TestUnconsentedCheckAdapter_Unwrap(t *testing.T) {
	fake := &fakeUpdateAdapter{name: "base-adapter"}
	wrapped := unconsentedCheckAdapter{Adapter: fake}
	if unwrapped, ok := any(wrapped).(interface{ Unwrap() adapters.Adapter }); !ok {
		t.Fatal("unconsentedCheckAdapter does not implement Unwrap()")
	} else if unwrapped.Unwrap() != fake {
		t.Errorf("expected unwrapped adapter %v, got %v", fake, unwrapped.Unwrap())
	}
}
