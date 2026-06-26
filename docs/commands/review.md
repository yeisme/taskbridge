# review 命令

`taskbridge review` 做一周视角的任务健康复盘，输出 work/life 覆盖率、unknown-domain backlog、逾期积压、Provider sync health、建议动作和后续 action file 入口。它默认只建议不写入，是"只看不动"的安全入口。

## 什么时候用

适合用 `review` 的情况：

- 想做任务健康检查，看看 work/life 覆盖、未知 domain 和逾期积压。
- 想生成建议动作但不立即执行。
- 想通过 action file 批量处理风险任务。

不适合用 `review` 的情况：

- 想自动处理逾期任务：用 `governance resolve-overdue`。
- 想看今日工作台：用 `today`。
- 想看任务列表：用 `list`。

## 子命令

`review` 当前没有子命令。

## 常用流程

### 只读复盘

```bash
taskbridge review
taskbridge review --format json
taskbridge sync pull --all && taskbridge review
```

### 生成并执行动作

```bash
taskbridge review --apply-file actions.json --dry-run
taskbridge review --apply-file actions.json --confirm
```

`--apply-file` 接受一个 action file，`--dry-run` 预览执行效果，`--confirm` 确认后执行写入。

## 输出模式

| 模式 | 用途 |
| --- | --- |
| 默认英文复盘摘要 | 人类查看风险和建议动作。 |
| `--format json` / `--json` | 输出 AI-native envelope，`data` 内含 `taskbridge.review.v1` 或 action execution result。 |
| `--agent` | 输出低 token key=value facts。 |

## 边界

- 不带 `--apply-file` 时只读，不写任务。
- `--apply-file` 必须显式 `--dry-run` 或 `--confirm`。
- review 的建议动作不会隐式写入；写回仍走 action file 的 dry-run/confirm gate。

## 常见错误

| 错误 | 原因 | 处理 |
| --- | --- | --- |
| 需要 `--confirm` | 执行 action file 但未确认。 | 审核动作后追加 `--confirm`。 |
| action file 不存在 | `--apply-file` 路径无效。 | 检查文件路径。 |

## 最短可用流程

```bash
taskbridge review
```
