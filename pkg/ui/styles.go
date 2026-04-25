// Package ui provides styled CLI output components using lipgloss
package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Color palette
var (
	// Primary colors
	PrimaryColor   = lipgloss.Color("#7C3AED") // Purple
	SecondaryColor = lipgloss.Color("#3B82F6") // Blue

	// Status colors
	SuccessColor = lipgloss.Color("#10B981") // Green
	ErrorColor   = lipgloss.Color("#EF4444") // Red
	WarningColor = lipgloss.Color("#F59E0B") // Yellow/Amber
	InfoColor    = lipgloss.Color("#6B7280") // Gray

	// UI colors
	BorderColor   = lipgloss.Color("#374151") // Dark gray
	HeaderBgColor = lipgloss.Color("#1F2937") // Darker gray
	AltRowColor   = lipgloss.Color("#1A1A2E") // Alternate row background
)

// Base styles
var (
	// TitleStyle for main titles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(PrimaryColor).
			MarginBottom(1).
			Padding(0, 1)

	// HeaderStyle for table headers
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(HeaderBgColor).
			Padding(0, 1)

	// RowStyle for table rows
	RowStyle = lipgloss.NewStyle().
			Padding(0, 1)

	// AltRowStyle for alternating table rows
	AltRowStyle = lipgloss.NewStyle().
			Background(AltRowColor).
			Padding(0, 1)

	// SelectedStyle for selected items
	SelectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#374151")).
			Foreground(lipgloss.Color("#FFFFFF"))

	// CardStyle for card-like containers
	CardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(BorderColor).
			Padding(1, 2).
			Margin(1, 0)
)

// Status styles
var (
	// SuccessStyle for success messages
	SuccessStyle = lipgloss.NewStyle().
			Foreground(SuccessColor).
			Bold(true)

	// ErrorStyle for error messages
	ErrorStyle = lipgloss.NewStyle().
			Foreground(ErrorColor).
			Bold(true)

	// WarningStyle for warning messages
	WarningStyle = lipgloss.NewStyle().
			Foreground(WarningColor)

	// InfoStyle for informational messages
	InfoStyle = lipgloss.NewStyle().
			Foreground(InfoColor)

	// EnabledStyle for enabled status
	EnabledStyle = SuccessStyle

	// DisabledStyle for disabled status
	DisabledStyle = lipgloss.NewStyle().
			Foreground(InfoColor)
)

// Label and value styles
var (
	// LabelStyle for field labels
	LabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF")).
			Bold(true)

	// ValueStyle for field values
	ValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E5E7EB"))
)

// Status text helpers
var (
	StatusEnabled       = SuccessStyle.Render("✅ 已启用")
	StatusDisabled      = DisabledStyle.Render("❌ 未启用")
	StatusConnected     = SuccessStyle.Render("✅ 已连接")
	StatusNotConfigured = DisabledStyle.Render("❌ 未配置")
	StatusExpired       = WarningStyle.Render("⚠️ 已过期")
)

// Success prints a success message
func Success(msg string) string {
	return SuccessStyle.Render("✓ " + msg)
}

// Error prints an error message
func Error(msg string) string {
	return ErrorStyle.Render("✗ " + msg)
}

// Warning prints a warning message
func Warning(msg string) string {
	return WarningStyle.Render("⚠ " + msg)
}

// Info prints an informational message
func Info(msg string) string {
	return InfoStyle.Render("ℹ " + msg)
}

// LabelValue formats a label-value pair
func LabelValue(label, value string) string {
	return LabelStyle.Render(label+": ") + ValueStyle.Render(value)
}

// Bold returns bold text
func Bold(text string) string {
	return lipgloss.NewStyle().Bold(true).Render(text)
}

// Dim returns dimmed text
func Dim(text string) string {
	return lipgloss.NewStyle().Faint(true).Render(text)
}

// Highlight returns highlighted text
func Highlight(text string) string {
	return lipgloss.NewStyle().
		Foreground(PrimaryColor).
		Bold(true).
		Render(text)
}

// Theme colors (Claude Code inspired)
var (
	ThemePurple  = lipgloss.Color("#7C3AED")
	ThemeGreen   = lipgloss.Color("#22C55E")
	ThemeOrange  = lipgloss.Color("#F59E0B")
	ThemeRed     = lipgloss.Color("#EF4444")
	ThemeGray    = lipgloss.Color("#94A3B8")
	ThemeBgDark  = lipgloss.Color("#1E1E2E")
	ThemeBgCard  = lipgloss.Color("#2D2D3F")
	ThemeText    = lipgloss.Color("#E2E8F0")
	ThemeDimText = lipgloss.Color("#64748B")
)

// TaskCard returns a card style with a colored left border based on task status.
func TaskCard(status string) lipgloss.Style {
	borderColor := ThemeGray
	switch status {
	case "completed":
		borderColor = ThemeGreen
	case "in_progress":
		borderColor = ThemeOrange
	case "todo":
		borderColor = ThemeGray
	case "cancelled":
		borderColor = ThemeRed
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderLeft(true).
		BorderForeground(lipgloss.Color(string(borderColor))).
		Padding(0, 1).
		MarginBottom(1)
}

// PriorityStyle returns a bold style colored by priority level.
func PriorityStyle(priority int) lipgloss.Style {
	colors := map[int]lipgloss.Color{
		0: ThemeRed,    // P0 Urgent
		1: ThemeOrange, // P1 High
		2: ThemeOrange, // P2 Medium
		3: ThemeGray,   // P3 Low
	}
	c, ok := colors[priority]
	if !ok {
		c = ThemeGray
	}
	return lipgloss.NewStyle().Foreground(c).Bold(true)
}

// QuadrantStyle returns a background-colored style for Eisenhower matrix quadrants.
func QuadrantStyle(quadrant int) lipgloss.Style {
	colors := map[int]lipgloss.Color{
		1: ThemeRed,    // Q1 Urgent+Important
		2: ThemeGreen,  // Q2 Important
		3: ThemeOrange, // Q3 Urgent
		4: ThemeGray,   // Q4 Neither
	}
	c, ok := colors[quadrant]
	if !ok {
		c = ThemeGray
	}
	return lipgloss.NewStyle().
		Background(c).
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(0, 1)
}

// ProgressBar renders a gradient progress bar with the given width and progress (0-100).
func ProgressBar(width int, progress int) string {
	filled := width * progress / 100
	if filled > width {
		filled = width
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	return lipgloss.NewStyle().
		Foreground(ThemePurple).
		Render(bar)
}

// StatusBarStyle returns a style for the TUI bottom status bar.
func StatusBarStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(ThemePurple).
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(0, 1)
}

// ThemeTitleStyle returns a style for TUI headers.
func ThemeTitleStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(ThemePurple).
		Bold(true).
		MarginBottom(1)
}

// DimStyle returns a style for secondary/dimmed text.
func DimStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(ThemeDimText)
}
