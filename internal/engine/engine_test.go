package engine

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/config"
)

type dummyAdapter struct {
	name string
}

func (d *dummyAdapter) Name() string                                { return d.name }
func (d *dummyAdapter) Detect() bool                                { return true }
func (d *dummyAdapter) Check() (adapters.UpdateInfo, error)         { return adapters.UpdateInfo{}, nil }
func (d *dummyAdapter) Update(dryRun bool) (adapters.Result, error) { return adapters.Result{}, nil }
func (d *dummyAdapter) Info() adapters.ToolInfo {
	return adapters.ToolInfo{ID: d.name, Name: d.name}
}

func TestCalculateWorkerCount_Clamping(t *testing.T) {
	tests := []struct {
		numCPU   int
		expected int
	}{
		{numCPU: -1, expected: 4},
		{numCPU: 0, expected: 4},
		{numCPU: 1, expected: 4},
		{numCPU: 2, expected: 4},
		{numCPU: 3, expected: 4},
		{numCPU: 4, expected: 4},
		{numCPU: 6, expected: 6},
		{numCPU: 8, expected: 8},
		{numCPU: 9, expected: 8},
		{numCPU: 16, expected: 8},
		{numCPU: 64, expected: 8},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("cpu_%d", tt.numCPU), func(t *testing.T) {
			got := CalculateWorkerCount(tt.numCPU)
			if got != tt.expected {
				t.Errorf("CalculateWorkerCount(%d) = %d; want %d", tt.numCPU, got, tt.expected)
			}
		})
	}
}

func TestEngine_ConstructorDefaults(t *testing.T) {
	cfg := &config.Config{}
	eng := New(cfg, "linux")
	if eng == nil {
		t.Fatal("New returned nil")
	}
	if eng.cfg != cfg {
		t.Errorf("eng.cfg = %v, want %v", eng.cfg, cfg)
	}
	if eng.osName != "linux" {
		t.Errorf("eng.osName = %q, want %q", eng.osName, "linux")
	}
	expectedWorkers := CalculateWorkerCount(runtime.NumCPU())
	if eng.numWorkers != expectedWorkers {
		t.Errorf("eng.numWorkers = %d, want %d", eng.numWorkers, expectedWorkers)
	}
	if eng.adapters != nil {
		t.Errorf("eng.adapters = %v, want nil", eng.adapters)
	}
}

func TestEngine_WithConcurrency(t *testing.T) {
	cfg := &config.Config{}

	t.Run("valid concurrency", func(t *testing.T) {
		eng := New(cfg, "darwin", WithConcurrency(6))
		if eng.numWorkers != 6 {
			t.Errorf("eng.numWorkers = %d, want 6", eng.numWorkers)
		}
	})

	t.Run("zero concurrency clamped to 1", func(t *testing.T) {
		eng := New(cfg, "windows", WithConcurrency(0))
		if eng.numWorkers != 1 {
			t.Errorf("eng.numWorkers = %d, want 1", eng.numWorkers)
		}
	})

	t.Run("negative concurrency clamped to 1", func(t *testing.T) {
		eng := New(cfg, "linux", WithConcurrency(-2))
		if eng.numWorkers != 1 {
			t.Errorf("eng.numWorkers = %d, want 1", eng.numWorkers)
		}
	})
}

func TestEngine_WithAdapters(t *testing.T) {
	cfg := &config.Config{}
	dummyList := []adapters.Adapter{
		&dummyAdapter{name: "fake1"},
		&dummyAdapter{name: "fake2"},
	}

	eng := New(cfg, "linux", WithAdapters(dummyList))
	if len(eng.adapters) != 2 {
		t.Fatalf("len(eng.adapters) = %d, want 2", len(eng.adapters))
	}
	if eng.adapters[0].Name() != "fake1" || eng.adapters[1].Name() != "fake2" {
		t.Errorf("eng.adapters = %+v, unexpected items", eng.adapters)
	}
}

func TestEngine_HeadlessImportIsolation(t *testing.T) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("failed to parse directory: %v", err)
	}

	disallowed := []string{
		"github.com/JhnFrankz/upp/internal/output",
		"github.com/spf13/cobra",
		"github.com/charmbracelet/bubbletea",
	}

	for _, pkg := range pkgs {
		for filename, file := range pkg.Files {
			for _, imp := range file.Imports {
				path := strings.Trim(imp.Path.Value, `"`)
				for _, d := range disallowed {
					if strings.Contains(path, d) {
						t.Errorf("file %s imports forbidden package %q", filename, path)
					}
				}
			}
		}
	}
}

func TestCheckStatus_String(t *testing.T) {
	tests := []struct {
		status   CheckStatus
		expected string
	}{
		{StatusAvailable, "available"},
		{StatusCurrent, "current"},
		{StatusSkipped, "skipped"},
		{StatusFailed, "failed"},
		{CheckStatus(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.status.String(); got != tt.expected {
			t.Errorf("CheckStatus(%d).String() = %q, want %q", tt.status, got, tt.expected)
		}
	}
}
