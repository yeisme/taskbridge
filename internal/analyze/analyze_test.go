package analyze

import (
	"testing"
	"time"

	"github.com/yeisme/taskbridge/internal/model"
)

func sampleTask(id, title string, status model.TaskStatus, quadrant model.Quadrant, priority model.Priority) model.Task {
	return model.Task{
		ID:        id,
		Title:     title,
		Status:    status,
		Quadrant:  quadrant,
		Priority:  priority,
		DueDate:   nil,
		UpdatedAt: time.Now(),
	}
}

func TestCalculateQuadrant_EmptyTasks(t *testing.T) {
	result := CalculateQuadrant(nil)
	if result.Summary.Total != 0 {
		t.Fatalf("total = %d, want 0", result.Summary.Total)
	}
	if result.Q1.Count != 0 || result.Q2.Count != 0 {
		t.Fatal("quadrant counts should be 0 for empty input")
	}
}

func TestCalculateQuadrant_SkipsCompleted(t *testing.T) {
	tasks := []model.Task{
		sampleTask("t1", "Q1 task", model.StatusTodo, model.QuadrantUrgentImportant, model.PriorityMedium),
		sampleTask("t2", "Completed", model.StatusCompleted, model.QuadrantUrgentImportant, model.PriorityMedium),
	}
	result := CalculateQuadrant(tasks)
	if result.Q1.Count != 1 {
		t.Fatalf("Q1 count = %d, want 1 (completed should be skipped)", result.Q1.Count)
	}
	if result.Summary.Total != 2 {
		t.Fatalf("total = %d, want 2", result.Summary.Total)
	}
	if result.Summary.Active != 1 {
		t.Fatalf("active = %d, want 1", result.Summary.Active)
	}
	if result.Summary.Completed != 1 {
		t.Fatalf("completed = %d, want 1", result.Summary.Completed)
	}
}

func TestCalculateQuadrant_Percentages(t *testing.T) {
	tasks := []model.Task{
		sampleTask("t1", "Q1", model.StatusTodo, model.QuadrantUrgentImportant, model.PriorityMedium),
		sampleTask("t2", "Q2", model.StatusTodo, model.QuadrantNotUrgentImportant, model.PriorityMedium),
		sampleTask("t3", "Q1", model.StatusTodo, model.QuadrantUrgentImportant, model.PriorityMedium),
	}
	result := CalculateQuadrant(tasks)
	if result.Q1.Percentage != 66.66666666666666 {
		t.Fatalf("Q1 percentage = %f, want 66.67", result.Q1.Percentage)
	}
	if result.Q2.Percentage != 33.33333333333333 {
		t.Fatalf("Q2 percentage = %f, want 33.33", result.Q2.Percentage)
	}
}

func TestCalculatePriority_EmptyTasks(t *testing.T) {
	result := CalculatePriority(nil)
	if result.Summary.Total != 0 {
		t.Fatalf("total = %d, want 0", result.Summary.Total)
	}
}

func TestCalculatePriority_DistributionAndScore(t *testing.T) {
	tasks := []model.Task{
		sampleTask("t1", "Urgent", model.StatusTodo, model.QuadrantUrgentImportant, model.PriorityUrgent),
		sampleTask("t2", "High", model.StatusTodo, model.QuadrantUrgentImportant, model.PriorityHigh),
		sampleTask("t3", "Completed", model.StatusCompleted, model.QuadrantUrgentImportant, model.PriorityMedium),
	}
	tasks[0].PriorityScore = 80
	tasks[1].PriorityScore = 60

	result := CalculatePriority(tasks)
	if result.Urgent.Count != 1 {
		t.Fatalf("urgent count = %d, want 1", result.Urgent.Count)
	}
	if result.High.Count != 1 {
		t.Fatalf("high count = %d, want 1", result.High.Count)
	}
	if result.Summary.Active != 2 {
		t.Fatalf("active = %d, want 2", result.Summary.Active)
	}
	if result.Summary.AvgPriorityScore != 70.0 {
		t.Fatalf("avg score = %f, want 70.0", result.Summary.AvgPriorityScore)
	}
}

func TestCalculateTime_NoDueDate(t *testing.T) {
	tasks := []model.Task{
		sampleTask("t1", "No due date", model.StatusTodo, model.QuadrantUrgentImportant, model.PriorityMedium),
	}
	result := CalculateTime(tasks)
	if result.NoDueDate.Count != 1 {
		t.Fatalf("no_due_date count = %d, want 1", result.NoDueDate.Count)
	}
}

func TestCalculateTime_Overdue(t *testing.T) {
	past := time.Now().AddDate(0, 0, -5)
	tasks := []model.Task{
		sampleTask("t1", "Overdue", model.StatusTodo, model.QuadrantUrgentImportant, model.PriorityMedium),
	}
	tasks[0].DueDate = &past

	result := CalculateTime(tasks)
	if result.Overdue.Count != 1 {
		t.Fatalf("overdue count = %d, want 1", result.Overdue.Count)
	}
}

func TestCalculateTime_Today(t *testing.T) {
	now := time.Now()
	tasks := []model.Task{
		sampleTask("t1", "Today", model.StatusTodo, model.QuadrantUrgentImportant, model.PriorityMedium),
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, now.Location())
	tasks[0].DueDate = &today

	result := CalculateTime(tasks)
	if result.Today.Count != 1 {
		t.Fatalf("today count = %d, want 1", result.Today.Count)
	}
}

func TestCalculateTrend_CompletedTasks(t *testing.T) {
	now := time.Now()
	completed := now.AddDate(0, 0, -1)
	tasks := []model.Task{
		sampleTask("t1", "Done yesterday", model.StatusCompleted, model.QuadrantUrgentImportant, model.PriorityMedium),
	}
	tasks[0].CompletedAt = &completed

	result := CalculateTrend(tasks)
	if len(result.Daily) == 0 {
		t.Fatal("daily trend should have at least one point")
	}
}

func TestCalculateReport_Aggregates(t *testing.T) {
	tasks := []model.Task{
		sampleTask("t1", "Active Q1", model.StatusTodo, model.QuadrantUrgentImportant, model.PriorityUrgent),
		sampleTask("t2", "Completed", model.StatusCompleted, model.QuadrantUrgentImportant, model.PriorityMedium),
	}
	result := CalculateReport(tasks)
	if result.Total != 2 {
		t.Fatalf("total = %d, want 2", result.Total)
	}
	if result.Active != 1 {
		t.Fatalf("active = %d, want 1", result.Active)
	}
	if result.Completed != 1 {
		t.Fatalf("completed = %d, want 1", result.Completed)
	}
	if result.Quadrant.Q1.Count != 1 {
		t.Fatalf("quadrant Q1 = %d, want 1", result.Quadrant.Q1.Count)
	}
}
