## Why

TaskBridge 的产品方向已经收敛为本地任务控制面，但当前实现已经同时触达 `today/next/review`、`review --apply-file` 和 `taskbridge agent execute --confirm`，写入能力早于完整的审计、输出合同和测试证据。现在需要先补齐安全地基，否则继续加 Provider、MCP 或项目闭环都会把不可信写入和不可解析输出固化成长期债务。

## What Changes

- 固化新用户和日常入口：`doctor -> demo today -> today -> next -> review`，让无 Provider 认证时也能看到每日工作台价值。
- 对齐 CLI 输出合同：保留既有 `--format json` 兼容路径，同时为控制面命令补 `--json` envelope 和 `--agent` key=value 输出；机器 stdout 不混入中文提示、ANSI、日志或进度。
- 强化 Agent 契约：`taskbridge agent *` 继续默认输出稳定 JSON，但错误路径必须返回非零 exit code，capabilities 和 schemas 必须反映真实可用能力。
- 为 `review --apply-file` 和 `taskbridge agent execute` 增加 action 执行审计 receipt，dry-run 不写任务，confirm 写入必须能回答为什么、改了什么、结果如何、证据在哪里。
- 增加命令级/process e2e 测试入口和 evidence wrapper，`task test:integration` 每次运行都写入脱敏证据到 `temp/integration-test-runs/<run-id>/`。
- 更新 TaskBridge 自有文档，主叙事从“多 Todo 平台 CLI”调整为“人和 Agent 共用的任务执行控制面”。

## Capabilities

### New Capabilities

- `task-control-plane-entry`: 体验入口、每日工作台、下一步推荐和复盘建议的用户可见控制面行为。
- `task-action-execution-audit`: action file dry-run/confirm 执行门禁、审计 receipt 和可复验写入结果。
- `task-agent-output-contract`: Agent JSON、`--json` envelope、`--agent` key=value、schema/capability 输出和 stdout/stderr 分离。
- `taskbridge-integration-test-evidence`: TaskBridge 命令级、process e2e 和 golden 输出测试的脱敏证据落盘规则。

### Modified Capabilities

无。当前子项目还没有归档到 `openspec/specs/` 的稳定 spec，本 change 新增能力合同。

## Impact

- 主要影响代码：`cmd/controlplane.go`、`cmd/agent.go`、`cmd/doctor.go`、`cmd/root.go`、`internal/controlplane/`、`internal/actionfile/`、`internal/agentcontract/`、可能新增 `internal/actionaudit/` 和 `internal/clioutput/`。
- 主要影响测试：命令级 stdout/stderr contract、action file dry-run/confirm、agent error exit contract、schema validation、evidence wrapper。
- 主要影响文档：`README.md`、`docs/task-control-plane-roadmap.md`、`docs/agent-contract.md`、`docs/cli-design.md`、`AGENTS.md`。
- 依赖关系：应先完成或并行收口 `taskbridge-code-quality-refactor` 的 shared output helper、命令级 harness 和 evidence 任务，避免重复实现输出分流和测试入口。
