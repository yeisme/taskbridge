package buildinfo

import (
	"runtime/debug"
	"strings"
)

// These variables are injected by ldflags for Taskfile and GoReleaser builds.
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

// EffectiveVersion returns the ldflags-injected version, or the Go module
// version for binaries installed with `go install module@version`.
func EffectiveVersion() string {
	return resolveVersion(Version, mainModuleVersion())
}

func resolveVersion(ldflagsVersion, moduleVersion string) string {
	ldflagsVersion = strings.TrimSpace(ldflagsVersion)
	if ldflagsVersion != "" && ldflagsVersion != "dev" && ldflagsVersion != "(devel)" {
		return strings.TrimPrefix(ldflagsVersion, "v")
	}

	moduleVersion = strings.TrimSpace(moduleVersion)
	if moduleVersion != "" && moduleVersion != "(devel)" {
		return strings.TrimPrefix(moduleVersion, "v")
	}

	if ldflagsVersion != "" {
		return ldflagsVersion
	}
	return "dev"
}

func mainModuleVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	return info.Main.Version
}
