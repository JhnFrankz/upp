package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/config"
	"github.com/JhnFrankz/upp/internal/doctor"
	"github.com/JhnFrankz/upp/internal/lock"
	"github.com/JhnFrankz/upp/internal/security"
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

func TestCommandStreams_UpdateWritesToCobraOutAndReadsFromCobraIn(t *testing.T) {
	root, gf := BuildRoot()
	AddCommands(root, gf)

	var stdoutBuf, stderrBuf bytes.Buffer
	stdinBuf := strings.NewReader("y\n")
	root.SetIn(stdinBuf)
	root.SetOut(&stdoutBuf)
	root.SetErr(&stderrBuf)
	root.SetArgs([]string{"update"})

	fakeTool := &fakeUpdateAdapter{
		name:       "stream-tool",
		trust:      security.TrustCustomUntrusted,
		command:    "sudo stream-tool --update",
		privileges: []string{"sudo"},
		info: adapters.UpdateInfo{
			CurrentVersion:  "1.0.0",
			LatestVersion:   "2.0.0",
			UpdateAvailable: true,
		},
		result: adapters.Result{
			Success: true,
			Before:  "1.0.0",
			After:   "2.0.0",
		},
	}

	oldDeps := cliDeps.update
	cliDeps.update = updateDeps{
		acquireLock: func() (*lock.Lock, error) { return nil, nil },
		buildAdapterList: func(cfg *config.Config, osName string) []adapters.Adapter {
			return []adapters.Adapter{fakeTool}
		},
		stdinIsTTY: func() bool { return false },
	}
	defer func() { cliDeps.update = oldDeps }()

	if err := root.Execute(); err != nil {
		t.Fatalf("root.Execute(update) error: %v", err)
	}

	outStr := stdoutBuf.String()
	if !strings.Contains(outStr, "Proceed? [y/N]") {
		t.Errorf("expected Cobra out buffer to contain confirmation prompt, got: %q", outStr)
	}
	if !strings.Contains(outStr, "Command: sudo stream-tool --update") {
		t.Errorf("expected Cobra out buffer to contain command detail, got: %q", outStr)
	}
	if !fakeTool.updated {
		t.Error("expected tool to be updated after confirmation read from Cobra in")
	}
}

func TestCommandStreams_UpdateQuietSuppressesPrintInfo(t *testing.T) {
	root, gf := BuildRoot()
	AddCommands(root, gf)

	var stdoutBuf, stderrBuf bytes.Buffer
	root.SetOut(&stdoutBuf)
	root.SetErr(&stderrBuf)
	root.SetArgs([]string{"update", "--quiet"})

	fakeTool := &fakeUpdateAdapter{
		name:    "quiet-tool",
		trust:   security.TrustCustomTrusted,
		command: "quiet-tool --update",
		info: adapters.UpdateInfo{
			CurrentVersion:  "1.0.0",
			LatestVersion:   "2.0.0",
			UpdateAvailable: true,
		},
		result: adapters.Result{
			Success: true,
			Before:  "1.0.0",
			After:   "2.0.0",
		},
	}

	oldDeps := cliDeps.update
	cliDeps.update = updateDeps{
		acquireLock: func() (*lock.Lock, error) { return nil, nil },
		buildAdapterList: func(cfg *config.Config, osName string) []adapters.Adapter {
			return []adapters.Adapter{fakeTool}
		},
		stdinIsTTY: func() bool { return false },
	}
	defer func() { cliDeps.update = oldDeps }()

	if err := root.Execute(); err != nil {
		t.Fatalf("root.Execute(update --quiet) error: %v", err)
	}

	outStr := stdoutBuf.String()
	if strings.Contains(outStr, "Command: quiet-tool --update") {
		t.Errorf("expected quiet mode to suppress printInfo in update, got: %q", outStr)
	}
}
