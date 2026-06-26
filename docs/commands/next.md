# next 命令

`taskbridge next` 推荐当前最值得推进的跨 Provider 下一步任务。它不是完整列表命令，默认保持 3-5 个以内，并在机器输出中保留 `source`、`domain`、project 和 sync risk 线索。

## 什么时候用

适合用 `next` 的情况：

- 不确定现在该做什么，想看推荐。
- 想获取少量高优先级、即将到期、覆盖 work/life domain 的下一步建议。

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
taskbridge sync pull --all && taskbridge next
taskbridge next --format json
```

`--limit` 控制返回数量，默认少量。排序会考虑 due date、priority、domain、source、project next 和 sync risk；有冲突或 Provider 状态不确定的任务只会建议 review/resolve，不会建议直接 complete/defer/reschedule。

## 输出模式

| 模式 | 用途 |
| --- | --- |
| 默认英文短列表 | 快速扫推荐和下一步。 |
| `--format json` / `--json` | 输出 AI-native envelope，`data` 内含 `taskbridge.next.v1`。 |
| `--agent` | 输出低 token key=value recommendation facts。 |

## 边界

- 只读命令，不写本地任务或远端 Provider。
- `domain` 不替代 Provider `source`；机器输出会同时保留两者。

## 常见错误

| 错误 | 原因 | 处理 |
| --- | --- | --- |
| 无推荐 | 本地无任务或全部已完成。 | 正常情况，或先同步 `taskbridge sync pull --all`。 |

## 最短可用流程

```bash
taskbridge next
```
