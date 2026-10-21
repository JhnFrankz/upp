package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/adapters/official"
	"github.com/JhnFrankz/upp/internal/engine"
	"github.com/JhnFrankz/upp/internal/output"
	"github.com/JhnFrankz/upp/internal/platform"
	"github.com/JhnFrankz/upp/internal/security"
)

// enforceRiskFor reports whether a planned row's REAL command risk must decide
// the confirmation decision instead of the TrustOfficial auto-proceed shortcut.
//
// Interactive: any command above RiskLow must be confirmed, whoever ships it,
// so no official row ever runs a dangerous command unseen — apt's
// `sudo apt install --only-upgrade apt` prompts exactly like pacman's.
//
// --ci cannot prompt, so upp answers only for the commands it ships. An official
// adapter's command is a fixed, reviewed string inside the binary, so it
// proceeds; a custom command comes from the user's config and cannot be vouched
// for, so anything above RiskLow fails closed (spec security-model: `--ci` MUST
// fail high-risk custom updates even when trusted).
func enforceRiskFor(trust security.TrustLevel, risk security.RiskLevel, ci bool) bool {
	if risk == security.RiskLow {
		return false
	}
	return !ci || trust != security.TrustOfficial
}

// allAdaptersArg returns a slice suitable for passing to variadic ...[]adapters.Adapter functions.
func (s *UpdateSession) allAdaptersArg() [][]adapters.Adapter {
	if s.AllAdapters != nil {
		return [][]adapters.Adapter{s.AllAdapters}
	}
	return nil
}

// ExecutePlannedUpdate is the single executor shared by the sequential and
// interactive update paths. It preserves the exact execution parity between
// them: (1) --quiet suppression; (2) security risk classification against the
// planned command; (3) confirm prompt; (4) PackageUpdater delegation for owned
// tools; (5) timeout wrapping.
//
// The single source of truth for the update command is PlannedUpdate.Command
// (design D2 — declared equals planned). The single source of truth for risk
// is PlannedUpdate.RiskCommand (design D3 — custom check/update elevated risk
// prompts on both paths). Manager elevation prompts iff the manager's own
// Info() declares privileges (design D4 — plan sets ManagerID on a manager
// self-row iff it declares privileges, so pacman prompts while apt/brew/winget
// keep their byte-identical auto-proceed decision).
func (s *UpdateSession) ExecutePlannedUpdate(p engine.PlannedUpdate, a adapters.Adapter, index, total int) output.ToolResult {
	if s == nil {
		return output.ToolResult{
			Name:   p.ToolName,
			Status: output.StatusFailed,
			Error:  fmt.Errorf("update session is nil"),
		}
	}
	s.ensureDefaults()

	if a == nil {
		return output.ToolResult{
			Name:   p.ToolName,
			Status: output.StatusFailed,
			Error:  fmt.Errorf("adapter is nil"),
		}
	}

	info := a.Info()

	if !s.GlobalFlags.Quiet && total > 1 {
		s.Renderer.Progress("Updating", index, total, info.Name)
	}

	riskLevel := security.ClassifyCommand(p.RiskCommand)
	decision := security.ConfirmAction(security.ConfirmConfig{
		ToolName:    p.ToolName,
		TrustLevel:  p.Trust,
		RiskLevel:   riskLevel,
		Command:     p.RiskCommand,
		Privileges:  p.Privileges,
		CI:          s.GlobalFlags.CI,
		Quiet:       s.GlobalFlags.Quiet,
		Reader:      s.In,
		Writer:      s.Out,
		EnforceRisk: enforceRiskFor(p.Trust, riskLevel, s.GlobalFlags.CI),
	})

	switch decision {
	case security.ConfirmDeny:
		return output.ToolResult{
			Name:   p.ToolName,
			Status: output.StatusDenied,
		}
	case security.ConfirmError:
		return output.ToolResult{
			Name:   p.ToolName,
			Status: output.StatusFailed,
			Error:  fmt.Errorf("CI mode: elevated risk requires confirmation"),
		}
	}

	var result adapters.Result
	var updateErr error
	if p.ManagerID != "" && p.PackageName != "" {
		owner := resolveOwnerByID(p.ManagerID, s.allAdaptersArg()...)
		if updater, ok := owner.(adapters.PackageUpdater); ok {
			result, updateErr = updater.UpdatePackage(s.Ctx, p.PackageName)
		} else {
			result, updateErr = a.Update(s.Ctx, false)
		}
	} else {
		canonOS := s.OSName
		if norm, err := platform.NormalizeOS(s.OSName); err == nil {
			canonOS = norm
		}

		// Owned tools with a PackageUpdater manager delegate to
		// updater.UpdatePackage(pkg); standalone tools run their own Update.
		owner := engine.ResolvingOwner(a, canonOS, s.allAdaptersArg()...)
		hasManager := false
		if info.Manager != nil {
			hasManager = info.Manager[s.OSName] != "" || info.Manager[canonOS] != ""
		}
		if owner != nil && hasManager {
			if updater, ok := owner.(adapters.PackageUpdater); ok {
				pkg := engine.OwnedPackage(a, canonOS)
				if pkg != "" {
					result, updateErr = updater.UpdatePackage(s.Ctx, pkg)
				} else {
					result, updateErr = a.Update(s.Ctx, false)
				}
			} else {
				result, updateErr = a.Update(s.Ctx, false)
			}
		} else {
			result, updateErr = a.Update(s.Ctx, false)
		}
	}

	if updateErr != nil {
		errStderr := result.Stderr
		if errStderr == "" {
			errStderr = updateErr.Error()
		}
		errMsg := result.Error
		if errMsg == nil {
			if errors.Is(updateErr, context.DeadlineExceeded) {
				errMsg = engine.TimeoutErr(p.ToolName, "update", updateErr)
			} else {
				errMsg = fmt.Errorf("update failed")
			}
		} else {
			errMsg = engine.TimeoutErr(p.ToolName, "update", errMsg)
		}
		return output.ToolResult{
			Name:   p.ToolName,
			Status: output.StatusFailed,
			Error:  errMsg,
			Stderr: errStderr,
		}
	}

	if result.Success {
		ver := result.After
		if ver == "" && p.LatestVersion != "" {
			ver = p.LatestVersion
		}
		fromVer := p.CurrentVersion
		if fromVer == "" {
			fromVer = result.Before
		}
		if fromVer != "" && ver != "" && fromVer != ver && !strings.Contains(ver, "→") {
			ver = fmt.Sprintf("%s → %s", fromVer, ver)
		}
		return output.ToolResult{
			Name:    p.ToolName,
			Status:  output.StatusUpdated,
			Version: ver,
		}
	}

	errMsg := result.Error
	if errMsg == nil {
		errMsg = fmt.Errorf("update failed")
	}
	errStderr := result.Stderr
	if errStderr == "" {
		errStderr = errMsg.Error()
	}
	return output.ToolResult{
		Name:   p.ToolName,
		Status: output.StatusFailed,
		Error:  engine.TimeoutErr(p.ToolName, "update", errMsg),
		Stderr: errStderr,
	}
}

// executePlannedUpdate is a legacy adapter function wrapping UpdateSession.ExecutePlannedUpdate
// to preserve 100% backward compatibility with internal test suites.
func executePlannedUpdate(ctx context.Context, gf *GlobalFlags, p engine.PlannedUpdate, a adapters.Adapter, index, total int, r *output.Renderer, osName string, in io.Reader, out io.Writer, allAdapters ...[]adapters.Adapter) output.ToolResult {
	var all []adapters.Adapter
	if len(allAdapters) > 0 && allAdapters[0] != nil {
		all = allAdapters[0]
	}
	sess := NewUpdateSession(ctx, gf, nil, r, nil, all, osName, in, out)
	return sess.ExecutePlannedUpdate(p, a, index, total)
}

// resolveOwnerByID looks up an owner adapter by ID directly from allAdapters (if provided).
// When an explicit allAdapters slice is provided, it does not fall back to the official
// registry, preventing resurrecting managers that the user disabled in config.
func resolveOwnerByID(managerID string, allAdapters ...[]adapters.Adapter) adapters.Adapter {
	if len(allAdapters) > 0 && allAdapters[0] != nil {
		for _, a := range allAdapters[0] {
			if a != nil && (a.Info().ID == managerID || a.Name() == managerID) {
				return a
			}
		}
		return nil
	}
	return official.AdapterByName(managerID)
}
