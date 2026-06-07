## Why

`cli/taskbridge` 有 lint 配置和 Taskfile，但任务命名偏历史（如 `dev` 中有 `goreleaser build --snapshot -- clean` 拼写风险）。根 `go-dev-toolchain-quality` 要求修正命名、补齐 `fmt-check`、`check`/`ci` 顺序，并评估可选 Air。

## What Changes

- 修正历史 Taskfile 命名问题。
- 补齐 `fmt-check` 任务。
- 规范 `check`/`ci` 顺序。
- 评估可选 Air 热加载。
- 保留 Docker 任务。

## Capabilities

### Modified Capabilities

- `taskbridge`: Go 开发工具链标准化，修正命名和补齐任务。

## Impact

- 修改文件：`Taskfile.yml`、`.golangci.yml`（复核）、`AGENTS.md`。

## Non-Goals

- 不改变 TaskBridge 的任务管理功能。
