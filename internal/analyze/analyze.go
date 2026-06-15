// Package analyze contains task analysis computation logic for quadrant,
// priority, time, trend, and comprehensive report analysis.
// The command layer (cmd/analyze.go) calls these functions and handles
// rendering and output projection; this package owns the data model and
// pure computation.
package analyze

import (
	"sort"
	"time"

	"github.com/yeisme/taskbridge/internal/model"
)

// QuadrantAnalysis four-quadrant analysis results
type QuadrantAnalysis struct {
	Q1      QuadrantData `json:"q1"`
	Q2      QuadrantData `json:"q2"`
	Q3      QuadrantData `json:"q3"`
	Q4      QuadrantData `json:"q4"`
	Summary SummaryData  `json:"summary,omitempty"`
}

// QuadrantData quadrant data
type QuadrantData struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Count       int      `json:"count"`
	Percentage  float64  `json:"percentage"`
	Tasks       []string `json:"tasks,omitempty"`
}

// PriorityAnalysis prioritization analysis results
type PriorityAnalysis struct {
	Urgent  PriorityData `json:"urgent"`
	High    PriorityData `json:"high"`
	Medium  PriorityData `json:"medium"`
	Low     PriorityData `json:"low"`
	None    PriorityData `json:"none"`
	Summary SummaryData  `json:"summary"`
}

// PriorityData priority data
type PriorityData struct {
	Count      int      `json:"count"`
	Percentage float64  `json:"percentage"`
	Tasks      []string `json:"tasks,omitempty"`
}

// TimeAnalysis time analysis results
type TimeAnalysis struct {
	Overdue   TimeData `json:"overdue"`
	Today     TimeData `json:"today"`
	Tomorrow  TimeData `json:"tomorrow"`
	ThisWeek  TimeData `json:"this_week"`
	NextWeek  TimeData `json:"next_week"`
	ThisMonth TimeData `json:"this_month"`
	Future    TimeData `json:"future"`
	NoDueDate TimeData `json:"no_due_date"`
}

// TimeData time data
type TimeData struct {
	Description string   `json:"description"`
	Count       int      `json:"count"`
	Tasks       []string `json:"tasks,omitempty"`
}

// TrendAnalysis completion trend analysis results
type TrendAnalysis struct {
	Daily   []TrendPoint `json:"daily"`
	Weekly  []TrendPoint `json:"weekly"`
	Monthly []TrendPoint `json:"monthly"`
}

// TrendPoint a single data point in trend analysis
type TrendPoint struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

// AnalyzeReport comprehensive analysis report
type AnalyzeReport struct {
	Total     int              `json:"total"`
	Active    int              `json:"active"`
	Completed int              `json:"completed"`
	Quadrant  QuadrantAnalysis `json:"quadrant"`
	Priority  PriorityAnalysis `json:"priority"`
	Time      TimeAnalysis     `json:"time"`
}

// SummaryData summary data
type SummaryData struct {
	Total            int     `json:"total"`
	Active           int     `json:"active"`
	Completed        int     `json:"completed"`
	AvgPriorityScore float64 `json:"avg_priority_score"`
}

// CalculateQuadrant computes quadrant distribution from tasks.
// Quadrant counting: skips completed tasks, groups active tasks by quadrant.
func CalculateQuadrant(tasks []model.Task) QuadrantAnalysis {
	analysis := QuadrantAnalysis{
		Q1: QuadrantData{Name: "Q1 Urgent and important", Description: "Do immediately", Tasks: []string{}},
		Q2: QuadrantData{Name: "Q2 Important not urgent", Description: "Schedule and protect time", Tasks: []string{}},
		Q3: QuadrantData{Name: "Q3 Urgent not important", Description: "Delegate or reduce", Tasks: []string{}},
		Q4: QuadrantData{Name: "Q4 Not urgent or important", Description: "Delete or defer", Tasks: []string{}},
	}

	total := len(tasks)
	for _, t := range tasks {
		if t.Status == model.StatusCompleted {
			continue
		}
		switch t.Quadrant {
		case model.QuadrantUrgentImportant:
			analysis.Q1.Count++
			analysis.Q1.Tasks = append(analysis.Q1.Tasks, t.Title)
		case model.QuadrantNotUrgentImportant:
			analysis.Q2.Count++
			analysis.Q2.Tasks = append(analysis.Q2.Tasks, t.Title)
		case model.QuadrantUrgentNotImportant:
			analysis.Q3.Count++
			analysis.Q3.Tasks = append(analysis.Q3.Tasks, t.Title)
		case model.QuadrantNotUrgentNotImportant:
			analysis.Q4.Count++
			analysis.Q4.Tasks = append(analysis.Q4.Tasks, t.Title)
		}
	}

	activeTotal := analysis.Q1.Count + analysis.Q2.Count + analysis.Q3.Count + analysis.Q4.Count
	if activeTotal > 0 {
		analysis.Q1.Percentage = float64(analysis.Q1.Count) / float64(activeTotal) * 100
		analysis.Q2.Percentage = float64(analysis.Q2.Count) / float64(activeTotal) * 100
		analysis.Q3.Percentage = float64(analysis.Q3.Count) / float64(activeTotal) * 100
		analysis.Q4.Percentage = float64(analysis.Q4.Count) / float64(activeTotal) * 100
	}

	analysis.Summary = SummaryData{Total: total, Active: activeTotal, Completed: total - activeTotal}
	return analysis
}

// CalculatePriority computes priority distribution from tasks.
// Priority counting: skips completed, groups by priority level, computes average score.
func CalculatePriority(tasks []model.Task) PriorityAnalysis {
	analysis := PriorityAnalysis{
		Urgent: PriorityData{Tasks: []string{}},
		High:   PriorityData{Tasks: []string{}},
		Medium: PriorityData{Tasks: []string{}},
		Low:    PriorityData{Tasks: []string{}},
		None:   PriorityData{Tasks: []string{}},
	}

	var totalScore int
	var scoreCount int

	for _, t := range tasks {
		if t.Status == model.StatusCompleted {
			analysis.Summary.Completed++
			continue
		}
		analysis.Summary.Active++

		switch t.Priority {
		case model.PriorityUrgent:
			analysis.Urgent.Count++
			analysis.Urgent.Tasks = append(analysis.Urgent.Tasks, t.Title)
		case model.PriorityHigh:
			analysis.High.Count++
			analysis.High.Tasks = append(analysis.High.Tasks, t.Title)
		case model.PriorityMedium:
			analysis.Medium.Count++
			analysis.Medium.Tasks = append(analysis.Medium.Tasks, t.Title)
		case model.PriorityLow:
			analysis.Low.Count++
			analysis.Low.Tasks = append(analysis.Low.Tasks, t.Title)
		default:
			analysis.None.Count++
			analysis.None.Tasks = append(analysis.None.Tasks, t.Title)
		}

		if t.PriorityScore > 0 {
			totalScore += t.PriorityScore
			scoreCount++
		}
	}

	analysis.Summary.Total = len(tasks)
	if scoreCount > 0 {
		analysis.Summary.AvgPriorityScore = float64(totalScore) / float64(scoreCount)
	}

	if analysis.Summary.Active > 0 {
		analysis.Urgent.Percentage = float64(analysis.Urgent.Count) / float64(analysis.Summary.Active) * 100
		analysis.High.Percentage = float64(analysis.High.Count) / float64(analysis.Summary.Active) * 100
		analysis.Medium.Percentage = float64(analysis.Medium.Count) / float64(analysis.Summary.Active) * 100
		analysis.Low.Percentage = float64(analysis.Low.Count) / float64(analysis.Summary.Active) * 100
		analysis.None.Percentage = float64(analysis.None.Count) / float64(analysis.Summary.Active) * 100
	}

	return analysis
}

// CalculateTime computes time distribution from tasks.
// 时间边界逻辑：使用本地日期截断（00:00:00），按 overdue/today/tomorrow/this week/next week/this month/future 分类。
func CalculateTime(tasks []model.Task) TimeAnalysis {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tomorrow := today.AddDate(0, 0, 1)
	thisWeekEnd := today.AddDate(0, 0, 7-int(today.Weekday()))
	nextWeekEnd := thisWeekEnd.AddDate(0, 0, 7)
	thisMonthEnd := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location()).Add(-time.Second)

	analysis := TimeAnalysis{
		Overdue:   TimeData{Description: "overdue", Tasks: []string{}},
		Today:     TimeData{Description: "today", Tasks: []string{}},
		Tomorrow:  TimeData{Description: "tomorrow", Tasks: []string{}},
		ThisWeek:  TimeData{Description: "this week", Tasks: []string{}},
		NextWeek:  TimeData{Description: "next week", Tasks: []string{}},
		ThisMonth: TimeData{Description: "this month", Tasks: []string{}},
		Future:    TimeData{Description: "future", Tasks: []string{}},
		NoDueDate: TimeData{Description: "no due date", Tasks: []string{}},
	}

	for _, t := range tasks {
		if t.Status == model.StatusCompleted {
			continue
		}

		if t.DueDate == nil {
			analysis.NoDueDate.Count++
			analysis.NoDueDate.Tasks = append(analysis.NoDueDate.Tasks, t.Title)
			continue
		}

		due := time.Date(t.DueDate.Year(), t.DueDate.Month(), t.DueDate.Day(), 0, 0, 0, 0, t.DueDate.Location())

		switch {
		case due.Before(today):
			analysis.Overdue.Count++
			analysis.Overdue.Tasks = append(analysis.Overdue.Tasks, t.Title)
		case due.Equal(today):
			analysis.Today.Count++
			analysis.Today.Tasks = append(analysis.Today.Tasks, t.Title)
		case due.Equal(tomorrow):
			analysis.Tomorrow.Count++
			analysis.Tomorrow.Tasks = append(analysis.Tomorrow.Tasks, t.Title)
		case due.Before(thisWeekEnd) || due.Equal(thisWeekEnd):
			analysis.ThisWeek.Count++
			analysis.ThisWeek.Tasks = append(analysis.ThisWeek.Tasks, t.Title)
		case due.Before(nextWeekEnd) || due.Equal(nextWeekEnd):
			analysis.NextWeek.Count++
			analysis.NextWeek.Tasks = append(analysis.NextWeek.Tasks, t.Title)
		case due.Before(thisMonthEnd) || due.Equal(thisMonthEnd):
			analysis.ThisMonth.Count++
			analysis.ThisMonth.Tasks = append(analysis.ThisMonth.Tasks, t.Title)
		default:
			analysis.Future.Count++
			analysis.Future.Tasks = append(analysis.Future.Tasks, t.Title)
		}
	}

	return analysis
}

// CalculateTrend computes completion trend from tasks.
// Groups completed tasks by day, week, and month based on CompletedAt timestamp.
func CalculateTrend(tasks []model.Task) TrendAnalysis {
	dailyMap := make(map[string]int)
	weeklyMap := make(map[string]int)
	monthlyMap := make(map[string]int)

	for _, t := range tasks {
		if t.Status != model.StatusCompleted || t.CompletedAt == nil {
			continue
		}
		completed := *t.CompletedAt
		dayKey := completed.Format("2006-01-02")
		_, week := completed.ISOWeek()
		weekKey := completed.Format("2006-W") + formatWeekNum(week)
		monthKey := completed.Format("2006-01")

		dailyMap[dayKey]++
		weeklyMap[weekKey]++
		monthlyMap[monthKey]++
	}

	return TrendAnalysis{
		Daily:   mapToSortedPoints(dailyMap),
		Weekly:  mapToSortedPoints(weeklyMap),
		Monthly: mapToSortedPoints(monthlyMap),
	}
}

// CalculateReport generates a comprehensive analysis report.
func CalculateReport(tasks []model.Task) AnalyzeReport {
	total := len(tasks)
	active := 0
	completed := 0
	for _, t := range tasks {
		if t.Status == model.StatusCompleted {
			completed++
		} else {
			active++
		}
	}

	return AnalyzeReport{
		Total:     total,
		Active:    active,
		Completed: completed,
		Quadrant:  CalculateQuadrant(tasks),
		Priority:  CalculatePriority(tasks),
		Time:      CalculateTime(tasks),
	}
}

func formatWeekNum(week int) string {
	if week < 10 {
		return "0" + string(rune('0'+week))
	}
	return string(rune('0'+week/10)) + string(rune('0'+week%10))
}

func mapToSortedPoints(m map[string]int) []TrendPoint {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	points := make([]TrendPoint, 0, len(keys))
	for _, k := range keys {
		points = append(points, TrendPoint{Label: k, Count: m[k]})
	}
	return points
}
