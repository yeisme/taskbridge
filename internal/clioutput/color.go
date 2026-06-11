package clioutput

type ColorMode string

const (
	ColorAuto   ColorMode = "auto"
	ColorAlways ColorMode = "always"
	ColorNever  ColorMode = "never"
)

func ShouldColor(mode ColorMode, stdoutIsTerminal bool, noColorEnv string) bool {
	switch mode {
	case ColorAlways:
		return true
	case ColorNever:
		return false
	default:
		return stdoutIsTerminal && noColorEnv == ""
	}
}
