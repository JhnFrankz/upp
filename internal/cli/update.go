package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/config"
	"github.com/JhnFrankz/upp/internal/engine"
	"github.com/JhnFrankz/upp/internal/lock"
	"github.com/JhnFrankz/upp/internal/output"
	"github.com/JhnFrankz/upp/internal/platform"
)

// NewUpdateCommand creates the `upp update` command.
func NewUpdateCommand(gf *GlobalFlags) *cobra.Command {
	uf := &UpdateFlags{}

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Apply updates to enabled tools",
		Long:  "Process each enabled tool: detect, check, confirm, and update.",
		RunE: func(cmd *cobra.Command, args []string) error {
			d := cliDeps.update
			if d.in == nil {
				d.in = cmd.InOrStdin()
			}
			if d.out == nil {
				d.out = cmd.OutOrStdout()
			}
			if d.errOut == nil {
				d.errOut = cmd.ErrOrStderr()
			}
			return runUpdateContext(cmd.Context(), gf, uf, d)
		},
	}

	cmd.Flags().BoolVarP(&uf.DryRun, "dry-run", "n", false, "show planned actions without executing")

	return cmd
}

// updateDeps carries the injectable seams for runUpdate (design D5),
// mirroring selfUpdateDeps. The zero value uses production
// behavior: the production adapter list builder, real TTY detection, and
// the real CheckboxSelector.
type updateDeps struct {
	acquireLock      func() (*lock.Lock, error)
	buildAdapterList func(cfg *config.Config, osName string) []adapters.Adapter
	// stdinIsTTY reports whether stdin is a TTY — the interactive gate
	// (design D2: TTY && !ci && !quiet && !dry-run). Zero value = production
	// stdinIsTTY().
	stdinIsTTY func() bool
	// selector runs the pending-set checkbox selector (design D2/D9) and
	// returns the selected tool IDs plus whether the user canceled. Zero
	// value = production CheckboxSelector over os.Stdout/os.Stdin.
	selector func(pending []output.SelectOption) ([]string, bool)
	in       io.Reader
	out      io.Writer
	errOut   io.Writer
}

// UpdateSession encapsulates the execution context, configuration flags,
// output streams, engine, and adapter inventory for an update invocation.
type UpdateSession struct {
	Ctx         context.Context
	GlobalFlags *GlobalFlags
	UpdateFlags *UpdateFlags
	Renderer    *output.Renderer
	Engine      *engine.Engine
	AllAdapters []adapters.Adapter
	OSName      string
	In          io.Reader
	Out         io.Writer
	ErrOut      io.Writer
}

// NewUpdateSession creates a populated UpdateSession with normalized defaults.
func NewUpdateSession(ctx context.Context, gf *GlobalFlags, uf *UpdateFlags, r *output.Renderer, eng *engine.Engine, allAdapters []adapters.Adapter, osName string, in io.Reader, out io.Writer, errOut ...io.Writer) *UpdateSession {
	if ctx == nil {
		ctx = context.Background()
	}
	if gf == nil {
		gf = &GlobalFlags{}
	}
	if uf == nil {
		uf = &UpdateFlags{}
	}
	if in == nil {
		in = os.Stdin
	}
	if out == nil {
		out = os.Stdout
	}
	var eo io.Writer
	if len(errOut) > 0 && errOut[0] != nil {
		eo = errOut[0]
	} else {
		eo = os.Stderr
	}
	if r == nil {
		r = output.NewRendererVerbose(out, gf.Quiet, gf.Verbose)
	}
	if eng == nil {
		var opts []engine.Option
		if len(allAdapters) > 0 {
			opts = append(opts, engine.WithAdapters(allAdapters))
		}
		eng = engine.New(nil, osName, opts...)
	}
	return &UpdateSession{
		Ctx:         ctx,
		GlobalFlags: gf,
		UpdateFlags: uf,
		Renderer:    r,
		Engine:      eng,
		AllAdapters: allAdapters,
		OSName:      osName,
		In:          in,
		Out:         out,
		ErrOut:      eo,
	}
}

func runUpdate(gf *GlobalFlags, uf *UpdateFlags, deps updateDeps) error {
	return runUpdateContext(context.Background(), gf, uf, deps)
}

func runUpdateContext(ctx context.Context, gf *GlobalFlags, uf *UpdateFlags, deps updateDeps) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	if !uf.DryRun {
		acquireLock := deps.acquireLock
		if acquireLock == nil {
			acquireLock = defaultAcquireLock
		}
		l, err := acquireLock()
		if err != nil {
			return err
		}
		if l != nil {
			defer func() { _ = l.Release() }()
		}
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("cannot load config: %w", err)
	}

	p, err := platform.Detect()
	if err != nil {
		return fmt.Errorf("cannot detect platform: %w", err)
	}

	var allAdapters []adapters.Adapter
	var opts []engine.Option
	if deps.buildAdapterList != nil {
		allAdapters = deps.buildAdapterList(cfg, p.OS)
		opts = append(opts, engine.WithAdapters(allAdapters))
	}
	eng := engine.New(cfg, p.OS, opts...)
	if allAdapters == nil {
		var err error
		allAdapters, err = eng.Resolve(engine.Filter{})
		if err != nil {
			return fmt.Errorf("cannot resolve tools: %w", err)
		}
	}

	in := deps.in
	if in == nil {
		in = os.Stdin
	}
	out := deps.out
	if out == nil {
		out = os.Stdout
	}
	errOut := deps.errOut
	if errOut == nil {
		errOut = os.Stderr
	}
	deps.in = in
	deps.out = out
	deps.errOut = errOut

	onlyList := ParseFilter(gf.Only)
	if len(onlyList) > 0 {
		FilterTools(adapterIDs(allAdapters), onlyList, errOut)
	}

	var filteredAdapters []adapters.Adapter
	if len(onlyList) == 0 {
		filteredAdapters = allAdapters
	} else {
		filteredAdapters = engine.FilterAdapters(allAdapters, onlyList)
	}

	r := output.NewRendererVerbose(out, gf.Quiet, gf.Verbose)

	if uf.DryRun {
		r.DryRunHeader()
	}

	sess := NewUpdateSession(ctx, gf, uf, r, eng, allAdapters, p.OS, in, out, errOut)

	// Interactive gate (design D2): selector only when stdin is a TTY and
	// --ci, --quiet, and --dry-run are all unset. Any other combination keeps
	// sequential behavior.
	if deps.stdinIsTTY == nil {
		deps.stdinIsTTY = stdinIsTTY
	}
	if deps.stdinIsTTY() && !gf.CI && !gf.Quiet && !uf.DryRun {
		return sess.RunInteractive(deps, filteredAdapters)
	}

	return sess.RunSequential(filteredAdapters)
}
