# Provider 连接指南

本文档详细介绍如何配置和连接各个 Todo 平台 provider。

## 快速选择

| Provider | 认证方式 | 配置难度 | 推荐场景 |
| --- | --- | --- | --- |
| [Todoist](#todoist) | API Token | ⭐ 简单 | 个人任务管理，快速上手 |
| [TickTick](#ticktick) | OpenAPI Token | ⭐ 简单 | 四象限爱好者 |
| [滴答清单](#滴答清单-dida365) | OpenAPI Token | ⭐ 简单 | 国内用户，四象限 |
| [Microsoft Todo](#microsoft-todo) | OAuth 2.0 | ⭐⭐ 中等 | Microsoft 365 用户 |
| [Google Tasks](#google-tasks) | OAuth 2.0 | ⭐⭐ 中等 | Google 生态用户 |
| [飞书任务](#飞书任务) | OAuth 2.0 | ⭐⭐⭐ 复杂 | 飞书团队协作 |

推荐首次使用从 Todoist 或 TickTick 开始，只需一个 API Token 即可。

## 目录

- [Google Tasks](#google-tasks)
- [Microsoft Todo](#microsoft-todo)
- [飞书任务](#飞书任务)
- [TickTick](#ticktick)
- [滴答清单 (Dida365)](#滴答清单-dida365)
- [Todoist](#todoist)
- [多 Provider 共存](#多-provider-共存)
- [常用命令](#常用命令)
- [故障排除](#故障排除)

## 凭证文件路径汇总

```
~/.taskbridge/
├── config.yaml                  # 主配置文件
├── credentials/                 # OAuth 凭证（需手动创建）
│   ├── google.json             # Google OAuth 客户端凭证
│   ├── microsoft.json          # Azure AD 应用凭证
│   └── feishu.json             # 飞书自建应用凭证
└── tokens/                      # 认证 token（自动生成）
    ├── google.json
    ├── microsoft.json
    ├── feishu.json
    ├── ticktick.json
    ├── dida.json
    └── todoist.json
```

---

## Google Tasks

### 前置要求

- Google Cloud Platform 账号
- 已启用的 Google Tasks API

### 步骤 1: 创建 Google Cloud 项目

1. 访问 [Google Cloud Console](https://console.cloud.google.com/)
2. 创建新项目或选择现有项目
3. 记录项目 ID

### 步骤 2: 启用 Google Tasks API

1. 在左侧菜单中选择 **API 和服务** > **库**
2. 搜索 "Tasks API"
3. 点击 **启用**

### 步骤 3: 配置 OAuth 同意屏幕

1. 转到 **API 和服务** > **OAuth 同意屏幕**
2. 选择用户类型（外部/内部）
3. 填写应用名称、支持邮箱等信息
4. 添加以下作用域：
   - `https://www.googleapis.com/auth/tasks`
   - `https://www.googleapis.com/auth/tasks.readonly`

### 步骤 4: 创建 OAuth 客户端凭证

1. 转到 **API 和服务** > **凭证**
2. 点击 **创建凭证** > **OAuth 客户端 ID**
3. 应用类型选择 **桌面应用**
4. 记录 **客户端 ID** 和 **客户端密钥**

### 步骤 5: 保存凭证文件

创建凭证文件 `~/.taskbridge/credentials/google_credentials.json`：

```json
{
  "client_id": "你的客户端ID.apps.googleusercontent.com",
  "client_secret": "你的客户端密钥",
  "redirect_url": "http://127.0.0.1:8080/callback"
}
```

### 步骤 6: 登录认证

```bash
taskbridge auth login google
```

系统会自动打开浏览器进行 OAuth 授权，完成后 token 将保存到统一 token store：`~/.taskbridge/credentials/tokens.json`。

### 步骤 7: 验证连接

```bash
taskbridge provider test google
```

---

## Microsoft Todo

### 前置要求

- Microsoft Azure 账号
- Microsoft 365 订阅（个人或工作账号）

### 步骤 1: 注册 Azure AD 应用

1. 访问 [Azure Portal](https://portal.azure.com/)
2. 转到 **Azure Active Directory** > **应用注册**
3. 点击 **新注册**
4. 填写应用名称，选择支持的账户类型
5. 记录 **应用程序(客户端) ID**

### 步骤 2: 配置身份验证

1. 在应用页面，点击 **身份验证**
2. 添加平台 > **Web**
3. 添加重定向 URI：`http://127.0.0.1:8080/callback`
4. 勾选 **访问令牌** 和 **ID 令牌**

### 步骤 3: 创建客户端密钥

1. 点击 **证书和密码**
2. 点击 **新客户端密码**
3. 记录生成的 **密钥值**（只显示一次）

### 步骤 4: 配置 API 权限

1. 点击 **API 权限**
2. 添加权限 > **Microsoft Graph**
3. 选择 **委托的权限**
4. 添加以下权限：
   - `Tasks.Read`
   - `Tasks.ReadWrite`
   - `Tasks.Read.Shared`
   - `Tasks.ReadWrite.Shared`

### 步骤 5: 保存凭证文件

创建凭证文件 `~/.taskbridge/credentials/microsoft_credentials.json`：

```json
{
  "client_id": "你的应用程序ID",
  "client_secret": "你的客户端密钥",
  "redirect_url": "http://127.0.0.1:8080/callback",
  "tenant_id": "common"
}
```

### 步骤 6: 登录认证

```bash
taskbridge auth login microsoft
```

### 步骤 7: 验证连接

```bash
taskbridge provider test microsoft
```

---

## 飞书任务

### 前置要求

- 飞书开发者账号
- 已创建的自建应用

### 步骤 1: 创建飞书应用

1. 访问 [飞书开放平台](https://open.feishu.cn/)
2. 点击 **创建企业自建应用**
3. 填写应用名称和描述
4. 记录 **App ID** 和 **App Secret**

### 步骤 2: 配置应用权限

1. 进入应用 > **权限管理**
2. 申请以下权限：
   - `task:tasklist:read` - 获取任务列表
   - `task:tasklist:write` - 创建和更新任务列表
   - `task:task:read` - 获取任务详情
   - `task:task:write` - 创建和更新任务

### 步骤 3: 配置重定向 URL

1. 进入 **安全设置**
2. 添加重定向 URL：`http://127.0.0.1:3456/callback`

### 步骤 4: 发布应用版本

1. 进入 **版本管理与发布**
2. 创建版本并提交审核
3. 审核通过后发布

### 步骤 5: 保存凭证文件

创建凭证文件 `~/.taskbridge/credentials/feishu_credentials.json`：

```json
{
  "app_id": "cli_xxxxxxxxxxxx",
  "app_secret": "xxxxxxxxxxxxxxxx",
  "redirect_url": "http://127.0.0.1:3456/callback",
  "scopes": [
    "task:tasklist:read",
    "task:tasklist:write",
    "task:task:read",
    "task:task:write"
  ]
}
```

### 步骤 6: 登录认证

```bash
taskbridge auth login feishu
```

### 步骤 7: 验证连接

```bash
taskbridge provider test feishu
```

---

## TickTick

TickTick 使用官方 OpenAPI Token 认证。

### 步骤 1: 获取 API Token

1. 打开 TickTick 开发者平台并登录
2. 创建或查看个人 API Token
3. 复制 Token
4. Token 通常以 `tp_` 开头

### 步骤 2: 登录认证

```bash
taskbridge auth login ticktick
```

按提示输入 API Token，认证成功后 token 将保存到统一 token store：`~/.taskbridge/credentials/tokens.json`。

### 步骤 3: 验证连接

```bash
taskbridge provider test ticktick
```

### 注意事项

- TickTick 使用官方静态 Token，无需刷新
- Token 有效期较长，但建议定期检查

---

## 滴答清单 (Dida365)

滴答清单是 TickTick 的国内版本，使用 OpenAPI Token 认证。

### 步骤 1: 获取 API Token

1. 打开滴答清单开发者平台并登录
2. 创建或查看个人 API Token
3. 复制 Token
4. Token 通常以 `dp_` 开头

### 步骤 2: 登录认证

```bash
taskbridge auth login dida
```

### 步骤 3: 验证连接

```bash
taskbridge provider test dida
```

### 别名支持

滴答清单支持以下别名：

- `dida` - 推荐使用
- `ticktick_cn` - TickTick 国内版
- `tick-cn` - 简写形式

---

## Todoist

Todoist 使用 API Token 认证。

### 步骤 1: 获取 API Token

1. 登录 [Todoist](https://todoist.com)
2. 点击右上角头像 > **设置**
3. 选择 **集成** 选项卡
4. 找到 **API Token** 部分
5. 复制显示的 Token

### 步骤 2: 登录认证

```bash
taskbridge auth login todoist
```

按提示输入 API Token。

### 步骤 3: 验证连接

```bash
taskbridge provider test todoist
```

---

## 多 Provider 共存

TaskBridge 支持同时连接多个 Provider。启用方式：

```bash
# 启用多个 Provider
taskbridge provider enable microsoft
taskbridge provider enable todoist
taskbridge provider enable ticktick

# 设置环境变量
export TASKBRIDGE_PROVIDERS=microsoft,todoist,ticktick

# 查看所有 Provider 状态
taskbridge auth status

# 按需同步不同 Provider
taskbridge sync pull microsoft
taskbridge sync pull todoist
```

### 跨 Provider 同步

```bash
# 查看 Microsoft 和 Todoist 之间的差异
taskbridge sync diff microsoft --target todoist --format json

# 双向同步
taskbridge sync bidirectional microsoft
```

每个 Provider 的任务通过 `source` 字段区分来源，不会混淆。

---

## 常用命令

### 查看认证状态

```bash
# 查看所有 provider 状态
taskbridge auth status

# 查看特定 provider 状态
taskbridge auth status google
taskbridge auth status microsoft
taskbridge auth status feishu
taskbridge auth status ticktick
taskbridge auth status dida
taskbridge auth status todoist
```

### 刷新 Token

```bash
taskbridge auth refresh google
taskbridge auth refresh microsoft
taskbridge auth refresh feishu
taskbridge auth refresh ticktick  # 静态 token，仅校验
taskbridge auth refresh dida      # 静态 token，仅校验
taskbridge auth refresh todoist
```

### 登出

```bash
taskbridge auth logout google
```

---

## 故障排除

### Token 过期

如果遇到 token 过期错误：

```bash
# 刷新 token
taskbridge auth refresh <provider>

# 或重新登录
taskbridge auth login <provider>
```

### 凭证文件未找到

确保凭证文件位于正确的位置：

```
~/.taskbridge/
├── credentials/
│   ├── google.json
│   ├── microsoft.json
│   └── feishu.json
└── tokens/
    ├── google.json
    ├── microsoft.json
    ├── feishu.json
    ├── ticktick.json
    ├── dida.json
    └── todoist.json
```

### 端口被占用

如果 OAuth 回调端口被占用，可以修改凭证文件中的 `redirect_url` 使用其他端口。

### 权限不足

确保在各个平台配置了正确的 API 权限/作用域。

### 同步异常

```bash
# 检查 Provider 连接
taskbridge provider test <provider>

# 查看同步状态
taskbridge sync status

# 查看同步冲突
taskbridge sync conflicts
```

### 全局诊断

```bash
taskbridge doctor
taskbridge doctor --format json
```

---

## 配置文件示例

完整的配置文件 `~/.taskbridge/config.yaml`：

```yaml
intelligence:
  enabled: true
  timezone: Asia/Shanghai

providers:
  microsoft:
    enabled: false
  google:
    enabled: false
  feishu:
    enabled: false
  ticktick:
    enabled: false
  dida:
    enabled: false
  todoist:
    enabled: false

storage:
  type: file
  path: ~/.taskbridge/data

sync:
  auto: false
  interval: 5m
```

启用需要使用的 provider 后即可开始同步任务。
