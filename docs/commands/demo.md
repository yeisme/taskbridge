# demo 命令

`taskbridge demo` 是新手体验入口。它使用内置 mock 数据展示 TaskBridge 控制面能力，不需要 Provider 认证、不读取 token、不调用远端 API、不写本地 task store。

## 什么时候用

适合用 `demo` 的情况：

- 第一次接触 TaskBridge，想快速看到控制面效果。
- 还没有认证任何 Provider，但想理解 today、next、review 的输出结构。
- 想验证 `--json` 或 `--agent` 输出格式是否符合脚本或 Agent 契约。

不适合用 `demo` 的情况：

- 已经有真实任务数据：直接用 `today`、`next`、`review`。
- 想执行真实任务操作：用 `task` 或 `review --apply-file`。

## 子命令

| 命令 | 用途 |
| --- | --- |
| `demo today` | 用 mock 数据展示每日工作台 |

## 常用流程

```bash
taskbridge demo today
taskbridge demo today --json
taskbridge demo today --format json
```

## 输出模式

| 模式 | 用途 |
| --- | --- |
| 默认 human | English 摘要 + task table + 推荐下一步 |
| `--json` | AI-native envelope (`spec_version`, `mode`, `command`, `status`, `facts`, `data`) |
| `--format json` | Legacy JSON payload (`taskbridge.today.v1`) |

## 边界

- 只读命令，不写本地 storage，不调用远端 API。
- 不读取 Provider token 或 credentials。
- 输出结构与 `today --mock` 完全一致，`demo today` 是面向用户的正式入口。

## 常见错误

| 错误 | 原因 | 处理 |
| --- | --- | --- |
| 未知子命令 | 输入了 `demo next` 等不存在的子命令。 | 当前只支持 `demo today`。 |

## 最短可用流程

```bash
taskbridge demo today
```
