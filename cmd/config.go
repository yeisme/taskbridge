package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yeisme/taskbridge/internal/clioutput"
	pkgconfig "github.com/yeisme/taskbridge/pkg/config"
	"gopkg.in/yaml.v3"
)

var (
	configShowSensitive bool
	configFormat        string
)

// configCmd configuration command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management",
	Long: `Manage TaskBridge runtime configuration. Configuration files are deprecated; prefer environment variables and command-line flags.

Subcommands:
  show     Show current configuration
  set      Set configuration items (deprecated)
  get      Get configuration items
  init     Initialize a configuration file (deprecated)
  validate Validate configuration

Examples:
  taskbridge config show
  taskbridge config set storage.path ./mydata
  taskbridge config get providers.google.enabled
  taskbridge config init`,
}

// configShowCmd shows configuration
var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Long:  `Display currently loaded configuration information`,
	RunE:  runConfigShow,
}

// configSetCmd set configuration
var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set configuration items",
	Long: `Set the specified configuration item.

Example:
  taskbridge config set storage.path ./mydata
  taskbridge config set providers.google.enabled true
  taskbridge config set sync.interval 10m`,
	Args: cobra.ExactArgs(2),
	RunE: runConfigSet,
}

// configGetCmd Get configuration
var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get configuration items",
	Long: `Get the value of the specified configuration item.

Example:
  taskbridge config get storage.path
  taskbridge config get providers.google.enabled`,
	Args: cobra.ExactArgs(1),
	RunE: runConfigGet,
}

// configInitCmd initializes the configuration
var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize configuration file",
	Long:  `Create a default configuration file in the current directory or specified location. Deprecated: use environment variables or flags instead`,
	RunE:  runConfigInit,
}

// configValidateCmd Verify configuration
var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Verify configuration",
	Long:  `Verify that the current configuration is valid`,
	RunE:  runConfigValidate,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configValidateCmd)

	configShowCmd.Flags().BoolVar(&configShowSensitive, "sensitive", false, "Show sensitive information")
	configShowCmd.Flags().StringVarP(&configFormat, "format", "f", "yaml", "Output format (yaml, json)")

	configInitCmd.Flags().StringVar(&cfgFile, "output", "", "Configuration file output path")
}

func buildConfigProjection(current *pkgconfig.Config) clioutput.Projection {
	p := clioutput.New("config.show")
	p.Summary = "TaskBridge configuration loaded."
	p.Facts["source"] = "default+env+flags"
	if current != nil {
		p.Facts["storage_type"] = current.Storage.Type
		p.Facts["storage_path"] = current.Storage.Path
		p.Facts["log_level"] = current.App.LogLevel
	}
	p.Data = current
	return p
}

func buildConfigGetProjection(key string, value any) clioutput.Projection {
	projection := clioutput.New("config.get")
	valueText := configValueText(value)
	if isSensitiveConfigKey(key) {
		valueText = clioutput.RedactedValue
		value = clioutput.RedactedValue
	}
	projection.Summary = "Configuration value loaded."
	projection.Facts["key"] = key
	projection.Facts["value"] = valueText
	projection.Data = map[string]any{"key": key, "value": value}
	return projection
}

func renderConfigGet(projection clioutput.Projection) string {
	return fmt.Sprintf("Configuration value\n\nKey: %v\nValue: %v\n", projection.Facts["key"], projection.Facts["value"])
}

func configValueText(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return fmt.Sprint(v)
	default:
		data, err := yaml.Marshal(value)
		if err != nil {
			return fmt.Sprint(value)
		}
		return strings.TrimSpace(string(data))
	}
}

func isSensitiveConfigKey(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	for _, needle := range []string{"token", "secret", "password", "authorization", "cookie"} {
		if strings.Contains(normalized, needle) {
			return true
		}
	}
	return false
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	projection := buildConfigProjection(cfg)
	if globalProjectionModeRequested() {
		return printProjection(configFormat, projection, nil)
	}

	switch strings.ToLower(strings.TrimSpace(configFormat)) {
	case "json":
		data, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return commandError("Serialization configuration failed", err)
		}
		fmt.Println(string(data))
	default:
		data, err := yaml.Marshal(cfg)
		if err != nil {
			return commandError("Serialization configuration failed", err)
		}
		fmt.Println(string(data))
		fmt.Fprintln(os.Stderr, "Configuration source: default value + environment variable + command line parameter (config.yaml is deprecated)")
	}
	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	_ = args
	return usageError("`taskbridge config set` is deprecated. Please use environment variables or command line parameters instead. Example: TASKBRIDGE_STORAGE_PATH=./data taskbridge list")
}

func runConfigGet(cmd *cobra.Command, args []string) error {
	key := args[0]

	//Simplify the implementation and get the value based on key
	var value interface{}
	parts := strings.Split(key, ".")

	switch parts[0] {
	case "storage":
		if len(parts) > 1 {
			switch parts[1] {
			case "type":
				value = cfg.Storage.Type
			case "path":
				value = cfg.Storage.Path
			case "file":
				if len(parts) > 2 && parts[2] == "format" {
					value = cfg.Storage.File.Format
				} else {
					value = cfg.Storage.File
				}
			case "nosql":
				if len(parts) > 2 && parts[2] == "url" {
					value = cfg.Storage.NoSQL.URL
				} else {
					value = cfg.Storage.NoSQL
				}
			default:
				value = cfg.Storage
			}
		} else {
			value = cfg.Storage
		}
	case "sync":
		if len(parts) > 1 {
			switch parts[1] {
			case "mode":
				value = cfg.Sync.Mode
			case "interval":
				value = cfg.Sync.Interval.String()
			case "conflict_resolution":
				value = cfg.Sync.ConflictResolution
			default:
				value = cfg.Sync
			}
		} else {
			value = cfg.Sync
		}
	case "intelligence":
		if len(parts) > 1 {
			switch parts[1] {
			case "timezone":
				value = cfg.Intelligence.Timezone
			case "enabled":
				value = cfg.Intelligence.Enabled
			default:
				value = cfg.Intelligence
			}
		} else {
			value = cfg.Intelligence
		}
	case "providers":
		if len(parts) > 1 {
			switch parts[1] {
			case "google":
				if len(parts) > 2 && parts[2] == "enabled" {
					value = cfg.Providers.Google.Enabled
				} else {
					value = cfg.Providers.Google
				}
			case "microsoft":
				value = cfg.Providers.Microsoft
			case "feishu":
				value = cfg.Providers.Feishu
			case "ticktick":
				value = cfg.Providers.TickTick
			case "dida":
				value = cfg.Providers.Dida
			case "todoist":
				value = cfg.Providers.Todoist
			default:
				value = cfg.Providers
			}
		} else {
			value = cfg.Providers
		}
	case "app":
		if len(parts) > 1 {
			switch parts[1] {
			case "name":
				value = cfg.App.Name
			case "version":
				value = cfg.App.Version
			case "log_level":
				value = cfg.App.LogLevel
			default:
				value = cfg.App
			}
		} else {
			value = cfg.App
		}
	default:
		return usageError("Unknown configuration item:" + key)
	}

	projection := buildConfigGetProjection(key, value)
	if outputJSON || outputAgent || outputEvents || outputExplain {
		return printProjection("text", projection, nil)
	}
	fmt.Fprint(os.Stdout, renderConfigGet(projection))
	return nil
}

func runConfigInit(cmd *cobra.Command, args []string) error {
	_ = cmd
	_ = args
	return usageError("`taskbridge config init` is deprecated. Please use environment variables or command line parameters instead. Example: TASKBRIDGE_PROVIDERS=microsoft,todoist taskbridge sync status")
}

func buildConfigValidateProjection(issues []pkgconfig.ValidationIssue) clioutput.Projection {
	p := clioutput.New("config.validate")
	errorCount, warningCount := 0, 0
	for _, issue := range issues {
		switch issue.Level {
		case pkgconfig.ValidationLevelError:
			errorCount++
		case pkgconfig.ValidationLevelWarning:
			warningCount++
		}
	}
	p.Summary = "Configuration validation completed."
	p.Facts["errors"] = errorCount
	p.Facts["warnings"] = warningCount
	p.Facts["issues"] = len(issues)
	p.Data = map[string]any{"issues": issues, "errors": errorCount, "warnings": warningCount}
	if errorCount > 0 {
		p.Status = clioutput.StatusFailed
		p.Error = &clioutput.OutputError{Code: "config_invalid", Message: "Configuration validation failed", Suggestion: "Fix the reported configuration errors and run taskbridge config validate again."}
	} else if warningCount > 0 {
		p.Status = clioutput.StatusPartial
	}
	return p
}

func runConfigValidate(cmd *cobra.Command, args []string) error {
	_ = cmd
	_ = args

	issues := cfg.Validate()
	projection := buildConfigValidateProjection(issues)
	if globalProjectionModeRequested() {
		if err := printProjection("text", projection, nil); err != nil {
			return err
		}
		if projection.Status == clioutput.StatusFailed {
			return &CLIError{Message: "Configuration verification failed", ExitCode: 1}
		}
		return nil
	}

	exitCode := writeValidationReport(os.Stdout, issues)
	if exitCode != 0 {
		return &CLIError{Message: "Configuration verification failed", ExitCode: exitCode}
	}
	return nil
}
