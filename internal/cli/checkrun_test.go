package cli

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/engine"
	"github.com/JhnFrankz/upp/internal/output"
)

// fakeDelayedAdapter introduces a controllable delay to test concurrency &
// ordering. Shared with integration_test.go, which lives in the same package.
type fakeDelayedAdapter struct {
	name     string
	delay    time.Duration
	info     adapters.UpdateInfo
	checkErr error
}

func (f *fakeDelayedAdapter) Name() string { return f.name }
func (f *fakeDelayedAdapter) Detect() bool { return true }
func (f *fakeDelayedAdapter) Check(ctx context.Context) (adapters.UpdateInfo, error) {
	if f.delay > 0 {
		time.Sleep(f.delay)
	}
	return f.info, f.checkErr
}
func (f *fakeDelayedAdapter) Update(ctx context.Context, dryRun bool) (adapters.Result, error) {
	return adapters.Result{Success: true}, nil
}
func (f *fakeDelayedAdapter) Info() adapters.ToolInfo {
	return adapters.ToolInfo{
		ID:           f.name,
		Name:         f.name,
		Trust:        adapters.TrustOfficial,
		UpdatePolicy: adapters.PolicyGated,
	}
}

func TestOutcomeToToolResult(t *testing.T) {
	testErr := errors.New("check failed")
	tests := []struct {
		name     string
		input    engine.CheckOutcome
		expected output.ToolResult
	}{
		{
			name: "StatusAvailable",
			input: engine.CheckOutcome{
				ToolName:        "tool-a",
				Status:          engine.StatusAvailable,
				CurrentVersion:  "1.0.0",
				LatestVersion:   "2.0.0",
				UpdateAvailable: true,
			},
			expected: output.ToolResult{
				Name:    "tool-a",
				Status:  output.StatusAvailable,
				Version: "1.0.0 → 2.0.0",
			},
		},
		{
			name: "StatusCurrent",
			input: engine.CheckOutcome{
				ToolName:       "tool-c",
				Status:         engine.StatusCurrent,
				CurrentVersion: "1.5.0",
			},
			expected: output.ToolResult{
				Name:    "tool-c",
				Status:  output.StatusCurrent,
				Version: "1.5.0",
			},
		},
		{
			name: "StatusSkipped",
			input: engine.CheckOutcome{
				ToolName: "tool-s",
				Status:   engine.StatusSkipped,
			},
			expected: output.ToolResult{
				Name:   "tool-s",
				Status: output.StatusSkipped,
			},
		},
		{
			name: "StatusFailed",
			input: engine.CheckOutcome{
				ToolName: "tool-f",
				Status:   engine.StatusFailed,
				Err:      testErr,
				Stderr:   "error details",
			},
			expected: output.ToolResult{
				Name:   "tool-f",
				Status: output.StatusFailed,
				Error:  testErr,
				Stderr: "error details",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := outcomeToToolResult(tt.input)
			if got.Name != tt.expected.Name {
				t.Errorf("Name = %q, want %q", got.Name, tt.expected.Name)
			}
			if got.Status != tt.expected.Status {
				t.Errorf("Status = %v, want %v", got.Status, tt.expected.Status)
			}
			if got.Version != tt.expected.Version {
				t.Errorf("Version = %q, want %q", got.Version, tt.expected.Version)
			}
			if got.Error != tt.expected.Error {
				t.Errorf("Error = %v, want %v", got.Error, tt.expected.Error)
			}
			if got.Stderr != tt.expected.Stderr {
				t.Errorf("Stderr = %q, want %q", got.Stderr, tt.expected.Stderr)
			}
		})
	}
}
