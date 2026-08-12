// Package adapters defines the core interface and types for tool adapters.
// Every tool adapter (official or custom) must implement the Adapter interface.
package adapters

// TrustLevel represents how much the system trusts a tool adapter.
type TrustLevel int

const (
	// TrustCustomUntrusted is for custom adapters, untrusted by default.
	// It is the ZERO value on purpose: an unset TrustLevel MUST resolve to the
	// least-privileged level so unset trust fails closed. The zero value MUST
	// stay the least-privileged tier — never insert a new level before it.
	TrustCustomUntrusted TrustLevel = 0
	// TrustCustomTrusted is for custom adapters marked trusted=true in config.
	// It must never alias TrustOfficial: trust level MUST NOT bypass the risk matrix.
	TrustCustomTrusted TrustLevel = 1
	// TrustOfficial is for official, built-in adapters.
	TrustOfficial TrustLevel = 2
)

// String returns a human-readable trust label.
func (t TrustLevel) String() string {
	switch t {
	case TrustOfficial:
		return "official"
	case TrustCustomTrusted:
		return "custom-trusted"
	case TrustCustomUntrusted:
		return "custom-untrusted"
	default:
		return "unknown"
	}
}

// Adapter is the contract every tool adapter must implement.
// Each adapter handles detection, version checking, and updating for one tool.
type Adapter interface {
	// Name returns the tool identifier (e.g., "apt", "brew", "nvm").
	Name() string
	// Detect returns true if the tool is installed on the current platform.
	Detect() bool
	// Check queries the tool for current and latest versions.
	Check() (UpdateInfo, error)
	// Update performs the update operation.
	Update(dryRun bool) (Result, error)
	// Info returns static metadata about the tool.
	Info() ToolInfo
}

// UpdateInfo holds version information returned by Check().
type UpdateInfo struct {
	CurrentVersion  string
	LatestVersion   string
	UpdateAvailable bool
}

// Result holds the outcome of an Update() call.
type Result struct {
	Success    bool
	Before     string
	After      string
	Error      error
	Privileges []string // e.g., ["sudo"]
}

// ToolInfo holds static metadata about a tool.
type ToolInfo struct {
	ID         string
	Name       string
	Platforms  []string
	Trust      TrustLevel
	Command    string   // real update command; empty for official adapters
	Privileges []string // e.g., ["sudo"]
}
