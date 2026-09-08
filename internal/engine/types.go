package engine

import (
	"github.com/JhnFrankz/upp/internal/adapters"
)

// CheckStatus represents the health and update status of a tool check.
type CheckStatus int

const (
	// StatusAvailable indicates a new update is available.
	StatusAvailable CheckStatus = iota
	// StatusCurrent indicates the tool is already up-to-date.
	StatusCurrent
	// StatusSkipped indicates the tool is not installed or skipped.
	StatusSkipped
	// StatusFailed indicates the check encountered an error or panic.
	StatusFailed
)

// String returns the string representation of CheckStatus.
func (s CheckStatus) String() string {
	switch s {
	case StatusAvailable:
		return "available"
	case StatusCurrent:
		return "current"
	case StatusSkipped:
		return "skipped"
	case StatusFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// CheckOutcome encapsulates the pure domain result of an adapter check.
type CheckOutcome struct {
	ToolID          string
	ToolName        string
	Status          CheckStatus
	CurrentVersion  string
	LatestVersion   string
	UpdateAvailable bool
	Err             error
	Stderr          string
	RawUpdateInfo   adapters.UpdateInfo
}

// CheckProgress represents a progress event emitted during concurrent checking.
type CheckProgress struct {
	Index   int
	Total   int
	Outcome CheckOutcome
}

// Filter specifies filtering criteria for tool discovery and planning.
type Filter struct {
	Only []string
	All  bool
}

// PlannedUpdate represents an individual pending update action.
type PlannedUpdate struct {
	ToolID         string
	ToolName       string
	ManagerID      string
	PackageName    string
	UpdatePolicy   adapters.UpdatePolicy
	CurrentVersion string
	LatestVersion  string
	RiskCommand    string
	Privileges     []string
	Trust          adapters.TrustLevel
}

// UpdatePlan represents the categorized result of update planning.
type UpdatePlan struct {
	Updates []PlannedUpdate
	Skipped []CheckOutcome
	Current []CheckOutcome
	Failed  []CheckOutcome
}
