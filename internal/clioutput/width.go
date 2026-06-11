package clioutput

import (
	"strings"

	"github.com/mattn/go-runewidth"
)

func DisplayWidth(s string) int {
	return runewidth.StringWidth(s)
}

func PadRight(s string, width int) string {
	padding := width - DisplayWidth(s)
	if padding <= 0 {
		return s
	}
	return s + strings.Repeat(" ", padding)
}

func PadLeft(s string, width int) string {
	padding := width - DisplayWidth(s)
	if padding <= 0 {
		return s
	}
	return strings.Repeat(" ", padding) + s
}

func TruncateDisplay(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if DisplayWidth(s) <= maxWidth {
		return s
	}
	if maxWidth <= 1 {
		return "…"[:maxWidth]
	}
	target := maxWidth - 1
	var b strings.Builder
	used := 0
	for _, r := range s {
		rw := DisplayWidth(string(r))
		if used+rw > target {
			break
		}
		b.WriteRune(r)
		used += rw
	}
	b.WriteString("…")
	return b.String()
}
