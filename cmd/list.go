package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/yeisme/taskbridge/internal/clioutput"
	"github.com/yeisme/taskbridge/internal/model"
	"github.com/yeisme/taskbridge/internal/provider"
	"github.com/yeisme/taskbridge/internal/sync"
	"github.com/yeisme/taskbridge/internal/taskoutput"
	"github.com/yeisme/taskbridge/internal/taskquery"
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

// listCmd list tasks command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "list tasks",
	Long: `List all tasks and support filtering by source, status, quadrant, etc.

Output format:
  - table: table format (default)
  - json: JSON format
  - markdown: Markdown format

Example:
  taskbridge list
  taskbridge list --format json
  taskbridge list --source google --status todo
  taskbridge list --quadrant 1
  taskbridge list --all`,
	RunE: runList,
}

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().StringVarP(&listSource, "source", "s", "", "Filter by source (google, microsoft, feishu, ticktick, dida, todoist)")
	listCmd.Flags().StringVarP(&listStatus, "status", "t", "", "Filter by status (todo, in_progress, completed, canceled)")
	listCmd.Flags().StringVarP(&listFormat, "format", "f", "table", "Output format (table, json, markdown, compact, tsv)")
	listCmd.Flags().IntVarP(&listQuadrant, "quadrant", "q", 0, "Filter by quadrant (1-4)")
	listCmd.Flags().IntVarP(&listPriority, "priority", "p", 0, "Filter by priority (1-4)")
	listCmd.Flags().StringVar(&listTag, "tag", "", "Filter by tag")
	listCmd.Flags().StringArrayVar(&listNames, "list", nil, "Filter by list name (can be specified repeatedly)")
	listCmd.Flags().StringArrayVar(&listIDs, "list-id", nil, "Filter by list ID (can be specified repeatedly)")
	listCmd.Flags().StringArrayVar(&listTaskIDs, "id", nil, "Filter by task ID (can be specified repeatedly)")
	listCmd.Flags().StringVar(&listQuery, "query", "", "Filter by keyword/natural language text (local match)")
	listCmd.Flags().BoolVarP(&listAll, "all", "a", false, "Show all tasks (including completed ones)")
	listCmd.Flags().BoolVar(&listSyncNow, "sync-now", false, "Before querying, synchronize the remote task to the local")
	listCmd.Flags().IntVar(&listLimit, "limit", 0, "Limit the number of returned tasks (0 means all)")
	listCmd.Flags().IntVar(&listOffset, "offset", 0, "Skip first N tasks")
	listCmd.Flags().StringVar(&listFields, "fields", "", "Select output fields (comma separated, such as: id, title, status)")
}

func runList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	statusChanged := cmd.Flags().Lookup("status") != nil && cmd.Flags().Lookup("status").Changed

	listQuery, err := taskquery.BuildListQuery(taskquery.ListOptions{
		Source:        listSource,
		Status:        listStatus,
		StatusChanged: statusChanged,
		Quadrant:      listQuadrant,
		Priority:      listPriority,
		Tag:           listTag,
		ListNames:     listNames,
		ListIDs:       listIDs,
		TaskIDs:       listTaskIDs,
		QueryText:     listQuery,
		All:           listAll,
	})
	if err != nil {
		return usageError(err.Error())
	}

	if listSyncNow {
		if err := syncNowForList(ctx, listQuery.Source); err != nil {
			return commandError("Sync failed", err)
		}
	}

	//Create storage
	store, cleanup, err := getStore()
	if err != nil {
		return commandError("Failed to create storage", err)
	}
	defer cleanup()

	//Query all matching tasks (without limit/offset, used for total statistics)
	allTasks, err := store.QueryTasks(ctx, listQuery.Query)
	if err != nil {
		return commandError("Query task failed", err)
	}
	totalCount := len(allTasks)

	//Build output context and automatically detect pipelines/AI mode
	limitChanged := cmd.Flags().Lookup("limit") != nil && cmd.Flags().Lookup("limit").Changed
	effectiveLimit := listLimit
	if !limitChanged {
		oc := taskoutput.NewOutputContext(listFormat, nil, listLimit, listOffset, false)
		effectiveLimit = oc.Limit
	}

	//Apply offset/limit slice
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
	} //Parsing the --fields parameter
	parsedFields, err := taskoutput.ParseFields(listFields)
	if err != nil {
		return usageError(err.Error())
	}

	projection := taskoutput.NewTaskBrowseProjection("task.list", tasks, parsedFields, totalCount, effectiveLimit, listOffset)
	if totalCount == 0 && listSyncNow {
		projection.Actions = nil
	}
	if globalProjectionModeRequested() {
		return printProjection(listFormat, projection, nil)
	}
	if listFormat == "json" {
		return renderListJSON(tasks, parsedFields, totalCount, effectiveLimit, listOffset)
	}
	// If there are no tasks, display a human prompt.
	if totalCount == 0 {
		fmt.Fprint(os.Stdout, clioutput.RenderSummary(projection))
		return nil
	}

	if err := taskoutput.RenderTasks(tasks, taskoutput.TaskRenderOptions{Format: listFormat, Fields: parsedFields, Writer: os.Stdout}); err != nil {
		return commandError("Output task failed", err)
	}

	//Pagination tip: when there are more results
	if page, ok := projection.Data.(taskoutput.TaskBrowsePage); ok && page.HasMore {
		fmt.Fprintf(os.Stdout, "Total %d tasks (display first %d, use --limit or --offset to page)\n", totalCount, len(tasks))
	}

	return nil
}

func renderListJSON(tasks []model.Task, fields []string, total, limit, offset int) error {
	payload := taskoutput.NewTaskBrowseProjection("task.list", tasks, fields, total, limit, offset).Data
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return commandError("Serialization task failed", err)
	}
	fmt.Fprintln(os.Stdout, string(data))
	return nil
}

func syncNowForList(ctx context.Context, source string) error {
	//When specifying a source, only this Provider will be synchronized
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
	} //When no source is specified, try to synchronize the certified Provider
	var synced int
	for _, p := range provider.GetAllProviderNames() {
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
		return fmt.Errorf("no certified Provider found to sync with")
	}
	return nil
}
