package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/yeisme/taskbridge/internal/clioutput"
	"github.com/yeisme/taskbridge/internal/model"
	"github.com/yeisme/taskbridge/internal/provider"
	"github.com/yeisme/taskbridge/internal/storage"
	"github.com/yeisme/taskbridge/internal/taskoutput"
	"github.com/yeisme/taskbridge/pkg/ui"
)

var (
	listsSource  string
	listsFormat  string
	listsSyncNow bool
)

type listSummary = taskoutput.TaskListSummary

var listsCmd = &cobra.Command{
	Use:   "lists",
	Short: "Make a to-do list",
	Long: `List locally available tasks (for easy access to list_id).

Example:
  taskbridge lists
  taskbridge lists --source ms
  taskbridge lists --sync-now --source microsoft
  taskbridge lists --format json`,
	RunE: runLists,
}

func init() {
	rootCmd.AddCommand(listsCmd)

	listsCmd.Flags().StringVarP(&listsSource, "source", "s", "", "Filter by source (abbreviations supported, such as ms/g/tick/todo)")
	listsCmd.Flags().StringVarP(&listsFormat, "format", "f", "table", "Output format (table, json)")
	listsCmd.Flags().BoolVar(&listsSyncNow, "sync-now", false, "Before querying, synchronize the remote task to the local")
}

func runLists(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	resolvedSource := ""
	if listsSource != "" {
		resolvedSource = provider.ResolveProviderName(listsSource)
		if !provider.IsValidProvider(resolvedSource) {
			return usageError("Unsupported sources:" + listsSource)
		}
	}

	if listsSyncNow {
		if err := syncNowForList(ctx, resolvedSource); err != nil {
			return commandError("Sync failed", err)
		}
	}

	store, cleanup, err := getStore()
	if err != nil {
		return commandError("Failed to create storage", err)
	}
	defer cleanup()

	lists, err := store.ListTaskLists(ctx)
	if err != nil {
		return commandError("Query list failed", err)
	}

	if resolvedSource != "" {
		filtered := make([]model.TaskList, 0, len(lists))
		for _, list := range lists {
			if list.Source == model.TaskSource(resolvedSource) {
				filtered = append(filtered, list)
			}
		}
		lists = filtered
	}

	taskCounts := map[string]int{}
	if len(lists) > 0 {
		var err error
		taskCounts, err = buildTaskCountByList(ctx, store, resolvedSource)
		if err != nil {
			return commandError("Failed to count the number of tasks", err)
		}
	}

	summaries := make([]listSummary, 0, len(lists))
	for _, list := range lists {
		summaries = append(summaries, listSummary{
			Provider:       string(list.Source),
			ListID:         list.ID,
			ListName:       list.Name,
			TaskCountLocal: taskCounts[list.ID],
		})
	}

	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].Provider == summaries[j].Provider {
			return summaries[i].ListName < summaries[j].ListName
		}
		return summaries[i].Provider < summaries[j].Provider
	})

	projection := taskoutput.NewTaskListsProjection("task.lists", summaries)
	if len(summaries) == 0 && listsSyncNow {
		projection.Actions = nil
	}
	if globalProjectionModeRequested() {
		return printProjection(listsFormat, projection, nil)
	}

	switch listsFormat {
	case "json":
		data, err := json.MarshalIndent(summaries, "", "  ")
		if err != nil {
			return commandError("Serialization failed", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
	default:
		if len(summaries) == 0 {
			fmt.Fprint(cmd.OutOrStdout(), clioutput.RenderSummary(projection))
			return nil
		}
		printListsTable(summaries)
	}
	return nil
}

func buildTaskCountByList(ctx context.Context, store storage.Storage, source string) (map[string]int, error) {
	opts := storage.ListOptions{}
	if source != "" {
		opts.Source = model.TaskSource(source)
	}

	tasks, err := store.ListTasks(ctx, opts)
	if err != nil {
		return nil, err
	}

	counts := make(map[string]int, len(tasks))
	for _, task := range tasks {
		if task.ListID == "" {
			continue
		}
		counts[task.ListID]++
	}
	return counts, nil
}

func printListsTable(lists []listSummary) {
	table := ui.NewSimpleTable(
		ui.Column{Header: "Provider", Width: 10, AlignLeft: true},
		ui.Column{Header: "ListID", Width: 30, AlignLeft: true},
		ui.Column{Header: "ListName", Width: 26, AlignLeft: true},
		ui.Column{Header: "TaskCount", Width: 9, AlignRight: true},
	)

	for _, list := range lists {
		table.AddRow(
			list.Provider,
			taskoutput.TruncateDisplay(list.ListID, 30),
			taskoutput.TruncateDisplay(list.ListName, 26),
			fmt.Sprintf("%d", list.TaskCountLocal),
		)
	}

	fmt.Println()
	fmt.Println(table.Render())
	fmt.Println()
	fmt.Printf("Total %d lists\n", len(lists))
}
