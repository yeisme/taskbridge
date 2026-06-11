# 命令文档预设

新增或重写 TaskBridge 命令文档时，按这个预设保持 Eikona、Cohors、Pinax 风格一致。

## 文件位置

- 命令组文档：`docs/commands/<command>.md`
- 命令地图：`docs/commands/README.md`
- 行为设计：相关 `docs/*.md` 或 OpenSpec change artifacts
- 实现任务：`openspec/changes/<change-id>/tasks.md`

## 命令文档结构

````markdown
# <command> 命令

`taskbridge <command>` 用一句话说明职责，并说明它和相邻命令的边界。

## 子命令

| 命令 | 用途 | 写入 |
| --- | --- | --- |
| `taskbridge <command> <subcommand>` | 用户什么时候用。 | 不写入 / 写本地 storage / 调 Provider。 |

## 常用流程

```bash
taskbridge <command> <subcommand> --json
```

## 输出模式

- 默认 human summary：中文，适合人读。
- `--json` / `--format json`：机器解析，不混入日志。
- `--agent`：如支持，输出稳定 key=value。

## 边界

- 写清楚只读、配置写入、本地任务写入、Provider 写入、action/audit 或危险动作确认边界。
- 写清楚 `--dry-run`、`--confirm`、`--force`、`--delete` 等高风险 flag 的语义。
- 示例必须是真实用户可运行命令，不展示本地 wrapper、shell alias 或 agent-only 前缀。
````

## 覆盖要求

- 每个 root command 必须有一个 `docs/commands/<command>.md`。
- 每个文档必须列出当前实现的主要子命令或说明该命令没有子命令。
- 新增 Cobra root command 时，同一个 change 必须更新 `docs/commands/README.md` 和对应命令页。
- 行为变更时，命令页必须同步更新输出模式、写入边界和常用流程。

