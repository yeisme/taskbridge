# TaskBridge CLI 子命令设计

## 命令概览

```text
taskbridge [command] [subcommand] [flags]

Commands:
  auth        认证管理
  provider    Provider 管理
  sync        任务同步
  list        列出任务
  lists       列出清单
  task        本地任务管理
  project     项目规划与落地
  analyze     任务分析
  governance  治理与智能辅助
  tui         交互式终端界面
  serve       后台服务
  config      配置管理
  version     版本信息
```

## project

```text
taskbridge project create <name>
taskbridge project list
taskbridge project split <project-id>
taskbridge project split-markdown <project-id> --file plan.md
taskbridge project confirm <project-id>
taskbridge project sync <project-id> --provider google
```

## governance

```text
taskbridge governance overdue-health
taskbridge governance resolve-overdue --action task_1:defer
taskbridge governance rebalance-longterm
taskbridge governance detect-decomposition
taskbridge governance decompose-task <task-id>
taskbridge governance achievement
```

## 控制面规划命令

以下命令属于四阶段路线设计，详见 [control plane roadmap](task-control-plane-roadmap.md)。实现时应保持 CLI 核心入口不变，并优先提供稳定 JSON 输出。

```text
taskbridge sync diff <source> --target <provider>
taskbridge sync conflicts
taskbridge sync audit <session-id>
taskbridge today
taskbridge inbox
taskbridge next
taskbridge review
taskbridge project review <project-id>
taskbridge project next <project-id>
taskbridge project adjust <project-id> --dry-run
taskbridge project done <project-id>
taskbridge project archive <project-id>
taskbridge agent capabilities
taskbridge agent schemas
taskbridge agent today
taskbridge agent plan <goal> --dry-run
taskbridge agent plan <goal> --dry-run=false
taskbridge agent execute --action-file actions.json --dry-run
```

## 输出原则

1. 普通命令从同一个 projection 渲染默认 human、`--json`、`--agent`、`--events` 和 `--explain`（如果支持）。
2. 默认 human CLI chrome 使用 English；用户任务内容、第三方 payload 和既有 machine fields 不被翻译。
3. `--json` 是新脚本推荐入口，输出 envelope；legacy `--format json` 按命令保留可解析兼容 payload。
4. `--agent` 是低 token key=value 输出，必须包含 `spec_version`、`mode=agent`、`command`、`status`。
5. CLI 是唯一主交互面，不再维护协议服务子命令。
6. 危险写入命令必须支持 dry-run 或明确确认参数。

## 输出与 Agent 体验

后续 CLI 输出设计以 [CLI 输出与 Agent 体验设计](cli-output-agent-design.md) 和 OpenSpec change `taskbridge-cli-output-polish` 为准：先生成统一 projection，再分别渲染默认 English summary/table/stats panel、`--json` envelope、普通命令 `--agent` key=value、长任务 `--events` NDJSON 和决策复核 `--explain`。

`taskbridge agent *` 继续遵守 [Agent 契约](agent-contract.md)，stdout 保持 `taskbridge.agent-result.v1` JSON；普通命令的 `--agent` 是低 token key=value 输出，不替代 `agent *` 安全执行入口。

## 命令文档

TaskBridge 命令文档按 Eikona、Cohors、Pinax 的方式独立维护在 [commands](commands/README.md) 目录。每个 root command 必须有单独页面，页面负责说明职责、子命令、常用流程、输出模式和写入边界；根 README 和路线文档不再承载完整命令手册。

新增或重写 Cobra root command 时，同一个 change 必须更新：

- `docs/commands/README.md` 命令地图。
- `docs/commands/<command>.md` 命令页面。
- 如输出或写入语义变化，更新相关设计文档和 contract tests。

## OpenSpec 任务集成

OpenSpec 任务应作为 TaskBridge 控制面的工程任务信号接入，设计见 [TaskBridge 与 OpenSpec 任务集成设计](openspec-taskbridge-integration.md)。OpenSpec 继续作为 change lifecycle 和 OpenSpec 命令的事实源；TaskBridge 只在 `today`、`next`、`review`、Agent 输出中消费 OpenSpec 状态并推荐原生 `openspec` 下一步命令。首阶段不新增 `taskbridge openspec *` wrapper，不直接写 OpenSpec 文件。
