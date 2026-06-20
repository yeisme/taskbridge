# doctor 命令

`taskbridge doctor` 检查 TaskBridge 本地环境，包括 storage path、project store 和 Provider 认证状态。它是首次使用和新环境诊断的推荐入口。

## 什么时候用

适合用 `doctor` 的情况：

- 首次安装 TaskBridge 后想确认环境是否就绪。
- 遇到问题想快速排查是配置、认证还是存储的问题。
- 升级后想确认新版本环境兼容。

不适合用 `doctor` 的情况：

- 想看推荐下一步：用 `quickstart`。
- 想查看配置详情：用 `config show`。
- 想测试 Provider 连接：用 `provider test`。

## 子命令

`doctor` 当前没有子命令。

## 常用流程

```bash
taskbridge doctor
taskbridge doctor --format json
```

doctor 会检查以下项目：

1. **存储路径** — `TASKBRIDGE_STORAGE_PATH` 是否可写。
2. **Provider 认证** — 各 Provider 凭证文件和 token 状态。
3. **项目存储** — project store 是否正常。
4. **配置完整性** — 必要的配置项是否齐全。

## 输出模式

| 模式 | 用途 |
| --- | --- |
| 默认英文检查结果 | 人类查看，附一条推荐下一步。 |
| `--format json` | 输出 `taskbridge.doctor.v1` 结构。 |

## 边界

- 允许做本地写权限探测，但不创建任务、不调用远端 Provider 写 API。
- JSON stdout 不能混入日志、提示或进度条。

## 常见错误

| 错误 | 原因 | 处理 |
| --- | --- | --- |
| 存储路径不可写 | 目录不存在或权限不足。 | 创建目录或调整权限。 |
| 凭证文件缺失 | 未完成 Provider 认证。 | 按 Provider 连接指南创建凭证文件。 |
| Token 过期 | OAuth token 已失效。 | 运行 `auth refresh <provider>`。 |

## 最短可用流程

```bash
taskbridge doctor
```
