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

// tuiCmd TUI command
var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "interactive terminal interface",
	Long: `Open the interactive terminal interface (TUI) to browse and update TaskBridge tasks.

Keyboard navigation:
  ↑/k move up    ↓/j move down
  ←/h left tab   →/l right tab
  Enter expand details    x complete/restore
  d delete with confirmation    q quit
  r refresh      / search
  1-4 filter by quadrant    a show all
  s switch sorting`,
	RunE: runTUI,
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}

// ViewType view type
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

// SortType sorting type
type SortType int

const (
	SortByDueDate SortType = iota
	SortByPriority
	SortByCreated
	SortByTitle
	SortCount
)

// InputMode input mode
type InputMode int

const (
	ModeNormal InputMode = iota
	ModeSearch
	ModeDetail
	ModeConfirmDelete
)

// Style - using pkg/ui theme system, retaining TUI-specific styles
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

// Model TUI model
type Model struct {
	// data
	tasks      []model.Task
	taskLists  []model.TaskList
	providers  map[model.TaskSource]provider.Provider
	store      storage.Storage
	googleProv provider.Provider

	// UI state
	currentView   ViewType
	filtered      []model.Task
	selected      int
	quadrant      int    // 0 = all, 1-4 = specific
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

// Initialize model
func initialModel() Model {
	return Model{
		loading:     true,
		quadrant:    0,
		currentView: ViewDashboard,
		sortBy:      SortByDueDate,
		inputMode:   ModeNormal,
	}
}

// Message type
type loadMsg struct {
	tasks      []model.Task
	taskLists  []model.TaskList
	providers  map[model.TaskSource]provider.Provider
	store      storage.Storage
	googleProv provider.Provider
	err        error
}

// Load data
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

// Initialization
func (m Model) Init() tea.Cmd {
	return loadData()
}

// Update
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

// handleSearchInput handles search input
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

// handleDetailInput handles task details mode
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

// handleConfirmDeleteInput handles deletion confirmation
func (m Model) handleConfirmDeleteInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch strings.ToLower(msg.String()) {
	case "y":
		if m.expandedTask != nil && m.store != nil {
			ctx := context.Background()
			if err := m.store.DeleteTask(ctx, m.expandedTask.ID); err != nil {
				m.err = fmt.Errorf("failed to delete task: %w", err)
				return m, nil
			}
			// Reload tasks
			tasks, err := m.store.ListTasks(ctx, storage.ListOptions{})
			if err != nil {
				m.err = fmt.Errorf("failed to reload tasks: %w", err)
				return m, nil
			}
			m.tasks = tasks
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

// handleNormalInput handles normal input
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

// toggleComplete toggles task completion status
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

	if err := m.store.SaveTask(ctx, &task); err != nil {
		m.err = fmt.Errorf("failed to update task: %w", err)
		return m, nil
	}

	// Reload tasks
	tasks, err := m.store.ListTasks(ctx, storage.ListOptions{})
	if err != nil {
		m.err = fmt.Errorf("failed to reload tasks: %w", err)
		return m, nil
	}
	m.tasks = tasks
	m.applyFilter()
	if m.selected >= len(m.filtered) && m.selected > 0 {
		m.selected = len(m.filtered) - 1
	}
	return m, nil
}

// getMaxItems gets the maximum number of items in the current view
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

// getSortName gets the sort name
func (m *Model) getSortName() string {
	switch m.sortBy {
	case SortByDueDate:
		return "due date"
	case SortByPriority:
		return "priority"
	case SortByCreated:
		return "creation time"
	case SortByTitle:
		return "title"
	default:
		return "unknown"
	}
}

// applyFilter applies filtering and sorting
func (m *Model) applyFilter() {
	m.filtered = nil

	for _, t := range m.tasks {
		// Quadrant filter
		if m.quadrant > 0 && int(t.Quadrant) != m.quadrant {
			continue
		}
		// Status filter
		if m.statusFilter != "" && string(t.Status) != m.statusFilter {
			continue
		}
		// Search filter
		if m.inputMode == ModeSearch && m.inputBuffer != "" {
			if !strings.Contains(strings.ToLower(t.Title), strings.ToLower(m.inputBuffer)) {
				continue
			}
		}

		m.filtered = append(m.filtered, t)
	}

	// Sort
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

// View rendering
func (m Model) View() string {
	if m.loading {
		return "\n ⏳ Loading...\n"
	}

	if m.err != nil {
		return fmt.Sprintf("\n ❌ Failed to load: %v\n", m.err)
	}

	var b strings.Builder

	// Render tab bar
	b.WriteString(m.renderTabs())
	b.WriteString("\n")

	// Render filter bar
	b.WriteString(m.renderFilterBar())
	b.WriteString("\n")

	// Render search input
	if m.inputMode == ModeSearch {
		b.WriteString(m.renderSearchInput())
		b.WriteString("\n")
	}

	// Render confirm delete dialog
	if m.inputMode == ModeConfirmDelete && m.expandedTask != nil {
		b.WriteString("\n")
		b.WriteString(confirmStyle.Render(fmt.Sprintf("⚠ Are you sure you want to delete task \"%s\"? (y/n)", m.expandedTask.Title)))
		b.WriteString("\n")
		return b.String()
	}

	// Render task details
	if m.inputMode == ModeDetail && m.expandedTask != nil {
		b.WriteString(m.renderTaskDetail(m.expandedTask))
	} else {
		// Render current view content
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

	// Help information
	if m.showHelp {
		b.WriteString("\n")
		b.WriteString(ui.DimStyle().Render(`
Shortcuts:
  ↑/k move up    ↓/j move down
  ←/h left tab   →/l right tab
  Tab switch view q quit
  Enter expand details x complete/restore
  d delete task  1-4 filter by quadrant
  a show all     / search
  r refresh      s switch sorting
  ? help
`))
	}

	// Status bar
	b.WriteString("\n")
	b.WriteString(m.renderStatusBar())

	return b.String()
}

// renderDashboardView renders the Dashboard view
func (m Model) renderDashboardView() string {
	var b strings.Builder

	b.WriteString(ui.ThemeTitleStyle().Render("📊 TaskBridge Dashboard"))
	b.WriteString("\n\n")

	// --- Today's Tasks ---
	todayTasks := m.getTodayTasks(5)
	b.WriteString(ui.ThemeTitleStyle().Foreground(ui.ThemePurple).Bold(true).Render("📅 Today's Tasks"))
	b.WriteString("\n")
	if len(todayTasks) == 0 {
		b.WriteString(ui.DimStyle().Render("No tasks for today, take a break 🎉\n"))
	} else {
		for _, t := range todayTasks {
			prioStyle := ui.PriorityStyle(int(t.Priority))
			mark := prioStyle.Render(t.Priority.Emoji())
			statusMark := "○"
			if t.Status == model.StatusCompleted {
				statusMark = completedStyle.Render("✓")
			}
			fmt.Fprintf(&b, "  %s %s %s\n", statusMark, mark, t.Title)
		}
	}
	b.WriteString("\n")

	// --- Overdue Tasks ---
	overdueTasks := m.getOverdueTasks()
	b.WriteString(lipgloss.NewStyle().Foreground(ui.ThemeRed).Bold(true).Render("⚠️ Overdue Tasks"))
	b.WriteString("\n")
	if len(overdueTasks) == 0 {
		b.WriteString(ui.DimStyle().Render("No overdue tasks ✨\n"))
	} else {
		for _, t := range overdueTasks {
			if t.DueDate != nil {
				b.WriteString(overdueStyle.Render(fmt.Sprintf("✗ %s (Due: %s)\n", t.Title, t.DueDate.Format("01-02"))))
			}
		}
	}
	b.WriteString("\n")

	// --- Four Quadrants Overview ---
	b.WriteString(ui.ThemeTitleStyle().Foreground(ui.ThemePurple).Bold(true).Render("📈 Four Quadrants Overview"))
	b.WriteString("\n")

	quadrants := []struct {
		q     model.Quadrant
		label string
		icon  string
	}{
		{model.QuadrantUrgentImportant, "Q1 Urgent + Important", "🔥"},
		{model.QuadrantNotUrgentImportant, "Q2 Important", "📋"},
		{model.QuadrantUrgentNotImportant, "Q3 Urgent", "⚡"},
		{model.QuadrantNotUrgentNotImportant, "Q4 Others", "🗑️"},
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
		fmt.Fprintf(&b, "  %s %s %d/%d\n", qBadge, bar, completed, count)
	}
	b.WriteString("\n")

	// --- Provider Status ---
	b.WriteString(ui.ThemeTitleStyle().Foreground(ui.ThemePurple).Bold(true).Render("🔌 Provider Status"))
	b.WriteString("\n")
	if len(m.providers) == 0 {
		b.WriteString(ui.DimStyle().Render("No registered providers\n"))
	} else {
		for name, p := range m.providers {
			var statusIcon string
			if p.IsAuthenticated() {
				statusIcon = lipgloss.NewStyle().Foreground(ui.ThemeGreen).Render("✓")
			} else {
				statusIcon = lipgloss.NewStyle().Foreground(ui.ThemeRed).Render("✗")
			}
			fmt.Fprintf(&b, "  %s %s\n", statusIcon, name)
		}
	}
	b.WriteString(ui.DimStyle().Render("Press s to sync | Tab for task list\n"))

	return b.String()
}

// getTodayTasks gets today's tasks (up to limit items)
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

// getOverdueTasks gets overdue and incomplete tasks
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

// renderTabs renders the tab bar
func (m Model) renderTabs() string {
	tabs := []string{"Dashboard", "Tasks", "Quadrants", "Projects", "Providers", "Auth"}
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

// renderFilterBar renders the filter bar
func (m Model) renderFilterBar() string {
	var parts []string

	// Quadrant filter indicator
	if m.quadrant > 0 {
		qStyle := ui.QuadrantStyle(m.quadrant)
		parts = append(parts, qStyle.Render(fmt.Sprintf("Q%d", m.quadrant)))
	} else {
		parts = append(parts, ui.DimStyle().Render("Q*"))
	}

	// Status filter indicator
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

	// Sort indicator
	parts = append(parts, ui.DimStyle().Render("sort: "+m.getSortName()))

	separator := ui.DimStyle().Render(" | ")
	return lipgloss.NewStyle().MarginBottom(1).Render(strings.Join(parts, separator))
}

// renderStatusBar renders the enhanced status bar
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

	// Progress
	percent := 0
	if total > 0 {
		percent = completed * 100 / total
	}

	bar := ui.ProgressBar(20, percent)

	left := fmt.Sprintf("%s | %d/%d completed", m.getViewName(), completed, total)
	if overdue > 0 {
		left += fmt.Sprintf(" | %d overdue", overdue)
	}
	left += " | Press ? for help | q to exit"

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

// getViewName gets the current view name
func (m Model) getViewName() string {
	names := []string{"Dashboard", "Tasks", "Quadrants", "Projects", "Providers", "Auth"}
	return names[m.currentView]
}

// renderSearchInput renders search input
func (m Model) renderSearchInput() string {
	return inputStyle.Render(fmt.Sprintf("🔍 Search: %s_", m.inputBuffer))
}

// renderTasksView renders the tasks view (using theme card style)
func (m Model) renderTasksView() string {
	var b strings.Builder

	title := "📋 Tasks"
	if m.quadrant > 0 {
		title = fmt.Sprintf("📋 Quadrant Q%d Tasks", m.quadrant)
	}
	b.WriteString(ui.ThemeTitleStyle().Render(title))
	b.WriteString("\n")

	if len(m.filtered) == 0 {
		b.WriteString("  📭 No tasks found\n")
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
					overdueMark = overdueStyle.Render(" ⚠ Overdue")
				}
			}

			subtaskStr := ""
			if len(t.SubtaskIDs) > 0 {
				subtaskStr = ui.DimStyle().Render(fmt.Sprintf(" [%d subtasks]", len(t.SubtaskIDs)))
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

// renderTaskDetail renders task details
func (m Model) renderTaskDetail(t *model.Task) string {
	var b strings.Builder

	b.WriteString(ui.ThemeTitleStyle().Render("📋 Task Details"))
	b.WriteString("\n\n")

	cardStyle := ui.TaskCard(string(t.Status))

	var content strings.Builder
	content.WriteString(detailKeyStyle.Render("Title: ") + detailValStyle.Render(t.Title) + "\n")
	content.WriteString(detailKeyStyle.Render("Status: ") + renderStatusBadge(t.Status) + "\n")
	content.WriteString(detailKeyStyle.Render("Priority: ") + ui.PriorityStyle(int(t.Priority)).Render(fmt.Sprintf("P%d %s", t.Priority, t.Priority.Emoji())) + "\n")
	content.WriteString(detailKeyStyle.Render("Quadrant: ") + ui.QuadrantStyle(int(t.Quadrant)).Render(fmt.Sprintf("Q%d", t.Quadrant)) + "\n")

	if t.Description != "" {
		content.WriteString(detailKeyStyle.Render("Description: ") + detailValStyle.Render(t.Description) + "\n")
	}
	if t.DueDate != nil {
		dueStr := t.DueDate.Format("2006-01-02")
		if t.DueDate.Before(time.Now()) && t.Status != model.StatusCompleted {
			dueStr = overdueStyle.Render(dueStr + " (Overdue)")
		}
		content.WriteString(detailKeyStyle.Render("Due Date: ") + detailValStyle.Render(dueStr) + "\n")
	}
	if t.ListName != "" {
		content.WriteString(detailKeyStyle.Render("List: ") + detailValStyle.Render(t.ListName) + "\n")
	}
	if len(t.Tags) > 0 {
		content.WriteString(detailKeyStyle.Render("Tags: ") + detailValStyle.Render(strings.Join(t.Tags, ", ")) + "\n")
	}
	if t.Progress > 0 {
		content.WriteString(detailKeyStyle.Render("Progress: ") + ui.ProgressBar(30, t.Progress) + fmt.Sprintf(" %d%%", t.Progress) + "\n")
	}
	content.WriteString(detailKeyStyle.Render("Source: ") + detailValStyle.Render(string(t.Source)) + "\n")
	content.WriteString(ui.DimStyle().Render(fmt.Sprintf("Created: %s | Updated: %s", t.CreatedAt.Format("01-02 15:04"), t.UpdatedAt.Format("01-02 15:04"))) + "\n")

	b.WriteString(cardStyle.BorderForeground(ui.ThemePurple).Render(content.String()))

	b.WriteString("\n")
	b.WriteString(ui.DimStyle().Render("Esc Back | x Complete/Restore | d Delete | ? Help"))

	return b.String()
}

// renderStatusBadge renders status badge
func renderStatusBadge(status model.TaskStatus) string {
	switch status {
	case model.StatusCompleted:
		return lipgloss.NewStyle().Foreground(ui.ThemeGreen).Bold(true).Render("✓ Completed")
	case model.StatusInProgress:
		return lipgloss.NewStyle().Foreground(ui.ThemeOrange).Bold(true).Render("◉ In Progress")
	case model.StatusTodo:
		return lipgloss.NewStyle().Foreground(ui.ThemeGray).Render("○ Todo")
	default:
		return detailValStyle.Render(string(status))
	}
}

// renderQuadrantView renders four quadrants view
func (m Model) renderQuadrantView() string {
	var b strings.Builder
	b.WriteString(ui.ThemeTitleStyle().Render("📊 Four Quadrants Analysis"))
	b.WriteString("\n\n")

	quadrantData := []struct {
		q     model.Quadrant
		label string
		icon  string
		desc  string
	}{
		{model.QuadrantUrgentImportant, "Q1", "🔥", "Urgent & Important (Do now)"},
		{model.QuadrantNotUrgentImportant, "Q2", "📋", "Important & Not Urgent (Plan)"},
		{model.QuadrantUrgentNotImportant, "Q3", "⚡", "Urgent & Not Important (Delegate)"},
		{model.QuadrantNotUrgentNotImportant, "Q4", "🗑️", "Not Urgent & Not Important (Eliminate/Postpone)"},
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
		fmt.Fprintf(&b, " [%d tasks]\n", count)
		b.WriteString(m.renderQuadrantTasks(qd.q))
		b.WriteString("\n")
	}

	return b.String()
}

// renderQuadrantTasks renders quadrant tasks
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
		b.WriteString("  (No tasks)\n")
	} else {
		remaining := 0
		for _, t := range m.tasks {
			if t.Quadrant == q {
				remaining++
			}
		}
		if remaining > 5 {
			fmt.Fprintf(&b, "  ... and %d more tasks\n", remaining-5)
		}
	}
	return b.String()
}

// renderProjectsView renders the projects view
func (m Model) renderProjectsView() string {
	var b strings.Builder
	b.WriteString(ui.ThemeTitleStyle().Render("📁 Projects (Task Lists)"))
	b.WriteString("\n\n")

	if len(m.taskLists) == 0 {
		b.WriteString(ui.DimStyle().Render("No projects found\n"))
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
		b.WriteString(ui.DimStyle().Render(fmt.Sprintf(" (%d tasks)\n", taskCount)))
	}

	return b.String()
}

// renderProvidersView renders the providers view
func (m Model) renderProvidersView() string {
	var b strings.Builder
	b.WriteString(ui.ThemeTitleStyle().Render("🔌 Provider Information"))
	b.WriteString("\n\n")

	if len(m.providers) == 0 {
		b.WriteString(ui.DimStyle().Render("No registered providers\n"))
		return b.String()
	}

	i := 0
	for name, p := range m.providers {
		prefix := "  "
		if i == m.selected {
			prefix = "▶ "
		}

		caps := p.Capabilities()
		status := lipgloss.NewStyle().Foreground(ui.ThemeRed).Render("❌ Not Authenticated")
		if p.IsAuthenticated() {
			status = lipgloss.NewStyle().Foreground(ui.ThemeGreen).Render("✅ Authenticated")
		}

		fmt.Fprintf(&b, "%s%s - %s\n", prefix, name, status)
		fmt.Fprintf(&b, "    Subtasks: %v | Tags: %v | Priority: %v\n",
			boolToCheck(caps.SupportsSubtasks),
			boolToCheck(caps.SupportsTags),
			boolToCheck(caps.SupportsPriority))
		fmt.Fprintf(&b, "    Due Date: %v | Reminder: %v | Progress: %v\n",
			boolToCheck(caps.SupportsDueDate),
			boolToCheck(caps.SupportsReminder),
			boolToCheck(caps.SupportsProgress))
		b.WriteString("\n")
		i++
	}

	return b.String()
}

// renderAuthView renders the auth view
func (m Model) renderAuthView() string {
	var b strings.Builder
	b.WriteString(ui.ThemeTitleStyle().Render("🔐 Authentication Status"))
	b.WriteString("\n\n")

	if len(m.providers) == 0 {
		b.WriteString(ui.DimStyle().Render("No registered providers\n"))
		return b.String()
	}

	i := 0
	for name, p := range m.providers {
		prefix := "  "
		if i == m.selected {
			prefix = "▶ "
		}

		if p.IsAuthenticated() {
			fmt.Fprintf(&b, "%s%s: %s\n", prefix, name, lipgloss.NewStyle().Foreground(ui.ThemeGreen).Render("✅ Authenticated"))
		} else {
			fmt.Fprintf(&b, "%s%s: %s\n", prefix, name, lipgloss.NewStyle().Foreground(ui.ThemeRed).Render("❌ Not Authenticated"))
			fmt.Fprintf(&b, "    Run taskbridge auth %s to authenticate\n", name)
		}
		b.WriteString("\n")
		i++
	}

	b.WriteString(ui.DimStyle().Render("Tip: Use taskbridge auth <provider> command to authenticate\n"))

	return b.String()
}

// boolToCheck converts boolean to check symbol
func boolToCheck(b bool) string {
	if b {
		return "✓"
	}
	return "✗"
}

func runTUI(cmd *cobra.Command, args []string) error {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return commandError("Failed to start TUI", err)
	}
	return nil
}
