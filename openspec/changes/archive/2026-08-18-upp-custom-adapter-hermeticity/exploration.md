# Exploration: Hermetic CustomAdapter Execution & Privileges Consistency

## 1. Context & Motivation

During test suite verification, running unit tests in `internal/adapters/` revealed a critical flaw:
- `TestCustomAdapter_Privileges` hangs for up to 10 minutes (the default `adapters.UpdateTimeout`) because `ca.Update(false)` directly invokes `shellExec("sudo mytool --update")`, which launches an interactive `sudo` subprocess. In non-interactive test and CI environments, `sudo` blocks waiting for tty credentials.
- Furthermore, `CustomAdapter.Update(true)` (dry-run mode) returns a `Result` struct where `Privileges` is omitted (`nil`), whereas `CustomAdapter.Update(false)` and `CustomAdapter.Info()` both populate `Privileges` via `detectPrivileges(c.command)`.
- Multiple tests in `internal/adapters/custom_test.go` spawn real operating system subshells (`echo`, `exit 1`, `sleep 2`, writing executable scripts into `$PATH`), causing them to be skipped on Windows (`runtime.GOOS == "windows"`) and violating hermetic testing guarantees.

This exploration analyzes how to decouple subprocess execution in `CustomAdapter` via a mockable test seam (mirroring the proven pattern in `internal/adapters/official/`), restore 100% test hermeticity across all platforms, and ensure `Privileges` consistency across dry-run and live updates.

---

## 2. Root Cause Analysis

### 2.1 Test Execution Hang on `sudo`
In [`internal/adapters/custom.go`](file:///home/jhan/projects/upp/internal/adapters/custom.go#L66-L90):
```go
func (c *CustomAdapter) Update(dryRun bool) (Result, error) {
    if dryRun {
        return Result{
            Success: true,
            Before:  c.command,
            After:   c.command,
        }, nil
    }

    privileges := detectPrivileges(c.command)

    _, err := shellExec(c.command)
    if err != nil {
        return Result{
            Success:    false,
            Error:      fmt.Errorf("update command failed for %s: %w", c.id, err),
            Privileges: privileges,
        }, nil
    }

    return Result{
        Success:    true,
        Privileges: privileges,
    }, nil
}
```
In [`internal/adapters/custom_test.go`](file:///home/jhan/projects/upp/internal/adapters/custom_test.go#L197-L209):
```go
func TestCustomAdapter_Privileges(t *testing.T) {
    ca, err := NewCustomAdapter("mytool", "sudo mytool --update", "", false)
    if err != nil {
        t.Fatal(err)
    }
    result, err := ca.Update(false)
    if err != nil {
        t.Fatal(err)
    }
    if len(result.Privileges) == 0 || result.Privileges[0] != "sudo" {
        t.Errorf("Update() Privileges = %v, want [sudo]", result.Privileges)
    }
}
```
1. `TestCustomAdapter_Privileges` calls `ca.Update(false)`.
2. `ca.Update(false)` invokes `shellExec("sudo mytool --update")` -> `shellExecWithTimeout("sudo mytool --update", 10*time.Minute)` -> `RunCommandWithTimeout`.
3. An actual subshell (`sh -c "sudo mytool --update"`) is started.
4. `sudo` blocks on stdin/tty waiting for authentication credentials.
5. Because `UpdateTimeout` is 10 minutes, the test process hangs for 10 minutes before the context deadline terminates it.

### 2.2 Privileges Omission on Dry-Run
In [`CustomAdapter.Update`](file:///home/jhan/projects/upp/internal/adapters/custom.go#L67-L73):
- When `dryRun == true`, `Update` returns early before calling `detectPrivileges(c.command)`.
- The returned `Result` contains `Success: true, Before: c.command, After: c.command`, but `Privileges` is left uninitialized (`nil`).
- In contrast:
  - `CustomAdapter.Info()` returns `Privileges: detectPrivileges(c.command)`.
  - `CustomAdapter.Update(false)` (on both failure and success) returns `Privileges: detectPrivileges(c.command)`.
- This creates an inconsistency: dry-run inspection cannot report required privileges, and testing privilege detection via dry-run was prevented.

### 2.3 Non-Hermetic Test Suite in `custom_test.go`
The following tests in `custom_test.go` currently rely on the host system shell and environment:
- `TestCustomAdapter_Detect_Found` / `TestCustomAdapter_Detect_NotFound`: calls real `exec.LookPath`.
- `TestCustomAdapter_Check_WithCheckCmd`: executes `sh -c "echo 1.2.3"`.
- `TestCustomAdapter_Update_Execute`: executes `sh -c "echo updated"`.
- `TestCustomAdapter_Update_Failure`: executes `sh -c "exit 1"`.
- `TestCustomAdapter_Privileges`: executes `sh -c "sudo mytool --update"`.
- `TestShellExec`: executes `sh -c "echo hello"`.
- `TestCustomAdapter_Detect_WithRealCommand`: creates temporary executable files and mutates `PATH`.
- Six tests contain `if runtime.GOOS == "windows" { t.Skip(...) }`, leaving custom adapter behavior untested on Windows.

---

## 3. Comparison with Official Adapters Architecture

The `internal/adapters/official` package previously faced the same challenge and resolved it with a clean package-level seam:
1. **Package Seam Variables** in [`internal/adapters/official/helper.go`](file:///home/jhan/projects/upp/internal/adapters/official/helper.go#L22-L58):
   - `runCmdFn = func(command string) (stdout, stderr string, err error)`
   - `runCmdArgsFn = func(name string, args ...string) (stdout, stderr string, err error)`
   - `lookPathFn = func(name string) bool`
   - Production leaf functions (`runCmd`, `runCmdArgs`, `lookPath`, `commandOutput`, `shellOutput`) delegate to these variables.
2. **Test Fake Injection** in [`internal/adapters/official/exec_mock_test.go`](file:///home/jhan/projects/upp/internal/adapters/official/exec_mock_test.go#L28-L61):
   - `setExecFakes(t *testing.T, f execFakes)` swaps the function pointers for the duration of the test and restores them in `t.Cleanup`.
3. **Outcome**:
   - Over 400 official adapter test cases run in **< 1 second**.
   - Zero real subprocesses spawned during adapter unit tests.
   - Tests execute identically across Linux, macOS, and Windows.

---

## 4. Proposed Design & Seam Architecture

### 4.1 Adapters Execution Seam (`internal/adapters`)

Introduce package-level seam variables in `internal/adapters/custom.go` (or `internal/adapters/exec.go`):

```go
var (
    shellExecWithTimeoutFn = func(command string, timeout time.Duration) (string, error) {
        return defaultShellExecWithTimeout(command, timeout)
    }
    lookPathFn = func(file string) bool {
        _, err := exec.LookPath(file)
        return err == nil
    }
)
```

Where `defaultShellExecWithTimeout` is the current implementation utilizing `RunCommandWithTimeout`.

Production calls in `CustomAdapter` delegate to the seam:
- `c.Detect()` -> `lookPathFn(base)`
- `c.Check()` -> `shellExecWithTimeoutFn(c.checkCmd, CheckTimeout)`
- `c.Update()` -> `shellExecWithTimeoutFn(c.command, UpdateTimeout)`

### 4.2 Privileges Consistency in `CustomAdapter.Update`

Compute `privileges` upfront before checking `dryRun`:

```go
func (c *CustomAdapter) Update(dryRun bool) (Result, error) {
    privileges := detectPrivileges(c.command)
    if dryRun {
        return Result{
            Success:    true,
            Before:     c.command,
            After:      c.command,
            Privileges: privileges,
        }, nil
    }

    _, err := shellExec(c.command)
    if err != nil {
        return Result{
            Success:    false,
            Error:      fmt.Errorf("update command failed for %s: %w", c.id, err),
            Privileges: privileges,
        }, nil
    }

    return Result{
        Success:    true,
        Privileges: privileges,
    }, nil
}
```

### 4.3 Test Harness for `internal/adapters`

Create `internal/adapters/exec_mock_test.go` (or helper in `custom_test.go`):

```go
type shellFakeResult struct {
    stdout string
    err    error
}

type execFakes struct {
    shell    map[string]shellFakeResult
    lookPath map[string]bool
}

func setExecFakes(t *testing.T, f execFakes) {
    t.Helper()
    origShell := shellExecWithTimeoutFn
    origLookPath := lookPathFn

    shellExecWithTimeoutFn = func(command string, timeout time.Duration) (string, error) {
        if r, ok := f.shell[command]; ok {
            return r.stdout, r.err
        }
        return "", fmt.Errorf("unexpected command in test: %s", command)
    }
    lookPathFn = func(file string) bool {
        return f.lookPath[file]
    }

    t.Cleanup(func() {
        shellExecWithTimeoutFn = origShell
        lookPathFn = origLookPath
    })
}
```

### 4.4 Refactoring `custom_test.go` for 100% Hermeticity

| Existing Test | Current Issue | Hermetic Refactoring |
|---|---|---|
| `TestCustomAdapter_Detect_Found` | Calls real `exec.LookPath("echo")` | Use fake `lookPath: {"mytool": true}` |
| `TestCustomAdapter_Detect_NotFound` | Calls real `exec.LookPath("nonexistent")` | Use fake `lookPath: {"nonexistent": false}` |
| `TestCustomAdapter_Check_WithCheckCmd` | Runs real `echo 1.2.3`, skips Windows | Use fake `shell: {"mytool --version": {stdout: "1.2.3"}}`, enable cross-platform |
| `TestCustomAdapter_Update_DryRun` | Does not check `Result.Privileges` | Assert `Privileges: ["sudo"]` on `"sudo mytool --update"` in dry-run |
| `TestCustomAdapter_Update_Execute` | Runs real `echo updated`, skips Windows | Use fake `shell: {"mytool --update": {stdout: "ok"}}`, enable cross-platform |
| `TestCustomAdapter_Update_Failure` | Runs real `exit 1`, skips Windows | Use fake `shell: {"fail": {err: errors.New("exit status 1")}}`, enable cross-platform |
| `TestCustomAdapter_Privileges` | Hangs 10m on real `sudo`, skips Windows | Table test exercising `dryRun: true` and `dryRun: false` with fake runner returning `[sudo]` |
| `TestCustomAdapter_Detect_WithRealCommand` | Creates files in tempdir, mutates `$PATH` | Replaced by table-driven hermetic lookPath fake test |
| `TestShellExec` | Runs real `echo hello`, skips Windows | Unit test testing `shellExec` delegation to `shellExecWithTimeoutFn` |
| `TestShellExec_UpdateTimeoutKills` / `TestCustomAdapter_Check_CheckTimeoutKills` | Runs real `sleep 2` | Keep as fast (100ms) integration test for timeout kill mechanics or test fake propagation |

---

## 5. Risk Assessment & Validation Strategy

1. **Risk of Regressions**:
   - Production execution behavior (`defaultShellExecWithTimeout` and `exec.LookPath`) remains strictly identical. Seam variables only get swapped in tests via `t.Cleanup`.
   - Backward compatibility: `NewCustomAdapter` signature and `Adapter` interface methods remain 100% unchanged.
2. **Privileges Contract**:
   - `Result.Privileges` is now guaranteed to match `ToolInfo.Privileges` in both dry-run and live executions.
3. **Verification Steps**:
   - Run `go test -v -timeout 5s ./internal/adapters` -> All tests must pass in < 50ms without timeout.
   - Run `go test ./...` -> Full repo tests pass hermetically across Linux, macOS, and Windows.
