package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/JhnFrankz/upp/internal/doctor"
)

func TestDoctorCommand_AllOK(t *testing.T) {
	var buf bytes.Buffer
	gf := &GlobalFlags{}

	testDeps := doctorDeps{
		diagnose: func(ctx context.Context, deps doctor.DoctorDeps) []doctor.CheckResult {
			return []doctor.CheckResult{
				{
					Category: "Configuration & Storage",
					Name:     "Config File",
					Status:   doctor.SeverityOK,
					Message:  "Configuration file is valid",
				},
				{
					Category: "Process Lock",
					Name:     "Process Lock",
					Status:   doctor.SeverityOK,
					Message:  "No active process lock",
				},
			}
		},
	}

	err := runDoctor(context.Background(), gf, &buf, testDeps)
	if err != nil {
		t.Fatalf("expected nil error on all OK, got: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Results: 2 passed, 0 warnings, 0 errors.") {
		t.Errorf("output missing summary, got:\n%s", out)
	}
}

func TestDoctorCommand_WarningsOnly(t *testing.T) {
	var buf bytes.Buffer
	gf := &GlobalFlags{}

	testDeps := doctorDeps{
		diagnose: func(ctx context.Context, deps doctor.DoctorDeps) []doctor.CheckResult {
			return []doctor.CheckResult{
				{
					Category: "Process Lock",
					Name:     "Process Lock",
					Status:   doctor.SeverityWarn,
					Message:  "Stale lock file detected",
					FixHint:  "rm /tmp/upp.lock",
				},
			}
		},
	}

	// Exit policy: exit 0 on OK or WARN
	err := runDoctor(context.Background(), gf, &buf, testDeps)
	if err != nil {
		t.Fatalf("expected nil error when only warnings present, got: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Results: 0 passed, 1 warnings, 0 errors.") {
		t.Errorf("output missing summary, got:\n%s", out)
	}
}

func TestDoctorCommand_HasErrors(t *testing.T) {
	var buf bytes.Buffer
	gf := &GlobalFlags{}

	testDeps := doctorDeps{
		diagnose: func(ctx context.Context, deps doctor.DoctorDeps) []doctor.CheckResult {
			return []doctor.CheckResult{
				{
					Category: "Configuration & Storage",
					Name:     "Config Directory",
					Status:   doctor.SeverityError,
					Message:  "Config directory is not writable",
				},
			}
		},
	}

	// Exit policy: exit non-zero (return error) on ERROR
	err := runDoctor(context.Background(), gf, &buf, testDeps)
	if err == nil {
		t.Fatal("expected non-nil error when errors are present, got nil")
	}

	out := buf.String()
	if !strings.Contains(out, "Results: 0 passed, 0 warnings, 1 errors.") {
		t.Errorf("output missing summary, got:\n%s", out)
	}
}

func TestDoctorCommand_QuietAllOK(t *testing.T) {
	var buf bytes.Buffer
	gf := &GlobalFlags{Quiet: true}

	testDeps := doctorDeps{
		diagnose: func(ctx context.Context, deps doctor.DoctorDeps) []doctor.CheckResult {
			return []doctor.CheckResult{
				{
					Category: "Configuration & Storage",
					Name:     "Config File",
					Status:   doctor.SeverityOK,
					Message:  "Configuration file is valid",
				},
			}
		},
	}

	err := runDoctor(context.Background(), gf, &buf, testDeps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	want := "upp doctor: all checks passed.\n"
	if out != want {
		t.Errorf("quiet output expected %q, got %q", want, out)
	}
}
