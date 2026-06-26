# TaskBridge 文档地图

本目录是 TaskBridge 子项目的产品、设计、运行、协议、实现和 release 文档真源。根仓库只保留跨项目 handoff 和治理索引，不维护 TaskBridge 文档镜像。

TaskBridge 是覆盖所有 Todo 软件的 CLI 任务中控台：本地文件存储是默认运行时存储，Provider adapter 接入 Microsoft Todo、Todoist、飞书任务、TickTick、滴答清单、Google Tasks 和本地任务，`today`/`next`/`review` 把工作与生活任务汇总成每日决策视图，Agent 通过 action file 安全执行写入。

## 当前状态

- 当前阶段：本地任务管理、Provider 连接、`sync pull --all` 聚合、跨 Provider daily hub、项目规划、治理分析和 Agent 集成已落地。
- 当前实现边界：支持 Microsoft Todo、Google Tasks、飞书任务、TickTick、滴答清单、Todoist 六个 Provider；支持本地任务 CRUD、单向/双向同步、冲突处理、项目拆分、治理建议和 Agent 安全执行。OmniFocus 和 Apple Reminders 仍在计划中。
- 用户可见命令使用 `taskbridge <command>` 统一入口；默认 human output 使用英文，Agent 和脚本应使用各命令 `--json`、`--agent`、`--events`、`--format json` 或 `taskbridge agent *` JSON，而不是解析 human output。

核心日常路径：

```bash
taskbridge sync pull --all
taskbridge today
taskbridge next
taskbridge review
```

## 文档分区

### 命令手册

- [命令地图](./commands/README.md)：按工作流解释每个 root command 管什么。
- [demo](./commands/demo.md)：无需认证的控制面 demo 入口。
- [agent](./commands/agent.md)：Agent 安全执行入口。
- [analyze](./commands/analyze.md)：四象限、优先级、趋势分析。
- [auth](./commands/auth.md)：Provider 认证和 token 管理。
- [completion](./commands/completion.md)：Shell 自动补全。
- [config](./commands/config.md)：配置查看。
- [doctor](./commands/doctor.md)：环境诊断。
- [governance](./commands/governance.md)：逾期治理、任务拆分、成就分析。
- [inbox](./commands/inbox.md)：待整理任务视图。
- [list](./commands/list.md)：任务筛选和浏览。
- [lists](./commands/lists.md)：清单结构查看。
- [next](./commands/next.md)：推荐下一步。
- [project](./commands/project.md)：项目规划和拆分。
- [provider](./commands/provider.md)：Provider 启用和测试。
- [quickstart](./commands/quickstart.md)：新手引导。
- [review](./commands/review.md)：任务健康复盘。
- [serve](./commands/serve.md)：后台服务。
- [sync](./commands/sync.md)：同步引擎。
- [task](./commands/task.md)：本地任务 CRUD。
- [today](./commands/today.md)：每日工作台。
- [tui](./commands/tui.md)：交互式终端界面。
- [version](./commands/version.md)：版本信息。

### Provider 与认证

- [Provider 连接指南](./provider-setup-guide.md)：各 Provider 的认证配置详细步骤。
- [Microsoft 设置指南](./microsoft-setup-guide.md)：Microsoft Todo/Azure AD 配置详解。
- [Token 自动刷新](./token-auto-refresh.md)：Token 刷新机制设计。
- [同步 Provider 简写](./sync-provider-shorthand.md)：Provider 名称简写和别名。

### 设计与规划

- [架构设计](./architecture.md)：系统架构和模块边界。
- [Agent 契约](./agent-contract.md)：Agent 安全执行契约。
- [CLI 设计](./cli-design.md)：CLI 交互设计原则。
- [CLI 输出契约](./cli-output-agent-design.md)：输出模式设计（`--json`、`--agent`、`--events`、`--explain`）。
- [多 Provider 存储](./multi-provider-storage.md)：多 Provider 数据模型和存储设计。
- [目标拆解 MVP](./goal-decomposition-mvp.md)：目标拆分功能设计。
- [任务控制面路线](./task-control-plane-roadmap.md)：四阶段路线规划。
- [Release 管理](./release-management.md)：版本发布流程。

### OpenSpec 集成

- [OpenSpec TaskBridge 集成](./openspec-taskbridge-integration.md)：OpenSpec 工程信号与 TaskBridge 每日控制面集成设计。

## 验证入口

只改文档时不默认跑 Go 测试。修改 Go 代码后执行：

```bash
go test ./...
go build -trimpath -ldflags="-s -w" -o taskbridge
```

如果已安装 Taskfile，也可以运行：

```bash
task check
```
