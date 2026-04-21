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

## 输出原则

1. 所有自动化导向命令默认支持 `--format json`
2. 人类阅读命令可以保留 text/json 双输出
3. CLI 是唯一主交互面，不再维护协议服务子命令
