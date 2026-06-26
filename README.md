# TaskBridge

<div align="center">

**CLI command center for all Todo apps**

</div>

---

TaskBridge brings Microsoft Todo, Todoist, Feishu Tasks, TickTick, Dida365, Google Tasks, and local tasks into one CLI daily hub. The main path is pull every authenticated provider, inspect today's work/life split, choose the next bounded recommendation, then write back only through explicit sync or action-file confirmation.

```bash
taskbridge sync pull --all
taskbridge today
taskbridge next
taskbridge review
```

## Local Workflow

### Installation

**Homebrew cask (macOS / Linux):**

```bash
brew tap yeisme/tap
brew install --cask taskbridge
```

**Scoop (Windows):**

```powershell
scoop bucket add yeisme https://github.com/yeisme/scoop-bucket
scoop install taskbridge
```

**GitHub Release archives:** Download the archive for your platform from [Releases](https://github.com/yeisme/taskbridge/releases), verify it with `checksums.txt`, unpack it, and place `taskbridge` on your `PATH`.

**Direct Linux package assets:** Download `.deb`, `.rpm`, or `.apk` files from [Releases](https://github.com/yeisme/taskbridge/releases). These are direct release assets, not repository-backed APT, YUM, or APK channels.

**Go install:**

```bash
go install github.com/yeisme/taskbridge@latest
```

**Build from source:**

```bash
git clone https://github.com/yeisme/taskbridge.git
cd taskbridge
task build
```

Package installers install only the `taskbridge` binary and static release metadata. They do not create provider credentials, OAuth tokens, local task stores, or remote Todo writes.

### Configuration

```bash
export TASKBRIDGE_HOME=~/.taskbridge
export TASKBRIDGE_STORAGE_PATH=~/.taskbridge/data
export TASKBRIDGE_PROVIDERS=microsoft,todoist
```

For first use, run diagnostics first, then try the demo:

```bash
taskbridge doctor
taskbridge quickstart
taskbridge demo today
```

After authenticating providers, use the daily hub path:

```bash
taskbridge sync pull --all
taskbridge today
taskbridge next
```

### Provider Authentication

```bash
taskbridge auth status
taskbridge auth login microsoft
taskbridge auth login google
taskbridge auth login feishu
taskbridge auth login ticktick
taskbridge auth login dida
taskbridge auth login todoist
taskbridge provider list
taskbridge provider enable todoist
taskbridge provider test todoist
```

For the detailed Provider connection guide, see [docs/provider-setup-guide.md](docs/provider-setup-guide.md). OAuth credential files are saved under `~/.taskbridge/credentials/<provider>_credentials.json`, and authentication tokens are saved in `~/.taskbridge/credentials/tokens.json`.

### Task Browsing and Management

```bash
taskbridge list
taskbridge list --all
taskbridge list --source microsoft --status todo
taskbridge list --query "today"
taskbridge list --format json
taskbridge lists
taskbridge lists --source microsoft
```

```bash
taskbridge task add "Organize OpenSpec output contract" --due 2026-06-10 --priority 3
taskbridge task show <task-id> --format json
taskbridge task edit <task-id> --due 2026-06-12
taskbridge task done <task-id>
taskbridge task undo <task-id>
```

`list` supports filtering by source, status, quadrant, priority, tag, list, and keyword; output formats include `table`, `json`, `markdown`, `compact`, and `tsv`. `task` targets the local store and does not write directly to remote Providers; remote synchronization must go through `sync`.

### Sync

```bash
taskbridge sync pull microsoft
taskbridge sync pull --all
taskbridge sync pull --all --dry-run --json
taskbridge sync push todoist
taskbridge sync bidirectional microsoft --dry-run
taskbridge sync status
taskbridge sync diff microsoft --target todoist --format json
taskbridge sync conflicts
taskbridge sync resolve <conflict-id>
taskbridge sync backup create
taskbridge sync backup restore <backup-id>
taskbridge sync audit <session-id> --format json
taskbridge sync watch microsoft --interval 10m
```

`sync pull` writes local storage; `sync push` writes remote Providers; `bidirectional` writes both ways. `--dry-run` does not write local storage or call remote write APIs. Remote overwrite, deletion, and conflict discard require explicit confirmation.

### Daily control plane

```bash
taskbridge today
taskbridge today --json
taskbridge next
taskbridge next --limit 3
taskbridge next --source openspec
taskbridge inbox
taskbridge inbox --limit 10 --source todoist
taskbridge review
taskbridge review --json
taskbridge review --apply-file actions.json --dry-run
taskbridge review --apply-file actions.json --confirm
```

New users can explore without any provider authentication:

```bash
taskbridge demo today
taskbridge demo today --json
```

`today` is the daily task workbench: must-do today, at-risk tasks, and suggested next steps in one view. `next` recommends the most valuable task to advance now. `inbox` lists tasks without a home, due date, or triage status. `review` runs a health check and suggests actions; by default it only suggests, never writes.


### Project Planning

```bash
taskbridge project create "Learn OpenClaw"
taskbridge project create "Ship TaskBridge control plane" --goal-text "Complete the four control-plane phases"
taskbridge project list
taskbridge project split <project-id> --max-tasks 10
taskbridge project split-markdown <project-id> --file plan.md
taskbridge project confirm <project-id>
taskbridge project sync <project-id>
taskbridge project review <project-id>
taskbridge project next <project-id>
taskbridge project adjust <project-id>
taskbridge project done <project-id>
taskbridge project archive <project-id>
```

`project create` creates a draft; `split` generates decomposition suggestions; `confirm` confirms and creates local tasks in storage; `sync` syncs to Providers. `adjust` is dry-run by default and requires confirmation before applying actions. `archive` does not delete historical data.

### Governance and Smart Assistance

```bash
taskbridge governance overdue-health --format json
taskbridge governance resolve-overdue --dry-run
taskbridge governance resolve-overdue --confirm
taskbridge governance rebalance-longterm --format json
taskbridge governance detect-decomposition --limit 10 --format json
taskbridge governance decompose-task <task-id> --format json
taskbridge governance decompose-task <task-id> --write-tasks
taskbridge governance achievement
```

`overdue-health` analyzes overdue task health; `resolve-overdue` handles overdue tasks in bulk; `rebalance-longterm` redistributes long-term unscheduled tasks; `detect-decomposition` identifies complex task candidates; `decompose-task` splits a task into execution steps; `achievement` analyzes completion. Bulk completion, bulk rescheduling, deletion, and remote overwrite must have a dry-run/confirm/action-file gate.

### Analysis

```bash
taskbridge analyze quadrant
taskbridge analyze priority
taskbridge analyze priority --json
taskbridge analyze time
taskbridge analyze trend
taskbridge analyze report --json
```

`analyze` provides quadrant, priority, time distribution, trend, and combined report analysis. Default human output uses the shared stats panel; `--json` outputs an envelope, while legacy `--format json` remains compatible with parseable payloads. These are read-only commands: they do not change tasks or sync Providers.

### Agent Integration

```bash
taskbridge agent capabilities
taskbridge agent today
taskbridge agent plan "Learn OpenClaw" --dry-run
taskbridge agent execute --action-file actions.json --dry-run
taskbridge agent execute --action-file actions.json --confirm
taskbridge agent schemas
```

`agent` is the Agent-safe execution entry point. stdout is always `taskbridge.agent-result.v1` JSON. Agents do not directly read or write `~/.taskbridge` data files and do not hold Provider tokens. Dangerous actions return `requires_confirmation=true` without writing when `--confirm` is missing.

Regular commands should prefer the `--json` envelope or `--agent` key=value output, such as `taskbridge version --json`, `taskbridge provider list --agent`, and `taskbridge config show --json`; Agent scripts should not parse human output.

### Background Service

```bash
taskbridge serve
taskbridge serve --token-refresh
taskbridge serve --sync --sync-interval 15m
```

`serve` starts a long-running background service for automatic token refresh and scheduled sync. Logs go to stderr or the logging system; stdout follows the JSON/events contract.

### Interactive Terminal

```bash
taskbridge tui
```

Starts an interactive terminal interface for browsing and operating on tasks. The TUI is not a scriptable machine-output entry point; Agents and CI should not depend on TUI output.

## Supported Platforms

| Platform        | Status    | Auth Method    | Notes            |
| --------------- | --------- | -------------- | ---------------- |
| Microsoft Todo  | ✅ Done   | OAuth 2.0      | Full support     |
| Google Tasks    | ✅ Done   | OAuth 2.0      | Basic support    |
| Feishu Tasks    | ✅ Done   | OAuth 2.0      | Full support     |
| TickTick        | ✅ Done   | OpenAPI Token  | Native quadrants |
| Dida            | ✅ Done   | OpenAPI Token  | China version    |
| Todoist         | ✅ Done   | API Token      | Full support     |
| OmniFocus       | 📋 Planned | —              | macOS only       |
| Apple Reminders | 📋 Planned | —              | macOS/iOS        |

## Project Structure

```text
taskbridge/
├── cmd/                    # CLI command entry points
├── internal/
│   ├── auth/               # Tokens and authentication
│   ├── model/              # Core data models
│   ├── project/            # Project and planning storage
│   ├── projectplanner/     # Goal decomposition and plan suggestions
│   ├── provider/           # Todo app adapters
│   │   ├── microsoft/      # Microsoft Todo
│   │   ├── google/         # Google Tasks
│   │   ├── feishu/         # Feishu Tasks
│   │   ├── ticktick/       # TickTick / Dida
│   │   └── todoist/        # Todoist
│   ├── storage/            # Storage layer
│   └── sync/               # Sync engine
├── pkg/
│   ├── config/             # Configuration management
│   ├── logger/             # Logging
│   ├── paths/              # Path conventions
│   └── ui/                 # CLI/TUI UI components
├── docs/                   # Design and usage docs
│   └── commands/           # Command manual
└── openspec/               # OpenSpec change management
```

## Tech Stack

- **Language**: Go 1.25+
- **CLI**: Cobra
- **Configuration**: Viper
- **Storage**: File storage / MongoDB (optional)

## Local Validation and Development Tools

```bash
task deps
task fmt-check
task lint
task test
task build
task check
```

When setting up a development machine for the first time, install helper tools:

```bash
task tools:install
```

Release configuration and local snapshot builds:

```bash
task release:check
task snapshot
task release:local
```

Optional hot reload (requires Air):

```bash
task dev:watch
```

When `task` is not installed, use equivalent commands:

```bash
go test ./...
mkdir -p dist && go build -trimpath -ldflags="-s -w" -o dist/taskbridge
```

## Documentation Entry Points

- [Subproject instructions](./AGENTS.md)
- [Documentation map](./docs/README.md)
- [Command manual](./docs/commands/README.md)
- [Provider connection guide](./docs/provider-setup-guide.md)
- [Architecture design](./docs/architecture.md)
- [Task control-plane roadmap](./docs/task-control-plane-roadmap.md)
- [OpenSpec](./openspec/config.yaml)
