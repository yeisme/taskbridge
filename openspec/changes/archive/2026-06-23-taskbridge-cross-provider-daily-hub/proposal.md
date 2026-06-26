# Proposal: TaskBridge Cross-Provider Daily Hub

## Why

TaskBridge 的核心爽点需要从“功能丰富的任务 CLI”收敛成“跨 Todo 软件的 CLI 任务中控台”：用户把 Microsoft To Do、Todoist、飞书任务、TickTick、滴答清单、Google Tasks 和本地任务统一拉回一个命令行入口，工作和生活任务都能在同一个 `today`/`next` 流程里被看见、排序和安全处理。

当前产品已经具备 Provider、同步、每日控制面、Agent action file、安全确认和 release 分发基础，但用户路径仍偏工程化：需要理解 provider/auth/sync/list/review/project 多个概念，才能感受到“所有任务统一照顾到”的价值。这个 change 把后续重点明确为跨 Provider 聚合、工作/生活分区、可解释推荐和安全写回。

## What Changes

- 定义 `sync pull --all` 的一键聚合入口，拉取所有已启用且已认证的 Provider，并在 dry-run/JSON 输出中明确每个 Provider 的结果。
- 为统一任务模型增加稳定的 task domain 概念：`work`、`life`、`personal`、`unknown`，用于 daily hub 分区、推荐和 review，不替代 Provider/source。
- 强化 `today` 输出结构，让默认视图按 Work、Life、Inbox、Overdue、Recommended next 汇总跨 Provider 任务，而不是按来源散落展示。
- 强化 `next` 的跨 Provider 推荐，限制默认数量，并要求推荐理由同时考虑 due date、priority、domain、source、project 和 sync risk。
- 强化 `review` 的工作/生活覆盖率和积压分析，输出建议动作但不隐式写入。
- 更新 README/命令文档首屏，把定位改为 “CLI command center for all Todo apps”。

## Compatibility

- `source`、`provider`、`list_id`、`tags`、已有 JSON envelope 和 Agent result schema 不删除、不重命名。
- 新增 domain 字段必须向后兼容：旧任务缺失 domain 时视为 `unknown`，可通过规则或用户命令逐步补全。
- `sync pull <provider>` 保持可用；`sync pull --all` 是增量入口。
- `today`/`next` 的机器输出可以新增字段，但不得破坏已有必需字段、exit code、stdout/stderr 分离和 JSON parseability。

## Non-Goals

- 不新增 Apple Reminders、OmniFocus 或新的 Todo Provider。
- 不实现复杂自动排程或日历时间块。
- 不让 Agent 或 MCP 绕过 CLI、Provider interface、dry-run/confirm/audit 边界。
- 不默认执行远端写入；写回仍由 `sync push --confirm` 或 action file confirm gate 控制。
- 不把 `domain` 设计成隐私敏感画像系统；只做任务分区和推荐解释。
