# TaskBridge

<div align="center">

**面向 AI 与多 Todo 平台的 CLI 工作流工具**

</div>

---

## 项目简介

TaskBridge 是一个纯 CLI 的任务桥接工具，用来连接各种 Todo 软件，并把任务同步、治理、拆分和分析能力统一到一个命令行入口。

它的核心目标是让用户和 AI 能够：

- 用统一模型查看不同平台的任务
- 在本地和远端 Todo 平台之间做可控同步
- 以 CLI 方式完成项目规划、任务拆分、逾期治理和成就分析
- 用脚本和 JSON 输出接入自动化流程

当前产品形态已经完全去掉协议服务层，主入口就是 `taskbridge` CLI。

### 支持的平台

| 平台            | 状态      | 特点       |
| --------------- | --------- | ---------- |
| Microsoft Todo  | ✅ 已完成 | 完整支持   |
| Google Tasks    | ✅ 已完成 | 基础支持   |
| 飞书任务        | ✅ 已完成 | 完整支持   |
| TickTick        | ✅ 已完成 | 原生四象限 |
| 滴答清单        | ✅ 已完成 | 国内版     |
| Todoist         | ✅ 已完成 | 完整支持   |
| OmniFocus       | 📋 计划中 | macOS 专用 |
| Apple Reminders | 📋 计划中 | macOS/iOS  |

> Provider 连接指南见 [docs/provider-setup-guide.md](docs/provider-setup-guide.md)

## 核心能力

### 1. 任务与同步

- `taskbridge auth`
- `taskbridge provider`
- `taskbridge list`
- `taskbridge lists`
- `taskbridge task`
- `taskbridge sync`

### 2. 项目规划

- `taskbridge project create`
- `taskbridge project list`
- `taskbridge project split`
- `taskbridge project split-markdown`
- `taskbridge project confirm`
- `taskbridge project sync`

### 3. 治理与智能辅助

- `taskbridge analyze quadrant`
- `taskbridge analyze priority`
- `taskbridge governance overdue-health`
- `taskbridge governance resolve-overdue`
- `taskbridge governance rebalance-longterm`
- `taskbridge governance detect-decomposition`
- `taskbridge governance decompose-task`
- `taskbridge governance achievement`

## 路线设计

- [TaskBridge 任务控制面四阶段路线设计](docs/task-control-plane-roadmap.md)

## 快速开始

### 安装

```bash
git clone https://github.com/yeisme/taskbridge.git
cd taskbridge
go mod tidy
go build -o taskbridge
```

### 配置

```bash
export TASKBRIDGE_HOME=~/.taskbridge
export TASKBRIDGE_STORAGE_PATH=~/.taskbridge/data
export TASKBRIDGE_PROVIDERS=microsoft,todoist
```

### 常用用法

```bash
./taskbridge provider list
./taskbridge sync status
./taskbridge list --sync-now --source microsoft
./taskbridge analyze quadrant
./taskbridge project create "学习 OpenClaw" --goal-text "我希望学习 openclaw"
./taskbridge project split <project-id> --max-tasks 10
./taskbridge governance overdue-health
./taskbridge governance achievement
```

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
│   ├── storage/            # 存储层
│   └── sync/               # 同步引擎
├── pkg/
│   ├── config/             # 配置管理
│   ├── logger/             # 日志
│   ├── paths/              # 路径约定
│   └── ui/                 # CLI/TUI UI 组件
└── docs/                   # 设计与使用文档
```

## 技术栈

- **语言**: Go 1.21+
- **CLI**: Cobra
- **配置**: Viper
- **存储**: 文件存储 / MongoDB（可选）

## 许可证

MIT License
