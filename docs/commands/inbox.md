# inbox 命令

`taskbridge inbox` 列出待整理任务，用于发现无项目、无日期、缺少下一步归属的任务。它是"收件箱"视图，帮助清理散落的未组织任务。

## 什么时候用

适合用 `inbox` 的情况：

- 有任务没有设置日期、项目或下一步。
- 想快速发现"被遗忘"的任务。
- 想在整理前先看看有哪些任务需要处理。

不适合用 `inbox` 的情况：

- 想整理或修改任务：用 `task` 或 `review --apply-file`。
- 想看所有任务：用 `list`。
- 想做每日工作台：用 `today`。

## 子命令

`inbox` 当前没有子命令。

## 常用流程

```bash
taskbridge inbox
taskbridge inbox --limit 10
taskbridge inbox --source todoist
taskbridge inbox --format json
```

`--limit` 控制显示数量。`--source` 按特定 Provider 筛选。

## 输出模式

| 模式 | 用途 |
| --- | --- |
| 默认英文短列表 | 快速浏览待整理任务。 |
| `--format json` | 输出 `taskbridge.inbox.v1` 结构。 |

## 边界

- 只读命令，不自动整理、改期或写回 Provider。
- 整理建议应进入 `review` 或 action file，不在 inbox 中静默执行。

## 常见错误

| 错误 | 原因 | 处理 |
| --- | --- | --- |
| 收件箱为空 | 所有任务都有归属和日期。 | 正常情况。 |

## 最短可用流程

```bash
taskbridge inbox
```
