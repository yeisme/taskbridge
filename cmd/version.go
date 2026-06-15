package cmd

import (
	"runtime"

	"github.com/spf13/cobra"
	"github.com/yeisme/taskbridge/internal/clioutput"
	"github.com/yeisme/taskbridge/pkg/buildinfo"
)

// versionCmd version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long: `Display TaskBridge version information.

Example:
  taskbridge version
  taskbridge version --json`,
	RunE: runVersion,
}

func buildVersionProjection() clioutput.Projection {
	p := clioutput.New("version.show")
	p.Summary = "TaskBridge - Local task execution control plane"
	p.Facts["version"] = buildinfo.Version
	p.Facts["git_commit"] = buildinfo.GitCommit
	p.Facts["build_date"] = buildinfo.BuildDate
	p.Facts["go_version"] = runtime.Version()
	p.Facts["platform"] = runtime.GOOS + "/" + runtime.GOARCH
	p.Data = map[string]any{
		"version":    buildinfo.Version,
		"git_commit": buildinfo.GitCommit,
		"build_date": buildinfo.BuildDate,
		"go_version": runtime.Version(),
		"platform":   runtime.GOOS + "/" + runtime.GOARCH,
	}
	return p
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func runVersion(_ *cobra.Command, _ []string) error {
	p := buildVersionProjection()
	return printProjection("text", p, nil)
}
