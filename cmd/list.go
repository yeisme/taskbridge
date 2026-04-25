package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/yeisme/taskbridge/internal/model"
	"github.com/yeisme/taskbridge/internal/provider"
	"github.com/yeisme/taskbridge/internal/storage"
	"github.com/yeisme/taskbridge/internal/sync"
	"github.com/yeisme/taskbridge/pkg/output"
	"github.com/yeisme/taskbridge/pkg/ui"
)

var (
	listSource   string
	listStatus   string
	listFormat   string
	listQuadrant int
	listPriority int
	listTag      string
	listNames    []string
	listIDs      []string
	listTaskIDs  []string
	listQuery    string
	listAll      bool
	listSyncNow  bool
	listLimit    int
	listOffset   int
	listFields   string
)

// listCmd 列出任务命令
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "列出任务",
	Long: `列出所有任务，支持按来源、状态、象限等条件筛选。

输出格式:
  - table: 表格格式（默认）
  - json: JSON 格式
  - markdown: Markdown 格式

示例:
  taskbridge list
  taskbridge list --format json
  taskbridge list --source google --status todo
  taskbridge list --quadrant 1
  taskbridge list --all`,
	RunE: runList,
}

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().StringVarP(&listSource, "source", "s", "", "按来源筛选（google, microsoft, feishu, ticktick, dida, todoist）")
	listCmd.Flags().StringVarP(&listStatus, "status", "t", "", "按状态筛选（todo, in_progress, completed, cancelled）")
	listCmd.Flags().StringVarP(&listFormat, "format", "f", "table", "输出格式（table, json, markdown, compact, tsv）")
	listCmd.Flags().IntVarP(&listQuadrant, "quadrant", "q", 0, "按象限筛选（1-4）")
	listCmd.Flags().IntVarP(&listPriority, "priority", "p", 0, "按优先级筛选（1-4）")
	listCmd.Flags().StringVar(&listTag, "tag", "", "按标签筛选")
	listCmd.Flags().StringArrayVar(&listNames, "list", nil, "按清单名称筛选（可重复指定）")
	listCmd.Flags().StringArrayVar(&listIDs, "list-id", nil, "按清单 ID 筛选（可重复指定）")
	listCmd.Flags().StringArrayVar(&listTaskIDs, "id", nil, "按任务 ID 筛选（可重复指定）")
	listCmd.Flags().StringVar(&listQuery, "query", "", "按关键词/自然语言文本过滤（本地匹配）")
	listCmd.Flags().BoolVarP(&listAll, "all", "a", false, "显示所有任务（包括已完成）")
	listCmd.Flags().BoolVar(&listSyncNow, "sync-now", false, "查询前先同步远程任务到本地")
	listCmd.Flags().IntVar(&listLimit, "limit", 0, "限制返回任务数量（0 表示全部）")
	listCmd.Flags().IntVar(&listOffset, "offset", 0, "跳过前 N 个任务")
	listCmd.Flags().StringVar(&listFields, "fields", "", "选择输出字段（逗号分隔，如: id,title,status）")
}

func runList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// 先解析 source（支持简写）
	resolvedSource := ""
	if listSource != "" {
		resolvedSource = provider.ResolveProviderName(listSource)
		if !provider.IsValidProvider(resolvedSource) {
			return usageError("不支持的来源: " + listSource)
		}
	}

	if listSyncNow {
		if err := syncNowForList(ctx, resolvedSource); err != nil {
			return commandError("同步失败", err)
		}
	}

	// 创建存储
	store, cleanup, err := getStore()
	defer cleanup()
	if err != nil {
		return commandError("创建存储失败", err)
	}

	// 构建查询
	query := storage.Query{}
	if resolvedSource != "" {
		query.Sources = []model.TaskSource{model.TaskSource(resolvedSource)}
	}
	query.ListIDs = sanitizeStringSlice(listIDs)
	query.ListNames = sanitizeStringSlice(listNames)
	query.TaskIDs = sanitizeStringSlice(listTaskIDs)
	query.QueryText = strings.TrimSpace(listQuery)

	statusChanged := cmd.Flags().Lookup("status") != nil && cmd.Flags().Lookup("status").Changed
	if statusChanged && listStatus != "" {
		for _, status := range splitCSV(listStatus) {
			query.Statuses = append(query.Statuses, model.TaskStatus(status))
		}
	}
	if listQuadrant > 0 && listQuadrant <= 4 {
		query.Quadrants = []model.Quadrant{model.Quadrant(listQuadrant)}
	}
	if listPriority > 0 && listPriority <= 4 {
		query.Priorities = []model.Priority{model.Priority(listPriority)}
	}
	if listTag != "" {
		query.Tags = []string{listTag}
	}
	// 当用户显式设置了 --status 时，严格尊重参数，不应用默认状态过滤。
	if !statusChanged && !listAll {
		// 默认只显示未完成任务
		query.Statuses = []model.TaskStatus{model.StatusTodo, model.StatusInProgress}
	}

	// 查询全部匹配任务（不带 limit/offset，用于统计总数）
	allTasks, err := store.QueryTasks(ctx, query)
	if err != nil {
		return commandError("查询任务失败", err)
	}
	totalCount := len(allTasks)

	// 如果没有任务，显示提示
	if totalCount == 0 {
		fmt.Println("📭 没有找到任务")
		if !listSyncNow {
			fmt.Println("💡 可尝试: taskbridge list --sync-now")
		}
		return nil
	}

	// 构建输出上下文，自动检测管道/AI 模式
	limitChanged := cmd.Flags().Lookup("limit") != nil && cmd.Flags().Lookup("limit").Changed
	effectiveLimit := listLimit
	if !limitChanged {
		oc := output.NewOutputContext(listFormat, nil, listLimit, listOffset, false)
		effectiveLimit = oc.Limit
	}

	// 应用 offset/limit 切片
	tasks := allTasks
	if listOffset > 0 {
		if listOffset >= len(tasks) {
			tasks = nil
		} else {
			tasks = tasks[listOffset:]
		}
	}
	if effectiveLimit > 0 && len(tasks) > effectiveLimit {
		tasks = tasks[:effectiveLimit]
	}

	// 解析 --fields 参数
	parsedFields, err := output.ParseFields(listFields)
	if err != nil {
		return usageError(err.Error())
	}

	// 按格式输出
	switch listFormat {
	case "json":
		printJSON(tasks, parsedFields)
	case "markdown":
		printMarkdown(tasks)
	case "compact":
		printCompact(os.Stdout, tasks, parsedFields)
	case "tsv":
		printTSV(os.Stdout, tasks, parsedFields)
	default:
		printTable(tasks)
	}

	// 分页提示：当有更多结果时
	if effectiveLimit > 0 && totalCount > effectiveLimit {
		fmt.Printf("共 %d 个任务（显示前 %d 条，使用 --limit 或 --offset 翻页）\n", totalCount, effectiveLimit)
	}

	return nil
}

// TaskOutput 任务输出格式
type TaskOutput struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Status        string   `json:"status"`
	ParentID      string   `json:"parent_id,omitempty"`
	SectionID     string   `json:"section_id,omitempty"`
	SectionName   string   `json:"section_name,omitempty"`
	SubtaskIDs    []string `json:"subtask_ids,omitempty"`
	Quadrant      string   `json:"quadrant,omitempty"`
	Priority      string   `json:"priority,omitempty"`
	DueDate       string   `json:"due_date,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	Source        string   `json:"source"`
	ListID        string   `json:"list_id,omitempty"`
	ListName      string   `json:"list_name,omitempty"`
	Progress      int      `json:"progress,omitempty"`
	PriorityScore int      `json:"priority_score,omitempty"`
}

// toOutput 转换为输出格式
func toOutput(t model.Task) TaskOutput {
	quadrantNames := map[model.Quadrant]string{
		model.QuadrantUrgentImportant:       "Q1-紧急重要",
		model.QuadrantNotUrgentImportant:    "Q2-重要不紧急",
		model.QuadrantUrgentNotImportant:    "Q3-紧急不重要",
		model.QuadrantNotUrgentNotImportant: "Q4-不紧急不重要",
	}

	priorityNames := map[model.Priority]string{
		model.PriorityUrgent: "P0-紧急",
		model.PriorityHigh:   "P1-高",
		model.PriorityMedium: "P2-中",
		model.PriorityLow:    "P3-低",
		model.PriorityNone:   "无",
	}

	var dueDate string
	if t.DueDate != nil {
		dueDate = t.DueDate.Format("2006-01-02")
	}
	parentID := ""
	if t.ParentID != nil {
		parentID = *t.ParentID
	}
	sectionID := getTaskCustomFieldString(t, "todoist_section_id")
	sectionName := getTaskCustomFieldString(t, "todoist_section_name")

	return TaskOutput{
		ID:            t.ID,
		Title:         t.Title,
		Status:        string(t.Status),
		ParentID:      parentID,
		SectionID:     sectionID,
		SectionName:   sectionName,
		SubtaskIDs:    t.SubtaskIDs,
		Quadrant:      quadrantNames[t.Quadrant],
		Priority:      priorityNames[t.Priority],
		DueDate:       dueDate,
		Tags:          t.Tags,
		Source:        string(t.Source),
		ListID:        t.ListID,
		ListName:      t.ListName,
		Progress:      t.Progress,
		PriorityScore: t.PriorityScore,
	}
}

func getTaskCustomFieldString(t model.Task, key string) string {
	if t.Metadata == nil || t.Metadata.CustomFields == nil {
		return ""
	}
	raw, ok := t.Metadata.CustomFields[key]
	if !ok {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(raw))
}

func printJSON(tasks []model.Task, fields []string) error {
	if len(fields) == 0 {
		// No field filter: output full TaskOutput
		out := make([]TaskOutput, len(tasks))
		for i, t := range tasks {
			out[i] = toOutput(t)
		}
		data, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return commandError("序列化失败", err)
		}
		fmt.Println(string(data))
		return nil

	}

	// Field filter: output only selected fields as maps
	result := make([]map[string]interface{}, len(tasks))
	for i, t := range tasks {
		m := make(map[string]interface{}, len(fields))
		for _, f := range fields {
			m[f] = jsonFieldValue(t, f)
		}
		result[i] = m
	}
	data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return commandError("序列化失败", err)
		}
		fmt.Println(string(data))
		return nil
	}

// jsonFieldValue extracts a field value for JSON output.
func jsonFieldValue(t model.Task, field string) interface{} {
	switch field {
	case "id":
		return t.ID
	case "title":
		return t.Title
	case "status":
		return string(t.Status)
	case "priority":
		return compactPriority(t.Priority)
	case "quadrant":
		return compactQuadrant(t.Quadrant)
	case "due_date":
		if t.DueDate != nil {
			return t.DueDate.Format("2006-01-02")
		}
		return nil
	case "source":
		return string(t.Source)
	case "list_name":
		return t.ListName
	case "tags":
		return t.Tags
	case "progress":
		return t.Progress
	case "description":
		return t.Description
	default:
		return nil
	}
}

func printTable(tasks []model.Task) {
	termWidth := detectTerminalWidth()
	// 固定列 + 动态标题/列表列，尽量吃满终端宽度。
	idW := 5
	statusW := 8
	quadrantW := 14
	priorityW := 7
	dueW := 10
	providerW := 10
	gapW := 2
	colCount := 8

	minTitleW := 28
	minListW := 14
	maxTitleW := 160
	maxListW := 60

	fixedW := idW + statusW + quadrantW + priorityW + dueW + providerW + (colCount-1)*gapW
	flexibleW := termWidth - fixedW
	if flexibleW < minTitleW+minListW {
		flexibleW = minTitleW + minListW
	}

	titleW := clampInt((flexibleW*2)/3, minTitleW, maxTitleW)
	listW := flexibleW - titleW
	if listW < minListW {
		deficit := minListW - listW
		titleW -= deficit
	}
	if titleW < minTitleW {
		titleW = minTitleW
	}
	listW = flexibleW - titleW
	if listW > maxListW {
		extra := listW - maxListW
		listW = maxListW
		titleW += extra
	}

	table := ui.NewSimpleTable(
		ui.Column{Header: "ID", Width: idW, AlignLeft: true},
		ui.Column{Header: "标题", Width: titleW, AlignLeft: true},
		ui.Column{Header: "状态", Width: statusW, AlignLeft: true},
		ui.Column{Header: "象限", Width: quadrantW, AlignLeft: true},
		ui.Column{Header: "优先级", Width: priorityW, AlignLeft: true},
		ui.Column{Header: "截止日期", Width: dueW, AlignLeft: true},
		ui.Column{Header: "Provider", Width: providerW, AlignLeft: true},
		ui.Column{Header: "List", Width: listW, AlignLeft: true},
	)

	fmt.Println()

	for _, t := range tasks {
		quadrant := quadrantShort(t.Quadrant)
		priority := priorityShort(t.Priority)
		status := statusShort(t.Status)
		dueDate := "-"
		if t.DueDate != nil {
			dueDate = t.DueDate.Format("01-02")
			if t.DueDate.Before(time.Now()) && t.Status != model.StatusCompleted {
				dueDate = "!" + dueDate
			}
		}

		title := truncateDisplay(t.Title, titleW)
		if t.Status == model.StatusCompleted {
			title = "✓ " + title
		}

		listName := "-"
		if t.ListName != "" {
			listName = truncateDisplay(t.ListName, listW)
		}

		table.AddRow(
			truncateDisplay(t.ID, idW),
			title,
			truncateDisplay(status, statusW),
			truncateDisplay(quadrant, quadrantW),
			truncateDisplay(priority, priorityW),
			truncateDisplay(dueDate, dueW),
			truncateDisplay(string(t.Source), providerW),
			listName,
		)
	}

	fmt.Println(table.Render())
	fmt.Println()
	fmt.Printf("共 %d 个任务\n", len(tasks))
}

func syncNowForList(ctx context.Context, source string) error {
	// 指定来源时，只同步该 Provider
	if source != "" {
		engine, err := getSyncEngineForProvider(source)
		if err != nil {
			return err
		}
		_, err = engine.Sync(ctx, sync.Options{
			Direction: sync.DirectionPull,
			Provider:  source,
		})
		return err
	}

	// 未指定来源时，尽量同步已认证 Provider
	providers := []string{"google", "microsoft", "feishu", "ticktick", "dida", "todoist"}
	var synced int
	for _, p := range providers {
		engine, err := getSyncEngineForProvider(p)
		if err != nil {
			continue
		}
		if _, err := engine.Sync(ctx, sync.Options{
			Direction: sync.DirectionPull,
			Provider:  p,
		}); err == nil {
			synced++
		}
	}
	if synced == 0 {
		return fmt.Errorf("未找到可同步的已认证 Provider")
	}
	return nil
}

func printMarkdown(tasks []model.Task) {
	fmt.Println("# 📋 任务列表")

	// 按象限分组
	quadrants := map[model.Quadrant][]model.Task{}
	for _, t := range tasks {
		quadrants[t.Quadrant] = append(quadrants[t.Quadrant], t)
	}

	quadrantNames := map[model.Quadrant]string{
		model.QuadrantUrgentImportant:       "🔥 紧急且重要 (Q1)",
		model.QuadrantNotUrgentImportant:    "📋 重要不紧急 (Q2)",
		model.QuadrantUrgentNotImportant:    "⚡ 紧急不重要 (Q3)",
		model.QuadrantNotUrgentNotImportant: "🗑️ 不紧急不重要 (Q4)",
	}

	quadrantOrder := []model.Quadrant{
		model.QuadrantUrgentImportant,
		model.QuadrantNotUrgentImportant,
		model.QuadrantUrgentNotImportant,
		model.QuadrantNotUrgentNotImportant,
	}

	for _, q := range quadrantOrder {
		qtasks := quadrants[q]
		if len(qtasks) > 0 {
			fmt.Printf("## %s\n\n", quadrantNames[q])
			for _, t := range qtasks {
				status := " "
				if t.Status == model.StatusCompleted {
					status = "x"
				}
				due := ""
				if t.DueDate != nil {
					due = fmt.Sprintf(" 📅 %s", t.DueDate.Format("2006-01-02"))
				}
				fmt.Printf("- [%s] %s%s\n", status, t.Title, due)
			}
			fmt.Println()
		}
	}

	// 统计
	fmt.Println("---")
	fmt.Printf("**总计**: %d 个任务\n", len(tasks))
}

// compactFieldMaxWidths defines max display width per field for compact output.
var compactFieldMaxWidths = map[string]int{
	"title":      40,
	"status":     12,
	"priority":   12,
	"quadrant":   12,
	"due_date":   12,
	"source":     12,
	"list_name":  12,
	"id":         12,
	"tags":       20,
	"progress":   8,
	"description": 40,
}

// defaultCompactFields is used when no --fields is specified.
var defaultCompactFields = []string{"title", "status", "priority", "quadrant", "due_date", "source", "list_name"}

// compactStatus returns English short code for task status.
func compactStatus(s model.TaskStatus) string {
	switch s {
	case model.StatusTodo:
		return "todo"
	case model.StatusInProgress:
		return "in_progress"
	case model.StatusCompleted:
		return "completed"
	case model.StatusCancelled:
		return "cancelled"
	case model.StatusDeferred:
		return "deferred"
	default:
		return string(s)
	}
}

// compactPriority returns P0/P1/P2/P3 short code.
func compactPriority(p model.Priority) string {
	switch p {
	case model.PriorityUrgent:
		return "P0"
	case model.PriorityHigh:
		return "P1"
	case model.PriorityMedium:
		return "P2"
	case model.PriorityLow:
		return "P3"
	default:
		return "-"
	}
}

// compactQuadrant returns Q1/Q2/Q3/Q4 short code.
func compactQuadrant(q model.Quadrant) string {
	switch q {
	case model.QuadrantUrgentImportant:
		return "Q1"
	case model.QuadrantNotUrgentImportant:
		return "Q2"
	case model.QuadrantUrgentNotImportant:
		return "Q3"
	case model.QuadrantNotUrgentNotImportant:
		return "Q4"
	default:
		return "-"
	}
}

// compactFieldValue extracts and formats a single field value for a task.
func compactFieldValue(t model.Task, field string) string {
	switch field {
	case "id":
		return t.ID
	case "title":
		return t.Title
	case "status":
		return compactStatus(t.Status)
	case "priority":
		return compactPriority(t.Priority)
	case "quadrant":
		return compactQuadrant(t.Quadrant)
	case "due_date":
		if t.DueDate != nil {
			return t.DueDate.Format("01-02")
		}
		return "-"
	case "source":
		return string(t.Source)
	case "list_name":
		if t.ListName != "" {
			return t.ListName
		}
		return "-"
	case "tags":
		if len(t.Tags) > 0 {
			return strings.Join(t.Tags, ",")
		}
		return "-"
	case "progress":
		return fmt.Sprintf("%d%%", t.Progress)
	case "description":
		if t.Description != "" {
			return t.Description
		}
		return "-"
	default:
		return "-"
	}
}

// printCompact outputs tasks in pipe-delimited compact format.
func printCompact(w io.Writer, tasks []model.Task, fields []string) {
	if len(fields) == 0 {
		fields = defaultCompactFields
	}

	// Header row
	headers := make([]string, len(fields))
	for i, f := range fields {
		headers[i] = truncateCompact(f, compactFieldMaxWidths[f])
	}
	fmt.Fprintln(w, strings.Join(headers, "|"))

	// Data rows
	for _, t := range tasks {
		row := make([]string, len(fields))
		for i, f := range fields {
			val := compactFieldValue(t, f)
			maxW := compactFieldMaxWidths[f]
			row[i] = truncateCompact(val, maxW)
		}
		fmt.Fprintln(w, strings.Join(row, "|"))
	}
}

// truncateCompact truncates a string to fit within maxWidth for compact display.
func truncateCompact(s string, maxWidth int) string {
	if maxWidth <= 0 || ui.StringWidth(s) <= maxWidth {
		return s
	}
	if maxWidth <= 3 {
		return strings.Repeat(".", maxWidth)
	}
	target := maxWidth - 3
	var b strings.Builder
	cur := 0
	for _, r := range s {
		rw := ui.StringWidth(string(r))
		if cur+rw > target {
			break
		}
		b.WriteRune(r)
		cur += rw
	}
	b.WriteString("...")
	return b.String()
}

// tsvDefaultFields is used when no --fields is specified for TSV output.
var tsvDefaultFields = []string{"id", "title", "status", "priority", "quadrant", "due_date", "source", "list_name"}

// escapeTSV escapes tabs and newlines in a string for TSV output.
func escapeTSV(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\t", "\\t")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	return s
}

// tsvFieldValue extracts a raw field value string for TSV output.
func tsvFieldValue(t model.Task, field string) string {
	switch field {
	case "id":
		return t.ID
	case "title":
		return t.Title
	case "status":
		return string(t.Status)
	case "priority":
		return compactPriority(t.Priority)
	case "quadrant":
		return compactQuadrant(t.Quadrant)
	case "due_date":
		if t.DueDate != nil {
			return t.DueDate.Format("2006-01-02")
		}
		return ""
	case "source":
		return string(t.Source)
	case "list_name":
		return t.ListName
	case "tags":
		if len(t.Tags) > 0 {
			return strings.Join(t.Tags, ",")
		}
		return ""
	case "progress":
		return fmt.Sprintf("%d", t.Progress)
	case "description":
		return t.Description
	default:
		return ""
	}
}

// printTSV outputs tasks in tab-separated format.
func printTSV(w io.Writer, tasks []model.Task, fields []string) {
	if len(fields) == 0 {
		fields = tsvDefaultFields
	}

	// Header row
	fmt.Fprintln(w, strings.Join(fields, "\t"))

	// Data rows
	for _, t := range tasks {
		row := make([]string, len(fields))
		for i, f := range fields {
			row[i] = escapeTSV(tsvFieldValue(t, f))
		}
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
}

func quadrantShort(q model.Quadrant) string {
	switch q {
	case model.QuadrantUrgentImportant:
		return "Q1-紧急重要"
	case model.QuadrantNotUrgentImportant:
		return "Q2-重要不紧急"
	case model.QuadrantUrgentNotImportant:
		return "Q3-紧急不重要"
	case model.QuadrantNotUrgentNotImportant:
		return "Q4-不紧急不重要"
	default:
		return "-"
	}
}

func priorityShort(p model.Priority) string {
	switch p {
	case model.PriorityUrgent:
		return "P0-紧急"
	case model.PriorityHigh:
		return "P1-高"
	case model.PriorityMedium:
		return "P2-中"
	case model.PriorityLow:
		return "P3-低"
	default:
		return "-"
	}
}

func statusShort(s model.TaskStatus) string {
	switch s {
	case model.StatusTodo:
		return "待办"
	case model.StatusInProgress:
		return "进行中"
	case model.StatusCompleted:
		return "已完成"
	case model.StatusCancelled:
		return "已取消"
	case model.StatusDeferred:
		return "已延期"
	default:
		return string(s)
	}
}

func detectTerminalWidth() int {
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 40 {
		return w
	}

	if v := os.Getenv("COLUMNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 40 {
			return n
		}
	}
	// 保守默认宽度，避免过窄换行。
	return 140
}

func truncateDisplay(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if ui.StringWidth(s) <= maxWidth {
		return s
	}
	if maxWidth <= 3 {
		return strings.Repeat(".", maxWidth)
	}

	target := maxWidth - 3
	var b strings.Builder
	cur := 0
	for _, r := range s {
		rw := ui.StringWidth(string(r))
		if cur+rw > target {
			break
		}
		b.WriteRune(r)
		cur += rw
	}
	b.WriteString("...")
	return b.String()
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func sanitizeStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		v := strings.TrimSpace(value)
		if v == "" {
			continue
		}
		key := strings.ToLower(v)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, v)
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		v := strings.TrimSpace(part)
		if v == "" {
			continue
		}
		result = append(result, v)
	}
	return result
}
