# analyze 命令

`taskbridge analyze` 做任务分析，包括四象限、优先级、时间分布、趋势和综合报告。它和 `governance` 的区别是：`analyze` 是只读分析报告，`governance` 会生成可执行的治理建议和实际动作。

## 什么时候用

适合用 `analyze` 的情况：

- 想了解任务在紧急/重要四象限中的分布。
- 想查看优先级分布是否合理。
- 想看任务的时间分布和完成趋势。
- 需要生成综合分析报告。

不适合用 `analyze` 的情况：

- 想处理逾期任务：用 `taskbridge governance overdue-health`。
- 想拆分复杂任务：用 `taskbridge governance decompose-task`。
- 想查看任务列表：用 `taskbridge list`。

## 子命令

| 命令 | 用途 | 写入 |
| --- | --- | --- |
| `taskbridge analyze quadrant` | 四象限分析。 | 不写入。 |
| `taskbridge analyze priority` | 优先级分析。 | 不写入。 |
| `taskbridge analyze time` | 时间分布分析。 | 不写入。 |
| `taskbridge analyze trend` | 趋势分析。 | 不写入。 |
| `taskbridge analyze report` | 生成综合报告。 | 不写入。 |

## 常用流程

```bash
taskbridge analyze quadrant
taskbridge analyze priority --format json
taskbridge analyze time
taskbridge analyze trend
taskbridge analyze report --format json
```

## 输出模式

| 模式 | 用途 |
| --- | --- |
| 默认英文摘要 | 人类查看分析结果。 |
| `--format json` | 机器解析分析结构。 |

## 边界

- 只读命令，不改任务、不同步 Provider。
- 分析逻辑应迁入 internal service，命令层只负责调用和渲染。

## 常见错误

| 错误 | 原因 | 处理 |
| --- | --- | --- |
| 无任务数据 | 本地 storage 为空。 | 先运行 `sync pull <provider>` 拉取任务。 |
| 无逾期/时间数据 | 任务缺少日期字段。 | 检查任务是否有 `due_date` 设置。 |

## 最短可用流程

```bash
taskbridge analyze quadrant
taskbridge analyze report --format json
```
