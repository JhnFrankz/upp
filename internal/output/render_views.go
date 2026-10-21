package output

import (
	"fmt"
	"strings"
	"text/tabwriter"
)

// ListEntry holds data for a single tool in the list output.
type ListEntry struct {
	ID      string // --only filter ID (e.g. "apt", "brew")
	Name    string
	Status  Status
	Version string
}

// DashboardData holds info for rendering the bare upp welcome dashboard.
type DashboardData struct {
	Version        string
	Platform       string
	EnabledTools   int
	AvailableTools int
}

// TargetItem represents a target file or directory for display in an operation plan.
type TargetItem struct {
	Type   string
	Path   string
	Exists bool
}

// ListPresenter defines the contract for rendering tool listing views.
type ListPresenter interface {
	ListTools(groups []Group)
	NoToolsConfigured()
	NoToolsMatchFilter(filter string)
}

// DashboardPresenter defines the contract for rendering welcome dashboard views.
type DashboardPresenter interface {
	Dashboard(data DashboardData)
	DashboardNoConfig(version, platform string)
}

// InitPresenter defines the contract for rendering initialization wizard steps.
type InitPresenter interface {
	InitHeader()
	InitDetected(name string)
	InitConfigGenerated(path string)
}

// SelfUpdatePresenter defines the contract for rendering self-update prompts and status.
type SelfUpdatePresenter interface {
	SelfUpdatePrompt(current, latest, target string)
	SelfUpdateDevBuild()
	SelfUpdateUpToDate(tag string)
	SelfUpdateDone(current, latest string)
}

// UninstallPresenter defines the contract for rendering uninstall plans and outcomes.
type UninstallPresenter interface {
	UninstallDryRunHeader()
	UninstallDryRunTarget(targetType, path string)
	UninstallRemoved(targetType, path string)
	UninstallPlan(targets []TargetItem)
	UninstallCanceled()
	UninstallDone()
}

// ViewPresenter aggregates all command view presentation interfaces.
type ViewPresenter interface {
	ListPresenter
	DashboardPresenter
	InitPresenter
	SelfUpdatePresenter
	UninstallPresenter
}

var (
	_ ListPresenter       = (*Renderer)(nil)
	_ DashboardPresenter  = (*Renderer)(nil)
	_ InitPresenter       = (*Renderer)(nil)
	_ SelfUpdatePresenter = (*Renderer)(nil)
	_ UninstallPresenter  = (*Renderer)(nil)
	_ ViewPresenter       = (*Renderer)(nil)
)

// --- List output ---

// ListTools renders a table of detected tools, grouped by owning manager
// when groups carry a non-empty Header (design: group rendering/wiring).
// Each group prints its manager header line once, then its child rows indented
// beneath it (the manager's own row leads, followed by the tools it owns).
// A standalone group (empty Header) prints only its rows.
func (r *Renderer) ListTools(groups []Group) {
	w := tabwriter.NewWriter(r.w, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
		r.cyan("ID"), "Name", "Status", "Version")

	for _, g := range groups {
		if g.Header != "" {
			_, _ = fmt.Fprintf(w, "%s\n", g.Header)
		}
		for _, t := range g.Items {
			status := r.statusLabel(t.Status)
			version := t.Version
			if version == "" {
				version = "-"
			}
			_, _ = fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n",
				t.ID, t.Name, status, version)
		}
	}
	_ = w.Flush()
}

// NoToolsConfigured prints the empty-list message for `upp list` when no
// tools are configured. Suppressed under --quiet like other list status
// output.
func (r *Renderer) NoToolsConfigured() {
	if r.quiet {
		return
	}
	_, _ = fmt.Fprintln(r.w, "no tools configured.")
}

// NoToolsMatchFilter prints the --only filter mismatch message for `upp
// list`: the filter matched none of the configured tools. Suppressed under
// --quiet like other list status output.
func (r *Renderer) NoToolsMatchFilter(filter string) {
	if r.quiet {
		return
	}
	_, _ = fmt.Fprintf(r.w, "no tools match --only filter: %s\n", filter)
}

// --- Dashboard output ---

// Dashboard renders the bare upp welcome dashboard.
func (r *Renderer) Dashboard(data DashboardData) {
	if r.quiet {
		return
	}
	_, _ = fmt.Fprintf(r.w, "%s upp %s (%s)\n\n", r.cyan("●"), data.Version, data.Platform)
	_, _ = fmt.Fprintf(r.w, "  Tools: %d enabled (%d configured for platform)\n\n", data.EnabledTools, data.AvailableTools)
	_, _ = fmt.Fprintln(r.w, "  Commands:")
	_, _ = fmt.Fprintf(r.w, "    %-14s %s\n", "upp update -n", "Preview pending updates (--dry-run)")
	_, _ = fmt.Fprintf(r.w, "    %-14s %s\n", "upp update", "Apply updates to all enabled tools")
	_, _ = fmt.Fprintf(r.w, "    %-14s %s\n", "upp list", "List configured tools and versions")
	_, _ = fmt.Fprintf(r.w, "    %-14s %s\n", "upp doctor", "Run environment and health diagnostics")
	_, _ = fmt.Fprintf(r.w, "    %-14s %s\n", "upp --help", "Show help and options")
}

// DashboardNoConfig renders the bare upp guidance when no config exists.
func (r *Renderer) DashboardNoConfig(version, platform string) {
	if r.quiet {
		return
	}
	_, _ = fmt.Fprintf(r.w, "%s upp %s (%s)\n\n", r.cyan("●"), version, platform)
	_, _ = fmt.Fprintln(r.w, "  No configuration found.")
	_, _ = fmt.Fprintln(r.w, "  Run \"upp init\" to detect installed tools and initialize your config.")
}

// --- Init output ---

// InitHeader prints the init wizard header.
func (r *Renderer) InitHeader() {
	_, _ = fmt.Fprintln(r.w, "upp init — detecting installed tools...")
	_, _ = fmt.Fprintln(r.w)
}

// InitDetected prints a detected tool.
func (r *Renderer) InitDetected(name string) {
	_, _ = fmt.Fprintf(r.w, "  %s %s\n", r.statusIcon(StatusCurrent), r.cyan(name))
}

// InitConfigGenerated prints the config generated message.
func (r *Renderer) InitConfigGenerated(path string) {
	_, _ = fmt.Fprintln(r.w)
	_, _ = fmt.Fprintf(r.w, "%s Config written to %s\n", r.statusIcon(StatusUpdated), path)
}

// --- Self-update output ---

// SelfUpdatePrompt prints the pre-replace confirmation (design D8):
// current → latest versions and the resolved binary path, then the
// Proceed question. It is never suppressed by quiet mode (spec flag
// semantics: --quiet must not hide the confirm prompt).
func (r *Renderer) SelfUpdatePrompt(current, latest, target string) {
	_, _ = fmt.Fprintf(r.w, "  Update upp from %s to %s?\n", current, latest)
	_, _ = fmt.Fprintf(r.w, "    Target: %s\n", target)
	_, _ = fmt.Fprint(r.w, "  Proceed? [y/N] ")
}

// SelfUpdateDevBuild prints the development-build message (spec R1:
// exit 0, no update claim). Always shown, including quiet mode.
func (r *Renderer) SelfUpdateDevBuild() {
	_, _ = fmt.Fprintln(r.w, "development build; self-update is only available for release builds")
}

// SelfUpdateUpToDate prints the already-up-to-date message with the
// current release tag (spec R1). Always shown.
func (r *Renderer) SelfUpdateUpToDate(tag string) {
	_, _ = fmt.Fprintf(r.w, "already up to date (%s)\n", tag)
}

// SelfUpdateDone prints the replacement success line: current → latest.
// Always shown.
func (r *Renderer) SelfUpdateDone(current, latest string) {
	_, _ = fmt.Fprintf(r.w, "upp updated: %s → %s\n", current, latest)
}

// --- Uninstall output ---

// UninstallDryRunHeader prints the dry-run header for uninstallation.
func (r *Renderer) UninstallDryRunHeader() {
	_, _ = fmt.Fprintf(r.w, "%s Dry run — no files will be removed\n\n", r.statusIcon(StatusAvailable))
}

// UninstallDryRunTarget prints a planned target to be removed during dry-run.
func (r *Renderer) UninstallDryRunTarget(targetType, path string) {
	_, _ = fmt.Fprintf(r.w, "  %s Would remove %s: %s\n", r.statusIcon(StatusAvailable), targetType, r.cyan(path))
}

// UninstallRemoved prints a successfully removed target.
func (r *Renderer) UninstallRemoved(targetType, path string) {
	if r.quiet {
		return
	}
	_, _ = fmt.Fprintf(r.w, "  %s Removed %s: %s\n", r.statusIcon(StatusUpdated), targetType, path)
}

// UninstallPlan renders the list of targets to be removed before interactive confirmation.
func (r *Renderer) UninstallPlan(targets []TargetItem) {
	_, _ = fmt.Fprintln(r.w, "The following targets will be removed:")
	hasExisting := false
	for _, t := range targets {
		if t.Exists {
			hasExisting = true
			break
		}
	}
	for _, t := range targets {
		if hasExisting && !t.Exists {
			continue
		}
		typeName := t.Type
		if len(typeName) > 0 {
			typeName = strings.ToUpper(typeName[:1]) + typeName[1:]
		}
		_, _ = fmt.Fprintf(r.w, "  - %s: %s\n", typeName, t.Path)
	}
	_, _ = fmt.Fprintln(r.w)
}

// UninstallCanceled prints the cancellation message when the user declines uninstallation.
func (r *Renderer) UninstallCanceled() {
	_, _ = fmt.Fprintln(r.w, "Uninstall canceled — no files were removed.")
}

// UninstallDone prints the uninstallation completion message.
func (r *Renderer) UninstallDone() {
	_, _ = fmt.Fprintln(r.w, "upp has been successfully uninstalled.")
}
