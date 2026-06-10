# TaskBridge Serve RPC and MCP Design

Date: 2026-06-10

## Problem

`taskbridge serve` currently acts as a local long-running daemon for token refresh, optional scheduled sync, and optional health checks. It does not expose a stable integration surface for local dashboards, GUI clients, or MCP-capable agents. Existing docs already allow a future MCP adapter, but with strict boundaries: CLI remains the core entrypoint, adapters must not read `~/.taskbridge` directly, must not hold provider tokens, and must not bypass Provider interfaces or confirmation gates.

## Decision

Add RPC and MCP as optional `serve` capabilities, with `serve` acting as a runtime supervisor. Use one shared application service underneath CLI, RPC, and MCP. Do not make MCP or RPC a second implementation of TaskBridge business logic.

Recommended approach: **RPC is the local service surface; MCP is a thin adapter over the same application service**.

## Goals

- Keep existing `taskbridge serve` behavior unchanged by default.
- Add opt-in local RPC for scripts, dashboards, future GUI, and health/status automation.
- Add opt-in MCP tools for Claude/Cursor/Codex-style agents.
- Preserve the existing Agent safety model: dry-run first, action file for writes, explicit confirmation for dangerous operations.
- Ensure RPC/MCP never directly read local TaskBridge data files or provider credentials.
- Keep logs, human banners, JSON responses, and MCP protocol output separated.

## Non-goals

- Do not turn TaskBridge into a SaaS web app.
- Do not make the core product depend on a constantly running MCP server.
- Do not make MCP the primary API for ordinary CLI users.
- Do not expose provider tokens, auth headers, raw provider payloads, or local credential file contents.
- Do not allow RPC/MCP handlers to call provider write APIs directly.
- Do not silently execute destructive, remote-write, conflict-discard, bulk-complete, or bulk-reschedule actions.

## User-facing CLI shape

Existing behavior remains:

```bash
taskbridge serve
```

This starts token auto-refresh using current defaults.

New opt-in flags:

```bash
taskbridge serve \
  --enable-auto-refresh \
  --enable-sync \
  --enable-health \
  --enable-rpc \
  --rpc-addr 127.0.0.1:8082 \
  --enable-mcp \
  --mcp-addr 127.0.0.1:8083
```

Proposed flags:

| Flag | Default | Meaning |
| --- | --- | --- |
| `--enable-rpc` | `false` | Start local RPC server. |
| `--rpc-addr` | `127.0.0.1:8082` | RPC listen address. |
| `--rpc-readonly` | `true` | Reject confirmed write operations through RPC unless disabled. |
| `--rpc-token` | empty | Optional bearer token or `env:NAME` token source. |
| `--enable-mcp` | `false` | Start MCP server. |
| `--mcp-addr` | `127.0.0.1:8083` | MCP listen address for HTTP transport. |
| `--mcp-readonly` | `true` | Reject confirmed write tools unless disabled. |
| `--mcp-transport` | `streamable-http` | MCP transport; initial implementation may support one transport only. |

If a user binds RPC or MCP to `0.0.0.0`, `serve` should print a clear warning to stderr because TaskBridge may expose task metadata and action execution surfaces.

## Architecture

```text
cmd/serve.go
  -> internal/runtime.Service
       -> auth.TokenManager
       -> sync.Scheduler
       -> health HTTP server
       -> rpc HTTP server
       -> mcp server

cmd/agent.go / cmd/today.go / cmd/next.go
  -> internal/agentservice.Service
       -> controlplane/project/sync/actionfile services
       -> storage/provider through existing interfaces

internal/rpc
  -> validates HTTP requests
  -> maps request/response envelopes
  -> calls internal/agentservice

internal/mcp
  -> registers MCP tools and schemas
  -> maps MCP tool calls/results
  -> calls internal/agentservice
```

The key cut is `internal/agentservice`: CLI commands, RPC handlers, and MCP tools all call the same service. RPC and MCP never shell out to `taskbridge` and never parse human output.

## Runtime lifecycle

`serve` should own all long-running components through one lifecycle object:

```text
runtime.Service.Start(ctx)
  - start token manager when enabled
  - start sync scheduler when enabled
  - start health server when enabled
  - start RPC server when enabled
  - start MCP server when enabled

runtime.Service.Stop(ctx)
  - gracefully shut down RPC/MCP/health servers
  - stop sync scheduler
  - stop token manager
```

Implementation should use context cancellation and a coordinated goroutine model such as `errgroup.WithContext`. Every server must have bounded graceful shutdown. A startup failure in any enabled component should fail `serve` instead of leaving an ambiguous half-running daemon.

## RPC contract

RPC is a local JSON HTTP API. Initial version uses stable JSON envelopes and only exposes safe TaskBridge operations.

Envelope:

```json
{
  "schema": "taskbridge.rpc-result.v1",
  "status": "ok",
  "request_id": "req_20260610_090000",
  "dry_run": true,
  "requires_confirmation": false,
  "result": {},
  "warnings": [],
  "errors": []
}
```

Error envelope keeps JSON shape:

```json
{
  "schema": "taskbridge.rpc-result.v1",
  "status": "error",
  "request_id": "req_20260610_090000",
  "dry_run": true,
  "requires_confirmation": false,
  "result": null,
  "warnings": [],
  "errors": [
    {
      "code": "provider_not_authenticated",
      "message": "Provider is not authenticated.",
      "next_action": "taskbridge auth login microsoft"
    }
  ]
}
```

Initial endpoints:

| Endpoint | Method | Behavior |
| --- | --- | --- |
| `/rpc/v1/status` | `GET` | Runtime status: token manager, scheduler, providers, uptime. |
| `/rpc/v1/capabilities` | `GET` | Safe commands, schemas, providers, confirmation rules. |
| `/rpc/v1/today` | `POST` | Same semantic result as agent-friendly today. |
| `/rpc/v1/next` | `POST` | Next recommended task/action. |
| `/rpc/v1/inbox` | `POST` | Tasks needing triage. |
| `/rpc/v1/review` | `POST` | Health review and suggested actions. |
| `/rpc/v1/sync/diff` | `POST` | Read-only sync diff. |
| `/rpc/v1/agent/plan` | `POST` | Plan preview or project draft write when explicitly allowed. |
| `/rpc/v1/agent/execute` | `POST` | Action file dry-run by default; confirmed writes gated. |

Write rules:

- `dry_run` defaults to `true` for plan and execute.
- `requires_confirmation=true` must propagate unchanged.
- If `--rpc-readonly=true`, confirmed writes return a structured error.
- Confirmed writes must still go through action file execution and existing Provider interfaces.

## MCP contract

MCP is a thin adapter over the same application service. It does not own business rules.

Initial tools:

| Tool | Behavior |
| --- | --- |
| `taskbridge_capabilities` | Return provider status, safe operations, schemas, confirmation policy. |
| `taskbridge_today` | Return daily workbench data. |
| `taskbridge_next` | Return recommended next steps. |
| `taskbridge_inbox` | Return triage queue. |
| `taskbridge_review` | Return health review and suggested action file content. |
| `taskbridge_sync_diff` | Return provider/local diff without writing. |
| `taskbridge_plan` | Generate project/task plan; dry-run by default. |
| `taskbridge_execute_action_file` | Execute action file in dry-run by default; confirmed writes gated. |
| `taskbridge_health` | Return runtime health snapshot. |

MCP adapter rules:

- It must not read `~/.taskbridge` files directly.
- It must not hold or return provider tokens.
- It must not bypass Provider interfaces.
- It must not bypass `requires_confirmation`.
- Tool schemas may be rich; `--agent` output remains low-token and separate.

## Security and privacy

Default network exposure is localhost only. Remote binding is explicit and warned.

Secrets and sensitive data are never returned:

- provider tokens
- auth headers
- OAuth refresh tokens
- raw provider payloads
- credential file paths when unnecessary
- hidden prompts or chain-of-thought

Optional token auth should support:

```bash
taskbridge serve --enable-rpc --rpc-token env:TASKBRIDGE_RPC_TOKEN
```

The same auth shape can be reused for MCP HTTP transport if needed.

## Observability

Health endpoints stay separate from RPC/MCP:

```text
/health
/healthz
/livez
/readyz
/status
```

RPC status can reuse the same runtime snapshot but must emit the RPC envelope. Logs and diagnostics go to stderr or the logger. Machine protocols must not include startup banners, tables, ANSI, or progress text.

Runtime status should include:

- start time
- uptime
- live/ready status
- token manager running state
- provider loaded/authenticated state
- token validity summary without token values
- scheduler running state
- last/next sync run metadata
- enabled RPC/MCP listeners without secrets

## Implementation phases

### Phase 1: Runtime boundary

- Move `serve` lifecycle logic out of `cmd/serve.go` into `internal/runtime`.
- Preserve current flags and behavior.
- Keep health endpoint behavior compatible.
- Add tests for lifecycle start/stop, invalid duration config, health snapshot, and graceful shutdown.

### Phase 2: Agent service extraction

- Extract shared `internal/agentservice` from existing `agent`, `today`, `next`, `review`, `sync diff`, and action-file code paths as needed.
- CLI commands call the service and render as they do today.
- Add tests that service results preserve existing agent contract semantics.

### Phase 3: RPC MVP

- Add `internal/rpc` server.
- Add `--enable-rpc`, `--rpc-addr`, `--rpc-readonly`, and optional `--rpc-token`.
- Implement `/rpc/v1/status`, `/rpc/v1/capabilities`, `/rpc/v1/today`, and `/rpc/v1/agent/execute` dry-run.
- Add tests for JSON envelope, read-only gate, auth failure, malformed request, and shutdown.

### Phase 4: MCP adapter

- Add `internal/mcp` with tool registry and schema mapping.
- Add `--enable-mcp`, `--mcp-addr`, `--mcp-readonly`, and `--mcp-transport`.
- Implement the initial tool set over `agentservice`.
- Add contract tests for tool discovery, tool schemas, dry-run behavior, and confirmation propagation.

### Phase 5: Documentation and operations

- Update `docs/commands/serve.md` with current flags and new RPC/MCP examples.
- Add `docs/rpc.md` for local HTTP API.
- Add `docs/mcp.md` for MCP client setup and tool boundaries.
- Add systemd/Docker-style examples using health checks.

## Verification plan

Focused tests:

```bash
go test ./cmd ./internal/auth ./internal/sync ./internal/controlplane ./internal/actionfile
```

After runtime/RPC/MCP changes:

```bash
go test ./internal/runtime ./internal/rpc ./internal/mcp ./internal/agentservice
```

Full gate before completion:

```bash
go test ./...
go build ./...
```

Concurrency-sensitive runtime changes should also run:

```bash
go test -race ./internal/runtime ./internal/rpc ./internal/mcp
```

Smoke checks:

```bash
taskbridge serve --enable-health --health-port 8081
taskbridge serve --enable-rpc --rpc-addr 127.0.0.1:8082
taskbridge serve --enable-mcp --mcp-addr 127.0.0.1:8083
```

For RPC smoke, verify JSON parses and contains `schema=taskbridge.rpc-result.v1`. For MCP smoke, verify a client can list tools and call `taskbridge_health` and `taskbridge_today` without writing.

## Open decisions

1. Whether first MCP transport should be streamable HTTP only, stdio only, or both.
2. Whether RPC/MCP confirmed writes should ever be allowed, or remain dry-run-only until a later phase.
3. Whether RPC auth is required in MVP or localhost binding is sufficient for the first version.
4. Whether `agentservice` should be extracted before RPC MVP or in parallel with the first RPC endpoints.

Default choices for implementation unless changed during review:

- RPC first, MCP second.
- Localhost-only by default.
- RPC/MCP read-only by default.
- Confirmed writes allowed only when readonly is explicitly disabled and the action file confirmation gate passes.
- MCP transport starts with streamable HTTP if a stable Go MCP library is available; otherwise start with stdio adapter and keep `serve` HTTP MCP as Phase 4b.
