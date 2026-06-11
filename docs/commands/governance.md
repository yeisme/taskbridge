# governance 命令

`taskbridge governance` 提供任务治理和智能辅助能力，包括逾期健康、长期任务调配、复杂任务识别、任务拆分和成就分析。它和 `analyze` 的区别是：`governance` 生成可执行的治理建议和实际动作，`analyze` 只做只读分析报告。

## 什么时候用

适合用 `governance` 的情况：

- 有大量逾期任务，需要分析健康度并批量处理。
- 长期无排期任务堆积，需要重新调配。
- 复杂任务缺少子任务，需要识别和拆分。
- 想查看一段时间内的任务完成成就。

不适合用 `governance` 的情况：

- 只想看四象限或优先级分析：用 `taskbridge analyze`。
- 只想查看任务列表：用 `taskbridge list`。
- 想手动添加子任务：用 `taskbridge task add`。

## 子命令

| 命令 | 用途 | 写入 |
| --- | --- | --- |
| `taskbridge governance overdue-health` | 分析逾期任务健康度。 | 不写入。 |
| `taskbridge governance resolve-overdue` | 批量处理逾期任务。 | dry-run 或确认后写。 |
| `taskbridge governance rebalance-longterm` | 调配长期无排期任务。 | dry-run 或写本地任务。 |
| `taskbridge governance detect-decomposition` | 识别复杂且缺少子任务的候选任务。 | 不写入。 |
| `taskbridge governance decompose-task <task-id>` | 将单个任务拆成执行步骤。 | 默认建议；`--write-tasks` 写本地任务。 |
| `taskbridge governance achievement` | 分析完成情况和成就反馈。 | 不写入。 |

## 逾期治理流程

```bash
taskbridge governance overdue-health --format json
taskbridge governance resolve-overdue --dry-run
taskbridge governance resolve-overdue --confirm
```

`overdue-health` 查看逾期分布和健康评分；`resolve-overdue` 生成批量处理建议，默认 dry-run 不写入，确认后执行。

## 复杂任务拆分流程

```bash
taskbridge governance detect-decomposition --limit 10 --format json
taskbridge governance decompose-task <task-id> --format json
taskbridge governance decompose-task <task-id> --write-tasks
```

`detect-decomposition` 扫描所有任务，找出复杂但缺少子任务的候选；`decompose-task` 对单个任务生成拆分建议，`--write-tasks` 会把建议作为子任务写入本地 storage。

## 成就分析

```bash
taskbridge governance achievement
taskbridge governance achievement --format json
```

查看任务完成趋势、成就反馈和激励统计。

## 输出模式

| 模式 | 用途 |
| --- | --- |
| 默认中文摘要 | 人类查看治理建议和健康分析。 |
| `--format json` | 机器解析治理结果。 |

## 边界

- 批量完成、批量改期、删除、远端覆盖必须有 dry-run/confirm/action file gate。
- 不让治理命令绕过 Provider 接口写远端。
- `decompose-task --write-tasks` 只写本地 storage，远端同步走 `sync`。

## 常见错误

| 错误 | 原因 | 处理 |
| --- | --- | --- |
| 无逾期任务 | `overdue-health` 没发现逾期任务。 | 正常情况，无需处理。 |
| 任务不存在 | `decompose-task` 的 task-id 无效。 | 用 `taskbridge list` 查找正确 id。 |
| 需要确认 | `resolve-overdue` 没有 `--confirm`。 | 审核建议后追加 `--confirm`。 |

## 最短可用流程

```bash
taskbridge governance overdue-health
taskbridge governance detect-decomposition --limit 5
taskbridge governance achievement
```
