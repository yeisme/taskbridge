## 0. Preflight

- [ ] 0.1 Owner: `cli/taskbridge`; Lane: sequential; Scope: 只读检查。运行 `git status --short`。
- [ ] 0.2 Owner: `cli/taskbridge`; Lane: sequential; Scope: 工具可用性。运行 `golangci-lint version`、`task --version`。

## 1. lint 配置复核

- [ ] 1.1 Owner: `cli/taskbridge`; Lane: A; Depends on: 0.2; Scope: 复核 `.golangci.yml`。确认基线 linters 和 formatters 覆盖。
- [ ] 1.2 Owner: `cli/taskbridge`; Lane: A; Depends on: 1.1; Scope: 全量 lint。运行 `golangci-lint run`。

## 2. Taskfile 修正

- [ ] 2.1 Owner: `cli/taskbridge`; Lane: B; Depends on: 0.1; Scope: 修正命名。修复 `dev` 中 `goreleaser build --snapshot -- clean` 拼写风险。
- [ ] 2.2 Owner: `cli/taskbridge`; Lane: B; Depends on: 2.1; Scope: 补齐 `fmt-check`。新增 `fmt-check`（golangci-lint fmt --diff）。
- [ ] 2.3 Owner: `cli/taskbridge`; Lane: B; Depends on: 2.2; Scope: CI 顺序。`check`/`ci` 顺序为 `deps`、`mod-check`、`fmt-check`、`lint`、`test`、`build`。
- [ ] 2.4 Owner: `cli/taskbridge`; Lane: B; Depends on: 2.3; Scope: 保留 Docker 任务。确认 Docker 编排任务保留不变。

## 3. 文档更新

- [ ] 3.1 Owner: `cli/taskbridge`; Lane: C; Depends on: 2.4; Scope: `AGENTS.md`。更新质量门禁命令。

## 4. 验证

- [ ] 4.1 Owner: `cli/taskbridge`; Lane: D; Depends on: 3.1; Scope: 全量验证。运行 `task fmt-check`、`task lint`、`task test`、`task build`。
