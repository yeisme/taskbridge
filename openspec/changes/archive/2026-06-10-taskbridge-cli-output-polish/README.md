# taskbridge-cli-output-polish

收敛 TaskBridge CLI 默认人类输出、机器输出合同、表格宽度、颜色控制和迁移测试计划。

## Artifacts

- `proposal.md`: 变更背景、目标、非目标、影响面和兼容策略。
- `design.md`: 输出 projection、renderer、视觉层和迁移边界设计。
- `tasks.md`: 可执行任务、并行 lane、验收命令和失败复验。
- `specs/taskbridge-cli-output-experience/spec.md`: CLI 输出体验和机器合同的可验收需求。

## Reference Baselines

- GitPulse: projection 一次构建，多 renderer 输出；`--agent`/`--json` 和 legacy format 明确分层。
- skillctl: 默认输出按命令类型分为 result table、detail card、governance report、write preview、event stream 和 explain report。
- TaskBridge: 当前已有 `pkg/ui`、`pkg/output`、`cmd/output_control.go` 和 `docs/cli-output-agent-design.md`，但缺少跨命令统一合同和迁移计划。
