package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/yeisme/taskbridge/internal/model"
	"github.com/yeisme/taskbridge/internal/provider"
	"github.com/yeisme/taskbridge/internal/sync"
	"github.com/yeisme/taskbridge/internal/taskquery"
	"github.com/yeisme/taskbridge/pkg/output"
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

type listJSONOutput struct {
	Tasks   interface{} `json:"tasks"`
	Total   int         `json:"total"`
	Limit   int         `json:"limit"`
	Offset  int         `json:"offset"`
	HasMore bool        `json:"has_more"`
}

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
			return commandError("同步失败", err)
		}
	}

	// 创建存储
	store, cleanup, err := getStore()
	if err != nil {
		return commandError("创建存储失败", err)
	}
	defer cleanup()

	// 查询全部匹配任务（不带 limit/offset，用于统计总数）
	allTasks, err := store.QueryTasks(ctx, listQuery.Query)
	if err != nil {
		return commandError("查询任务失败", err)
	}
	totalCount := len(allTasks)

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

	if listFormat == "json" {
		return renderListJSON(tasks, parsedFields, totalCount, effectiveLimit, listOffset)
	}

	// 如果没有任务，显示提示
	if totalCount == 0 {
		fmt.Println("📭 没有找到任务")
		if !listSyncNow {
			fmt.Println("💡 可尝试: taskbridge list --sync-now")
		}
		return nil
	}

	if err := output.RenderTasks(tasks, output.TaskRenderOptions{Format: listFormat, Fields: parsedFields, Writer: os.Stdout}); err != nil {
		return commandError("输出任务失败", err)
	}

	// 分页提示：当有更多结果时
	if effectiveLimit > 0 && totalCount > effectiveLimit {
		fmt.Printf("共 %d 个任务（显示前 %d 条，使用 --limit 或 --offset 翻页）\n", totalCount, effectiveLimit)
	}

	return nil
}

func renderListJSON(tasks []model.Task, fields []string, total, limit, offset int) error {
	payload := listJSONOutput{
		Tasks:   output.TaskJSONRows(tasks, fields),
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		HasMore: offset+len(tasks) < total,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return commandError("序列化任务失败", err)
	}
	fmt.Fprintln(os.Stdout, string(data))
	return nil
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
		return fmt.Errorf("未找到可同步的已认证 Provider")
	}
	return nil
}
