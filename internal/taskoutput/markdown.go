package taskoutput

import (
	"fmt"
	"io"
	"strings"

	"github.com/yeisme/taskbridge/internal/model"
)

// PrintTasksMarkdown renders tasks as a Markdown table.
func PrintTasksMarkdown(w io.Writer, tasks []model.Task) {
	fmt.Fprintln(w, "| ID | Title | Status | Priority | Quadrant | Due | Source | List |")
	fmt.Fprintln(w, "| --- | --- | --- | --- | --- | --- | --- | --- |")
	for _, t := range tasks {
		due := ""
		if t.DueDate != nil {
			due = t.DueDate.Format("2006-01-02")
		}
		fmt.Fprintf(
			w,
			"| %s | %s | %s | %s | %s | %s | %s | %s |\n",
			markdownCell(t.ID),
			markdownCell(t.Title),
			markdownCell(string(t.Status)),
			markdownCell(compactPriority(t.Priority)),
			markdownCell(compactQuadrant(t.Quadrant)),
			markdownCell(due),
			markdownCell(string(t.Source)),
			markdownCell(t.ListName),
		)
	}
	fmt.Fprintf(w, "\n**Total**: %d tasks\n", len(tasks))
}

func markdownCell(s string) string {
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return s
}
