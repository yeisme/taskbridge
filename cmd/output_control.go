package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/yeisme/taskbridge/internal/clioutput"
)

func wantsJSON(format string) bool {
	return outputJSON || strings.EqualFold(strings.TrimSpace(format), "json") || IsQuietMode()
}

func resolveOutputFormat(format string) string {
	if outputJSON {
		return "json"
	}
	if outputAgent {
		return "agent"
	}
	if outputEvents {
		return "events"
	}
	if outputExplain {
		return "explain"
	}
	return strings.ToLower(strings.TrimSpace(format))
}

func globalProjectionModeRequested() bool {
	return outputJSON || outputAgent || outputEvents || outputExplain
}

func printStructured(format string, value interface{}, renderText func()) error {
	if wantsJSON(format) {
		data, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return commandError("Serialized output failed", err)
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

func printProjection(format string, projection clioutput.Projection, renderText func()) error {
	switch resolveOutputFormat(format) {
	case "json":
		data, err := clioutput.RenderJSON(projection)
		if err != nil {
			return commandError("Serialized output failed", err)
		}
		fmt.Fprintln(os.Stdout, string(data))
		return nil
	case "agent", "ai":
		fmt.Fprint(os.Stdout, clioutput.RenderAgent(projection))
		return nil
	case "events":
		return usageError("--events is not supported for this command")
	case "explain":
		fmt.Fprint(os.Stdout, clioutput.RenderExplain(projection))
		return nil
	}

	if wantsJSON(format) {
		data, err := clioutput.RenderJSON(projection)
		if err != nil {
			return commandError("Serialized output failed", err)
		}
		fmt.Fprintln(os.Stdout, string(data))
		return nil
	}
	if renderText != nil {
		renderText()
		return nil
	}
	fmt.Fprint(os.Stdout, clioutput.RenderSummary(projection))
	return nil
}
