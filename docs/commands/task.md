# task 命令

`taskbridge task` 管理本地任务 CRUD 和完成状态。它面向本地 store，不绕过 Provider 接口直接写远端 Todo 平台。远端同步必须走 `sync` 命令。

## 什么时候用

适合用 `task` 的情况：

- 想在本地快速创建、修改或完成一个任务。
- 想查看某个任务的详细信息。
- 想撤销误操作（如误完成）。

不适合用 `task` 的情况：

- 想直接在远端 Provider 创建任务：用 `sync push` 或在远端 App 创建后 `sync pull`。
- 想批量处理任务：用 `governance` 或 `review --apply-file`。
- 想查看任务列表：用 `list`。

## 子命令

| 命令 | 用途 | 写入 |
| --- | --- | --- |
| `taskbridge task add <title>` | 新增本地任务。 | 写本地 storage。 |
| `taskbridge task edit <task-id>` | 修改标题、日期、优先级或象限。 | 写本地 storage。 |
| `taskbridge task delete <task-id>` | 删除本地任务。 | 写本地 storage；高风险。 |
| `taskbridge task done <task-id>` | 标记完成。 | 写本地 storage。 |
| `taskbridge task undo <task-id>` | 撤销完成。 | 写本地 storage。 |
| `taskbridge task show <task-id>` | 查看详情。 | 不写入。 |

## 常用流程

### 创建任务

```bash
taskbridge task add "整理 OpenSpec 输出契约" --due 2026-06-10 --priority 3
taskbridge task add "写单元测试" --due 2026-06-12 --priority 2
```

### 查看和修改

```bash
taskbridge task show <task-id> --format json
taskbridge task edit <task-id> --due 2026-06-15
taskbridge task edit <task-id> --priority 1
```

### 完成和撤销

```bash
taskbridge task done <task-id>
taskbridge task undo <task-id>
```

### 删除

```bash
taskbridge task delete <task-id>
```

## 输出模式

| 模式 | 用途 |
| --- | --- |
| 默认中文摘要 | 人类查看操作结果。 |
| `--format json` | 机器解析任务详情。 |

## 边界

- 不直接写远端 Provider；远端同步必须走 `sync`。
- 删除、批量完成、批量改期等后续高风险能力必须有确认或 action file gate。

## 常见错误

| 错误 | 原因 | 处理 |
| --- | --- | --- |
| 任务不存在 | task-id 无效。 | 用 `taskbridge list` 查找正确 id。 |
| 日期格式错误 | `--due` 需要合法日期格式。 | 使用 `YYYY-MM-DD` 格式。 |
| 优先级越界 | `--priority` 超出范围。 | 使用 1-5 的整数。 |

## 最短可用流程

```bash
taskbridge task add "完成文档" --due 2026-06-10
taskbridge task done <task-id>
```
