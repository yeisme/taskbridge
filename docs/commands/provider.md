# provider 命令

`taskbridge provider` 管理 Provider 目录、启用状态和能力信息。它和 `auth` 的区别是：`provider` 管理 Provider 的功能启用和配置，`auth` 管理认证 token。

## 什么时候用

适合用 `provider` 的情况：

- 想查看当前支持哪些 Provider 以及启用状态。
- 想启用或禁用某个 Provider。
- 想查看某个 Provider 的能力详情（支持哪些操作）。
- 想测试 Provider 连接和认证是否正常。

不适合用 `provider` 的情况：

- 想登录或刷新 token：用 `auth`。
- 想同步任务：用 `sync`。
- 想查看任务列表：用 `list`。

## 子命令

| 命令 | 用途 | 写入 |
| --- | --- | --- |
| `taskbridge provider list` | 查看支持的 Provider 和状态。 | 不写入。 |
| `taskbridge provider info <provider>` | 查看 Provider 能力详情。 | 不写入。 |
| `taskbridge provider enable <provider>` | 启用 Provider 配置。 | 修改运行时配置状态。 |
| `taskbridge provider disable <provider>` | 禁用 Provider 配置。 | 修改运行时配置状态。 |
| `taskbridge provider test <provider>` | 测试启用和认证状态。 | 不写远端。 |

## 支持的 Provider

| Provider | 标识 | 认证方式 | 特点 |
| --- | --- | --- | --- |
| Microsoft Todo | `microsoft` | OAuth 2.0 | 完整支持，Azure AD 应用 |
| Google Tasks | `google` | OAuth 2.0 | 基础支持，Google Cloud 项目 |
| Feishu Tasks | `feishu` | OAuth 2.0 | 完整支持，飞书自建应用 |
| TickTick | `ticktick` | OpenAPI Token | 原生四象限支持 |
| 滴答清单 | `dida` | OpenAPI Token | 国内版，别名 `ticktick_cn` |
| Todoist | `todoist` | API Token | 完整支持 |

## Provider 管理流程

```bash
taskbridge provider list
taskbridge provider enable todoist
taskbridge auth login todoist
taskbridge provider test todoist
taskbridge provider info todoist
```

启用和认证是两个独立步骤：`provider enable` 修改配置状态，`auth login` 完成认证。两者都完成后才能正常使用。

## 输出模式

| 模式 | 用途 |
| --- | --- |
| 默认 human table/card | 人类浏览 Provider 列表。 |
| `--format json` | 机器解析 Provider 状态。 |

## 边界

- 不保存 token；token 管理在 `auth`。
- 不绕过 Provider 接口执行远端写入。
- `provider test` 只做连接和认证校验，不读写远端任务数据。

## 常见错误

| 错误 | 原因 | 处理 |
| --- | --- | --- |
| Provider 未启用 | 尝试同步未启用的 Provider。 | 先运行 `provider enable <name>`。 |
| 认证失败 | `provider test` 报告认证异常。 | 运行 `auth login <provider>` 重新认证。 |
| Provider 不支持某操作 | 某些 Provider 能力有限。 | 用 `provider info <name>` 查看支持的操作。 |

## 最短可用流程

```bash
taskbridge provider list
taskbridge provider enable todoist
taskbridge auth login todoist
taskbridge provider test todoist
```
