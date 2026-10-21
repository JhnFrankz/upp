package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderer_DoctorResults_AllOK(t *testing.T) {
	results := []DiagnosticResult{
		{
			Category: "Configuration & Storage",
			Name:     "Config File",
			Status:   DiagnosticSeverityOK,
			Message:  "Configuration file is valid",
		},
		{
			Category: "Network",
			Name:     "api.github.com",
			Status:   DiagnosticSeverityOK,
			Message:  "Reachable (status 200)",
		},
	}

	t.Run("normal mode", func(t *testing.T) {
		var buf bytes.Buffer
		r := NewRendererForced(&buf, false, false, false, false)
		r.DoctorResults(results, false, false)

		out := buf.String()
		if !strings.Contains(out, "Configuration & Storage") {
			t.Errorf("missing category header, got: %s", out)
		}
		if !strings.Contains(out, "[✓] Config File: Configuration file is valid") {
			t.Errorf("missing check line, got: %s", out)
		}
		if !strings.Contains(out, "Results: 2 passed, 0 warnings, 0 errors.") {
			t.Errorf("missing or incorrect summary, got: %s", out)
		}
	})

	t.Run("quiet mode", func(t *testing.T) {
		var buf bytes.Buffer
		r := NewRendererForced(&buf, false, false, true, false)
		r.DoctorResults(results, true, false)

		out := buf.String()
		want := "upp doctor: all checks passed.\n"
		if out != want {
			t.Errorf("quiet all-ok expected %q, got %q", want, out)
		}
	})
}

func TestRenderer_DoctorResults_WarningsAndErrors(t *testing.T) {
	results := []DiagnosticResult{
		{
			Category: "Configuration & Storage",
			Name:     "Config File",
			Status:   DiagnosticSeverityOK,
			Message:  "Configuration file is valid",
		},
		{
			Category: "Process Lock",
			Name:     "Process Lock",
			Status:   DiagnosticSeverityWarn,
			Message:  "Stale lock file detected (PID: 999 is dead)",
			FixHint:  "Remove stale lock file: rm /path/to/upp.lock",
		},
		{
			Category: "Configuration & Storage",
			Name:     "Config Directory",
			Status:   DiagnosticSeverityError,
			Message:  "Config directory is not writable",
			Detail:   "permission denied",
			FixHint:  "chmod 755 ~/.config/upp",
		},
	}

	t.Run("normal mode", func(t *testing.T) {
		var buf bytes.Buffer
		r := NewRendererForced(&buf, false, false, false, false)
		r.DoctorResults(results, false, false)

		out := buf.String()
		if !strings.Contains(out, "[✓] Config File: Configuration file is valid") {
			t.Errorf("missing OK check line, got: %s", out)
		}
		if !strings.Contains(out, "[!] Process Lock: Stale lock file detected") {
			t.Errorf("missing warning line, got: %s", out)
		}
		if !strings.Contains(out, "Hint: Remove stale lock file: rm /path/to/upp.lock") {
			t.Errorf("missing fix hint, got: %s", out)
		}
		if !strings.Contains(out, "[✗] Config Directory: Config directory is not writable") {
			t.Errorf("missing error line, got: %s", out)
		}
		if !strings.Contains(out, "Hint: chmod 755 ~/.config/upp") {
			t.Errorf("missing error fix hint, got: %s", out)
		}
		if !strings.Contains(out, "Results: 1 passed, 1 warnings, 1 errors.") {
			t.Errorf("incorrect summary, got: %s", out)
		}
	})

	t.Run("quiet mode", func(t *testing.T) {
		var buf bytes.Buffer
		r := NewRendererForced(&buf, false, false, true, false)
		r.DoctorResults(results, true, false)

		out := buf.String()
		// In quiet mode, only warnings and errors are rendered
		if strings.Contains(out, "Config File: Configuration file is valid") {
			t.Errorf("quiet mode should suppress OK results, got: %s", out)
		}
		if !strings.Contains(out, "[!] Process Lock: Stale lock file detected") {
			t.Errorf("quiet mode should show warnings, got: %s", out)
		}
		if !strings.Contains(out, "[✗] Config Directory: Config directory is not writable") {
			t.Errorf("quiet mode should show errors, got: %s", out)
		}
		if !strings.Contains(out, "Results: 1 passed, 1 warnings, 1 errors.") {
			t.Errorf("quiet mode with issues should show summary, got: %s", out)
		}
	})

	t.Run("verbose mode", func(t *testing.T) {
		var buf bytes.Buffer
		r := NewRendererForced(&buf, false, false, false, true)
		r.DoctorResults(results, false, true)

		out := buf.String()
		if !strings.Contains(out, "permission denied") {
			t.Errorf("verbose mode should display Detail, got: %s", out)
		}
	})
}

func TestDoctorIcon(t *testing.T) {
	var buf bytes.Buffer
	rPlain := NewRendererForced(&buf, false, false, false, false)
	if got := rPlain.doctorIcon(DiagnosticSeverityOK); got != doctorIconOKPlain {
		t.Errorf("doctorIcon plain OK = %q, want %q", got, doctorIconOKPlain)
	}
	if got := rPlain.doctorIcon(DiagnosticSeverityWarn); got != doctorIconWarnPlain {
		t.Errorf("doctorIcon plain Warn = %q, want %q", got, doctorIconWarnPlain)
	}
	if got := rPlain.doctorIcon(DiagnosticSeverityError); got != doctorIconErrorPlain {
		t.Errorf("doctorIcon plain Error = %q, want %q", got, doctorIconErrorPlain)
	}
	if got := rPlain.doctorIcon(DiagnosticSeverity(999)); got != doctorIconUnknown {
		t.Errorf("doctorIcon plain unknown = %q, want %q", got, doctorIconUnknown)
	}

	rColor := NewRendererForced(&buf, true, false, false, false)
	if got := rColor.doctorIcon(DiagnosticSeverityOK); got != doctorIconOKColor {
		t.Errorf("doctorIcon color OK = %q, want %q", got, doctorIconOKColor)
	}
	if got := rColor.doctorIcon(DiagnosticSeverityWarn); got != doctorIconWarnColor {
		t.Errorf("doctorIcon color Warn = %q, want %q", got, doctorIconWarnColor)
	}
	if got := rColor.doctorIcon(DiagnosticSeverityError); got != doctorIconErrorColor {
		t.Errorf("doctorIcon color Error = %q, want %q", got, doctorIconErrorColor)
	}
	if got := rColor.doctorIcon(DiagnosticSeverity(999)); got != doctorIconUnknown {
		t.Errorf("doctorIcon color unknown = %q, want %q", got, doctorIconUnknown)
	}
}

func TestRenderer_DoctorPresenterInterface(t *testing.T) {
	var buf bytes.Buffer
	var p DiagnosticPresenter = NewRenderer(&buf, false)
	p.DoctorResults([]DiagnosticResult{
		{
			Category: "General",
			Name:     "Check",
			Status:   DiagnosticSeverityOK,
			Message:  "Pass",
		},
	}, false, false)

	if !strings.Contains(buf.String(), "Pass") {
		t.Errorf("DiagnosticPresenter interface expected output with 'Pass', got: %s", buf.String())
	}
}
