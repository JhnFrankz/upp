package doctor

// Severity indicates the outcome of a diagnostic check.
type Severity int

const (
	SeverityOK Severity = iota
	SeverityWarn
	SeverityError
)

func (s Severity) String() string {
	switch s {
	case SeverityOK:
		return "OK"
	case SeverityWarn:
		return "WARN"
	case SeverityError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// CheckResult represents the outcome of a single diagnostic check.
type CheckResult struct {
	Category string // e.g. "Configuration & Storage", "Process Lock", "Package Managers", "Tool Paths", "Network"
	Name     string
	Status   Severity
	Message  string
	Detail   string
	FixHint  string
}

// HasErrors returns true if any check result has SeverityError status.
func HasErrors(results []CheckResult) bool {
	for _, r := range results {
		if r.Status == SeverityError {
			return true
		}
	}
	return false
}

// HasWarnings returns true if any check result has SeverityWarn status.
func HasWarnings(results []CheckResult) bool {
	for _, r := range results {
		if r.Status == SeverityWarn {
			return true
		}
	}
	return false
}
