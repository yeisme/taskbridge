## 0. Preflight

- [x] 0.1 Owner: `cli/taskbridge`; Lane: sequential; Scope: 只读检查。运行 `git status --short`；发现工作区已有多项既有改动，本次仅维护工具链相关文件并修复 race detector 暴露的 `RefreshAll` 并发写 map。
- [x] 0.2 Owner: `cli/taskbridge`; Lane: sequential; Scope: 工具可用性。运行 `task --version`、`golangci-lint version`；GoReleaser 初始缺失，已通过 `task tools:install` 安装。

## 1. lint 配置复核

- [x] 1.1 Owner: `cli/taskbridge`; Lane: A; Depends on: 0.2; Scope: 复核 `.golangci.yaml`。已覆盖 errcheck、govet、ineffassign、staticcheck、unused、misspell、revive 和 gofmt/goimports。
- [x] 1.2 Owner: `cli/taskbridge`; Lane: A; Depends on: 1.1; Scope: 全量 lint。已运行 `task lint`，输出 `0 issues.`。

## 2. Taskfile 修正

- [x] 2.1 Owner: `cli/taskbridge`; Lane: B; Depends on: 0.1; Scope: 修正命名。`dev` 不再直接调用 GoReleaser；快照构建统一为 `task snapshot`。
- [x] 2.2 Owner: `cli/taskbridge`; Lane: B; Depends on: 2.1; Scope: 补齐格式、lint、race、工具安装和发布辅助任务。新增/保留 `tools:install`、`fmt-check`、`lint:new`、`test:race`、`release:check`、`snapshot`、`release:local`、`dev:watch`；`task build` 输出到 `dist/taskbridge` 并 smoke-run `dist/taskbridge --help`。
- [x] 2.3 Owner: `cli/taskbridge`; Lane: B; Depends on: 2.2; Scope: CI 顺序。`check`/`ci` 顺序为 `deps`、`mod-check`、`fmt-check`、`lint`、`test`、`build`。
- [x] 2.4 Owner: `cli/taskbridge`; Lane: B; Depends on: 2.3; Scope: 保留 Docker 任务。Docker 编排任务保留，并将 Dockerfile builder 对齐到 Go 1.25 + `-trimpath`。

## 3. 文档更新

- [x] 3.1 Owner: `cli/taskbridge`; Lane: C; Depends on: 2.4; Scope: `AGENTS.md` 和 `README.md`。已更新质量门禁、工具安装和 GoReleaser/Air 入口说明。

## 4. 验证

- [x] 4.1 Owner: `cli/taskbridge`; Lane: D; Depends on: 3.1; Scope: 全量验证。已运行 `task check`、`task lint:new`、`task release:check`、`task snapshot`、`task test:race`、`openspec validate taskbridge-go-dev-toolchain --strict`。
