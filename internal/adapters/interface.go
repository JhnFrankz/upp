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

// UpdatePolicy controls when update() may run for a tool adapter.
type UpdatePolicy int

const (
	// PolicyGated means update() runs only when check() reported
	// update_available=true (apt, npm, pnpm, nvm).
	// It is the ZERO value on purpose: an unset UpdatePolicy MUST resolve
	// to the most conservative behavior (fail closed), mirroring the
	// TrustLevel convention above — never insert a new value before it.
	PolicyGated UpdatePolicy = 0
	// PolicyAlwaysUpdate means update() always runs when requested,
	// regardless of the check() result (brew, bun, docker, gh, go,
	// opencode, winget, scoop, and custom adapters).
	PolicyAlwaysUpdate UpdatePolicy = 1
)

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

// PackageChecker is implemented by manager adapters (apt/brew/winget) that can
// report whether a specific owned package has an update. It answers the
// package-system availability question (`apt-cache policy <pkg>`,
// `brew outdated --json <pkg>`, `winget upgrade <pkg>`) for an owned tool's
// package under that manager — NOT the manager's own self check (design D2).
// One helper serves both an owned tool's delegated Check() (interactive
// pending) and the manager-group bulk path.
type PackageChecker interface {
	// CheckPackage reports the current vs latest version of an owned package
	// and whether an update is available.
	CheckPackage(packageName string) (UpdateInfo, error)
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

// Kind distinguishes manager adapters (which own other tools) from
// owned/standalone tool adapters. It is the declared ownership kind a tool
// reports through ToolInfo (spec Tool Ownership Declaration).
type Kind int

const (
	// KindTool is the ZERO value on purpose: a tool with no manager (or an
	// owned tool that is itself not a manager). An unset Kind MUST resolve to
	// the tool tier so an undeclared ownership kind fails open toward
	// standalone behavior, mirroring the TrustLevel/UpdatePolicy convention.
	KindTool Kind = iota
	// KindManager is for manager adapters (apt, brew, winget, scoop) that
	// declare owning a set of tools per platform.
	KindManager
)

// ToolInfo holds static metadata about a tool.
type ToolInfo struct {
	ID             string
	Name           string
	Platforms      []string
	Trust          TrustLevel
	UpdatePolicy   UpdatePolicy
	Kind           Kind
	Manager        map[string]string // platform -> owning manager ID (nil for standalone)
	ManagerPackage map[string]string // platform -> package name under that platform's manager
	Command        string            // real update command; empty for official adapters
	Privileges     []string          // e.g., ["sudo"]
}
