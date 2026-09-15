# Delta for Tool Adapter Interface

## MODIFIED Requirements

### Requirement: Adapter Interface

Every adapter MUST implement four operations:

- `detect() → bool` — is this tool installed?
- `check() → UpdateInfo` — current version + latest available + update available?
- `update() → Result` — perform the update, return success/failure + details
- `list() → ToolInfo` — return installed tool info (name, version, source, owning manager, kind)

`ToolInfo` MUST carry an owning `Manager` map keyed by platform and a `Kind` (`KindManager` for manager adapters, `KindTool` otherwise). A tool with a resolving owner on the current platform reports that manager; a tool with no owner reports no manager. A `ToolInfo` whose `Kind=KindTool` and that has a resolving owner on the current platform MUST also declare a per-manager package-name entry (see Per-Manager Package Mapping), so the owned tool's package under its manager is known.

Every manager adapter MUST declare its real update commands — the manager self-update command and the per-package update command — including any privilege elevation, and MUST declare the privileges each command requires. The pacman adapter MUST declare `Privileges: ["sudo"]`. Plan metadata consumed by the confirmation gate and transparency display (`RiskCommand`, plan privileges) MUST be derived from these declarations and MUST byte-equal the command actually executed; the system MUST NOT synthesize plan commands from templates (e.g. `<manager> upgrade <pkg>`). A custom tool whose update delegates to its manager MUST inherit the manager's real declared command and privileges as its own for confirmation and display.

(Previously: adapters had no declared real-update-command surface; the engine synthesized plan commands (`UpdateCmdName`, e.g. `<manager> upgrade <pkg>`), which diverged from execution for pacman (`sudo pacman -S --noconfirm <pkg>` executed, sudo-free string planned) and for custom manager-delegated tools.)

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Tool installed | `nvm` is on PATH | `detect()` called | Returns `true` |
| Tool missing | `nvm` not on PATH | `detect()` called | Returns `false` |
| Update available | Node.js v18 installed, v20 latest | `check()` called | `update_available=true`, versions returned |
| No update | Node.js v20 installed, v20 latest | `check()` called | `update_available=false` |
| Update succeeds | Update command exits 0 | `update()` called | `success=true`, before/after versions returned |
| Update fails | Update command exits non-zero | `update()` called | `success=false`, error message returned |
| ToolInfo carries owner | docker installed on Linux | `list()` called | `ToolInfo` includes `Manager["linux"]="apt"`, `Kind=KindTool` |
| Manager carries kind | apt installed on Linux | `list()` called | `ToolInfo.Kind=KindManager` |
| Owned tool carries package | gh owned by apt on Linux | `list()` called | `ToolInfo` declares package `gh` under `Manager["linux"]="apt"` |
| pacman declares sudo | pacman adapter on Linux | Adapter declaration read | Declares `Privileges=["sudo"]` and the per-package command `sudo pacman -S --noconfirm <pkg>` |
| Declared equals executed | pacman package row planned | Plan `RiskCommand` compared against the command the pacman update path executes | Byte-identical command; identical privileges |
| apt/brew/winget byte-stable | apt, brew, winget rows planned | Plan `RiskCommand` compared against executed commands | Byte-identical to the commands executed today (no prompt churn) |
| Custom delegated inherits declaration | Custom tool with `manager = "pacman"` | Plan built for the row | `RiskCommand` and privileges equal the manager's real self-update declaration (`sudo pacman -S --noconfirm pacman`, sudo) |

(Previously: `list() → ToolInfo` returned only name/version/source; `ToolInfo` had no `Manager` or `Kind` field, and no per-manager package-name field existed.)
