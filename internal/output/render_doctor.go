package output

import (
	"io"
	"strconv"
	"strings"
)

const (
	doctorIconOKPlain    = "[✓]"
	doctorIconWarnPlain  = "[!]"
	doctorIconErrorPlain = "[✗]"
	doctorIconUnknown    = "[?]"

	doctorIconOKColor    = "\033[32m[✓]\033[0m"
	doctorIconWarnColor  = "\033[33m[!]\033[0m"
	doctorIconErrorColor = "\033[31m[✗]\033[0m"
)

// DiagnosticSeverity represents the severity of a diagnostic result.
type DiagnosticSeverity int

const (
	DiagnosticSeverityOK DiagnosticSeverity = iota
	DiagnosticSeverityWarn
	DiagnosticSeverityError
)

// DiagnosticResult holds a single diagnostic check outcome for display.
type DiagnosticResult struct {
	Name     string
	Status   DiagnosticSeverity
	Category string
	Message  string
	Detail   string
	FixHint  string
}

// DiagnosticPresenter defines the contract for presenting doctor environment and health checks.
type DiagnosticPresenter interface {
	DoctorResults(results []DiagnosticResult, quiet, verbose bool)
}

var _ DiagnosticPresenter = (*Renderer)(nil)

func (r *Renderer) doctorIcon(s DiagnosticSeverity) string {
	if !r.color {
		switch s {
		case DiagnosticSeverityOK:
			return doctorIconOKPlain
		case DiagnosticSeverityWarn:
			return doctorIconWarnPlain
		case DiagnosticSeverityError:
			return doctorIconErrorPlain
		default:
			return doctorIconUnknown
		}
	}
	switch s {
	case DiagnosticSeverityOK:
		return doctorIconOKColor
	case DiagnosticSeverityWarn:
		return doctorIconWarnColor
	case DiagnosticSeverityError:
		return doctorIconErrorColor
	default:
		return doctorIconUnknown
	}
}

// DoctorResults renders diagnostics results grouped by category.
func (r *Renderer) DoctorResults(results []DiagnosticResult, quiet, verbose bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	hasErrors := false
	hasWarnings := false
	for _, res := range results {
		switch res.Status {
		case DiagnosticSeverityError:
			hasErrors = true
		case DiagnosticSeverityWarn:
			hasWarnings = true
		}
	}

	if quiet && !hasErrors && !hasWarnings {
		_, _ = io.WriteString(r.w, "upp doctor: all checks passed.\n")
		return
	}

	type categorySpan struct {
		name  string
		start int
		end   int
	}
	categories := make([]categorySpan, 0, 8)
	indicesBuf := make([]int, 0, len(results))
	passedCount := 0
	warnCount := 0
	errCount := 0
	numDisplay := 0
	for idx := range results {
		switch results[idx].Status {
		case DiagnosticSeverityOK:
			passedCount++
		case DiagnosticSeverityWarn:
			warnCount++
		case DiagnosticSeverityError:
			errCount++
		}
		if quiet && results[idx].Status == DiagnosticSeverityOK {
			continue
		}
		numDisplay++
		cat := results[idx].Category
		if cat == "" {
			cat = "General"
		}
		found := false
		for i := range categories {
			if categories[i].name == cat {
				// Insert into indicesBuf at categories[i].end, and shift subsequent spans
				insertPos := categories[i].end
				indicesBuf = append(indicesBuf, 0)
				copy(indicesBuf[insertPos+1:], indicesBuf[insertPos:])
				indicesBuf[insertPos] = idx
				categories[i].end++
				for j := i + 1; j < len(categories); j++ {
					categories[j].start++
					categories[j].end++
				}
				found = true
				break
			}
		}
		if !found {
			start := len(indicesBuf)
			indicesBuf = append(indicesBuf, idx)
			categories = append(categories, categorySpan{
				name:  cat,
				start: start,
				end:   start + 1,
			})
		}
	}

	var sb strings.Builder
	sb.Grow(numDisplay*96 + 256)

	firstCategory := true
	for i := range categories {
		if !firstCategory {
			sb.WriteByte('\n')
		}
		firstCategory = false
		sb.WriteString(categories[i].name)
		sb.WriteByte('\n')

		for _, idx := range indicesBuf[categories[i].start:categories[i].end] {
			item := &results[idx]
			icon := r.doctorIcon(item.Status)
			sb.WriteString("  ")
			sb.WriteString(icon)
			sb.WriteByte(' ')
			if item.Name != "" {
				sb.WriteString(item.Name)
				sb.WriteString(": ")
				sb.WriteString(item.Message)
				sb.WriteByte('\n')
			} else {
				sb.WriteString(item.Message)
				sb.WriteByte('\n')
			}

			if verbose && item.Detail != "" {
				rem := item.Detail
				for len(rem) > 0 {
					var line string
					line, rem, _ = strings.Cut(rem, "\n")
					if line != "" {
						sb.WriteString("    ")
						sb.WriteString(line)
						sb.WriteByte('\n')
					}
				}
			}

			if item.FixHint != "" {
				sb.WriteString("    Hint: ")
				sb.WriteString(item.FixHint)
				sb.WriteByte('\n')
			}
		}
	}

	sb.WriteString("\nResults: ")
	sb.WriteString(strconv.Itoa(passedCount))
	sb.WriteString(" passed, ")
	sb.WriteString(strconv.Itoa(warnCount))
	sb.WriteString(" warnings, ")
	sb.WriteString(strconv.Itoa(errCount))
	sb.WriteString(" errors.\n")
	_, _ = io.WriteString(r.w, sb.String())
}
