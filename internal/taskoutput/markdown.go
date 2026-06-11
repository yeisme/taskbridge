package taskoutput

import (
	"fmt"
	"io"

	"github.com/yeisme/taskbridge/internal/model"
)

// PrintTasksMarkdown renders tasks as Markdown grouped by quadrant.
func PrintTasksMarkdown(w io.Writer, tasks []model.Task) {
	fmt.Fprintln(w, "# 📋 Task list")

	quadrants := map[model.Quadrant][]model.Task{}
	for _, t := range tasks {
		quadrants[t.Quadrant] = append(quadrants[t.Quadrant], t)
	}

	quadrantNames := map[model.Quadrant]string{
		model.QuadrantUrgentImportant:       "🔥 Urgent and important (Q1)",
		model.QuadrantNotUrgentImportant:    "📋 Important not urgent (Q2)",
		model.QuadrantUrgentNotImportant:    "⚡ Urgent not important (Q3)",
		model.QuadrantNotUrgentNotImportant: "🗑️ Not urgent or important (Q4)",
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
			fmt.Fprintf(w, "## %s\n\n", quadrantNames[q])
			for _, t := range qtasks {
				status := " "
				if t.Status == model.StatusCompleted {
					status = "x"
				}
				due := ""
				if t.DueDate != nil {
					due = fmt.Sprintf(" 📅 %s", t.DueDate.Format("2006-01-02"))
				}
				fmt.Fprintf(w, "- [%s] %s%s\n", status, t.Title, due)
			}
			fmt.Fprintln(w)
		}
	}

	fmt.Fprintln(w, "---")
	fmt.Fprintf(w, "**Total**: %d tasks\n", len(tasks))
}
