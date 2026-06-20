# next 命令

`taskbridge next` 推荐当前最值得推进的下一步任务。它不是完整列表命令，默认应保持少量、可执行。

## 什么时候用

适合用 `next` 的情况：

- 不确定现在该做什么，想看推荐。
- 想获取少量高优先级、即将到期的下一步建议。

不适合用 `next` 的情况：

- 想看完整任务列表：用 `list`。
- 想看每日工作台全貌：用 `today`。
- 想分析任务健康度：用 `review`。

## 子命令

`next` 当前没有子命令。

## 常用流程

```bash
taskbridge next
taskbridge next --limit 3
taskbridge next --source openspec
taskbridge next --format json
```

`--limit` 控制返回数量，默认少量。`--source openspec` 只消费 OpenSpec 工程任务信号。

## 输出模式

| 模式 | 用途 |
| --- | --- |
| 默认英文短列表 | 快速扫推荐和下一步。 |
| `--format json` | 输出 `taskbridge.next.v1` 结构。 |

## 边界

- 只读命令，不写本地任务或远端 Provider。
- `--source openspec` 只消费信号，不新增 `taskbridge openspec *` wrapper。

## 常见错误

| 错误 | 原因 | 处理 |
| --- | --- | --- |
| 无推荐 | 本地无任务或全部已完成。 | 正常情况，或先同步 `sync pull`。 |

## 最短可用流程

```bash
taskbridge next
```
