package cmd

import (
	"fmt"
	"io"

	pkgconfig "github.com/yeisme/taskbridge/pkg/config"
	"github.com/yeisme/taskbridge/pkg/ui"
)

func writeValidationReport(out io.Writer, issues []pkgconfig.ValidationIssue) int {
	errorCount := 0
	warningCount := 0

	for _, issue := range issues {
		switch issue.Level {
		case pkgconfig.ValidationLevelError:
			errorCount++
		case pkgconfig.ValidationLevelWarning:
			warningCount++
		}
	}

	fmt.Fprintln(out, "Configuration validation")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Summary")
	summary := ui.NewSimpleTable(
		ui.Column{Header: "Level", AlignLeft: true},
		ui.Column{Header: "Count", AlignRight: true},
	)
	summary.AddRow("Errors", fmt.Sprint(errorCount))
	summary.AddRow("Warnings", fmt.Sprint(warningCount))
	fmt.Fprint(out, summary.Render())

	writeIssues := func(title string, level string) {
		fmt.Fprintln(out)
		fmt.Fprintln(out, title)
		table := ui.NewSimpleTable(
			ui.Column{Header: "Field", AlignLeft: true},
			ui.Column{Header: "Message", AlignLeft: true},
		)
		for _, issue := range issues {
			if issue.Level == level {
				table.AddRow(issue.Field, issue.Message)
			}
		}
		fmt.Fprint(out, table.Render())
	}

	if errorCount > 0 {
		writeIssues("Errors", pkgconfig.ValidationLevelError)
	}
	if warningCount > 0 {
		writeIssues("Warnings", pkgconfig.ValidationLevelWarning)
	}

	if errorCount > 0 {
		return 1
	}

	return 0
}
