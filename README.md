# TaskBridge

<div align="center">

**面向 AI 与多 Todo 平台的 CLI 工作流工具**

</div>

---

## 本地工作流

### 安装

```bash
git clone https://github.com/yeisme/taskbridge.git
cd taskbridge
go build -trimpath -ldflags="-s -w" -o taskbridge
```

### 配置

```bash
export TASKBRIDGE_HOME=~/.taskbridge
export TASKBRIDGE_STORAGE_PATH=~/.taskbridge/data
export TASKBRIDGE_PROVIDERS=microsoft,todoist
```

首次使用建议先运行诊断：

```bash
taskbridge doctor
taskbridge quickstart
```

### Provider 认证

```bash
taskbridge auth status
taskbridge auth login microsoft
taskbridge auth login google
taskbridge auth login feishu
taskbridge auth login ticktick
taskbridge auth login dida
taskbridge auth login todoist
taskbridge provider list
taskbridge provider enable todoist
taskbridge provider test todoist
```

Provider 连接详细指南见 [docs/provider-setup-guide.md](docs/provider-setup-guide.md)。OAuth 凭证文件保存在 `~/.taskbridge/credentials/<provider>_credentials.json`，认证 token 保存在 `~/.taskbridge/credentials/tokens.json`。

### 任务浏览与管理

```bash
taskbridge list
taskbridge list --all
taskbridge list --source microsoft --status todo
taskbridge list --query "今天"
taskbridge list --format json
taskbridge lists
taskbridge lists --source microsoft
```

```bash
taskbridge task add "整理 OpenSpec 输出契约" --due 2026-06-10 --priority 3
taskbridge task show <task-id> --format json
taskbridge task edit <task-id> --due 2026-06-12
taskbridge task done <task-id>
taskbridge task undo <task-id>
```

`list` 支持按来源、状态、象限、优先级、标签、清单、关键词过滤；输出格式支持 `table`、`json`、`markdown`、`compact`、`tsv`。`task` 面向本地 store，不直接写远端 Provider；远端同步必须走 `sync`。

### 同步

```bash
taskbridge sync pull microsoft
taskbridge sync push todoist
taskbridge sync bidirectional microsoft --target todoist
taskbridge sync status
taskbridge sync diff microsoft --target todoist --format json
taskbridge sync conflicts
taskbridge sync resolve <conflict-id>
taskbridge sync backup create
taskbridge sync backup restore <backup-id>
taskbridge sync audit <session-id> --format json
taskbridge sync watch microsoft --interval 10m
```

`sync pull` 写本地 storage；`sync push` 写远端 Provider；`bidirectional` 双向写入。`--dry-run` 不写本地 storage，不调用远端写 API。远端覆盖、删除、冲突丢弃必须显式确认。

### 每日控制面

```bash
taskbridge today
taskbridge today --mock
taskbridge today --format json
taskbridge next
taskbridge next --limit 3
taskbridge next --source openspec
taskbridge inbox
taskbridge inbox --limit 10 --source todoist
taskbridge review
taskbridge review --format json
taskbridge review --apply-file actions.json --dry-run
taskbridge review --apply-file actions.json --confirm
```

`today` 是每日任务工作台，把今日必须做、即将失控、建议下一步放在一个入口。`next` 推荐当前最值得推进的下一步。`inbox` 列出无归属、缺日期或待整理任务。`review` 做任务健康复盘，默认只建议不写入。

### 项目规划

```bash
taskbridge project create "学习 OpenClaw"
taskbridge project create "发布 TaskBridge 控制面" --goal-text "希望完成控制面四阶段"
taskbridge project list
taskbridge project split <project-id> --max-tasks 10
taskbridge project split-markdown <project-id> --file plan.md
taskbridge project confirm <project-id>
taskbridge project sync <project-id>
taskbridge project review <project-id>
taskbridge project next <project-id>
taskbridge project adjust <project-id>
taskbridge project done <project-id>
taskbridge project archive <project-id>
```

`project create` 创建草稿；`split` 生成拆分建议；`confirm` 确认落库创建本地任务；`sync` 同步到 Provider。`adjust` 默认 dry-run，有 action 时必须确认后应用。`archive` 不删除历史数据。

### 治理与智能辅助

```bash
taskbridge governance overdue-health --format json
taskbridge governance resolve-overdue --dry-run
taskbridge governance resolve-overdue --confirm
taskbridge governance rebalance-longterm --format json
taskbridge governance detect-decomposition --limit 10 --format json
taskbridge governance decompose-task <task-id> --format json
taskbridge governance decompose-task <task-id> --write-tasks
taskbridge governance achievement
```

`overdue-health` 分析逾期任务健康度；`resolve-overdue` 批量处理逾期任务；`rebalance-longterm` 调配长期无排期任务；`detect-decomposition` 识别复杂候选任务；`decompose-task` 拆成执行步骤；`achievement` 分析完成情况。批量完成、批量改期、删除、远端覆盖必须有 dry-run/confirm/action file gate。

### 分析

```bash
taskbridge analyze quadrant
taskbridge analyze priority
taskbridge analyze priority --json
taskbridge analyze time
taskbridge analyze trend
taskbridge analyze report --json
```

`analyze` 提供四象限、优先级、时间分布、趋势和综合报告分析。默认 human 输出使用共享 stats panel；`--json` 输出 envelope，legacy `--format json` 仍保持可解析兼容 payload。只读命令，不改任务、不同步 Provider。

### Agent 集成

```bash
taskbridge agent capabilities
taskbridge agent today
taskbridge agent plan "学习 OpenClaw" --dry-run
taskbridge agent execute --action-file actions.json --dry-run
taskbridge agent execute --action-file actions.json --confirm
taskbridge agent schemas
```

`agent` 是 Agent 安全执行入口。stdout 永远是 `taskbridge.agent-result.v1` JSON。Agent 不直接读写 `~/.taskbridge` 数据文件，不持有 Provider token。危险动作没有 `--confirm` 时返回 `requires_confirmation=true`，不能写入。

普通命令优先使用 `--json` envelope 或 `--agent` key=value，例如 `taskbridge version --json`、`taskbridge provider list --agent`、`taskbridge config show --json`；Agent 脚本不要解析 human output。

### 后台服务

```bash
taskbridge serve
taskbridge serve --token-refresh
taskbridge serve --sync --sync-interval 15m
```

`serve` 启动长运行后台服务，用于 token 自动刷新和定时同步。日志写 stderr 或日志系统；stdout 遵守 JSON/events 契约。

### 交互式终端

```bash
taskbridge tui
```

启动交互式终端界面浏览和操作任务。TUI 不作为脚本机器输出入口，Agent 和 CI 不应依赖 TUI 输出。

## 支持的平台

| 平台            | 状态      | 认证方式       | 特点       |
| --------------- | --------- | -------------- | ---------- |
| Microsoft Todo  | ✅ 已完成 | OAuth 2.0      | 完整支持   |
| Google Tasks    | ✅ 已完成 | OAuth 2.0      | 基础支持   |
| Feishu Tasks    | ✅ 已完成 | OAuth 2.0      | 完整支持   |
| TickTick        | ✅ 已完成 | OpenAPI Token  | 原生四象限 |
| 滴答清单        | ✅ 已完成 | OpenAPI Token  | 国内版     |
| Todoist         | ✅ 已完成 | API Token      | 完整支持   |
| OmniFocus       | 📋 计划中 | —              | macOS 专用 |
| Apple Reminders | 📋 计划中 | —              | macOS/iOS  |

## 项目结构

```text
taskbridge/
├── cmd/                    # CLI 命令入口
├── internal/
│   ├── auth/               # token 与认证
│   ├── model/              # 核心数据模型
│   ├── project/            # 项目与规划存储
│   ├── projectplanner/     # 目标拆分与计划建议
│   ├── provider/           # Todo 软件适配器
│   │   ├── microsoft/      # Microsoft Todo
│   │   ├── google/         # Google Tasks
│   │   ├── feishu/         # Feishu Tasks
│   │   ├── ticktick/       # TickTick / 滴答清单
│   │   └── todoist/        # Todoist
│   ├── storage/            # 存储层
│   └── sync/               # 同步引擎
├── pkg/
│   ├── config/             # 配置管理
│   ├── logger/             # 日志
│   ├── paths/              # 路径约定
│   └── ui/                 # CLI/TUI UI 组件
├── docs/                   # 设计与使用文档
│   └── commands/           # 命令手册
└── openspec/               # OpenSpec 变更管理
```

## 技术栈

- **语言**: Go 1.25+
- **CLI**: Cobra
- **配置**: Viper
- **存储**: 文件存储 / MongoDB（可选）

## 本地验证与开发工具

```bash
task deps
task fmt-check
task lint
task test
task build
task check
```

首次配置开发机时可安装辅助工具：

```bash
task tools:install
```

发布配置和本地快照构建：

```bash
task release:check
task snapshot
task release:local
```

可选热加载（需安装 Air）：

```bash
task dev:watch
```

没有安装 `task` 时，使用等价命令：

```bash
go test ./...
mkdir -p dist && go build -trimpath -ldflags="-s -w" -o dist/taskbridge
```

## 文档入口

- [子项目指令](./AGENTS.md)
- [文档地图](./docs/README.md)
- [命令手册](./docs/commands/README.md)
- [Provider 连接指南](./docs/provider-setup-guide.md)
- [架构设计](./docs/architecture.md)
- [任务控制面路线](./docs/task-control-plane-roadmap.md)
- [OpenSpec](./openspec/config.yaml)
