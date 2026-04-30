package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func wantsJSON(format string) bool {
	return strings.EqualFold(strings.TrimSpace(format), "json") || IsQuietMode()
}

func printStructured(format string, value interface{}, renderText func()) error {
	if wantsJSON(format) {
		data, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return commandError("序列化输出失败", err)
		}
		fmt.Fprintln(os.Stdout, string(data))
		return nil
	}
	if renderText != nil {
		renderText()
		return nil
	}
	return printResult(value)
}
