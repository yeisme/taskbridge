package clioutput

import (
	"fmt"
	"sort"
	"strings"
)

func RenderStatPanel(panel StatPanel) string {
	labelWidth, valueWidth, percentWidth := 0, 0, 0
	labels := make([]string, 0, len(panel.Rows))
	for _, row := range panel.Rows {
		label := strings.TrimSpace(strings.Join([]string{row.Icon, row.Label}, " "))
		labels = append(labels, label)
		if w := DisplayWidth(label); w > labelWidth {
			labelWidth = w
		}
		if w := DisplayWidth(row.Value); w > valueWidth {
			valueWidth = w
		}
		if w := DisplayWidth(row.Percent); w > percentWidth {
			percentWidth = w
		}
	}

	var rows []string
	for i, row := range panel.Rows {
		parts := []string{PadRight(labels[i], labelWidth)}
		if row.Value != "" || valueWidth > 0 {
			parts = append(parts, PadLeft(row.Value, valueWidth))
		}
		if row.Percent != "" || percentWidth > 0 {
			parts = append(parts, PadLeft(row.Percent, percentWidth))
		}
		if row.Hint != "" {
			parts = append(parts, row.Hint)
		}
		rows = append(rows, strings.Join(parts, "  "))
	}

	width := DisplayWidth(panel.Title)
	for _, row := range rows {
		if w := DisplayWidth(row); w > width {
			width = w
		}
	}
	if panel.Footer != "" {
		if w := DisplayWidth(panel.Footer); w > width {
			width = w
		}
	}
	if width < 42 {
		width = 42
	}

	var b strings.Builder
	if panel.Title != "" {
		b.WriteString(panel.Title)
		b.WriteString("\n\n")
	}
	b.WriteString("╭")
	b.WriteString(strings.Repeat("─", width+2))
	b.WriteString("╮\n")
	for _, row := range rows {
		b.WriteString("│ ")
		b.WriteString(PadRight(row, width))
		b.WriteString(" │\n")
	}
	b.WriteString("╰")
	b.WriteString(strings.Repeat("─", width+2))
	b.WriteString("╯\n")
	if panel.Footer != "" {
		b.WriteString("\n")
		b.WriteString(panel.Footer)
		b.WriteString("\n")
	}
	return b.String()
}

func RenderSummary(p Projection) string {
	var b strings.Builder
	b.WriteString("Status\n")
	if strings.TrimSpace(p.Summary) != "" {
		b.WriteString(p.Summary)
		b.WriteString("\n")
	} else {
		b.WriteString(string(p.Status))
		b.WriteString("\n")
	}

	if len(p.Preview) > 0 || len(p.Tables) > 0 || len(p.Stats) > 0 {
		b.WriteString("\nHighlights\n")
		for _, item := range p.Preview {
			if strings.TrimSpace(item.Value) == "" {
				continue
			}
			b.WriteString("- ")
			if strings.TrimSpace(item.Label) != "" {
				b.WriteString(item.Label)
				b.WriteString(": ")
			}
			b.WriteString(item.Value)
			b.WriteString("\n")
		}
	}

	if len(p.Facts) > 0 {
		b.WriteString("\nFacts\n")
		keys := make([]string, 0, len(p.Facts))
		for key := range p.Facts {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		rows := make([][2]string, 0, len(keys))
		for _, key := range keys {
			rows = append(rows, [2]string{key, fmt.Sprint(p.Facts[key])})
		}
		b.WriteString(renderKeyValueTable("Fact", "Value", rows))
		b.WriteString("\n")
	}

	if len(p.Risks) > 0 {
		b.WriteString("\nRisks\n")
		for _, risk := range p.Risks {
			if strings.TrimSpace(risk) == "" {
				continue
			}
			b.WriteString("- ")
			b.WriteString(risk)
			b.WriteString("\n")
		}
	}

	for _, action := range p.Actions {
		if strings.TrimSpace(action.Command) == "" {
			continue
		}
		b.WriteString("\nRecommended next step\n")
		b.WriteString(action.Command)
		b.WriteString("\n")
		break
	}
	return b.String()
}

func RenderExplain(p Projection) string {
	var b strings.Builder
	b.WriteString("Conclusion\n")
	if strings.TrimSpace(p.Summary) != "" {
		b.WriteString(p.Summary)
	} else {
		b.WriteString(string(p.Status))
	}
	b.WriteString("\n")

	if len(p.Evidence) > 0 {
		b.WriteString("\nEvidence\n")
		for _, evidence := range p.Evidence {
			if strings.TrimSpace(evidence) == "" {
				continue
			}
			b.WriteString("- ")
			b.WriteString(evidence)
			b.WriteString("\n")
		}
	}

	if p.Confidence != nil {
		b.WriteString("\nConfidence\n")
		b.WriteString(fmt.Sprintf("%.2f", *p.Confidence))
		b.WriteString("\n")
	}

	if len(p.Risks) > 0 {
		b.WriteString("\nRisks\n")
		for _, risk := range p.Risks {
			if strings.TrimSpace(risk) == "" {
				continue
			}
			b.WriteString("- ")
			b.WriteString(risk)
			b.WriteString("\n")
		}
	}

	for _, action := range p.Actions {
		if strings.TrimSpace(action.Command) == "" {
			continue
		}
		b.WriteString("\nRecommended next step\n")
		b.WriteString(action.Command)
		b.WriteString("\n")
		break
	}
	return b.String()
}
func renderKeyValueTable(leftHeader, rightHeader string, rows [][2]string) string {
	leftWidth := DisplayWidth(leftHeader)
	rightWidth := DisplayWidth(rightHeader)
	for _, row := range rows {
		if w := DisplayWidth(row[0]); w > leftWidth {
			leftWidth = w
		}
		if w := DisplayWidth(row[1]); w > rightWidth {
			rightWidth = w
		}
	}
	var b strings.Builder
	b.WriteString("┌")
	b.WriteString(strings.Repeat("─", leftWidth+2))
	b.WriteString("┬")
	b.WriteString(strings.Repeat("─", rightWidth+2))
	b.WriteString("┐\n")
	b.WriteString("│ ")
	b.WriteString(PadRight(leftHeader, leftWidth))
	b.WriteString(" │ ")
	b.WriteString(PadRight(rightHeader, rightWidth))
	b.WriteString(" │\n")
	b.WriteString("├")
	b.WriteString(strings.Repeat("─", leftWidth+2))
	b.WriteString("┼")
	b.WriteString(strings.Repeat("─", rightWidth+2))
	b.WriteString("┤\n")
	for _, row := range rows {
		b.WriteString("│ ")
		b.WriteString(PadRight(row[0], leftWidth))
		b.WriteString(" │ ")
		b.WriteString(PadRight(row[1], rightWidth))
		b.WriteString(" │\n")
	}
	b.WriteString("└")
	b.WriteString(strings.Repeat("─", leftWidth+2))
	b.WriteString("┴")
	b.WriteString(strings.Repeat("─", rightWidth+2))
	b.WriteString("┘\n")
	return b.String()
}
