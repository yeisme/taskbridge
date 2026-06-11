# version 命令

`taskbridge version` 显示版本、Git 提交、构建时间、Go 版本和平台信息。

## 什么时候用

- 确认当前 TaskBridge 版本。
- 提交 issue 时需要附上版本信息。
- 升级后确认新版本生效。

## 子命令

`version` 当前没有子命令。

## 常用流程

```bash
taskbridge version
taskbridge version --json
```

## 输出模式

| 模式 | 用途 |
| --- | --- |
| 默认中文版本摘要 | 人类查看。 |
| `--json` | 机器解析版本对象。 |

## 边界

- 只读命令，不访问 storage、Provider 或 token。

## 最短可用流程

```bash
taskbridge version
```
