package cli

import (
	"context"
	"errors"
	"io"
	"os"

	"github.com/JhnFrankz/upp/internal/doctor"
	"github.com/JhnFrankz/upp/internal/output"
	"github.com/spf13/cobra"
)

type doctorDeps struct {
	diagnose func(ctx context.Context, deps doctor.DoctorDeps) []doctor.CheckResult
	deps     doctor.DoctorDeps
}

// NewDoctorCommand creates the `upp doctor` command.
func NewDoctorCommand(gf *GlobalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Run environment and configuration diagnostics",
		Long: "Inspect configuration validity, storage permissions, orphaned process locks, " +
			"package manager conflicts, active tool PATH shadowing, and upstream network connectivity.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor(cmd.Context(), gf, os.Stdout, cliDeps.doctor)
		},
	}
	return cmd
}

// NewDoctorCmd is an alias for NewDoctorCommand.
var NewDoctorCmd = NewDoctorCommand

func runDoctor(ctx context.Context, gf *GlobalFlags, out io.Writer, deps doctorDeps) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if deps.diagnose == nil {
		deps.diagnose = doctor.Diagnose
	}

	results := deps.diagnose(ctx, deps.deps)

	r := output.NewRendererVerbose(out, gf.Quiet, gf.Verbose)
	r.DoctorResults(results, gf.Quiet, gf.Verbose)

	if doctor.HasErrors(results) {
		return errors.New("doctor found issues that require attention")
	}

	return nil
}
