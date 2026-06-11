## Why

`cli/taskbridge` 有 lint 配置和 Taskfile，但任务命名偏历史（如 `dev` 中有 `goreleaser build --snapshot -- clean` 拼写风险）。根 `go-dev-toolchain-quality` 要求修正命名、补齐 `fmt-check`、`check`/`ci` 顺序，并评估可选 Air。

## What Changes

- 统一 Taskfile 的 Go 开发、验证、发布和可选热加载入口。
- 补齐 `tools:install`、`fmt-check`、`lint:new`、`test:race`、`release:check`、`snapshot`、`release:local`、`dev:watch` 任务。
- 规范 `check`/`ci` 顺序。
- 对齐本地 build、Docker build、GoReleaser build 的 `-trimpath` 和 `pkg/buildinfo` ldflags。
- 保留 Docker 任务。

## Capabilities

### Modified Capabilities

- `taskbridge`: Go 开发工具链标准化，修正命名和补齐任务。

## Impact

- 修改文件：`Taskfile.yml`、`.golangci.yaml`、`.goreleaser.yaml`、`.air.toml`、`Dockerfile`、`AGENTS.md`、`README.md`。

## Non-Goals

- 不改变 TaskBridge 的任务管理功能。
