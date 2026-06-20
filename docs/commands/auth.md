# auth 命令

`taskbridge auth` 管理 Provider 认证、token 状态和刷新。它通过各 Provider 的认证流程保存 token，不把 token 打到 stdout。OAuth 凭证保存在 `~/.taskbridge/credentials/<provider>_credentials.json`，认证 token 保存在统一 token store：`~/.taskbridge/credentials/tokens.json`。

## 什么时候用

适合用 `auth` 的情况：

- 首次使用需要登录某个 Provider。
- Token 过期需要刷新或重新登录。
- 想查看当前各 Provider 的认证状态。
- 想登出某个 Provider。

不适合用 `auth` 的情况：

- 想查看或测试 Provider 功能：用 `provider list` 和 `provider test`。
- 想同步任务：用 `sync`。

## 子命令

| 命令 | 用途 | 写入 |
| --- | --- | --- |
| `taskbridge auth login <provider>` | 登录指定 Provider。 | 写 token store。 |
| `taskbridge auth logout <provider>` | 删除本地 token。 | 写 token store。 |
| `taskbridge auth status` | 查看所有 Provider 认证状态。 | 不写入。 |
| `taskbridge auth show <provider>` | 查看单个 Provider 认证详情。 | 不写入。 |
| `taskbridge auth refresh <provider>` | 刷新或校验 token。 | 可能写 token store。 |

## 各 Provider 登录

### Microsoft Todo

```bash
taskbridge auth login microsoft
```

需要先在 Azure Portal 注册应用并保存凭证到 `~/.taskbridge/credentials/microsoft_credentials.json`。登录时自动打开浏览器进行 OAuth 授权。

### Google Tasks

```bash
taskbridge auth login google
```

需要先在 Google Cloud Console 创建项目、启用 Tasks API 并保存凭证到 `~/.taskbridge/credentials/google_credentials.json`。

### Feishu Tasks

```bash
taskbridge auth login feishu
```

需要先在飞书开放平台创建自建应用并保存凭证到 `~/.taskbridge/credentials/feishu_credentials.json`。

### TickTick

```bash
taskbridge auth login ticktick
```

按提示输入 API Token（以 `tp_` 开头），无需 OAuth。

### 滴答清单 (Dida365)

```bash
taskbridge auth login dida
```

按提示输入 API Token（以 `dp_` 开头），无需 OAuth。支持别名 `dida`、`ticktick_cn`、`tick-cn`。

### Todoist

```bash
taskbridge auth login todoist
```

按提示输入 API Token，无需 OAuth。

## Token 管理流程

```bash
taskbridge auth status
taskbridge auth login <provider>
taskbridge auth show <provider>
taskbridge auth refresh <provider>
taskbridge auth logout <provider>
```

## 输出模式

| 模式 | 用途 |
| --- | --- |
| 默认英文状态 | 人类查看认证状态和修复建议。 |
| `--format json` | 机器解析认证状态。 |

认证相关机器输出必须脱敏，不输出 token、secret、cookie 或 auth header。

## 边界

- JSON stdout 不能混入浏览器提示或人类说明。
- 缺凭证、token 读取失败、未实现认证必须返回非零错误，不能打印错误后成功退出。
- 不保存原始 secret 到 stdout、stderr 或日志。

## 常见错误

| 错误 | 原因 | 处理 |
| --- | --- | --- |
| 凭证文件未找到 | `~/.taskbridge/credentials/<provider>_credentials.json` 不存在。 | 按 Provider 连接指南创建凭证文件。 |
| Token 过期 | OAuth token 已失效。 | 运行 `auth refresh <provider>` 或重新 `auth login`。 |
| 端口被占用 | OAuth 回调端口冲突。 | 修改凭证文件中的 `redirect_url` 端口。 |
| 权限不足 | Provider API 权限未配置。 | 检查 Provider 控制台是否添加了正确的作用域。 |

## 最短可用流程

```bash
taskbridge auth status
taskbridge auth login todoist
taskbridge auth status
```
