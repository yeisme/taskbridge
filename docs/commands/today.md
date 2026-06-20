# today 命令

`taskbridge today` 是每日任务工作台。它把今日必须做、即将失控、建议下一步、项目下一步和后续 OpenSpec 工程信号放在一个默认入口里。

## 什么时候用

适合用 `today` 的情况：

- 每日开始工作时想快速了解今天该做什么。
- 想在一个入口看到今日任务、风险和推荐下一步。
- 想用 mock 数据体验 TaskBridge 功能。

不适合用 `today` 的情况：

- 想看完整的任务列表：用 `list`。
- 想只看推荐下一步：用 `next`。
- 想做任务健康复盘：用 `review`。

## 子命令

`today` 当前没有子命令。

## 常用流程

```bash
taskbridge today
taskbridge today --json
taskbridge today --source microsoft
taskbridge today --source openspec
```

`--mock` 使用内置模拟数据（保留给测试和兼容路径）；新用户应使用 `taskbridge demo today` 体验控制面。`--source openspec` 在 today 视图中加入 OpenSpec 工程任务信号。

## 输出模式

| 模式 | 用途 |
| --- | --- |
| 默认英文摘要 | 突出状态、重点、风险和推荐下一步。 |
| `--format json` | 输出 `taskbridge.today.v1` 结构。 |

## 边界

- 只读命令，不写本地 storage，不调用远端写 API。
- `--mock` 使用内置模拟数据，不读取真实 token 或任务。
- `--source openspec` 只消费 OpenSpec 信号，不新增 `taskbridge openspec *` wrapper。

## 常见错误

| 错误 | 原因 | 处理 |
| --- | --- | --- |
| 无任务 | 本地 storage 为空。 | 先运行 `sync pull <provider>` 拉取任务。 |
| Provider 未认证 | 指定了 `--source` 但未登录。 | 先运行 `auth login <provider>`。 |

## 最短可用流程

```bash
taskbridge today
```
