package output

import (
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/JhnFrankz/upp/internal/doctor"
)

func BenchmarkRenderer_DoctorResults(b *testing.B) {
	categories := []string{"System", "Network", "Configuration", "Tools"}
	results := make([]doctor.CheckResult, 15)
	for i := 0; i < 15; i++ {
		cat := categories[i%len(categories)]
		status := doctor.SeverityOK
		msg := "All checks passed successfully"
		hint := ""
		detail := ""
		switch i % 3 {
		case 1:
			status = doctor.SeverityWarn
			msg = "Non-critical configuration notice detected"
			hint = "Run check again with --fix if applicable"
			detail = "diagnostic detail line 1\ndiagnostic detail line 2"
		case 2:
			status = doctor.SeverityError
			msg = "Required component is missing or misconfigured"
			hint = "Install missing dependency"
			detail = "error log trace line 1\nerror log trace line 2"
		}
		results[i] = doctor.CheckResult{
			Category: cat,
			Name:     fmt.Sprintf("check-%02d", i),
			Status:   status,
			Message:  msg,
			FixHint:  hint,
			Detail:   detail,
		}
	}

	r := NewRenderer(io.Discard, false)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r.DoctorResults(results, false, false)
	}
}

func BenchmarkRenderer_DoctorResults_Color(b *testing.B) {
	categories := []string{"System", "Network", "Configuration", "Tools"}
	results := make([]doctor.CheckResult, 15)
	for i := 0; i < 15; i++ {
		cat := categories[i%len(categories)]
		status := doctor.SeverityOK
		msg := "All checks passed successfully"
		hint := ""
		detail := ""
		switch i % 3 {
		case 1:
			status = doctor.SeverityWarn
			msg = "Non-critical configuration notice detected"
			hint = "Run check again with --fix if applicable"
			detail = "diagnostic detail line 1\ndiagnostic detail line 2"
		case 2:
			status = doctor.SeverityError
			msg = "Required component is missing or misconfigured"
			hint = "Install missing dependency"
			detail = "error log trace line 1\nerror log trace line 2"
		}
		results[i] = doctor.CheckResult{
			Category: cat,
			Name:     fmt.Sprintf("check-%02d", i),
			Status:   status,
			Message:  msg,
			FixHint:  hint,
			Detail:   detail,
		}
	}

	r := NewRendererForced(io.Discard, true, false, false, false)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r.DoctorResults(results, false, false)
	}
}

func BenchmarkRenderer_Summary(b *testing.B) {
	statuses := []Status{
		StatusUpdated,
		StatusCurrent,
		StatusSkipped,
		StatusFailed,
		StatusDeselected,
	}
	results := make([]ToolResult, 20)
	for i := 0; i < 20; i++ {
		st := statuses[i%len(statuses)]
		var err error
		if st == StatusFailed {
			err = errors.New("command exited with code 1")
		}
		results[i] = ToolResult{
			Name:    fmt.Sprintf("tool-%02d", i),
			Status:  st,
			Version: "1.2.3",
			Error:   err,
		}
	}
	summary := Summary{Results: results}
	r := NewRenderer(io.Discard, false)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r.Summary(summary)
	}
}

func BenchmarkRenderer_ToolLine_Verbose(b *testing.B) {
	results := []ToolResult{
		{Name: "brew", Status: StatusUpdated, Version: "4.1.0"},
		{Name: "cargo", Status: StatusCurrent, Version: "1.75.0"},
		{Name: "npm", Status: StatusFailed, Error: errors.New("exit 1"), Stderr: "npm ERR! connection refused\n"},
		{Name: "pip", Status: StatusSkipped},
		{Name: "go", Status: StatusAvailable, Version: "1.22.0"},
		{Name: "rustup", Status: StatusDeselected},
	}
	r := NewRendererForced(io.Discard, true, true, false, true)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r.ToolLine(results[i%len(results)])
	}
}

func BenchmarkRenderer_ToolLine_Quiet(b *testing.B) {
	results := []ToolResult{
		{Name: "brew", Status: StatusUpdated, Version: "4.1.0"},
		{Name: "cargo", Status: StatusCurrent, Version: "1.75.0"},
		{Name: "npm", Status: StatusFailed, Error: errors.New("exit 1")},
		{Name: "pip", Status: StatusSkipped},
		{Name: "go", Status: StatusAvailable, Version: "1.22.0"},
		{Name: "rustup", Status: StatusDeselected},
	}
	r := NewRendererForced(io.Discard, false, true, true, false)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r.ToolLine(results[i%len(results)])
	}
}

func BenchmarkRenderer_Progress(b *testing.B) {
	r := NewRendererForced(io.Discard, true, true, false, false)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r.Progress("Updating", 3, 10, "golang")
	}
}

func BenchmarkRenderer_ProgressInPlace(b *testing.B) {
	r := NewRendererForced(io.Discard, true, true, false, false)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r.ProgressInPlace("Updating", 3, 10, "golang")
	}
}
