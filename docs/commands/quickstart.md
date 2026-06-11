# quickstart 命令

`taskbridge quickstart` 根据当前本地状态输出一条推荐下一步，帮助新用户快速进入可见价值路径。它会检测当前环境状态并给出最合适的建议。

## 什么时候用

适合用 `quickstart` 的情况：

- 首次使用 TaskBridge，不知道从哪里开始。
- 想让 TaskBridge 告诉你当前最该做什么。
- 完成上一步后想看下一步建议。

## 子命令

`quickstart` 当前没有子命令。

## 典型路径

新用户通常的引导路径：

1. `taskbridge doctor` → 确认环境就绪
2. `taskbridge auth login <provider>` → 完成认证
3. `taskbridge provider enable <provider>` → 启用 Provider
4. `taskbridge sync pull <provider>` → 拉取任务
5. `taskbridge today` → 开始使用

每一步完成后运行 `taskbridge quickstart`，它会根据当前状态推荐下一步。

## 常用流程

```bash
taskbridge quickstart
taskbridge quickstart --format json
```

## 输出模式

| 模式 | 用途 |
| --- | --- |
| 默认输出 | 只展示一条建议命令。 |
| `--format json` | 输出 `taskbridge.quickstart.v1`。 |

## 边界

- 只读命令，不修复配置、不写 storage、不触发 Provider 认证。
- 推荐命令必须是真实用户可运行命令。

## 最短可用流程

```bash
taskbridge quickstart
```
