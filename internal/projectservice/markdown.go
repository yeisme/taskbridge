package projectservice

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/yeisme/taskbridge/internal/project"
)

var (
	unorderedListItemPattern = regexp.MustCompile(`^[-*+]\s+(.+)$`)
	orderedListItemPattern   = regexp.MustCompile(`^\d+[.)]\s+(.+)$`)
	orderedTitlePrefix       = regexp.MustCompile(`^\d+[.)]\s+`)
	markdownCheckboxPrefix   = regexp.MustCompile(`^\s*\[\s*[xX ]?\s*\]\s*`)
)

type MarkdownParseStats struct {
	TotalNodes   int `json:"total_nodes"`
	LeafTasks    int `json:"leaf_tasks"`
	IgnoredLines int `json:"ignored_lines"`
}

type markdownNode struct {
	Title        string
	Indent       int
	SiblingIndex int
	Children     []*markdownNode
}

type SplitMarkdownInput struct {
	ProjectID   string
	Markdown    string
	HorizonDays int
	MaxTasks    int
}

func (s *Service) SplitProjectMarkdown(ctx context.Context, in SplitMarkdownInput) (map[string]interface{}, error) {
	item, err := s.ProjectStore.GetProject(ctx, strings.TrimSpace(in.ProjectID))
	if err != nil {
		return nil, err
	}
	markdown := strings.TrimSpace(in.Markdown)
	if markdown == "" {
		return nil, fmt.Errorf("请传入 --file 或 --markdown")
	}
	root, stats, warnings := parseMarkdownTaskTree(markdown)
	horizonDays := normalizeHorizon(in.HorizonDays, item.HorizonDays)
	maxTasks := normalizeMarkdownMaxTasks(in.MaxTasks)
	tasks, phases, buildWarnings := buildPlanTasksFromMarkdown(root, item.ID, horizonDays, maxTasks)
	warnings = append(warnings, buildWarnings...)
	stats.LeafTasks = countLeafNodes(root)
	if stats.LeafTasks == 0 {
		return nil, fmt.Errorf("no valid markdown task leaf nodes found")
	}
	if len(tasks) < stats.LeafTasks {
		warnings = append(warnings, fmt.Sprintf("leaf tasks truncated by max_tasks=%d", maxTasks))
	}
	suggestion := &project.PlanSuggestion{
		PlanID:       generatePlanID(),
		ProjectID:    item.ID,
		GoalText:     item.GoalText,
		GoalType:     item.GoalType,
		Status:       project.StatusSplitSuggested,
		TasksPreview: tasks,
		Phases:       phases,
		Confidence:   0.9,
		Warnings:     warnings,
		CreatedAt:    time.Now(),
	}
	if err := s.ProjectStore.SavePlan(ctx, suggestion); err != nil {
		return nil, err
	}
	item.Status = project.StatusSplitSuggested
	item.LatestPlanID = suggestion.PlanID
	item.HorizonDays = horizonDays
	if err := s.ProjectStore.SaveProject(ctx, item); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"project_id":    item.ID,
		"plan_id":       suggestion.PlanID,
		"status":        suggestion.Status,
		"confidence":    suggestion.Confidence,
		"tasks_preview": suggestion.TasksPreview,
		"phases":        suggestion.Phases,
		"warnings":      suggestion.Warnings,
		"stats":         stats,
	}, nil
}

func ReadMarkdownInput(filePath, inline string) (string, error) {
	markdown := strings.TrimSpace(inline)
	if filePath != "" {
		bytes, err := os.ReadFile(filePath)
		if err != nil {
			return "", err
		}
		markdown = string(bytes)
	}
	return markdown, nil
}

func parseMarkdownTaskTree(markdown string) (*markdownNode, MarkdownParseStats, []string) {
	root := &markdownNode{Title: "__root__", Indent: -1}
	stack := []*markdownNode{root}
	stats := MarkdownParseStats{}
	warnings := make([]string, 0)
	previousIndent := -1

	for i, raw := range strings.Split(markdown, "\n") {
		line := strings.ReplaceAll(raw, "\t", "  ")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			stats.IgnoredLines++
			continue
		}
		indent := countLeadingSpaces(line)
		title, ok := parseMarkdownListTitle(strings.TrimSpace(line[indent:]))
		if !ok {
			stats.IgnoredLines++
			warnings = append(warnings, fmt.Sprintf("line %d ignored: not a markdown list item", i+1))
			continue
		}
		if previousIndent >= 0 && indent > previousIndent+2 {
			warnings = append(warnings, fmt.Sprintf("line %d indent jump from %d to %d; attached to nearest parent", i+1, previousIndent, indent))
		}
		previousIndent = indent
		for len(stack) > 1 && indent <= stack[len(stack)-1].Indent {
			stack = stack[:len(stack)-1]
		}
		parent := stack[len(stack)-1]
		node := &markdownNode{Title: title, Indent: indent, SiblingIndex: len(parent.Children) + 1}
		parent.Children = append(parent.Children, node)
		stack = append(stack, node)
		stats.TotalNodes++
	}
	return root, stats, warnings
}

func buildPlanTasksFromMarkdown(root *markdownNode, projectID string, horizonDays, maxTasks int) ([]project.PlanTask, []string, []string) {
	tasks := make([]project.PlanTask, 0)
	phases := make([]string, 0)
	warnings := make([]string, 0)
	phaseSeen := make(map[string]bool)
	for _, child := range root.Children {
		phase := child.Title
		collectMarkdownTasks(child, projectID, []string{child.Title}, []int{child.SiblingIndex}, phase, "", &tasks, &phases, phaseSeen)
	}
	for i := range tasks {
		tasks[i].DueOffsetDays = distributeDueOffset(i, len(tasks), horizonDays)
	}
	if len(tasks) > maxTasks {
		tasks = tasks[:maxTasks]
	}
	return tasks, phases, warnings
}

func collectMarkdownTasks(node *markdownNode, projectID string, pathTitles []string, pathIndexes []int, phase, parentPlanTaskID string, tasks *[]project.PlanTask, phases *[]string, phaseSeen map[string]bool) {
	if node == nil {
		return
	}
	if !phaseSeen[phase] {
		*phases = append(*phases, phase)
		phaseSeen[phase] = true
	}
	planTaskID := fmt.Sprintf("%s-%s", projectID, strings.ReplaceAll(strings.Join(pathTitles, "-"), " ", "-"))
	if len(node.Children) == 0 {
		*tasks = append(*tasks, project.PlanTask{ID: planTaskID, ParentID: parentPlanTaskID, Title: node.Title, Description: fmt.Sprintf("由 Markdown 任务树导入：%s", strings.Join(pathTitles, " > ")), EstimateMinutes: clampMarkdownEstimate(len(node.Children)), Priority: clampMarkdownPriority(len(pathTitles)), Quadrant: 2, Tags: []string{"markdown-import"}, Phase: phase})
		return
	}
	*tasks = append(*tasks, project.PlanTask{ID: planTaskID, ParentID: parentPlanTaskID, Title: node.Title, Description: fmt.Sprintf("由 Markdown 任务树导入：%s", strings.Join(pathTitles, " > ")), EstimateMinutes: clampMarkdownEstimate(len(node.Children)), Priority: clampMarkdownPriority(len(pathTitles)), Quadrant: 2, Tags: []string{"markdown-import"}, Phase: phase})
	for _, child := range node.Children {
		collectMarkdownTasks(child, projectID, append(append([]string{}, pathTitles...), child.Title), append(append([]int{}, pathIndexes...), child.SiblingIndex), phase, planTaskID, tasks, phases, phaseSeen)
	}
}

func countLeafNodes(root *markdownNode) int {
	if root == nil {
		return 0
	}
	if len(root.Children) == 0 {
		return 1
	}
	total := 0
	for _, child := range root.Children {
		total += countLeafNodes(child)
	}
	return total
}

func countLeadingSpaces(line string) int {
	count := 0
	for _, r := range line {
		if r != ' ' {
			break
		}
		count++
	}
	return count
}
func parseMarkdownListTitle(value string) (string, bool) {
	if matches := unorderedListItemPattern.FindStringSubmatch(value); len(matches) == 2 {
		return normalizeMarkdownTitle(matches[1]), true
	}
	if matches := orderedListItemPattern.FindStringSubmatch(value); len(matches) == 2 {
		return normalizeMarkdownTitle(matches[1]), true
	}
	return "", false
}
func normalizeMarkdownTitle(title string) string {
	out := strings.TrimSpace(title)
	for {
		next := strings.TrimSpace(orderedTitlePrefix.ReplaceAllString(out, ""))
		if next == out {
			break
		}
		out = next
	}
	out = sanitizeMarkdownText(out)
	return out
}
func sanitizeMarkdownText(text string) string {
	out := strings.TrimSpace(text)
	if out == "" {
		return out
	}
	out = markdownCheckboxPrefix.ReplaceAllString(out, "")
	out = strings.ReplaceAll(out, "**", "")
	out = strings.ReplaceAll(out, "__", "")
	out = strings.Join(strings.Fields(out), " ")
	return out
}
func normalizeMarkdownMaxTasks(value int) int {
	if value <= 0 {
		return 200
	}
	if value > 500 {
		return 500
	}
	return value
}
func distributeDueOffset(index, total, horizonDays int) int {
	if horizonDays <= 0 {
		horizonDays = 14
	}
	if total <= 1 {
		return 1
	}
	offset := int(mathRound(float64(index+1) * float64(horizonDays) / float64(total)))
	if offset < 1 {
		return 1
	}
	if offset > horizonDays {
		return horizonDays
	}
	return offset
}
func clampMarkdownEstimate(children int) int {
	switch {
	case children >= 3:
		return 120
	case children > 0:
		return 90
	default:
		return 60
	}
}
func clampMarkdownPriority(depth int) int {
	switch {
	case depth <= 1:
		return 3
	case depth == 2:
		return 2
	default:
		return 1
	}
}
func mathRound(value float64) float64 {
	if value < 0 {
		return float64(int(value - 0.5))
	}
	return float64(int(value + 0.5))
}
