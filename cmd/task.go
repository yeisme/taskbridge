package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/yeisme/taskbridge/internal/clioutput"
	"github.com/yeisme/taskbridge/internal/model"
	"github.com/yeisme/taskbridge/internal/taskoutput"
)

var (
	taskListID   string
	taskDueDate  string
	taskPriority int
	taskQuadrant int
	taskFormat   string
)

// taskCmd task management command
var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "task management",
	Long: `Manage local tasks.

Subcommands:
  add      Add task
  edit     Edit task
  delete   Delete task
  done     Complete the task
  show     Show task details
  move     Move task to another list

Examples:
  taskbridge task add "Complete report" --due 2024-01-15 --priority 3
  taskbridge task done <task-id>
  taskbridge task show <task-id>`,
}

// taskAddCmd Add task
var taskAddCmd = &cobra.Command{
	Use:   "add <title>",
	Short: "Add task",
	Long: `Add a new task to local storage.

Examples:
  taskbridge task add "Complete project report"
  taskbridge task add "Reply to email" --due 2024-01-15 --priority 3 --quadrant 1`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskAdd,
}

// taskEditCmd Edit task
var taskEditCmd = &cobra.Command{
	Use:   "edit <task-id>",
	Short: "Edit task",
	Long: `Edit an existing task.

Examples:
  taskbridge task edit <task-id> --title "New title"
  taskbridge task edit <task-id> --due 2024-01-20 --priority 2`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskEdit,
}

// taskDeleteCmd Delete task
var taskDeleteCmd = &cobra.Command{
	Use:   "delete <task-id>",
	Short: "Delete task",
	Long: `Delete the specified task.

Example:
  taskbridge task delete <task-id>
  taskbridge task delete <task-id> --force`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskDelete,
}

// taskDoneCmd Complete the task
var taskDoneCmd = &cobra.Command{
	Use:   "done <task-id>",
	Short: "Complete the task",
	Long: `Mark the task as completed.

Example:
  taskbridge task done <task-id>`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskDone,
}

// taskShowCmd Show task details
var taskShowCmd = &cobra.Command{
	Use:   "show <task-id>",
	Short: "Show task details",
	Long: `Displays detailed information for a specified task.

Example:
  taskbridge task show <task-id>
  taskbridge task show <task-id> --format json`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskShow,
}

// taskUndoCmd Undo complete
var taskUndoCmd = &cobra.Command{
	Use:   "undo <task-id>",
	Short: "Undo complete",
	Long: `Revert completed tasks to unfinished status.

Example:
  taskbridge task undo <task-id>`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskUndo,
}

func init() {
	rootCmd.AddCommand(taskCmd)
	taskCmd.AddCommand(taskAddCmd)
	taskCmd.AddCommand(taskEditCmd)
	taskCmd.AddCommand(taskDeleteCmd)
	taskCmd.AddCommand(taskDoneCmd)
	taskCmd.AddCommand(taskShowCmd)
	taskCmd.AddCommand(taskUndoCmd)

	//add command options
	taskAddCmd.Flags().StringVar(&taskListID, "list", "", "Task list ID")
	taskAddCmd.Flags().StringVar(&taskDueDate, "due", "", "Deadline (YYYY-MM-DD)")
	taskAddCmd.Flags().IntVarP(&taskPriority, "priority", "p", 0, "Priority (1-4)")
	taskAddCmd.Flags().IntVarP(&taskQuadrant, "quadrant", "q", 0, "Quadrant (1-4)")

	taskEditCmd.Flags().String("title", "", "New title")
	taskEditCmd.Flags().StringVar(&taskDueDate, "due", "", "Deadline (YYYY-MM-DD)")
	taskEditCmd.Flags().IntVarP(&taskPriority, "priority", "p", 0, "Priority (1-4)")
	taskEditCmd.Flags().IntVarP(&taskQuadrant, "quadrant", "q", 0, "Quadrant (1-4)")

	//show command options
	taskShowCmd.Flags().StringVarP(&taskFormat, "format", "f", "text", "Output format (text, json)")

	taskDeleteCmd.Flags().Bool("force", false, "Delete without confirmation")
}

func runTaskAdd(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	title := args[0]

	//Create storage
	store, cleanup, err := getStore()
	if err != nil {
		return commandError("Failed to create storage", err)
	}
	defer cleanup()

	//Create tasks
	task := &model.Task{
		ID:        generateID(),
		Title:     title,
		Status:    model.StatusTodo,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Source:    model.SourceLocal,
		ListID:    taskListID,
		Priority:  model.Priority(taskPriority),
		Quadrant:  model.Quadrant(taskQuadrant),
	} //parse deadline
	if taskDueDate != "" {
		due, err := time.Parse("2006-01-02", taskDueDate)
		if err != nil {
			return commandError("Invalid date format", err)
		}
		task.DueDate = &due
	}

	//Calculate priority score
	task.CalculatePriorityScore() //Save task
	if err := store.SaveTask(ctx, task); err != nil {
		return commandError("Failed to save task", err)
	}

	projection := buildTaskWriteReceipt("task.add", "created", *task)
	return printProjection("text", projection, func() {
		fmt.Print(renderTaskWriteReceipt(projection))
	})
}

func runTaskEdit(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	taskID := args[0]

	//Create storage
	store, cleanup, err := getStore()
	if err != nil {
		return commandError("Failed to create storage", err)
	}
	defer cleanup()

	//Get tasks
	task, err := store.GetTask(ctx, taskID)
	if err != nil {
		return commandError("Failed to get task", err)
	}

	//Update field
	if title, _ := cmd.Flags().GetString("title"); title != "" {
		task.Title = title
	}
	if taskDueDate != "" {
		due, err := time.Parse("2006-01-02", taskDueDate)
		if err != nil {
			return commandError("Invalid date format", err)
		}
		task.DueDate = &due
	}
	if taskPriority > 0 {
		task.Priority = model.Priority(taskPriority)
	}
	if taskQuadrant > 0 {
		task.Quadrant = model.Quadrant(taskQuadrant)
	}

	task.UpdatedAt = time.Now()
	task.CalculatePriorityScore()

	//Save task
	if err := store.SaveTask(ctx, task); err != nil {
		return commandError("Failed to save task", err)
	}

	projection := buildTaskWriteReceipt("task.edit", "updated", *task)
	return printProjection("text", projection, func() {
		fmt.Print(renderTaskWriteReceipt(projection))
	})
}

func runTaskDelete(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	taskID := args[0]

	//Create storage
	store, cleanup, err := getStore()
	if err != nil {
		return commandError("Failed to create storage", err)
	}
	defer cleanup()

	//Check if the task exists
	task, err := store.GetTask(ctx, taskID)
	if err != nil {
		return commandError("Task does not exist", err)
	}

	//Confirm deletion
	force, _ := cmd.Flags().GetBool("force")
	if !force {
		promptWriter := taskConfirmationWriter()
		fmt.Fprintf(promptWriter, "Delete task %q? (y/N): ", task.Title)
		var confirm string
		if _, err := fmt.Scanln(&confirm); err != nil {
			fmt.Fprintln(promptWriter, "Canceled")
			return nil
		}
		if confirm != "y" && confirm != "Y" {
			fmt.Fprintln(promptWriter, "Canceled")
			return nil
		}
	}

	// Delete task
	if err := store.DeleteTask(ctx, taskID); err != nil {
		return commandError("Failed to delete task", err)
	}

	projection := buildTaskWriteReceipt("task.delete", "deleted", *task)
	return printProjection("text", projection, func() {
		fmt.Print(renderTaskWriteReceipt(projection))
	})
}

func runTaskDone(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	taskID := args[0]

	//Create storage
	store, cleanup, err := getStore()
	if err != nil {
		return commandError("Failed to create storage", err)
	}
	defer cleanup()

	//Get tasks
	task, err := store.GetTask(ctx, taskID)
	if err != nil {
		return commandError("Failed to get task", err)
	}

	//Mark complete
	now := time.Now()
	task.Status = model.StatusCompleted
	task.CompletedAt = &now
	task.UpdatedAt = now

	//Save task
	if err := store.SaveTask(ctx, task); err != nil {
		return commandError("Failed to save task", err)
	}

	projection := buildTaskWriteReceipt("task.done", "completed", *task)
	return printProjection("text", projection, func() {
		fmt.Print(renderTaskWriteReceipt(projection))
	})
}

func runTaskUndo(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	taskID := args[0]

	//Create storage
	store, cleanup, err := getStore()
	if err != nil {
		return commandError("Failed to create storage", err)
	}
	defer cleanup()

	//Get tasks
	task, err := store.GetTask(ctx, taskID)
	if err != nil {
		return commandError("Failed to get task", err)
	}

	// Undo complete
	task.Status = model.StatusTodo
	task.CompletedAt = nil
	task.UpdatedAt = time.Now()

	//Save task
	if err := store.SaveTask(ctx, task); err != nil {
		return commandError("Failed to save task", err)
	}

	projection := buildTaskWriteReceipt("task.undo", "reopened", *task)
	return printProjection("text", projection, func() {
		fmt.Print(renderTaskWriteReceipt(projection))
	})
}

func runTaskShow(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	taskID := args[0]

	//Create storage
	store, cleanup, err := getStore()
	if err != nil {
		return commandError("Failed to create storage", err)
	}
	defer cleanup()

	//Get tasks
	task, err := store.GetTask(ctx, taskID)
	if err != nil {
		return commandError("Failed to get task", err)
	}

	projection := buildTaskDetailProjection(*task)
	return printProjection(taskFormat, projection, func() {
		fmt.Print(renderTaskDetail(projection))
	})
}

func buildTaskWriteReceipt(command, operation string, task model.Task) clioutput.Projection {
	taskProjection := taskoutput.ToTaskProjection(task)
	projection := clioutput.New(command)
	projection.Summary = fmt.Sprintf("Task %s.", operation)
	projection.Facts["operation"] = operation
	projection.Facts["task_id"] = task.ID
	projection.Facts["title"] = task.Title
	projection.Facts["status"] = string(task.Status)
	projection.Facts["source"] = string(task.Source)
	projection.Data = map[string]any{
		"operation": operation,
		"task":      taskProjection,
	}
	projection.Evidence = []string{task.ID}
	return projection
}

func buildTaskDetailProjection(task model.Task) clioutput.Projection {
	taskProjection := taskoutput.ToTaskProjection(task)
	projection := clioutput.New("task.show")
	projection.Summary = "Task details loaded."
	projection.Facts["task_id"] = task.ID
	projection.Facts["title"] = task.Title
	projection.Facts["status"] = string(task.Status)
	projection.Facts["source"] = string(task.Source)
	if taskProjection.DueDate != "" {
		projection.Facts["due_date"] = taskProjection.DueDate
	}
	projection.Data = map[string]any{
		"task": taskProjection,
	}
	projection.Preview = []clioutput.PreviewItem{
		{Label: "ID", Value: taskProjection.ID},
		{Label: "Title", Value: taskProjection.Title},
		{Label: "Status", Value: taskProjection.Status},
		{Label: "Priority", Value: taskProjection.Priority},
		{Label: "Quadrant", Value: taskProjection.Quadrant},
		{Label: "Due date", Value: taskProjection.DueDate},
		{Label: "List", Value: taskProjection.ListName},
		{Label: "Tags", Value: fmt.Sprint(taskProjection.Tags)},
		{Label: "Progress", Value: taskProgressValue(taskProjection.Progress)},
		{Label: "Source", Value: taskProjection.Source},
	}
	return projection
}

func renderTaskWriteReceipt(projection clioutput.Projection) string {
	return clioutput.RenderSummary(projection)
}

func renderTaskDetail(projection clioutput.Projection) string {
	if len(projection.Preview) == 0 {
		return clioutput.RenderSummary(projection)
	}
	var b strings.Builder
	b.WriteString("Task details\n")
	for _, item := range projection.Preview {
		if item.Value == "" || item.Value == "[]" {
			continue
		}
		fmt.Fprintf(&b, "- %s: %s\n", item.Label, item.Value)
	}
	return b.String()
}

func taskProgressValue(progress int) string {
	if progress <= 0 {
		return ""
	}
	return fmt.Sprintf("%d%%", progress)
}

func taskConfirmationWriter() *os.File {
	switch resolveOutputFormat("text") {
	case "json", "agent", "ai", "events", "explain":
		return os.Stderr
	}
	if IsQuietMode() {
		return os.Stderr
	}
	return os.Stdout
}

// generateID Generate task ID
func generateID() string {
	return fmt.Sprintf("task_%d", time.Now().UnixNano())
}
