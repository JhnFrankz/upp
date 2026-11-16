package official

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
)

func TestWingetAndScoopCoalesceConcurrentQueries(t *testing.T) {
	t.Run("winget upgrade runs once across concurrent Check and CheckPackage calls", func(t *testing.T) {
		var upgradeCalls atomic.Int32
		setExecFakes(t, execFakes{
			lookPath: map[string]bool{"winget": true},
		})
		orig := runCmdArgsFn
		runCmdArgsFn = func(ctx context.Context, name string, args ...string) (string, string, error) {
			if name == "winget" && len(args) == 1 && args[0] == "--version" {
				return "v1.8.2301", "", nil
			}
			if name == "winget" && len(args) == 1 && args[0] == "upgrade" {
				upgradeCalls.Add(1)
				return "winget  Microsoft.AppInstaller  v1.8.2301  v1.8.2311  winget\ngh  GitHub.cli  2.45.0  2.46.0  winget\nDocker.Docker  Docker.Docker  4.28.0  4.29.0  winget\nGoLang.Go  GoLang.Go  1.22.0  1.22.1  winget\n", "", nil
			}
			return orig(ctx, name, args...)
		}

		w := &WingetAdapter{}
		var wg sync.WaitGroup
		wg.Add(4)
		go func() { defer wg.Done(); _, _ = w.Check(context.Background()) }()
		go func() { defer wg.Done(); _, _ = w.CheckPackage(context.Background(), "GitHub.cli") }()
		go func() { defer wg.Done(); _, _ = w.CheckPackage(context.Background(), "Docker.Docker") }()
		go func() { defer wg.Done(); _, _ = w.CheckPackage(context.Background(), "GoLang.Go") }()
		wg.Wait()

		if got := upgradeCalls.Load(); got != 1 {
			t.Errorf("winget upgrade invoked %d times, want 1", got)
		}
	})

	t.Run("scoop status runs once across concurrent Check and CheckPackage calls", func(t *testing.T) {
		var statusCalls atomic.Int32
		setExecFakes(t, execFakes{
			lookPath: map[string]bool{"scoop": true},
		})
		orig := runCmdArgsFn
		runCmdArgsFn = func(ctx context.Context, name string, args ...string) (string, string, error) {
			if name == "scoop" && len(args) == 1 && args[0] == "status" {
				statusCalls.Add(1)
				return "Name   Installed Version Latest Version Missing Dependencies Info\n----   ----------------- -------------- -------------------- ----\nscoop  0.4.0             0.4.1\ngh     2.45.0            2.46.0\n", "", nil
			}
			return orig(ctx, name, args...)
		}

		s := &ScoopAdapter{}
		var wg sync.WaitGroup
		wg.Add(3)
		go func() { defer wg.Done(); _, _ = s.Check(context.Background()) }()
		go func() { defer wg.Done(); _, _ = s.CheckPackage(context.Background(), "gh") }()
		go func() { defer wg.Done(); _, _ = s.CheckPackage(context.Background(), "docker") }()
		wg.Wait()

		if got := statusCalls.Load(); got != 1 {
			t.Errorf("scoop status invoked %d times, want 1", got)
		}
	})
}
