package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/output"
	"github.com/JhnFrankz/upp/internal/security"
)

// unconsentedCheckAdapter delegates every Adapter method to the wrapped adapter
// except Check, which returns an empty result without running the declared
// check command.
//
// Authorization must NOT drop adapters from the check list: the sequential
// update path indexes outcomes by filtered-adapter position (update.go:231) and
// the interactive board indexes its labels the same way, so a shorter list
// desynchronizes both. Neutralizing Check keeps length and order intact.
type unconsentedCheckAdapter struct {
	adapters.Adapter
}

func (a unconsentedCheckAdapter) Check(ctx context.Context) (adapters.UpdateInfo, error) {
	return adapters.UpdateInfo{}, nil
}

// AuthorizeChecks runs check authorization on adapterList using the session's
// flags, renderer, and IO streams.
func (s *UpdateSession) AuthorizeChecks(adapterList []adapters.Adapter) ([]adapters.Adapter, error) {
	return authorizeChecks(s.GlobalFlags, adapterList, s.Renderer, s.In, s.Out)
}

// authorizeChecks neutralizes the check of every adapter whose declared check
// command needs consent that was not granted.
//
// A custom tool's check command is arbitrary shell, so it is gated by its real
// risk exactly like an update command (spec security-model: custom check-command
// gate). Interactive runs prompt through security.ConfirmAction; --ci denies
// with a non-zero error. A denial neutralizes only that check: the tool still
// reports as detected with an unknown version, and the run continues.
func authorizeChecks(gf *GlobalFlags, adapterList []adapters.Adapter, r *output.Renderer, in io.Reader, out io.Writer) ([]adapters.Adapter, error) {
	authorized := make([]adapters.Adapter, 0, len(adapterList))
	for _, a := range adapterList {
		inner := a
		if u, ok := a.(interface{ Unwrap() adapters.Adapter }); ok {
			inner = u.Unwrap()
		}

		// Official adapters never declare arbitrary custom check commands.
		// Only custom adapters can carry a shell check command requiring user consent.
		if _, isCustom := inner.(*adapters.CustomAdapter); !isCustom {
			if custom, ok := inner.(interface{ IsCustom() bool }); !ok || !custom.IsCustom() {
				authorized = append(authorized, a)
				continue
			}
		}

		// If a custom adapter's manager implements PackageChecker, Check()
		// delegates directly to the manager's CheckPackage without running
		// the custom tool's checkCmd shell command. Consent is not needed.
		if ca, ok := inner.(*adapters.CustomAdapter); ok {
			if ca.ManagerChecker() != nil {
				authorized = append(authorized, a)
				continue
			}
		}

		info := a.Info()
		if !security.CheckNeedsConsent(info.CheckCommand) {
			authorized = append(authorized, a)
			continue
		}

		decision := security.ConfirmAction(security.ConfirmConfig{
			ToolName:   info.Name,
			TrustLevel: info.Trust,
			RiskLevel:  security.ClassifyCommand(info.CheckCommand),
			Command:    info.CheckCommand,
			Privileges: info.Privileges,
			CI:         gf.CI,
			Quiet:      gf.Quiet,
			Reader:     in,
			Writer:     out,
		})

		switch decision {
		case security.ConfirmError:
			return nil, fmt.Errorf(
				"tool %q: check command requires confirmation and cannot run in --ci mode: %s",
				info.Name, info.CheckCommand)
		case security.ConfirmDeny:
			r.Warning(fmt.Sprintf("skipping version check for %q: check command not confirmed", info.Name))
			authorized = append(authorized, unconsentedCheckAdapter{Adapter: a})
		default: // ConfirmAuto, ConfirmProceed
			authorized = append(authorized, a)
		}
	}
	return authorized, nil
}
