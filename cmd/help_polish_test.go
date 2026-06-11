package cmd

import (
	"io"
	"strings"
	"testing"
)

func TestCommandHelpChromeIsEnglish(t *testing.T) {
	commands := map[string]interface {
		Help() error
		SetOut(io.Writer)
		SetErr(io.Writer)
	}{
		"config":        configCmd,
		"provider":      providerCmd,
		"provider list": providerListCmd,
		"provider test": providerTestCmd,
		"tui":           tuiCmd,
	}

	for name, command := range commands {
		output := helpOutput(t, command)
		for _, disallowed := range []string{"管理", "子命令", "配置", "示例", "测试指定", "认证state", "已弃用", "启动", "使用键盘", "删除", "恢复", "退出"} {
			if strings.Contains(output, disallowed) {
				t.Fatalf("%s help should use English CLI chrome, found %q in:\n%s", name, disallowed, output)
			}
		}
	}
}
