# serve 命令

`taskbridge serve` 启动后台服务，用于 token 自动刷新和定时同步等运行时能力。它是长运行进程，适合在后台持续运行。

## 什么时候用

适合用 `serve` 的情况：

- 想让 token 自动刷新，避免手动 `auth refresh`。
- 想定时自动同步任务。
- 需要后台持续运行的 TaskBridge 服务。

不适合用 `serve` 的情况：

- 只需要一次性同步：用 `sync pull/push`。
- 只需要一次性 token 刷新：用 `auth refresh`。
- 在 CI/CD 中使用：直接调用具体命令即可。

## 子命令

`serve` 当前没有子命令。

## 常用流程

### 启动后台服务

```bash
taskbridge serve
```

### 启用 token 刷新

```bash
taskbridge serve --token-refresh
```

### 定时同步

```bash
taskbridge serve --sync --sync-interval 15m
```

### 完整后台

```bash
taskbridge serve --token-refresh --sync --sync-interval 10m
```

## 输出模式

- 默认 human logs/diagnostics 写 stderr 或日志系统。
- 后续如支持机器模式，stdout 需要遵守 JSON/events 契约。

## 边界

- 会启动长运行进程，可能刷新 token 或拉取/同步任务。
- 不应把 token、provider payload 或 auth header 输出到 stdout/stderr。
- 需要 Ctrl+C 或信号停止。

## 常见错误

| 错误 | 原因 | 处理 |
| --- | --- | --- |
| 端口冲突 | 如有 HTTP 服务端口冲突。 | 检查是否有其他 serve 实例在运行。 |
| Token 刷新失败 | Provider token 已完全过期。 | 先手动 `auth login <provider>` 重新认证。 |

## 最短可用流程

```bash
taskbridge serve --token-refresh
```
