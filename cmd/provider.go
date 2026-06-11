package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yeisme/taskbridge/internal/clioutput"
	"github.com/yeisme/taskbridge/internal/provider"
	"github.com/yeisme/taskbridge/pkg/ui"
)

// providerCmd Provider management command
var providerCmd = &cobra.Command{
	Use:   "provider",
	Short: "Provider management",
	Long: `Manage Todo providers.

Subcommands:
  list       List all providers
  enable     Enable a provider
  disable    Disable a provider
  configure  Configure a provider
  test       Test provider connection

Examples:
  taskbridge provider list
  taskbridge provider enable google
  taskbridge provider test google`,
}

// providerListCmd lists Providers
var providerListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all Providers",
	Long:  `List all supported providers and their status`,
	RunE:  runProviderList,
}

// providerEnableCmd Enable Provider
var providerEnableCmd = &cobra.Command{
	Use:   "enable <provider>",
	Short: "Enable Provider",
	Long:  `Enable the specified Provider`,
	Args:  cobra.ExactArgs(1),
	RunE:  runProviderEnable,
}

// providerDisableCmd Disable Provider
var providerDisableCmd = &cobra.Command{
	Use:   "disable <provider>",
	Short: "Disable Provider",
	Long:  `Disable the specified Provider`,
	Args:  cobra.ExactArgs(1),
	RunE:  runProviderDisable,
}

// providerTestCmd Test Provider
var providerTestCmd = &cobra.Command{
	Use:   "test <provider>",
	Short: "Test Provider connection",
	Long:  `Test the selected provider connection and authentication status`,
	Args:  cobra.ExactArgs(1),
	RunE:  runProviderTest,
}

// providerInfoCmd displays Provider information
var providerInfoCmd = &cobra.Command{
	Use:   "info <provider>",
	Short: "Show provider details",
	Long:  `Display details and capabilities of the specified Provider`,
	Args:  cobra.ExactArgs(1),
	RunE:  runProviderInfo,
}

func init() {
	rootCmd.AddCommand(providerCmd)
	providerCmd.AddCommand(providerListCmd)
	providerCmd.AddCommand(providerEnableCmd)
	providerCmd.AddCommand(providerDisableCmd)
	providerCmd.AddCommand(providerTestCmd)
	providerCmd.AddCommand(providerInfoCmd)
}

// ProviderInfo Provider information
type ProviderInfo struct {
	Name         string
	ShortName    string
	DisplayName  string
	Description  string
	AuthType     string
	Enabled      bool
	Connected    bool
	Capabilities []string
}

// getProviderInfos gets all Provider information
func getProviderInfos() map[string]ProviderInfo {
	return map[string]ProviderInfo{
		"google": {
			Name:        "google",
			ShortName:   "google",
			DisplayName: "Google Tasks",
			Description: "Google task management service",
			AuthType:    "OAuth2",
			Enabled:     cfg.Providers.Google.Enabled,
		},
		"microsoft": {
			Name:        "microsoft",
			ShortName:   "ms",
			DisplayName: "Microsoft To Do",
			Description: "Microsoft Task Management Service",
			AuthType:    "OAuth2",
			Enabled:     cfg.Providers.Microsoft.Enabled,
		},
		"feishu": {
			Name:        "feishu",
			ShortName:   "feishu",
			DisplayName: "Feishu Tasks",
			Description: "Feishu task management",
			AuthType:    "App ID/Secret",
			Enabled:     cfg.Providers.Feishu.Enabled,
		},
		"ticktick": {
			Name:        "ticktick",
			ShortName:   "tick",
			DisplayName: "TickTick",
			Description: "TickTick task management",
			AuthType:    "API Token",
			Enabled:     cfg.Providers.TickTick.Enabled,
		},
		"dida": {
			Name:        "dida",
			ShortName:   "tick_cn",
			DisplayName: "Dida365",
			Description: "Tick-tock list (domestic)",
			AuthType:    "API Token",
			Enabled:     cfg.Providers.Dida.Enabled,
		},
		"todoist": {
			Name:        "todoist",
			ShortName:   "todo",
			DisplayName: "Todoist",
			Description: "Todoist task management",
			AuthType:    "API Token",
			Enabled:     cfg.Providers.Todoist.Enabled,
		},
	}
}

type ProviderListRow struct {
	Name        string `json:"name"`
	ShortName   string `json:"short_name"`
	Status      string `json:"status"`
	AuthType    string `json:"auth_type"`
	Description string `json:"description"`
}

func buildProviderListProjection() clioutput.Projection {
	providers := getProviderInfos()
	rows := make([]ProviderListRow, 0, len(providers))
	connected := 0
	for _, name := range provider.GetAllProviderNames() {
		p := providers[name]
		status := providerStatusLabel(name, p)
		if status == "Connected" {
			connected++
		}
		rows = append(rows, ProviderListRow{p.DisplayName, p.ShortName, status, p.AuthType, p.Description})
	}
	projection := clioutput.New("provider.list")
	projection.Summary = "Providers listed."
	projection.Facts["count"] = len(rows)
	projection.Facts["connected"] = connected
	projection.Data = map[string]any{"providers": rows}
	return projection
}

func providerStatusLabel(name string, p ProviderInfo) string {
	snapshot := getProviderAuthSnapshot(name)
	switch snapshot.StatusText {
	case "✅ Connected":
		return "Connected"
	case "⚠️ Expired":
		return "Expired"
	case "❌ Not authenticated":
		return "Not authenticated"
	default:
		if p.Enabled {
			return "Enabled"
		}
		return "Disabled"
	}
}

func statusWithIcon(status string) string {
	switch status {
	case "Connected":
		return "✅ Connected"
	case "Expired":
		return "⚠️ Expired"
	case "Not authenticated":
		return "❌ Not authenticated"
	case "Enabled":
		return "🟢 Enabled"
	case "Disabled":
		return "⚪ Disabled"
	case "Not enabled":
		return "⚪ Not enabled"
	case "Not configured":
		return "⚪ Not configured"
	case "Token error":
		return "⚠️ Token error"
	default:
		return status
	}
}

func buildProviderWriteProjection(command, providerName, state string) clioutput.Projection {
	projection := clioutput.New(command)
	projection.Summary = fmt.Sprintf("%s Provider %s %s.", providerWriteStatusIcon(state), providerName, state)
	projection.Facts["provider"] = providerName
	projection.Facts["state"] = state
	projection.Data = map[string]any{"provider": providerName, "state": state}
	switch state {
	case "enabled":
		projection.Actions = []clioutput.Action{
			{Name: "login", Command: "taskbridge auth login " + providerName},
			{Name: "test", Command: "taskbridge provider test " + providerName},
		}
	case "disabled":
		projection.Actions = []clioutput.Action{{Name: "enable", Command: "taskbridge provider enable " + providerName}}
	}
	return projection
}

func providerWriteStatusIcon(state string) string {
	switch state {
	case "enabled":
		return "✅"
	case "disabled":
		return "⚪"
	default:
		return "•"
	}
}

func renderProviderWriteReceipt(projection clioutput.Projection) string {
	var b strings.Builder
	b.WriteString("Status\n")
	b.WriteString(projection.Summary)
	b.WriteString("\n\nFacts\n")
	b.WriteString("- provider: ")
	b.WriteString(fmt.Sprint(projection.Facts["provider"]))
	b.WriteString("\n- state: ")
	b.WriteString(fmt.Sprint(projection.Facts["state"]))
	b.WriteString("\n")
	if len(projection.Actions) > 0 {
		b.WriteString("\nRecommended next steps\n")
		for _, action := range projection.Actions {
			if action.Command == "" {
				continue
			}
			b.WriteString("- ")
			b.WriteString(action.Command)
			b.WriteString("\n")
		}
	}
	return b.String()
}

func setProviderEnabled(providerName string, enabled bool) {
	switch providerName {
	case "google":
		cfg.Providers.Google.Enabled = enabled
	case "microsoft":
		cfg.Providers.Microsoft.Enabled = enabled
	case "feishu":
		cfg.Providers.Feishu.Enabled = enabled
	case "ticktick":
		cfg.Providers.TickTick.Enabled = enabled
	case "dida":
		cfg.Providers.Dida.Enabled = enabled
	case "todoist":
		cfg.Providers.Todoist.Enabled = enabled
	}
}

func renderProviderList(projection clioutput.Projection) string {
	data, _ := projection.Data.(map[string]any)
	rows, _ := data["providers"].([]ProviderListRow)
	table := ui.NewTable("Provider", "Alias", "Status", "Auth", "Description")
	for _, row := range rows {
		table.AddRow(row.Name, row.ShortName, statusWithIcon(row.Status), row.AuthType, row.Description)
	}
	return "\n" + table.Render() + "\n\nHint: use 'taskbridge provider info <alias>' for details.\n"
}

func runProviderList(_ *cobra.Command, _ []string) error {
	projection := buildProviderListProjection()
	return printProjection("text", projection, func() {
		fmt.Print(renderProviderList(projection))
	})
}

func runProviderEnable(_ *cobra.Command, args []string) error {
	//Parse Provider name (supports abbreviation)
	providerName := provider.ResolveProviderName(args[0])

	//Check if Provider exists
	if !provider.IsValidProvider(providerName) {
		return usageError("Unknown Provider:" + args[0])
	}

	setProviderEnabled(providerName, true)
	projection := buildProviderWriteProjection("provider.enable", providerName, "enabled")
	if outputJSON || outputAgent || outputEvents || outputExplain {
		return printProjection("text", projection, nil)
	}
	fmt.Print(renderProviderWriteReceipt(projection))
	return nil
}

func runProviderDisable(_ *cobra.Command, args []string) error {
	//Parse Provider name (supports abbreviation)
	providerName := provider.ResolveProviderName(args[0])

	//Check if Provider exists
	if !provider.IsValidProvider(providerName) {
		return usageError("Unknown Provider:" + args[0])
	}

	setProviderEnabled(providerName, false)
	projection := buildProviderWriteProjection("provider.disable", providerName, "disabled")
	if outputJSON || outputAgent || outputEvents || outputExplain {
		return printProjection("text", projection, nil)
	}
	fmt.Print(renderProviderWriteReceipt(projection))
	return nil
}

func buildProviderTestProjection(providerName string) clioutput.Projection {
	providerInfos := getProviderInfos()
	p := providerInfos[providerName]
	status := providerStatusLabel(providerName, p)
	projection := clioutput.New("provider.test")
	projection.Summary = "Provider connection check completed."
	projection.Facts["provider"] = providerName
	projection.Facts["enabled"] = p.Enabled
	projection.Facts["status"] = status
	if !p.Enabled {
		projection.Status = clioutput.StatusPartial
		projection.Actions = []clioutput.Action{{Name: "enable", Command: "taskbridge provider enable " + providerName}}
	} else if status != "Connected" {
		projection.Status = clioutput.StatusPartial
		projection.Actions = []clioutput.Action{{Name: "login", Command: "taskbridge auth login " + providerName}}
	}
	projection.Data = map[string]any{"provider": providerName, "enabled": p.Enabled, "status": status}
	return projection
}

func renderProviderTest(projection clioutput.Projection) string {
	return clioutput.RenderSummary(projection)
}

func runProviderTest(_ *cobra.Command, args []string) error {
	providerName := provider.ResolveProviderName(args[0])
	if !provider.IsValidProvider(providerName) {
		return usageError("Unknown Provider:" + args[0])
	}
	projection := buildProviderTestProjection(providerName)
	return printProjection("text", projection, func() {
		fmt.Print(renderProviderTest(projection))
	})
}

func buildProviderInfoProjection(providerName string) clioutput.Projection {
	def, _ := provider.GetProviderDefinition(providerName)
	providerInfos := getProviderInfos()
	p := providerInfos[providerName]
	status := "Not enabled"
	if p.Enabled {
		status = "Enabled"
	}
	projection := clioutput.New("provider.info")
	projection.Summary = def.DisplayName + " provider details."
	projection.Facts["provider"] = providerName
	projection.Facts["status"] = status
	projection.Facts["auth_type"] = p.AuthType
	projection.Data = map[string]any{
		"name":         def.Name,
		"display_name": def.DisplayName,
		"description":  def.Description,
		"auth_type":    p.AuthType,
		"status":       status,
		"capabilities": getProviderCapabilities(providerName),
	}
	return projection
}

func renderProviderInfo(projection clioutput.Projection) string {
	data, _ := projection.Data.(map[string]any)
	return clioutput.RenderSummary(clioutput.Projection{
		SpecVersion: clioutput.SpecVersion,
		Command:     projection.Command,
		Status:      clioutput.StatusSuccess,
		Summary:     projection.Summary,
		Facts: map[string]any{
			"Provider": data["display_name"],
			"Status":   statusWithIcon(fmt.Sprint(data["status"])),
			"Auth":     data["auth_type"],
		},
		Actions: []clioutput.Action{{Name: "login", Command: "taskbridge auth login " + fmt.Sprint(projection.Facts["provider"])}},
	})
}

func runProviderInfo(cmd *cobra.Command, args []string) error {
	providerName := provider.ResolveProviderName(args[0])
	if !provider.IsValidProvider(providerName) {
		return usageError("Unknown Provider:" + args[0])
	}
	projection := buildProviderInfoProjection(providerName)
	return printProjection("text", projection, func() {
		fmt.Print(renderProviderInfo(projection))
	})
}

func getProviderCapabilities(providerName string) []string {
	switch providerName {
	case "google":
		return []string{
			"expiration date",
			"task list",
			"Subtasks (limited support)",
			"priority (not supported)",
			"labels (not supported)",
			"delta sync (not supported)",
		}
	case "microsoft":
		return []string{
			"expiration date",
			"task list",
			"subtask",
			"priority",
			"remind",
			"labels (not supported)",
		}
	case "feishu":
		return []string{
			"expiration date",
			"task list",
			"priority",
			"Label",
			"Subtasks (limited support)",
		}
	case "ticktick":
		return []string{
			"expiration date",
			"task list",
			"subtask",
			"priority",
			"Label",
			"remind",
		}
	case "dida":
		return []string{
			"expiration date",
			"task list",
			"subtask",
			"priority",
			"Label",
			"remind",
		}
	case "todoist":
		return []string{
			"expiration date",
			"project",
			"subtask",
			"priority",
			"Label",
		}
	default:
		return []string{"unknown"}
	}
}
