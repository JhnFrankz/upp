package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/config"
	"github.com/JhnFrankz/upp/internal/doctor"
	"github.com/JhnFrankz/upp/internal/uninstall"
)

func TestCommandStreams_ListWritesToCobraOut(t *testing.T) {
	root, gf := BuildRoot()
	AddCommands(root, gf)

	var stdoutBuf, stderrBuf bytes.Buffer
	root.SetOut(&stdoutBuf)
	root.SetErr(&stderrBuf)
	root.SetArgs([]string{"list", "--only", "nonexistent-tool"})

	oldDeps := cliDeps.list
	cliDeps.list = listDeps{
		buildAdapterList: func(*config.Config, string) []adapters.Adapter {
			return nil
		},
	}
	defer func() { cliDeps.list = oldDeps }()

	if err := root.Execute(); err != nil {
		t.Fatalf("root.Execute(list) error: %v", err)
	}

	if stdoutBuf.Len() == 0 && stderrBuf.Len() == 0 {
		t.Error("expected output to be written to Cobra out/err buffers")
	}
}

func TestCommandStreams_DoctorWritesToCobraOut(t *testing.T) {
	root, gf := BuildRoot()
	AddCommands(root, gf)

	var stdoutBuf bytes.Buffer
	root.SetOut(&stdoutBuf)
	root.SetArgs([]string{"doctor"})

	oldDeps := cliDeps.doctor
	cliDeps.doctor = doctorDeps{
		diagnose: func(ctx context.Context, deps doctor.DoctorDeps) []doctor.CheckResult {
			return []doctor.CheckResult{
				{Category: "Test", Name: "Diagnostic", Status: doctor.SeverityOK, Message: "all good"},
			}
		},
	}
	defer func() { cliDeps.doctor = oldDeps }()

	if err := root.Execute(); err != nil {
		t.Fatalf("root.Execute(doctor) error: %v", err)
	}

	if !strings.Contains(stdoutBuf.String(), "Diagnostic") {
		t.Errorf("expected Cobra out buffer to contain diagnostic result, got: %q", stdoutBuf.String())
	}
}

func TestCommandStreams_UninstallWritesToCobraOut(t *testing.T) {
	root, gf := BuildRoot()
	AddCommands(root, gf)

	var stdoutBuf bytes.Buffer
	root.SetOut(&stdoutBuf)
	root.SetArgs([]string{"uninstall", "--dry-run", "-y"})

	oldDeps := cliDeps.uninstall
	cliDeps.uninstall = uninstallDeps{
		discover: func(execPath, configDir, cacheDir string) ([]uninstall.Target, error) {
			return nil, nil
		},
	}
	defer func() { cliDeps.uninstall = oldDeps }()

	if err := root.Execute(); err != nil {
		t.Fatalf("root.Execute(uninstall) error: %v", err)
	}

	if !strings.Contains(stdoutBuf.String(), "Dry run") {
		t.Errorf("expected Cobra out buffer to contain uninstall dry run header, got: %q", stdoutBuf.String())
	}
}
