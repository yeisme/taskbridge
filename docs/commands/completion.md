# completion 和 help 命令

`taskbridge completion` 生成 shell 自动补全脚本。`taskbridge help` 和 `taskbridge <command> --help` 是当前二进制参数事实来源。

## 支持的 Shell

| Shell | 命令 |
| --- | --- |
| Bash | `taskbridge completion bash` |
| Zsh | `taskbridge completion zsh` |
| Fish | `taskbridge completion fish` |
| PowerShell | `taskbridge completion powershell` |

## 安装补全

### Bash

```bash
taskbridge completion bash > /etc/bash_completion.d/taskbridge
source ~/.bashrc
```

### Zsh

```bash
taskbridge completion zsh > "${fpath[1]}/_taskbridge"
autoload -U compinit && compinit
```

### Fish

```bash
taskbridge completion fish > ~/.config/fish/completions/taskbridge.fish
```

## 输出模式

- completion 输出 shell 脚本到 stdout，不包装成 JSON envelope。
- help 输出人类帮助文本。

## 边界

- completion/help 不访问 storage，不触发远端调用，不写配置。
- 补全脚本基于当前二进制的 Cobra 命令树，升级后需重新生成。

## 最短可用流程

```bash
taskbridge completion zsh
taskbridge help
```
