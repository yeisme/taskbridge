package cmd

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/yeisme/taskbridge/internal/model"
	"github.com/yeisme/taskbridge/internal/provider"
	"github.com/yeisme/taskbridge/internal/provider/google"
	"github.com/yeisme/taskbridge/internal/storage"
	"github.com/yeisme/taskbridge/pkg/ui"
)

// tuiCmd TUI 命令
var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "交互式终端界面",
	Long: `启动交互式终端界面（TUI）查看 TaskBridge 任务。

使用键盘导航:
  ↑/k  上移      ↓/j  下移
  ←/h  左侧标签  →/l  右侧标签
  Enter 展开详情  x  完成/恢复
  d    删除(带确认) q  退出
  r    刷新      /    搜索
  1-4  按象限筛选 a    显示全部
  s    排序切换`,
	RunE: runTUI,
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}

// ViewType 视图类型
type ViewType int

const (
	ViewDashboard ViewType = iota
	ViewTasks
	ViewQuadrant
	ViewProjects
	ViewProviders
	ViewAuth
	ViewCount
)

// SortType 排序类型
type SortType int

const (
	SortByDueDate SortType = iota
	SortByPriority
	SortByCreated
	SortByTitle
	SortCount
)

// InputMode 输入模式
type InputMode int

const (
	ModeNormal InputMode = iota
	ModeSearch
	ModeDetail
	ModeConfirmDelete
)

// 样式 - 使用 pkg/ui 主题系统，保留 TUI 专用样式
var (
	selectedStyle = lipgloss.NewStyle().
			Foreground(ui.ThemePurple).
			Bold(true)

	completedStyle = lipgloss.NewStyle().
			Foreground(ui.ThemeDimText).
			Strikethrough(true)

	tabStyle = lipgloss.NewStyle().
			Foreground(ui.ThemeGray).
			Padding(0, 2)

	activeTabStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(ui.ThemePurple).
			Padding(0, 2).
			Bold(true)

	inputStyle = lipgloss.NewStyle().
			Foreground(ui.ThemePurple).
			Bold(true)

	overdueStyle = lipgloss.NewStyle().
			Foreground(ui.ThemeRed).
			Bold(true)

	detailKeyStyle = lipgloss.NewStyle().
			Foreground(ui.ThemePurple).Bold(true)

	detailValStyle = lipgloss.NewStyle().
			Foreground(ui.ThemeText)

	confirmStyle = lipgloss.NewStyle().
			Foreground(ui.ThemeRed).
			Bold(true)

	quadrantLabelStyles = map[int]lipgloss.Style{
		1: lipgloss.NewStyle().Foreground(ui.ThemeRed),
		2: lipgloss.NewStyle().Foreground(ui.ThemeGreen),
		3: lipgloss.NewStyle().Foreground(ui.ThemeOrange),
		4: lipgloss.NewStyle().Foreground(ui.ThemeGray),
	}
)

// Model TUI 模型
type Model struct {
	// 数据
	tasks      []model.Task
	taskLists  []model.TaskList
	providers  map[model.TaskSource]provider.Provider
	store      storage.Storage
	googleProv provider.Provider

	// UI 状态
	currentView   ViewType
	filtered      []model.Task
	selected      int
	quadrant      int // 0 = all, 1-4 = specific
	statusFilter  string // "" = all, "todo", "in_progress", "completed"
	width         int
	height        int
	loading       bool
	err           error
	showHelp      bool
	sortBy        SortType
	inputMode     InputMode
	inputBuffer   string
	expandedTask  *model.Task
	confirmDelete bool
}

// 初始化模型
func initialModel() Model {
	return Model{
		loading:     true,
		quadrant:    0,
		currentView: ViewDashboard,
		sortBy:      SortByDueDate,
		inputMode:   ModeNormal,
	}
}

// 消息类型
type loadMsg struct {
	tasks      []model.Task
	taskLists  []model.TaskList
	providers  map[model.TaskSource]provider.Provider
	store      storage.Storage
	googleProv provider.Provider
	err        error
}

// 加载数据
func loadData() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		store, _, err := getStore()
		if err != nil {
			return loadMsg{err: err}
		}

		tasks, err := store.ListTasks(ctx, storage.ListOptions{})
		if err != nil {
			return loadMsg{err: err}
		}

		taskLists, err := store.ListTaskLists(ctx)
		if err != nil {
			taskLists = []model.TaskList{}
		}

		providers := provider.GlobalRegistry.GetAll()

		var googleProv provider.Provider
		gp, err := google.NewProviderFromHome()
		if err == nil && gp.IsAuthenticated() {
			googleProv = gp
		}

		return loadMsg{tasks: tasks, taskLists: taskLists, providers: providers, store: store, googleProv: googleProv}
	}
}

// Init 初始化
func (m Model) Init() tea.Cmd {
	return loadData()
}

// Update 更新
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.inputMode {
		case ModeSearch:
			return m.handleSearchInput(msg)
		case ModeDetail:
			return m.handleDetailInput(msg)
		case ModeConfirmDelete:
			return m.handleConfirmDeleteInput(msg)
		default:
			return m.handleNormalInput(msg)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case loadMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.tasks = msg.tasks
			m.taskLists = msg.taskLists
			m.providers = msg.providers
			m.store = msg.store
			m.googleProv = msg.googleProv
			m.applyFilter()
		}
	}

	return m, nil
}

// handleSearchInput 处理搜索输入
func (m Model) handleSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.inputMode = ModeNormal
		m.inputBuffer = ""
		m.applyFilter()
	case "enter":
		m.inputMode = ModeNormal
		m.applyFilter()
	case "backspace":
		if len(m.inputBuffer) > 0 {
			m.inputBuffer = m.inputBuffer[:len(m.inputBuffer)-1]
			m.applyFilter()
		}
	default:
		if len(msg.String()) == 1 {
			m.inputBuffer += msg.String()
			m.applyFilter()
		}
	}
	return m, nil
}

// handleDetailInput 处理任务详情模式
func (m Model) handleDetailInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter":
		m.inputMode = ModeNormal
		m.expandedTask = nil
	case "x":
		return m.toggleComplete()
	case "d":
		if m.expandedTask != nil {
			m.inputMode = ModeConfirmDelete
			m.confirmDelete = true
		}
	}
	return m, nil
}

// handleConfirmDeleteInput 处理删除确认
func (m Model) handleConfirmDeleteInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch strings.ToLower(msg.String()) {
	case "y":
		if m.expandedTask != nil && m.store != nil {
			ctx := context.Background()
			_ = m.store.DeleteTask(ctx, m.expandedTask.ID)
			// Reload tasks
			tasks, err := m.store.ListTasks(ctx, storage.ListOptions{})
			if err == nil {
				m.tasks = tasks
			}
		}
		m.inputMode = ModeNormal
		m.expandedTask = nil
		m.confirmDelete = false
		m.applyFilter()
		if m.selected >= len(m.filtered) && m.selected > 0 {
			m.selected = len(m.filtered) - 1
		}
	case "n", "esc":
		m.inputMode = ModeDetail
		m.confirmDelete = false
	}
	return m, nil
}

// handleNormalInput 处理正常输入
func (m Model) handleNormalInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "?":
		m.showHelp = !m.showHelp
	case "r":
		m.loading = true
		return m, loadData()
	case "/":
		m.inputMode = ModeSearch
		m.inputBuffer = ""
	case "s":
		m.sortBy = (m.sortBy + 1) % SortCount
		m.applyFilter()
	case "a":
		m.quadrant = 0
		m.statusFilter = ""
		m.applyFilter()
		m.selected = 0
	case "1":
		m.quadrant = 1
		m.applyFilter()
		m.selected = 0
	case "2":
		m.quadrant = 2
		m.applyFilter()
		m.selected = 0
	case "3":
		m.quadrant = 3
		m.applyFilter()
		m.selected = 0
	case "4":
		m.quadrant = 4
		m.applyFilter()
		m.selected = 0
	case "up", "k":
		if m.selected > 0 {
			m.selected--
		}
	case "down", "j":
		maxItems := m.getMaxItems()
		if m.selected < maxItems-1 {
			m.selected++
		}
	case "left", "h":
		if m.currentView > 0 {
			m.currentView--
			m.selected = 0
		}
	case "right", "l":
		if m.currentView < ViewCount-1 {
			m.currentView++
			m.selected = 0
		}
	case "tab":
		m.currentView = (m.currentView + 1) % ViewCount
		m.selected = 0
	case "enter":
		if m.currentView == ViewTasks && len(m.filtered) > 0 && m.selected < len(m.filtered) {
			m.expandedTask = &m.filtered[m.selected]
			m.inputMode = ModeDetail
		}
	case "x":
		return m.toggleComplete()
	case "d":
		if m.currentView == ViewTasks && len(m.filtered) > 0 && m.selected < len(m.filtered) {
			m.expandedTask = &m.filtered[m.selected]
			m.inputMode = ModeConfirmDelete
			m.confirmDelete = true
		}
	}
	return m, nil
}

// toggleComplete 切换任务完成状态
func (m Model) toggleComplete() (tea.Model, tea.Cmd) {
	if m.currentView != ViewTasks || len(m.filtered) == 0 || m.selected >= len(m.filtered) || m.store == nil {
		return m, nil
	}

	task := m.filtered[m.selected]
	ctx := context.Background()

	if task.Status == model.StatusCompleted {
		task.Status = model.StatusTodo
	} else {
		task.Status = model.StatusCompleted
		now := time.Now()
		task.CompletedAt = &now
	}

	_ = m.store.SaveTask(ctx, &task)

	// Reload tasks
	tasks, err := m.store.ListTasks(ctx, storage.ListOptions{})
	if err == nil {
		m.tasks = tasks
	}
	m.applyFilter()
	if m.selected >= len(m.filtered) && m.selected > 0 {
		m.selected = len(m.filtered) - 1
	}
	return m, nil
}

// getMaxItems 获取当前视图的最大项目数
func (m *Model) getMaxItems() int {
	switch m.currentView {
	case ViewTasks, ViewQuadrant:
		return len(m.filtered)
	case ViewProviders:
		return len(m.providers)
	case ViewProjects:
		return len(m.taskLists)
	default:
		return 0
	}
}

// getSortName 获取排序名称
func (m *Model) getSortName() string {
	switch m.sortBy {
	case SortByDueDate:
		return "截止日期"
	case SortByPriority:
		return "优先级"
	case SortByCreated:
		return "创建时间"
	case SortByTitle:
		return "标题"
	default:
		return "未知"
	}
}

// applyFilter 应用筛选和排序
func (m *Model) applyFilter() {
	m.filtered = nil

	for _, t := range m.tasks {
		// 象限筛选
		if m.quadrant > 0 && int(t.Quadrant) != m.quadrant {
			continue
		}

		// 状态筛选
		if m.statusFilter != "" && string(t.Status) != m.statusFilter {
			continue
		}

		// 搜索筛选
		if m.inputMode == ModeSearch && m.inputBuffer != "" {
			if !strings.Contains(strings.ToLower(t.Title), strings.ToLower(m.inputBuffer)) {
				continue
			}
		}

		m.filtered = append(m.filtered, t)
	}

	// 排序
	sort.Slice(m.filtered, func(i, j int) bool {
		switch m.sortBy {
		case SortByDueDate:
			if m.filtered[i].DueDate == nil && m.filtered[j].DueDate == nil {
				return false
			}
			if m.filtered[i].DueDate == nil {
				return false
			}
			if m.filtered[j].DueDate == nil {
				return true
			}
			return m.filtered[i].DueDate.Before(*m.filtered[j].DueDate)
		case SortByPriority:
			return m.filtered[i].Priority < m.filtered[j].Priority
		case SortByCreated:
			return m.filtered[i].CreatedAt.After(m.filtered[j].CreatedAt)
		case SortByTitle:
			return m.filtered[i].Title < m.filtered[j].Title
		default:
			return false
		}
	})
}

// View 渲染
func (m Model) View() string {
	if m.loading {
		return "\n  ⏳ 加载中...\n"
	}

	if m.err != nil {
		return fmt.Sprintf("\n  ❌ 加载失败: %v\n", m.err)
	}

	var b strings.Builder

	// 渲染标签栏
	b.WriteString(m.renderTabs())
	b.WriteString("\n")

	// 渲染筛选栏
	b.WriteString(m.renderFilterBar())
	b.WriteString("\n")

	// 渲染搜索输入
	if m.inputMode == ModeSearch {
		b.WriteString(m.renderSearchInput())
		b.WriteString("\n")
	}

	// 渲染确认删除对话框
	if m.inputMode == ModeConfirmDelete && m.expandedTask != nil {
		b.WriteString("\n")
		b.WriteString(confirmStyle.Render(fmt.Sprintf("  ⚠ 确定要删除任务 \"%s\"？(y/n)", m.expandedTask.Title)))
		b.WriteString("\n")
		return b.String()
	}

	// 渲染任务详情
	if m.inputMode == ModeDetail && m.expandedTask != nil {
		b.WriteString(m.renderTaskDetail(m.expandedTask))
	} else {
		// 渲染当前视图内容
		switch m.currentView {
		case ViewDashboard:
			b.WriteString(m.renderDashboardView())
		case ViewTasks:
			b.WriteString(m.renderTasksView())
		case ViewQuadrant:
			b.WriteString(m.renderQuadrantView())
		case ViewProjects:
			b.WriteString(m.renderProjectsView())
		case ViewProviders:
			b.WriteString(m.renderProvidersView())
		case ViewAuth:
			b.WriteString(m.renderAuthView())
		}
	}

	// 帮助信息
	if m.showHelp {
		b.WriteString("\n")
		b.WriteString(ui.DimStyle().Render(`
快捷键:
  ↑/k  上移      ↓/j  下移
  ←/h  左标签    →/l  右标签
  Tab  切换视图  q    退出
  Enter 展开详情 x    完成/恢复
  d    删除任务  1-4  按象限
  a    显示全部  /    搜索
  r    刷新      s    切换排序
  ?    帮助(当前)
`))
	}

	// 状态栏
	b.WriteString("\n")
	b.WriteString(m.renderStatusBar())

	return b.String()
}

// renderDashboardView 渲染仪表盘视图
func (m Model) renderDashboardView() string {
	var b strings.Builder

	b.WriteString(ui.ThemeTitleStyle().Render("📊 TaskBridge 仪表盘"))
	b.WriteString("\n\n")

	// --- 今日任务 ---
	todayTasks := m.getTodayTasks(5)
	b.WriteString(ui.ThemeTitleStyle().Foreground(ui.ThemePurple).Bold(true).Render("📅 今日任务"))
	b.WriteString("\n")
	if len(todayTasks) == 0 {
		b.WriteString(ui.DimStyle().Render("  没有今日任务，休息一下吧 🎉\n"))
	} else {
		for _, t := range todayTasks {
			prioStyle := ui.PriorityStyle(int(t.Priority))
			mark := prioStyle.Render(t.Priority.Emoji())
			statusMark := "○"
			if t.Status == model.StatusCompleted {
				statusMark = completedStyle.Render("✓")
			}
			b.WriteString(fmt.Sprintf("  %s %s %s\n", statusMark, mark, t.Title))
		}
	}
	b.WriteString("\n")

	// --- 逾期任务 ---
	overdueTasks := m.getOverdueTasks()
	b.WriteString(lipgloss.NewStyle().Foreground(ui.ThemeRed).Bold(true).Render("⚠️ 逾期任务"))
	b.WriteString("\n")
	if len(overdueTasks) == 0 {
		b.WriteString(ui.DimStyle().Render("  没有逾期任务 ✨\n"))
	} else {
		for _, t := range overdueTasks {
			if t.DueDate != nil {
				b.WriteString(overdueStyle.Render(fmt.Sprintf("  ✗ %s (截止: %s)\n", t.Title, t.DueDate.Format("01-02"))))
			}
		}
	}
	b.WriteString("\n")

	// --- 四象限概览 ---
	b.WriteString(ui.ThemeTitleStyle().Foreground(ui.ThemePurple).Bold(true).Render("📈 四象限概览"))
	b.WriteString("\n")

	quadrants := []struct {
		q     model.Quadrant
		label string
		icon  string
	}{
		{model.QuadrantUrgentImportant, "Q1 紧急+重要", "🔥"},
		{model.QuadrantNotUrgentImportant, "Q2 重要", "📋"},
		{model.QuadrantUrgentNotImportant, "Q3 紧急", "⚡"},
		{model.QuadrantNotUrgentNotImportant, "Q4 其他", "🗑️"},
	}

	for _, qd := range quadrants {
		count := 0
		completed := 0
		for _, t := range m.tasks {
			if t.Quadrant == qd.q {
				count++
				if t.Status == model.StatusCompleted {
					completed++
				}
			}
		}
		pct := 0
		if count > 0 {
			pct = completed * 100 / count
		}
		qBadge := ui.QuadrantStyle(int(qd.q)).Render(fmt.Sprintf("%s %s", qd.icon, qd.label))
		bar := ui.ProgressBar(15, pct)
		b.WriteString(fmt.Sprintf("  %s %s %d/%d\n", qBadge, bar, completed, count))
	}
	b.WriteString("\n")

	// --- 同步状态 ---
	b.WriteString(ui.ThemeTitleStyle().Foreground(ui.ThemePurple).Bold(true).Render("🔌 Provider 状态"))
	b.WriteString("\n")
	if len(m.providers) == 0 {
		b.WriteString(ui.DimStyle().Render("  没有注册的 Provider\n"))
	} else {
		for name, p := range m.providers {
			var statusIcon string
			if p.IsAuthenticated() {
				statusIcon = lipgloss.NewStyle().Foreground(ui.ThemeGreen).Render("✓")
			} else {
				statusIcon = lipgloss.NewStyle().Foreground(ui.ThemeRed).Render("✗")
			}
			b.WriteString(fmt.Sprintf("  %s %s\n", statusIcon, name))
		}
	}
	b.WriteString(ui.DimStyle().Render("  按 s 同步 | Tab 切换到任务列表\n"))

	return b.String()
}

// getTodayTasks 获取今日任务 (最多 limit 条)
func (m Model) getTodayTasks(limit int) []model.Task {
	var result []model.Task
	today := time.Now().Truncate(24 * time.Hour)
	tomorrow := today.Add(24 * time.Hour)

	for _, t := range m.tasks {
		if t.Status == model.StatusCompleted {
			continue
		}
		if t.DueDate != nil && !t.DueDate.Before(today) && t.DueDate.Before(tomorrow) {
			result = append(result, t)
			if len(result) >= limit {
				break
			}
		}
	}
	return result
}

// getOverdueTasks 获取逾期未完成任务
func (m Model) getOverdueTasks() []model.Task {
	var result []model.Task
	now := time.Now()
	for _, t := range m.tasks {
		if t.Status == model.StatusCompleted {
			continue
		}
		if t.DueDate != nil && t.DueDate.Before(now) {
			result = append(result, t)
		}
	}
	return result
}

// renderTabs 渲染标签栏
func (m Model) renderTabs() string {
	tabs := []string{"仪表盘", "任务", "四象限", "项目", "Provider", "认证"}
	var renderedTabs []string

	for i, tab := range tabs {
		if i == int(m.currentView) {
			renderedTabs = append(renderedTabs, activeTabStyle.Render(tab))
		} else {
			renderedTabs = append(renderedTabs, tabStyle.Render(tab))
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)
}

// renderFilterBar 渲染筛选栏
func (m Model) renderFilterBar() string {
	var parts []string

	// 象限筛选指示
	if m.quadrant > 0 {
		qStyle := ui.QuadrantStyle(m.quadrant)
		parts = append(parts, qStyle.Render(fmt.Sprintf("Q%d", m.quadrant)))
	} else {
		parts = append(parts, ui.DimStyle().Render("Q*"))
	}

	// 状态筛选指示
	if m.statusFilter != "" {
		statusStyle := lipgloss.NewStyle().Bold(true)
		switch m.statusFilter {
		case "completed":
			statusStyle = statusStyle.Foreground(ui.ThemeGreen)
		case "in_progress":
			statusStyle = statusStyle.Foreground(ui.ThemeOrange)
		case "todo":
			statusStyle = statusStyle.Foreground(ui.ThemeGray)
		}
		parts = append(parts, statusStyle.Render(m.statusFilter))
	}

	// 排序指示
	parts = append(parts, ui.DimStyle().Render("sort:"+m.getSortName()))

	separator := ui.DimStyle().Render(" | ")
	return lipgloss.NewStyle().MarginBottom(1).Render(strings.Join(parts, separator))
}

// renderStatusBar 渲染增强状态栏
func (m Model) renderStatusBar() string {
	total := len(m.tasks)
	completed := 0
	overdue := 0
	for _, t := range m.tasks {
		if t.Status == model.StatusCompleted {
			completed++
		} else if t.DueDate != nil && t.DueDate.Before(time.Now()) {
			overdue++
		}
	}

	// 进度
	percent := 0
	if total > 0 {
		percent = completed * 100 / total
	}

	bar := ui.ProgressBar(20, percent)

	left := fmt.Sprintf(" %s | %d/%d 完成", m.getViewName(), completed, total)
	if overdue > 0 {
		left += fmt.Sprintf(" | %d 逾期", overdue)
	}
	left += " | 按 ? 查看帮助 | q 退出"

	right := bar

	// Use theme status bar style
	statusContent := lipgloss.NewStyle().
		Background(ui.ThemePurple).
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(0, 1).
		Width(m.width).
		Render(lipgloss.JoinHorizontal(lipgloss.Bottom, left, "  ", right))

	return statusContent
}

// getViewName 获取当前视图名称
func (m Model) getViewName() string {
	names := []string{"仪表盘", "任务列表", "四象限视图", "项目列表", "Provider 信息", "认证状态"}
	return names[m.currentView]
}

// renderSearchInput 渲染搜索输入
func (m Model) renderSearchInput() string {
	return inputStyle.Render(fmt.Sprintf("🔍 搜索: %s_", m.inputBuffer))
}

// renderTasksView 渲染任务视图 (使用主题卡片样式)
func (m Model) renderTasksView() string {
	var b strings.Builder

	title := "📋 任务列表"
	if m.quadrant > 0 {
		title = fmt.Sprintf("📋 象限 Q%d 任务", m.quadrant)
	}
	b.WriteString(ui.ThemeTitleStyle().Render(title))
	b.WriteString("\n")

	if len(m.filtered) == 0 {
		b.WriteString("  📭 没有找到任务\n")
		return b.String()
	}

	for i, t := range m.filtered {
		cardStyle := ui.TaskCard(string(t.Status))

		var line string
		if t.Status == model.StatusCompleted {
			line = completedStyle.Render("✓ " + t.Title)
		} else {
			priorityMark := t.Priority.Emoji()
			overdueMark := ""
			dueDateStr := ""

			if t.DueDate != nil {
				dueDateStr = ui.DimStyle().Render(fmt.Sprintf(" [%s]", t.DueDate.Format("01-02")))
				if t.DueDate.Before(time.Now()) {
					overdueMark = overdueStyle.Render(" ⚠逾期")
				}
			}

			subtaskStr := ""
			if len(t.SubtaskIDs) > 0 {
				subtaskStr = ui.DimStyle().Render(fmt.Sprintf(" [%d子任务]", len(t.SubtaskIDs)))
			}

			prioStyle := ui.PriorityStyle(int(t.Priority))
			titleText := prioStyle.Render(priorityMark) + " " + t.Title

			if i == m.selected {
				titleText = selectedStyle.Render("▶ " + titleText)
			}
			line = titleText + dueDateStr + overdueMark + subtaskStr
		}

		if i == m.selected {
			b.WriteString(cardStyle.BorderForeground(ui.ThemePurple).Render(line))
		} else {
			b.WriteString(cardStyle.Render(line))
		}
		b.WriteString("\n")
	}

	return b.String()
}

// renderTaskDetail 渲染任务详情
func (m Model) renderTaskDetail(t *model.Task) string {
	var b strings.Builder

	b.WriteString(ui.ThemeTitleStyle().Render("📋 任务详情"))
	b.WriteString("\n\n")

	cardStyle := ui.TaskCard(string(t.Status))

	var content strings.Builder
	content.WriteString(detailKeyStyle.Render("标题: ") + detailValStyle.Render(t.Title) + "\n")
	content.WriteString(detailKeyStyle.Render("状态: ") + renderStatusBadge(t.Status) + "\n")
	content.WriteString(detailKeyStyle.Render("优先级: ") + ui.PriorityStyle(int(t.Priority)).Render(fmt.Sprintf("P%d %s", t.Priority, t.Priority.Emoji())) + "\n")
	content.WriteString(detailKeyStyle.Render("象限: ") + ui.QuadrantStyle(int(t.Quadrant)).Render(fmt.Sprintf("Q%d", t.Quadrant)) + "\n")

	if t.Description != "" {
		content.WriteString(detailKeyStyle.Render("描述: ") + detailValStyle.Render(t.Description) + "\n")
	}
	if t.DueDate != nil {
		dueStr := t.DueDate.Format("2006-01-02")
		if t.DueDate.Before(time.Now()) && t.Status != model.StatusCompleted {
			dueStr = overdueStyle.Render(dueStr + " (已逾期)")
		}
		content.WriteString(detailKeyStyle.Render("截止: ") + detailValStyle.Render(dueStr) + "\n")
	}
	if t.ListName != "" {
		content.WriteString(detailKeyStyle.Render("列表: ") + detailValStyle.Render(t.ListName) + "\n")
	}
	if len(t.Tags) > 0 {
		content.WriteString(detailKeyStyle.Render("标签: ") + detailValStyle.Render(strings.Join(t.Tags, ", ")) + "\n")
	}
	if t.Progress > 0 {
		content.WriteString(detailKeyStyle.Render("进度: ") + ui.ProgressBar(30, t.Progress) + fmt.Sprintf(" %d%%", t.Progress) + "\n")
	}
	content.WriteString(detailKeyStyle.Render("来源: ") + detailValStyle.Render(string(t.Source)) + "\n")
	content.WriteString(ui.DimStyle().Render(fmt.Sprintf("创建: %s | 更新: %s", t.CreatedAt.Format("01-02 15:04"), t.UpdatedAt.Format("01-02 15:04"))) + "\n")

	b.WriteString(cardStyle.BorderForeground(ui.ThemePurple).Render(content.String()))

	b.WriteString("\n")
	b.WriteString(ui.DimStyle().Render("  Esc 返回 | x 完成/恢复 | d 删除 | ? 帮助"))

	return b.String()
}

// renderStatusBadge 渲染状态徽章
func renderStatusBadge(status model.TaskStatus) string {
	switch status {
	case model.StatusCompleted:
		return lipgloss.NewStyle().Foreground(ui.ThemeGreen).Bold(true).Render("✓ 已完成")
	case model.StatusInProgress:
		return lipgloss.NewStyle().Foreground(ui.ThemeOrange).Bold(true).Render("◉ 进行中")
	case model.StatusTodo:
		return lipgloss.NewStyle().Foreground(ui.ThemeGray).Render("○ 待办")
	default:
		return detailValStyle.Render(string(status))
	}
}

// renderQuadrantView 渲染四象限视图
func (m Model) renderQuadrantView() string {
	var b strings.Builder
	b.WriteString(ui.ThemeTitleStyle().Render("📊 四象限分析"))
	b.WriteString("\n\n")

	quadrantData := []struct {
		q     model.Quadrant
		label string
		icon  string
		desc  string
	}{
		{model.QuadrantUrgentImportant, "Q1", "🔥", "紧急且重要 (立即做)"},
		{model.QuadrantNotUrgentImportant, "Q2", "📋", "重要不紧急 (计划做)"},
		{model.QuadrantUrgentNotImportant, "Q3", "⚡", "紧急不重要 (授权做)"},
		{model.QuadrantNotUrgentNotImportant, "Q4", "🗑️", "不紧急不重要 (删除/延后)"},
	}

	for _, qd := range quadrantData {
		count := 0
		for _, t := range m.tasks {
			if t.Quadrant == qd.q {
				count++
			}
		}

		style := quadrantLabelStyles[int(qd.q)]
		b.WriteString(style.Render(fmt.Sprintf("%s %s - %s", qd.icon, qd.label, qd.desc)))
		fmt.Fprintf(&b, " [%d个任务]\n", count)
		b.WriteString(m.renderQuadrantTasks(qd.q))
		b.WriteString("\n")
	}

	return b.String()
}

// renderQuadrantTasks 渲染象限任务
func (m Model) renderQuadrantTasks(q model.Quadrant) string {
	var b strings.Builder
	count := 0
	for _, t := range m.tasks {
		if t.Quadrant == q && count < 5 {
			dueStr := ""
			if t.DueDate != nil {
				dueStr = ui.DimStyle().Render(fmt.Sprintf(" [%s]", t.DueDate.Format("01-02")))
			}
			if t.Status == model.StatusCompleted {
				b.WriteString(completedStyle.Render("  • ✓ "+t.Title) + dueStr + "\n")
			} else {
				b.WriteString("  • " + t.Title + dueStr + "\n")
			}
			count++
		}
	}
	if count == 0 {
		b.WriteString("  (暂无任务)\n")
	} else {
		remaining := 0
		for _, t := range m.tasks {
			if t.Quadrant == q {
				remaining++
			}
		}
		if remaining > 5 {
			fmt.Fprintf(&b, "  ... 还有 %d 个任务\n", remaining-5)
		}
	}
	return b.String()
}

// renderProjectsView 渲染项目视图
func (m Model) renderProjectsView() string {
	var b strings.Builder
	b.WriteString(ui.ThemeTitleStyle().Render("📁 项目列表 (任务列表)"))
	b.WriteString("\n\n")

	if len(m.taskLists) == 0 {
		b.WriteString(ui.DimStyle().Render("没有找到项目\n"))
		return b.String()
	}

	for i, list := range m.taskLists {
		prefix := "  "
		if i == m.selected {
			prefix = "▶ "
		}

		taskCount := 0
		for _, t := range m.tasks {
			if t.ListID == list.ID || t.ListName == list.Name {
				taskCount++
			}
		}

		if i == m.selected {
			b.WriteString(selectedStyle.Render(fmt.Sprintf("%s📁 %s", prefix, list.Name)))
		} else {
			fmt.Fprintf(&b, "%s📁 %s", prefix, list.Name)
		}
		b.WriteString(ui.DimStyle().Render(fmt.Sprintf(" (%d个任务)\n", taskCount)))
	}

	return b.String()
}

// renderProvidersView 渲染 Provider 视图
func (m Model) renderProvidersView() string {
	var b strings.Builder
	b.WriteString(ui.ThemeTitleStyle().Render("🔌 Provider 信息"))
	b.WriteString("\n\n")

	if len(m.providers) == 0 {
		b.WriteString(ui.DimStyle().Render("没有注册的 Provider\n"))
		return b.String()
	}

	i := 0
	for name, p := range m.providers {
		prefix := "  "
		if i == m.selected {
			prefix = "▶ "
		}

		caps := p.Capabilities()
		status := lipgloss.NewStyle().Foreground(ui.ThemeRed).Render("❌ 未认证")
		if p.IsAuthenticated() {
			status = lipgloss.NewStyle().Foreground(ui.ThemeGreen).Render("✅ 已认证")
		}

		fmt.Fprintf(&b, "%s%s - %s\n", prefix, name, status)
		fmt.Fprintf(&b, "    子任务: %v | 标签: %v | 优先级: %v\n",
			boolToCheck(caps.SupportsSubtasks),
			boolToCheck(caps.SupportsTags),
			boolToCheck(caps.SupportsPriority))
		fmt.Fprintf(&b, "    截止日期: %v | 提醒: %v | 进度: %v\n",
			boolToCheck(caps.SupportsDueDate),
			boolToCheck(caps.SupportsReminder),
			boolToCheck(caps.SupportsProgress))
		b.WriteString("\n")
		i++
	}

	return b.String()
}

// renderAuthView 渲染认证视图
func (m Model) renderAuthView() string {
	var b strings.Builder
	b.WriteString(ui.ThemeTitleStyle().Render("🔐 认证状态"))
	b.WriteString("\n\n")

	if len(m.providers) == 0 {
		b.WriteString(ui.DimStyle().Render("没有注册的 Provider\n"))
		return b.String()
	}

	i := 0
	for name, p := range m.providers {
		prefix := "  "
		if i == m.selected {
			prefix = "▶ "
		}

		if p.IsAuthenticated() {
			fmt.Fprintf(&b, "%s%s: %s\n", prefix, name, lipgloss.NewStyle().Foreground(ui.ThemeGreen).Render("✅ 已认证"))
		} else {
			fmt.Fprintf(&b, "%s%s: %s\n", prefix, name, lipgloss.NewStyle().Foreground(ui.ThemeRed).Render("❌ 未认证"))
			fmt.Fprintf(&b, "    运行 taskbridge auth %s 进行认证\n", name)
		}
		b.WriteString("\n")
		i++
	}

	b.WriteString(ui.DimStyle().Render("提示: 使用 taskbridge auth <provider> 命令进行认证\n"))

	return b.String()
}

// boolToCheck 布尔值转勾选符号
func boolToCheck(b bool) string {
	if b {
		return "✓"
	}
	return "✗"
}

func runTUI(cmd *cobra.Command, args []string) error {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return commandError("启动 TUI 失败", err)
	}
	return nil
}
