# Delta for tool-adapter

## ADDED Requirements

### Requirement: Custom Adapter Privileges & Execution

Custom adapters MUST analyze configured command strings statically for privilege escalation tokens (e.g., `sudo`, `doas`, `runas`, `admin`). Custom adapter `Update(dryRun)` MUST populate `Result.Privileges` with detected privileges for both dry-run (`dryRun=true`) and live execution (`dryRun=false`).

Custom adapter `Check()` and `Update()` MUST fail closed with a structured error when the base command executable is not detected on PATH (`Detect() == false`), avoiding unnecessary subshell invocations. When the base binary is present on PATH:
- `Update(true)` MUST return a success `Result` with before/after command strings and detected privileges without invoking any subprocess.
- `Update(false)` MUST execute the update command via the platform shell bounded by timeout and return the execution `Result` with detected privileges.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Dry-run with sudo | Custom tool configured with `sudo apt upgrade` and binary present on PATH | `Update(dryRun=true)` called | Returns success `Result` with `Privileges=["sudo"]`, before/after set, no subprocess spawned |
| Live update with sudo | Custom tool configured with `sudo apt upgrade` and binary present on PATH | `Update(dryRun=false)` called | Executes command via shell, returns `Result` with `Privileges=["sudo"]` |
| Missing binary on check | Custom tool whose base command binary is missing from PATH (`Detect() == false`) | `Check()` called | Fails closed with structured error without invoking check subshell |
| Missing binary on update | Custom tool whose base command binary is missing from PATH (`Detect() == false`) | `Update(dryRun=false)` called | Fails closed returning `Result` with `Success=false` and structured error without invoking update subshell |
| Present binary executes | Custom tool with base binary present on PATH | `Update(dryRun=false)` called | Executes shell command bounded by timeout, returns exit status and detected privileges |
| Present binary dry-run | Custom tool with base binary present on PATH | `Update(dryRun=true)` called | Returns preview `Result` with `Success=true`, before/after commands, and detected privileges |
