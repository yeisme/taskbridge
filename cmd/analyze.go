package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yeisme/taskbridge/internal/analyze"
	"github.com/yeisme/taskbridge/internal/clioutput"

	"github.com/yeisme/taskbridge/internal/model"
	"github.com/yeisme/taskbridge/internal/storage"
)

var (
	analyzeFormat string
)

// analyzeCmd analysis command
var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Task analysis",
	Long: `Analyze task data with quadrant, priority, time, trend, and comprehensive reports.

Subcommands:
  quadrant   Four-quadrant analysis (Eisenhower matrix)
  priority   Priority distribution analysis
  time       Time distribution analysis
  trend      Completion trend analysis
  report     Comprehensive analysis report

Examples:
  taskbridge analyze quadrant
  taskbridge analyze priority --format json
  taskbridge analyze report`,
}

// analyzeQuadrantCmd four quadrant analysis
var analyzeQuadrantCmd = &cobra.Command{
	Use:   "quadrant",
	Short: "Four-quadrant analysis (Eisenhower matrix)",
	Long: `Analyze active task distribution with the Eisenhower matrix:

  Q1 Urgent and important       - Do immediately
  Q2 Important not urgent       - Schedule and protect time
  Q3 Urgent not important       - Delegate or reduce
  Q4 Not urgent and not important - Delete or defer`,
	RunE: runAnalyzeQuadrant,
}

// analyzePriorityCmd Prioritization analysis
var analyzePriorityCmd = &cobra.Command{
	Use:   "priority",
	Short: "Priority distribution analysis",
	Long:  `Analyze active tasks by priority and average priority score`,
	RunE:  runAnalyzePriority,
}

// analyzeTimeCmd time analysis
var analyzeTimeCmd = &cobra.Command{
	Use:   "time",
	Short: "Time distribution analysis",
	Long:  `Analyze task distribution by deadline and creation time`,
	RunE:  runAnalyzeTime,
}

// analyzeTrendCmd trend analysis
var analyzeTrendCmd = &cobra.Command{
	Use:   "trend",
	Short: "Completion trend analysis",
	Long:  `Analyze task completion trends`,
	RunE:  runAnalyzeTrend,
}

// analyzeReportCmd comprehensive report
var analyzeReportCmd = &cobra.Command{
	Use:   "report",
	Short: "Comprehensive analysis report",
	Long:  `Generate a comprehensive analysis report across task status, quadrant, priority, and time`,
	RunE:  runAnalyzeReport,
}

func init() {
	rootCmd.AddCommand(analyzeCmd)
	analyzeCmd.AddCommand(analyzeQuadrantCmd)
	analyzeCmd.AddCommand(analyzePriorityCmd)
	analyzeCmd.AddCommand(analyzeTimeCmd)
	analyzeCmd.AddCommand(analyzeTrendCmd)
	analyzeCmd.AddCommand(analyzeReportCmd)

	//General options
	for _, cmd := range []*cobra.Command{analyzeQuadrantCmd, analyzePriorityCmd, analyzeTimeCmd, analyzeTrendCmd, analyzeReportCmd} {
		cmd.Flags().StringVarP(&analyzeFormat, "format", "f", "text", "Output format (text, json)")
	}
}

// Type aliases for the analyze package DTOs.
// The cmd layer uses these aliases for renderer compatibility while
// delegating computation to internal/analyze.
type (
	QuadrantAnalysis = analyze.QuadrantAnalysis
	QuadrantData     = analyze.QuadrantData
	PriorityAnalysis = analyze.PriorityAnalysis
	PriorityData     = analyze.PriorityData
	TimeAnalysis     = analyze.TimeAnalysis
	TimeData         = analyze.TimeData
	SummaryData      = analyze.SummaryData
)

// TrendAnalysis trend analysis results (cmd-specific JSON contract)
type TrendAnalysis struct {
	DailyCompletions []DayData `json:"daily_completions"`
	WeeklyAverage    float64   `json:"weekly_average"`
	TotalCompleted   int       `json:"total_completed"`
}

// DayData daily data
type DayData struct {
	Date      string `json:"date"`
	Completed int    `json:"completed"`
}

// AnalyzeReport comprehensive report (cmd-specific JSON contract with flat fields)
type AnalyzeReport struct {
	Total      int    `json:"total"`
	Active     int    `json:"active"`
	Completed  int    `json:"completed"`
	Q1         int    `json:"q1"`
	Q2         int    `json:"q2"`
	Q3         int    `json:"q3"`
	Q4         int    `json:"q4"`
	Urgent     int    `json:"urgent"`
	High       int    `json:"high"`
	Medium     int    `json:"medium"`
	Low        int    `json:"low"`
	Overdue    int    `json:"overdue"`
	TodayTasks int    `json:"today_tasks"`
	ThisWeek   int    `json:"this_week"`
	Generated  string `json:"generated"`
}

// getTasksForAnalysis Gets the tasks used for analysis
func getTasksForAnalysis() ([]model.Task, error) {
	ctx := context.Background()
	store, cleanup, err := getStore()
	if err != nil {
		return nil, fmt.Errorf("failed to create storage: %w", err)
	}
	defer cleanup()

	//Get all tasks
	tasks, err := store.ListTasks(ctx, storage.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	return tasks, nil
}

func printAnalyzeJSON(v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return commandError("Serialized output failed", err)
	}
	fmt.Println(string(data))
	return nil
}

func buildAnalyzeProjection(command, summary string, data any) clioutput.Projection {
	p := clioutput.New(command)
	p.Summary = summary
	p.Data = data
	switch v := data.(type) {
	case PriorityAnalysis:
		p.Facts["total"] = v.Summary.Total
		p.Facts["active"] = v.Summary.Active
		p.Facts["completed"] = v.Summary.Completed
		p.Facts["avg_priority_score"] = v.Summary.AvgPriorityScore
	case QuadrantAnalysis:
		p.Facts["total"] = v.Summary.Total
		p.Facts["active"] = v.Summary.Active
		p.Facts["completed"] = v.Summary.Completed
	case AnalyzeReport:
		p.Facts["total"] = v.Total
		p.Facts["active"] = v.Active
		p.Facts["completed"] = v.Completed
	}
	return p
}

func renderQuadrantAnalysis(analysis QuadrantAnalysis) string {
	active := analysis.Q1.Count + analysis.Q2.Count + analysis.Q3.Count + analysis.Q4.Count
	return clioutput.RenderStatPanel(clioutput.StatPanel{
		Title: "📊 Four-quadrant analysis (Eisenhower matrix)",
		Rows: []clioutput.StatRow{
			{Icon: "🔥", Label: "Q1 Urgent and important", Value: fmt.Sprintf("%d", analysis.Q1.Count), Percent: fmt.Sprintf("%.1f%%", analysis.Q1.Percentage), Hint: analysis.Q1.Description},
			{Icon: "📋", Label: "Q2 Important not urgent", Value: fmt.Sprintf("%d", analysis.Q2.Count), Percent: fmt.Sprintf("%.1f%%", analysis.Q2.Percentage), Hint: analysis.Q2.Description},
			{Icon: "⚡", Label: "Q3 Urgent not important", Value: fmt.Sprintf("%d", analysis.Q3.Count), Percent: fmt.Sprintf("%.1f%%", analysis.Q3.Percentage), Hint: analysis.Q3.Description},
			{Icon: "🗑️", Label: "Q4 Not urgent or important", Value: fmt.Sprintf("%d", analysis.Q4.Count), Percent: fmt.Sprintf("%.1f%%", analysis.Q4.Percentage), Hint: analysis.Q4.Description},
		},
		Footer: renderAnalyzeSummaryFooter(analysis.Summary, active),
	})
}

func renderAnalyzeSummaryFooter(summary SummaryData, fallbackActive int) string {
	if summary.Total == 0 && summary.Active == 0 && summary.Completed == 0 && fallbackActive > 0 {
		summary.Active = fallbackActive
		summary.Total = fallbackActive
	}
	footer := fmt.Sprintf("Total: %d tasks | Active: %d | Completed: %d", summary.Total, summary.Active, summary.Completed)
	if summary.AvgPriorityScore > 0 {
		footer += fmt.Sprintf(" | Average priority score: %.1f", summary.AvgPriorityScore)
	}
	return footer
}

func runAnalyzeQuadrant(_ *cobra.Command, _ []string) error {
	tasks, err := getTasksForAnalysis()
	if err != nil {
		return err
	}

	analysis := analyze.CalculateQuadrant(tasks)

	projection := buildAnalyzeProjection("analyze.quadrant", "Quadrant analysis completed.", analysis)
	return printProjectionWithLegacyJSON(analyzeFormat, analysis, projection, func() {
		fmt.Print(renderQuadrantAnalysis(analysis))
	})
}

func renderPriorityAnalysis(analysis PriorityAnalysis) string {
	return clioutput.RenderStatPanel(clioutput.StatPanel{
		Title: "📊 Prioritization analysis",
		Rows: []clioutput.StatRow{
			{Icon: "🔴", Label: "Urgent (P0)", Value: fmt.Sprintf("%d", analysis.Urgent.Count), Percent: fmt.Sprintf("%.1f%%", analysis.Urgent.Percentage)},
			{Icon: "🟠", Label: "High (P1)", Value: fmt.Sprintf("%d", analysis.High.Count), Percent: fmt.Sprintf("%.1f%%", analysis.High.Percentage)},
			{Icon: "🟡", Label: "Medium (P2)", Value: fmt.Sprintf("%d", analysis.Medium.Count), Percent: fmt.Sprintf("%.1f%%", analysis.Medium.Percentage)},
			{Icon: "🔵", Label: "Low (P3)", Value: fmt.Sprintf("%d", analysis.Low.Count), Percent: fmt.Sprintf("%.1f%%", analysis.Low.Percentage)},
			{Icon: "⚪", Label: "No priority", Value: fmt.Sprintf("%d", analysis.None.Count), Percent: fmt.Sprintf("%.1f%%", analysis.None.Percentage)},
		},
		Footer: renderAnalyzeSummaryFooter(analysis.Summary, analysis.Summary.Active),
	})
}

func runAnalyzePriority(_ *cobra.Command, _ []string) error {
	tasks, err := getTasksForAnalysis()
	if err != nil {
		return err
	}

	analysis := analyze.CalculatePriority(tasks)

	projection := buildAnalyzeProjection("analyze.priority", "Priority analysis completed.", analysis)
	return printProjectionWithLegacyJSON(analyzeFormat, analysis, projection, func() {
		fmt.Print(renderPriorityAnalysis(analysis))
	})
}

func renderTimeAnalysis(analysis TimeAnalysis) string {
	return clioutput.RenderStatPanel(clioutput.StatPanel{
		Title: "📊 Time distribution analysis",
		Rows: []clioutput.StatRow{
			{Icon: "⚠️", Label: "Overdue", Value: fmt.Sprintf("%d", analysis.Overdue.Count)},
			{Icon: "🔥", Label: "Today", Value: fmt.Sprintf("%d", analysis.Today.Count)},
			{Icon: "📅", Label: "Tomorrow", Value: fmt.Sprintf("%d", analysis.Tomorrow.Count)},
			{Icon: "📆", Label: "This week", Value: fmt.Sprintf("%d", analysis.ThisWeek.Count)},
			{Icon: "📋", Label: "Next week", Value: fmt.Sprintf("%d", analysis.NextWeek.Count)},
			{Icon: "🗓️", Label: "This month", Value: fmt.Sprintf("%d", analysis.ThisMonth.Count)},
			{Icon: "📁", Label: "Future", Value: fmt.Sprintf("%d", analysis.Future.Count)},
			{Icon: "❓", Label: "No due date", Value: fmt.Sprintf("%d", analysis.NoDueDate.Count)},
		},
	})
}

func runAnalyzeTime(_ *cobra.Command, _ []string) error {
	tasks, err := getTasksForAnalysis()
	if err != nil {
		return err
	}

	analysis := analyze.CalculateTime(tasks)

	projection := buildAnalyzeProjection("analyze.time", "Time analysis completed.", analysis)
	return printProjectionWithLegacyJSON(analyzeFormat, analysis, projection, func() {
		fmt.Print(renderTimeAnalysis(analysis))
	})
}

func renderTrendAnalysis(analysis TrendAnalysis) string {
	rows := make([]clioutput.StatRow, 0, len(analysis.DailyCompletions))
	for _, day := range analysis.DailyCompletions {
		bar := strings.Repeat("█", day.Completed)
		if day.Completed == 0 {
			bar = "░"
		}
		rows = append(rows, clioutput.StatRow{Label: day.Date, Value: fmt.Sprintf("%d", day.Completed), Hint: bar})
	}
	return clioutput.RenderStatPanel(clioutput.StatPanel{
		Title:  "📊 Trend analysis (last 7 days)",
		Rows:   rows,
		Footer: fmt.Sprintf("This week completed: %d tasks | Daily average: %.1f", analysis.TotalCompleted, analysis.WeeklyAverage),
	})
}

func runAnalyzeTrend(cmd *cobra.Command, args []string) error {
	tasks, err := getTasksForAnalysis()
	if err != nil {
		return err
	}

	//Statistics of completion status in the past 7 days
	now := time.Now()
	dailyCompletions := make(map[string]int)

	for i := 0; i < 7; i++ {
		date := now.AddDate(0, 0, -i)
		dateStr := date.Format("2006-01-02")
		dailyCompletions[dateStr] = 0
	}

	var totalCompleted int
	for _, t := range tasks {
		if t.Status == model.StatusCompleted && t.CompletedAt != nil {
			dateStr := t.CompletedAt.Format("2006-01-02")
			if _, ok := dailyCompletions[dateStr]; ok {
				dailyCompletions[dateStr]++
				totalCompleted++
			}
		}
	}

	//Convert to ordered list
	var trendData []DayData
	var dates []string
	for d := range dailyCompletions {
		dates = append(dates, d)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(dates)))

	for _, d := range dates {
		trendData = append(trendData, DayData{
			Date:      d,
			Completed: dailyCompletions[d],
		})
	}

	weeklyAvg := float64(totalCompleted) / 7.0

	analysis := TrendAnalysis{
		DailyCompletions: trendData,
		WeeklyAverage:    weeklyAvg,
		TotalCompleted:   totalCompleted,
	}

	projection := buildAnalyzeProjection("analyze.trend", "Trend analysis completed.", analysis)
	return printProjectionWithLegacyJSON(analyzeFormat, analysis, projection, func() {
		fmt.Print(renderTrendAnalysis(analysis))
	})
}

func renderAnalyzeReport(report AnalyzeReport) string {
	return clioutput.RenderStatPanel(clioutput.StatPanel{
		Title: "📊 TaskBridge comprehensive analysis report",
		Rows: []clioutput.StatRow{
			{Icon: "📋", Label: "Task summary", Value: fmt.Sprintf("%d", report.Total), Hint: fmt.Sprintf("Active %d / Completed %d", report.Active, report.Completed)},
			{Icon: "🎯", Label: "Quadrants", Value: fmt.Sprintf("Q1 %d", report.Q1), Hint: fmt.Sprintf("Q2 %d / Q3 %d / Q4 %d", report.Q2, report.Q3, report.Q4)},
			{Icon: "🔴", Label: "Priority", Value: fmt.Sprintf("P0 %d", report.Urgent), Hint: fmt.Sprintf("P1 %d / P2 %d / P3 %d", report.High, report.Medium, report.Low)},
			{Icon: "⏰", Label: "Time", Value: fmt.Sprintf("Overdue %d", report.Overdue), Hint: fmt.Sprintf("Today %d / This week %d", report.TodayTasks, report.ThisWeek)},
		},
		Footer: "Generated: " + report.Generated,
	})
}

func runAnalyzeReport(cmd *cobra.Command, args []string) error {
	tasks, err := getTasksForAnalysis()
	if err != nil {
		return err
	}

	report := AnalyzeReport{Total: len(tasks), Generated: time.Now().Format("2006-01-02 15:04:05")}
	for _, t := range tasks {
		if t.Status == model.StatusCompleted {
			report.Completed++
			continue
		}
		report.Active++
		switch t.Quadrant {
		case model.QuadrantUrgentImportant:
			report.Q1++
		case model.QuadrantNotUrgentImportant:
			report.Q2++
		case model.QuadrantUrgentNotImportant:
			report.Q3++
		case model.QuadrantNotUrgentNotImportant:
			report.Q4++
		}
		switch t.Priority {
		case model.PriorityUrgent:
			report.Urgent++
		case model.PriorityHigh:
			report.High++
		case model.PriorityMedium:
			report.Medium++
		case model.PriorityLow:
			report.Low++
		}
		if t.DueDate == nil {
			continue
		}
		now := time.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		if t.DueDate.Before(today) {
			report.Overdue++
		} else if t.DueDate.Format("2006-01-02") == today.Format("2006-01-02") {
			report.TodayTasks++
		} else if t.DueDate.Before(today.AddDate(0, 0, 7)) {
			report.ThisWeek++
		}
	}

	projection := buildAnalyzeProjection("analyze.report", "Comprehensive analysis report generated.", report)
	return printProjectionWithLegacyJSON(analyzeFormat, report, projection, func() {
		fmt.Print(renderAnalyzeReport(report))
	})
}
