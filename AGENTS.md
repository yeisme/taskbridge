# TaskBridge Subproject Instructions

## Tech Stack

- Go 1.25+
- Cobra CLI
- File storage is the default runtime store; MongoDB is optional
- The unified task model lives in `internal/model`

## Architecture Boundaries

- Put CLI command entry points in `cmd/`.
- Keep business logic in `internal/` first; the CLI layer should only parse arguments, call services, and render output.
- Prefer JSON structs or maps for stable user-visible output; avoid concatenated text that cannot be parsed.
- Local persistence should reuse `internal/persistence/atomicjson.go` or existing store patterns.

## Prohibited

- Do not bypass the Provider interface to write directly to remote Todo platforms.
- Do not let Agent or MCP adapters read or write `~/.taskbridge` data files directly.
- Do not mix progress bars, prompts, or logs into `--format json` output.
- Do not silently perform deletion, bulk completion, bulk rescheduling, remote overwrite, or conflict discard.

## Tests and Quality Gates

After code changes, run at least:

```bash
task test
```

When CLI compilation, release configuration, or command surfaces are involved, run:

```bash
task build
task release:check
```

When CLI process e2e, golden tests, or output contracts are involved, run:

```bash
task test:integration
```

Before submitting, run:

```bash
task check
```

- New commands have unit tests or repeatable CLI behavior tests.
- `--dry-run` does not write local storage or call remote write APIs.
- JSON mode writes only to stdout; logs and prompts go to stderr.
- GoReleaser, local builds, and Docker builds must inject the same `pkg/buildinfo` ldflags.
