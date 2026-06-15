# TaskBridge Agent Contract

Updated: 2026-06-15

## 目标

Agent 只能通过 TaskBridge CLI 的稳定 JSON 契约读取状态、生成动作预览和执行已确认动作。Agent 不直接读写 `~/.taskbridge` 数据文件，不持有 Provider token，也不绕过 CLI 调用 Provider。

## 命令边界

允许 Agent 调用：

- `taskbridge agent capabilities`
- `taskbridge agent today`
- `taskbridge agent plan <goal> --dry-run`
- `taskbridge agent plan <goal> --dry-run=false`
- `taskbridge agent execute --action-file actions.json --dry-run`
- `taskbridge agent execute --action-file actions.json --confirm`
- `taskbridge agent schemas`
- 只读命令：`today`、`next`、`inbox`、`review`、`sync diff`

默认禁止：

- 直接读写本地 store 文件。
- 直接调用 Provider 写 API。
- 在没有 `--confirm` 的情况下执行危险动作。
- 在 JSON stdout 中混入日志或提示文本。

## Result Envelope

所有 `taskbridge agent *` 命令 stdout 必须是 `taskbridge.agent-result.v1`：

```json
{
  "schema": "taskbridge.agent-result.v1",
  "status": "ok",
  "request_id": "req_20260501_090000",
  "dry_run": true,
  "requires_confirmation": false,
  "result": {},
  "warnings": [],
  "errors": []
}
```

错误也必须保持 JSON：

```json
{
  "schema": "taskbridge.agent-result.v1",
  "status": "error",
  "request_id": "req_20260501_090000",
  "dry_run": false,
  "requires_confirmation": false,
  "result": null,
  "warnings": [],
  "errors": [
    {
      "code": "store_init_failed",
      "message": "初始化存储失败",
      "next_action": "taskbridge doctor"
    }
  ]
}
```

## Action File

Agent 写入必须先生成 action file，再通过 `agent execute` 执行。

```json
{
  "schema": "taskbridge.actions.v1",
  "source": "review",
  "created_at": "2026-05-01T09:00:00+08:00",
  "actions": [
    {
      "id": "act_001",
      "type": "defer_task",
      "task_id": "task_123",
      "due_date": "2026-05-02",
      "reason": "任务逾期且仍有价值，建议推迟",
      "requires_confirmation": true
    }
  ]
}
```

危险动作包括：

- `delete_task`
- `complete_task`
- `defer_task`
- `reschedule_task`
- `remote_write`
- `conflict_discard`

危险动作无 `--confirm` 时必须返回 `requires_confirmation=true`，不能写入。

`agent execute` 默认 dry-run；传入 `--confirm` 时视为确认真实执行，不需要额外传 `--dry-run=false`。

`agent plan` 默认 dry-run，只返回项目拆分预览；传入 `--dry-run=false` 时只允许写入本地 project draft 和 plan suggestion，不创建真实任务，不调用 Provider 写 API。

普通 `review --apply-file` 不允许隐式执行，必须显式传入 `--dry-run` 或 `--confirm`。

## Schema 文件

- [agent-result.schema.json](schemas/agent-result.schema.json)
- [actions.schema.json](schemas/actions.schema.json)
- [today.schema.json](schemas/today.schema.json)

## Exit Codes

Agent commands follow the AI-native CLI output contract for exit behavior:

| Exit code | Meaning |
| --- | --- |
| 0 | Success or dry-run completed |
| 1 | Execution error (failed actions, store init failure, invalid action file) |
| 2 | Usage error (missing required flag, invalid arguments) |

On failure:
- stdout contains a valid `taskbridge.agent-result.v1` JSON envelope with `status: "error"`.
- stderr contains human-readable diagnostics only.
- No tokens, Authorization headers, or Provider payloads are written to stdout or stderr.

## Audit Receipts

Every `review --apply-file` and `agent execute` attempt writes an audit receipt to:

```text
<storage.path>/audit/actions/<session_id>.json
```

Receipt schema is `taskbridge.action-audit.v1` with fields:

```json
{
  "schema_version": "taskbridge.action-audit.v1",
  "session_id": "req_20260615_060053",
  "command": "agent execute",
  "action_file": "actions.json",
  "dry_run": true,
  "confirm": false,
  "status": "error",
  "started_at": "2026-06-15T06:00:53Z",
  "finished_at": "2026-06-15T06:00:53Z",
  "duration_ms": 12,
  "stats": {"total": 2, "updated": 0, "skipped": 2, "errors": 2, "confirmed": 0},
  "operations": [
    {"action_id": "a1", "type": "defer_task", "task_id": "t1", "status": "previewed", "dry_run": true, "confirmed": false}
  ],
  "errors": ["action a1 get task failed: task not found: t1"],
  "redaction": "task titles and fields are local task data; no tokens, authorization headers, or provider payloads are recorded"
}
```

Read receipts via:

```bash
taskbridge audit show <session-id>
taskbridge audit list
```

Receipts are CLI-authored records. Agents and users must not hand-write or modify receipt JSON.

## Schema Index

`taskbridge agent schemas` outputs a validator-friendly schema index with name, description, and commands for each schema:

```json
{
  "schema": "taskbridge.agent-schema-index.v1",
  "schemas": [
    {"name": "taskbridge.agent-result.v1", "description": "...", "commands": ["agent today", "agent execute"]},
    ...
  ]
}
```

Covered schemas:
- `taskbridge.agent-result.v1`
- `taskbridge.agent-capabilities.v1`
- `taskbridge.today.v1`
- `taskbridge.actions.v1`
- `taskbridge.action-result.v1`
- `taskbridge.action-audit.v1`
- `taskbridge.sync-session.v1`
- `taskbridge.project-review.v1`

## MCP Adapter (Deferred)

MCP adapter is intentionally deferred. It will only be built after:
- `--json`, `--agent`, and `--events` output modes are stable.
- Action audit receipts are proven reliable.
- `agent schemas` output is validator-friendly.
- Integration evidence confirms dry-run does not write and confirm is auditable.

The MCP adapter will be a thin wrapper that:
- Exposes schema/capabilities.
- Calls CLI or internal application service.
- Does not directly read/write `~/.taskbridge`.
- Does not hold Provider tokens.
- Does not bypass confirm/audit gates.

Remote Provider write, bidirectional sync auto-resolve, and Apple Reminders/OmniFocus providers are also deferred. They will be tracked in separate OpenSpec changes.
