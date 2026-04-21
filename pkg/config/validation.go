package config

import (
	"fmt"
	"strings"
)

const (
	ValidationLevelError   = "error"
	ValidationLevelWarning = "warning"
)

type ValidationIssue struct {
	Level   string
	Field   string
	Message string
}

func (c *Config) Validate() []ValidationIssue {
	issues := make([]ValidationIssue, 0)
	if c == nil {
		return append(issues, ValidationIssue{
			Level:   ValidationLevelError,
			Field:   "config",
			Message: "配置为空",
		})
	}

	addIssue := func(level, field, message string) {
		issues = append(issues, ValidationIssue{Level: level, Field: field, Message: message})
	}

	if strings.TrimSpace(c.Storage.Type) == "" {
		addIssue(ValidationLevelError, "storage.type", "不能为空")
	}
	if c.Storage.Type == "file" && strings.TrimSpace(c.Storage.Path) == "" {
		addIssue(ValidationLevelError, "storage.path", "不能为空（文件存储模式）")
	}

	switch strings.ToLower(strings.TrimSpace(c.Sync.Mode)) {
	case "once", "interval", "realtime":
	case "":
		addIssue(ValidationLevelError, "sync.mode", "不能为空")
	default:
		addIssue(ValidationLevelError, "sync.mode", fmt.Sprintf("无效值: %s", c.Sync.Mode))
	}

	if strings.TrimSpace(c.Intelligence.Timezone) == "" {
		addIssue(ValidationLevelWarning, "intelligence.timezone", "为空时将回退到系统时区")
	}
	if c.Intelligence.Overdue.WarningThreshold < 0 {
		addIssue(ValidationLevelError, "intelligence.overdue.warning_threshold", "不能小于 0")
	}
	if c.Intelligence.Overdue.OverloadThreshold < c.Intelligence.Overdue.WarningThreshold {
		addIssue(ValidationLevelError, "intelligence.overdue.overload_threshold", "不能小于 warning_threshold")
	}
	if c.Intelligence.Overdue.SevereDays < 1 {
		addIssue(ValidationLevelError, "intelligence.overdue.severe_days", "必须大于等于 1")
	}
	if c.Intelligence.LongTerm.ShortTermMin < 0 {
		addIssue(ValidationLevelError, "intelligence.long_term.short_term_min", "不能小于 0")
	}
	if c.Intelligence.LongTerm.ShortTermMax < c.Intelligence.LongTerm.ShortTermMin {
		addIssue(ValidationLevelError, "intelligence.long_term.short_term_max", "不能小于 short_term_min")
	}
	if c.Intelligence.Decompose.ComplexityThreshold < 0 || c.Intelligence.Decompose.ComplexityThreshold > 100 {
		addIssue(ValidationLevelError, "intelligence.decomposition.complexity_threshold", "必须在 0-100 范围内")
	}
	if c.Intelligence.Achievement.StreakGoalPerDay < 1 {
		addIssue(ValidationLevelError, "intelligence.achievement.streak_goal_per_day", "必须大于等于 1")
	}

	return issues
}
