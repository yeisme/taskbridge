package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/yeisme/taskbridge/internal/clioutput"
	"github.com/yeisme/taskbridge/internal/controlplane"
	"github.com/yeisme/taskbridge/internal/project"
)

type noPlanProjectStore struct{}

func (noPlanProjectStore) SaveProject(context.Context, *project.Project) error { return nil }
func (noPlanProjectStore) GetProject(context.Context, string) (*project.Project, error) {
	return nil, errors.New("not found")
}
func (noPlanProjectStore) ListProjects(context.Context, string) ([]project.Project, error) {
	return nil, nil
}
func (noPlanProjectStore) SavePlan(context.Context, *project.PlanSuggestion) error { return nil }
func (noPlanProjectStore) GetPlan(context.Context, string, string) (*project.PlanSuggestion, error) {
	return nil, errors.New("not found")
}
func (noPlanProjectStore) GetLatestPlan(context.Context, string) (*project.PlanSuggestion, error) {
	return nil, errors.New("not found")
}

func TestRenderProjectListUsesHumanTableNotJSON(t *testing.T) {
	items := []project.Project{{
		ID:          "proj-1",
		Name:        "Learn OpenClaw",
		Status:      project.StatusConfirmed,
		GoalType:    project.GoalTypeLearning,
		HorizonDays: 14,
		CreatedAt:   time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
	}}
	projection := buildProjectListProjection(context.Background(), noPlanProjectStore{}, items)

	out := renderProjectList(items, projection)

	for _, want := range []string{"Projects", "ID", "Name", "Status", "Goal", "proj-1", "Learn OpenClaw", "confirmed", "learning"} {
		if !strings.Contains(out, want) {
			t.Fatalf("project list output missing %q:\n%s", want, out)
		}
	}
	if strings.HasPrefix(strings.TrimSpace(out), "{") || strings.Contains(out, "\"latest_plan_id\"") || strings.Contains(out, "项目列表") {
		t.Fatalf("project list default should be a human table, not JSON or legacy prose:\n%s", out)
	}
}

func TestRenderDoctorResultUsesCheckTableNotJSON(t *testing.T) {
	result := controlplane.DoctorResult{
		Schema: "taskbridge.doctor.v1",
		Status: "warning",
		Checks: []controlplane.DoctorCheck{
			{ID: "storage_path", Status: "ok", Message: "storage path is writable"},
			{ID: "provider_auth", Status: "warning", Message: "no providers enabled", NextAction: "taskbridge auth status"},
		},
		NextAction: "taskbridge auth status",
	}
	out := renderDoctorResult(result, buildDoctorProjection(result))

	for _, want := range []string{"Doctor checks", "Check", "Status", "Message", "storage_path", "provider_auth", "no providers enabled", "Recommended next step"} {
		if !strings.Contains(out, want) {
			t.Fatalf("doctor output missing %q:\n%s", want, out)
		}
	}
	if strings.HasPrefix(strings.TrimSpace(out), "{") || strings.Contains(out, "\"checks\"") {
		t.Fatalf("doctor default should be a human table, not JSON:\n%s", out)
	}
}

func TestRenderControlTaskListUsesHumanTableNotJSON(t *testing.T) {
	result := &controlplane.ListResult{
		Schema: controlplane.SchemaNext,
		Status: "ok",
		Count:  1,
		Tasks: []controlplane.TaskRef{{
			ID:       "task-1",
			Title:    "Review rollout plan",
			Status:   "active",
			Source:   "local",
			Priority: 3,
			Reason:   "earliest due task",
		}},
	}
	projection := buildTaskListProjection("Suggest next steps", result)

	out := renderControlTaskList(result, projection)

	for _, want := range []string{"Next steps", "Task", "Title", "Status", "Priority", "task-1", "Review rollout plan", "earliest due task"} {
		if !strings.Contains(out, want) {
			t.Fatalf("control task list output missing %q:\n%s", want, out)
		}
	}
	if strings.HasPrefix(strings.TrimSpace(out), "{") || strings.Contains(out, "\"tasks\"") {
		t.Fatalf("control task list default should be a human table, not JSON:\n%s", out)
	}
}

func TestPrintProjectionWithLegacyJSONKeepsFormatJSONParseable(t *testing.T) {
	oldStdout := os.Stdout
	oldJSON, oldAgent := outputJSON, outputAgent
	defer func() {
		os.Stdout = oldStdout
		outputJSON, outputAgent = oldJSON, oldAgent
	}()
	outputJSON, outputAgent = false, false

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe: %v", err)
	}
	os.Stdout = w

	legacy := map[string]interface{}{"schema": "legacy.v1", "count": 1}
	projection := clioutput.New("project.list")
	err = printProjectionWithLegacyJSON("json", legacy, projection, func() {
		t.Fatal("renderText called for --format json")
	})
	w.Close()

	var buf bytes.Buffer
	if _, readErr := buf.ReadFrom(r); readErr != nil {
		t.Fatalf("ReadFrom: %v", readErr)
	}
	if err != nil {
		t.Fatalf("printProjectionWithLegacyJSON: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &parsed); err != nil {
		t.Fatalf("legacy --format json is not parseable: %v\n%s", err, buf.String())
	}
	if parsed["schema"] != "legacy.v1" || parsed["count"] != float64(1) {
		t.Fatalf("unexpected legacy JSON: %#v", parsed)
	}
}
