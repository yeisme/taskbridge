# config 命令

`taskbridge config` 查看当前配置和兼容旧配置命令。当前项目主要通过环境变量和命令行参数配置运行时。

## 配置层级

配置优先级从高到低：

1. **命令行参数** — `--source microsoft` 等
2. **环境变量** — `TASKBRIDGE_HOME`、`TASKBRIDGE_STORAGE_PATH`、`TASKBRIDGE_PROVIDERS`
3. **配置文件** — `~/.taskbridge/config.yaml`

## 环境变量

| 变量 | 用途 | 默认值 |
| --- | --- | --- |
| `TASKBRIDGE_HOME` | TaskBridge 主目录。 | `~/.taskbridge` |
| `TASKBRIDGE_STORAGE_PATH` | 本地存储路径。 | `$TASKBRIDGE_HOME/data` |
| `TASKBRIDGE_PROVIDERS` | 启用的 Provider 列表，逗号分隔。 | （空） |

## 子命令

| 命令 | 用途 | 写入 |
| --- | --- | --- |
| `taskbridge config show` | 显示当前生效配置。 | 不写入。 |
| `taskbridge config get <key>` | 获取配置项。 | 不写入。 |
| `taskbridge config set <key> <value>` | 已弃用，提示使用环境变量/flag。 | 不写入或返回 usage error。 |
| `taskbridge config init` | 已弃用，提示使用环境变量/flag。 | 不写入或返回 usage error。 |
| `taskbridge config validate` | 验证配置。 | 不写入。 |

## 常用流程

```bash
taskbridge config show
taskbridge config show --format json
taskbridge config validate
TASKBRIDGE_STORAGE_PATH=./data taskbridge config show --format json
```

## 配置文件示例

`~/.taskbridge/config.yaml`：

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

## 输出模式

| 格式 | 用途 |
| --- | --- |
| `yaml`（默认） | 人类浏览配置。 |
| `json` | 机器解析配置。 |

配置输出必须脱敏，不打印 token、secret 或 cookie。

## 边界

- 普通只读命令不应隐式创建或修改配置文件。
- 新配置写入能力如恢复，必须明确目标文件和写入范围。

## 常见错误

| 错误 | 原因 | 处理 |
| --- | --- | --- |
| 配置文件不存在 | 首次使用未创建配置文件。 | TaskBridge 会使用默认值，不影响使用。 |
| 环境变量未生效 | 命令行参数优先级更高。 | 检查是否有冲突的 flag。 |

## 最短可用流程

```bash
taskbridge config show
```
