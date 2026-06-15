// Package cmd provides CLI commands
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/yeisme/taskbridge/internal/provider"
	"github.com/yeisme/taskbridge/pkg/buildinfo"
	"github.com/yeisme/taskbridge/pkg/config"
	"github.com/yeisme/taskbridge/pkg/logger"
)

func init() {
	//Set Windows console output to UTF-8
	if err := os.Setenv("LANG", "en_US.UTF-8"); err != nil {
		//Non-critical errors, only logged
		fmt.Fprintf(os.Stderr, "Warning: Failed to set environment variables: %v\n", err)
	}
}

var (
	cfgFile       string
	verbose       bool
	quiet         bool
	storagePath   string
	storageType   string
	logLevel      string
	providers     string
	outputJSON    bool
	outputAgent   bool
	outputEvents  bool
	outputExplain bool
	colorMode     string
	cfg           *config.Config
)

// rootCmd root command
var rootCmd = &cobra.Command{
	Use:     "taskbridge",
	Short:   "Local task execution control plane for humans and agents",
	Version: buildinfo.Version,
	Long: `TaskBridge is a local task execution control plane shared by humans and agents.
It connects multiple Todo platforms and AI to unify task models, synchronization,
and safe execution workflows.

Supported platforms:
  - Microsoft Todo
  - Google Tasks
  - Feishu Tasks
  - TickTick
  - Dida365 (domestic TickTick)
  - Todoist

Get started:
  taskbridge doctor              # Check environment
  taskbridge demo today          # Preview the daily workbench (no auth needed)
  taskbridge today               # Open the daily workbench
  taskbridge next                # Get the next recommended step
  taskbridge review              # Run a task health review
  taskbridge agent capabilities  # Show Agent-callable commands`,
	SilenceErrors: true,
	SilenceUsage:  true,
}

// Execute execute command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, formatCLIError(err))
		os.Exit(cliExitCode(err))
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "Configuration file path (deprecated, no longer read)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	rootCmd.PersistentFlags().BoolVar(&quiet, "quiet", false, "Streamlined output (pipe/script friendly)")
	rootCmd.PersistentFlags().StringVar(&storagePath, "storage-path", "", "Task storage path (available environment variable TASKBRIDGE_STORAGE_PATH)")
	rootCmd.PersistentFlags().StringVar(&storageType, "storage-type", "", "Storage type: file|mongodb (available environment variable TASKBRIDGE_STORAGE_TYPE)")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "", "Log level: debug|info|warn|error (available environment variable TASKBRIDGE_LOG_LEVEL)")
	rootCmd.PersistentFlags().StringVar(&providers, "providers", "", "Enabled providers, comma separated (available environment variable TASKBRIDGE_PROVIDERS)")
	rootCmd.PersistentFlags().BoolVar(&outputJSON, "json", false, "Output an AI-native JSON envelope")
	rootCmd.PersistentFlags().BoolVar(&outputAgent, "agent", false, "Output stable key=value facts for agents")
	rootCmd.PersistentFlags().BoolVar(&outputEvents, "events", false, "Output newline-delimited JSON events when supported")
	rootCmd.PersistentFlags().BoolVar(&outputExplain, "explain", false, "Output an explainable decision summary when supported")
	rootCmd.PersistentFlags().StringVar(&colorMode, "color", "auto", "Color mode: auto|always|never")
	_ = rootCmd.PersistentFlags().MarkDeprecated("config", "Configuration files are deprecated, use environment variables and command line parameters instead")
}

// initConfig initialization configuration
func initConfig() {
	cfg = config.DefaultConfig()

	//1) Environment variable coverage
	if v := strings.TrimSpace(os.Getenv("TASKBRIDGE_STORAGE_PATH")); v != "" {
		cfg.Storage.Path = v
	}
	if v := strings.TrimSpace(os.Getenv("TASKBRIDGE_STORAGE_TYPE")); v != "" {
		cfg.Storage.Type = v
	}
	if v := strings.TrimSpace(os.Getenv("TASKBRIDGE_STORAGE_FORMAT")); v != "" {
		cfg.Storage.File.Format = v
	}
	if v := strings.TrimSpace(os.Getenv("TASKBRIDGE_LOG_LEVEL")); v != "" {
		cfg.App.LogLevel = v
	}
	applyProvidersFromList(strings.TrimSpace(os.Getenv("TASKBRIDGE_PROVIDERS")))

	//2) Command line parameters override environment variables
	if storagePath != "" {
		cfg.Storage.Path = storagePath
	}
	if storageType != "" {
		cfg.Storage.Type = storageType
	}
	if logLevel != "" {
		cfg.App.LogLevel = logLevel
	}
	if verbose {
		cfg.App.LogLevel = "debug"
	}
	if providers != "" {
		applyProvidersFromList(providers)
	}

	//Initialize the global log level to avoid debugging logs being misjudged as errors
	if err := logger.Init(&logger.Config{
		Level:      cfg.App.LogLevel,
		Format:     "json",
		Output:     "stderr",
		TimeFormat: "",
		Caller:     false,
	}); err != nil {
		//Log initialization failure should not interrupt the main process and fall back to info
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		fmt.Fprintf(os.Stderr, "Warning: Failed to initialize log, fell back to info level: %v\n", err)
	}
}

func applyProvidersFromList(value string) {
	if strings.TrimSpace(value) == "" {
		return
	}

	//After clearing, enable according to the list
	cfg.Providers.Google.Enabled = false
	cfg.Providers.Microsoft.Enabled = false
	cfg.Providers.Feishu.Enabled = false
	cfg.Providers.TickTick.Enabled = false
	cfg.Providers.Dida.Enabled = false
	cfg.Providers.Todoist.Enabled = false

	for _, raw := range strings.Split(value, ",") {
		name := provider.ResolveProviderName(strings.TrimSpace(raw))
		switch name {
		case "google":
			cfg.Providers.Google.Enabled = true
		case "microsoft":
			cfg.Providers.Microsoft.Enabled = true
		case "feishu":
			cfg.Providers.Feishu.Enabled = true
		case "ticktick":
			cfg.Providers.TickTick.Enabled = true
		case "dida":
			cfg.Providers.Dida.Enabled = true
		case "todoist":
			cfg.Providers.Todoist.Enabled = true
		case "":
			// ignore empty entry
		default:
			fmt.Fprintf(os.Stderr, "Warning: Ignoring unknown provider: %s\n", raw)
		}
	}
}

// GetConfig Get configuration
func GetConfig() *config.Config {
	return cfg
}

// IsQuietMode returns true when --quiet is explicitly set.
// Scripts should request machine output with --json/--agent instead of relying on pipe detection.
func IsQuietMode() bool {
	return quiet
}
