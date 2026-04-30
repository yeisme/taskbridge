package projectservice

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/yeisme/taskbridge/internal/model"
	"github.com/yeisme/taskbridge/internal/project"
	"github.com/yeisme/taskbridge/internal/projectplanner"
	"github.com/yeisme/taskbridge/internal/storage"
)

type Service struct {
	TaskStore    storage.Storage
	ProjectStore project.Store
}

type CreateInput struct {
	Name        string
	Description string
	ParentID    string
	GoalText    string
	HorizonDays int
	ListID      string
	Source      string
}

type SplitInput struct {
	ProjectID          string
	GoalText           string
	HorizonDays        int
	MaxTasks           int
	AIHint             string
	RequireDeliverable bool
	MinEstimateMinutes int
	MaxEstimateMinutes int
	MinTasks           int
	ConstraintMaxTasks int
	MinPracticeTasks   int
}

type ConfirmInput struct {
	ProjectID  string
	PlanID     string
	WriteTasks bool
}

type DraftPlanInput struct {
	Name        string
	Description string
	ParentID    string
	GoalText    string
	HorizonDays int
	ListID      string
	Source      string
	MaxTasks    int
}

type DraftPlanResult struct {
	Project *project.Project
	Plan    *project.PlanSuggestion
}

func (s *Service) CreateProject(ctx context.Context, in CreateInput) (*project.Project, error) {
	goalText := strings.TrimSpace(in.GoalText)
	if goalText == "" {
		goalText = strings.TrimSpace(in.Name)
	}
	now := time.Now()
	item := &project.Project{
		ID:           generateProjectID(),
		Name:         strings.TrimSpace(in.Name),
		Description:  strings.TrimSpace(in.Description),
		ParentID:     strings.TrimSpace(in.ParentID),
		GoalText:     goalText,
		GoalType:     projectplanner.DetectGoalType(goalText),
		Status:       project.StatusDraft,
		ListID:       strings.TrimSpace(in.ListID),
		Source:       strings.TrimSpace(in.Source),
		HorizonDays:  normalizeHorizon(in.HorizonDays, 14),
		LatestPlanID: "",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.ProjectStore.SaveProject(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *Service) CreateProjectDraftPlan(ctx context.Context, in DraftPlanInput) (*DraftPlanResult, error) {
	item, err := s.CreateProject(ctx, CreateInput{
		Name:        in.Name,
		Description: in.Description,
		ParentID:    in.ParentID,
		GoalText:    in.GoalText,
		HorizonDays: in.HorizonDays,
		ListID:      in.ListID,
		Source:      in.Source,
	})
	if err != nil {
		return nil, err
	}
	plan, err := s.savePlanForProject(ctx, item, SplitInput{
		ProjectID:   item.ID,
		GoalText:    item.GoalText,
		HorizonDays: item.HorizonDays,
		MaxTasks:    in.MaxTasks,
	})
	if err != nil {
		return nil, err
	}
	return &DraftPlanResult{Project: item, Plan: plan}, nil
}

func (s *Service) ListProjects(ctx context.Context, status string) ([]project.Project, error) {
	return s.ProjectStore.ListProjects(ctx, strings.TrimSpace(status))
}

func (s *Service) SplitProject(ctx context.Context, in SplitInput) (map[string]interface{}, error) {
	item, err := s.ProjectStore.GetProject(ctx, strings.TrimSpace(in.ProjectID))
	if err != nil {
		return nil, err
	}
	suggestion, err := s.savePlanForProject(ctx, item, in)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"project_id":    item.ID,
		"plan_id":       suggestion.PlanID,
		"status":        suggestion.Status,
		"confidence":    suggestion.Confidence,
		"constraints":   suggestion.Constraints,
		"tasks_preview": suggestion.TasksPreview,
		"phases":        suggestion.Phases,
		"warnings":      suggestion.Warnings,
	}, nil
}

func (s *Service) savePlanForProject(ctx context.Context, item *project.Project, in SplitInput) (*project.PlanSuggestion, error) {
	goalText := strings.TrimSpace(in.GoalText)
	if goalText == "" {
		goalText = item.GoalText
	}
	suggestion := projectplanner.Decompose(projectplanner.DecomposeInput{
		ProjectID:   item.ID,
		ProjectName: item.Name,
		GoalText:    goalText,
		GoalType:    item.GoalType,
		HorizonDays: normalizeHorizon(in.HorizonDays, item.HorizonDays),
		MaxTasks:    in.MaxTasks,
		AIHint:      strings.TrimSpace(in.AIHint),
		Constraints: project.PlanConstraints{
			RequireDeliverable: in.RequireDeliverable,
			MinEstimateMinutes: in.MinEstimateMinutes,
			MaxEstimateMinutes: in.MaxEstimateMinutes,
			MinTasks:           in.MinTasks,
			MaxTasks:           in.ConstraintMaxTasks,
			MinPracticeTasks:   in.MinPracticeTasks,
		},
	})
	suggestion.PlanID = generatePlanID()
	if err := s.ProjectStore.SavePlan(ctx, suggestion); err != nil {
		return nil, err
	}
	item.GoalText = goalText
	item.GoalType = suggestion.GoalType
	item.Status = project.StatusSplitSuggested
	item.LatestPlanID = suggestion.PlanID
	item.HorizonDays = normalizeHorizon(in.HorizonDays, item.HorizonDays)
	if err := s.ProjectStore.SaveProject(ctx, item); err != nil {
		return nil, err
	}
	return suggestion, nil
}

func (s *Service) ConfirmProject(ctx context.Context, in ConfirmInput) (map[string]interface{}, error) {
	item, err := s.ProjectStore.GetProject(ctx, strings.TrimSpace(in.ProjectID))
	if err != nil {
		return nil, err
	}
	var plan *project.PlanSuggestion
	if strings.TrimSpace(in.PlanID) != "" {
		plan, err = s.ProjectStore.GetPlan(ctx, item.ID, strings.TrimSpace(in.PlanID))
	} else {
		plan, err = s.ProjectStore.GetLatestPlan(ctx, item.ID)
	}
	if err != nil {
		return nil, err
	}
	if in.WriteTasks && len(plan.ConfirmedTaskIDs) > 0 {
		return map[string]interface{}{"project_id": item.ID, "status": project.StatusConfirmed, "created_task_ids": plan.ConfirmedTaskIDs, "count": len(plan.ConfirmedTaskIDs)}, nil
	}
	createdTaskIDs := make([]string, 0)
	if in.WriteTasks {
		now := time.Now()
		planToLocalID := make(map[string]string, len(plan.TasksPreview))
		for idx, planTask := range plan.TasksPreview {
			task := &model.Task{
				ID:          generateTaskID(),
				Title:       planTask.Title,
				Description: planTask.Description,
				Status:      model.StatusTodo,
				CreatedAt:   now,
				UpdatedAt:   now,
				Source:      model.SourceLocal,
				ListID:      item.ListID,
				Priority:    clampTaskPriority(planTask.Priority),
				Quadrant:    clampTaskQuadrant(planTask.Quadrant),
				Tags:        append([]string{}, planTask.Tags...),
			}
			dueDate := now.AddDate(0, 0, max(1, planTask.DueOffsetDays))
			task.DueDate = &dueDate
			task.Metadata = &model.TaskMetadata{
				Version:    "1.0",
				Quadrant:   int(task.Quadrant),
				Priority:   int(task.Priority),
				LocalID:    task.ID,
				SyncSource: "local",
				CustomFields: map[string]interface{}{
					"tb_project_id": item.ID,
					"tb_plan_id":    plan.PlanID,
					"tb_goal_type":  string(item.GoalType),
					"tb_phase":      planTask.Phase,
					"tb_step_index": idx + 1,
				},
			}
			if strings.TrimSpace(planTask.ParentID) != "" {
				if parentLocalID, ok := planToLocalID[strings.TrimSpace(planTask.ParentID)]; ok {
					parentID := parentLocalID
					task.ParentID = &parentID
				}
			}
			if strings.TrimSpace(planTask.ID) != "" {
				task.Metadata.CustomFields["tb_plan_task_id"] = strings.TrimSpace(planTask.ID)
			}
			if strings.TrimSpace(planTask.ParentID) != "" {
				task.Metadata.CustomFields["tb_parent_plan_task_id"] = strings.TrimSpace(planTask.ParentID)
			}
			if err := s.TaskStore.SaveTask(ctx, task); err != nil {
				return nil, err
			}
			createdTaskIDs = append(createdTaskIDs, task.ID)
			if strings.TrimSpace(planTask.ID) != "" {
				planToLocalID[strings.TrimSpace(planTask.ID)] = task.ID
			}
		}
	}
	now := time.Now()
	plan.Status = project.StatusConfirmed
	plan.ConfirmedTaskIDs = createdTaskIDs
	plan.ConfirmedAt = &now
	if err := s.ProjectStore.SavePlan(ctx, plan); err != nil {
		return nil, err
	}
	item.Status = project.StatusConfirmed
	item.LatestPlanID = plan.PlanID
	if err := s.ProjectStore.SaveProject(ctx, item); err != nil {
		return nil, err
	}
	return map[string]interface{}{"project_id": item.ID, "status": item.Status, "created_task_ids": createdTaskIDs, "count": len(createdTaskIDs)}, nil
}

func ProjectListText(ctx context.Context, store project.Store, items []project.Project) []string {
	lines := make([]string, 0, len(items))
	if len(items) == 0 {
		return []string{"暂无项目"}
	}
	lines = append(lines, "项目列表:")
	for _, item := range items {
		summary := ""
		if plan, err := store.GetLatestPlan(ctx, item.ID); err == nil {
			summary = fmt.Sprintf(" | %d phases / %d tasks", len(plan.Phases), len(plan.TasksPreview))
		}
		lines = append(lines, fmt.Sprintf("- %s | %s | %s%s", item.ID, item.Name, item.Status, summary))
	}
	return lines
}

func normalizeHorizon(value, fallback int) int {
	if value <= 0 {
		value = fallback
	}
	if value <= 0 {
		value = 14
	}
	if value < 7 {
		return 7
	}
	if value > 30 {
		return 30
	}
	return value
}

func clampTaskPriority(priority int) model.Priority {
	if priority < 0 {
		return model.Priority(0)
	}
	if priority > 4 {
		return model.Priority(4)
	}
	return model.Priority(priority)
}
func clampTaskQuadrant(quadrant int) model.Quadrant {
	if quadrant < 1 {
		return model.QuadrantNotUrgentImportant
	}
	if quadrant > 4 {
		return model.QuadrantNotUrgentNotImportant
	}
	return model.Quadrant(quadrant)
}
func generateProjectID() string { return fmt.Sprintf("proj_%d", time.Now().UnixNano()) }
func generatePlanID() string    { return fmt.Sprintf("plan_%d", time.Now().UnixNano()) }
func generateTaskID() string    { return fmt.Sprintf("task_%d", time.Now().UnixNano()) }

func SortProjectsByCreated(items []project.Project) {
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
}
