package cmd

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/yeisme/taskbridge/internal/project"
	"github.com/yeisme/taskbridge/pkg/config"
)

func TestAgentPlanNonDryRunCreatesProjectDraftAndPlan(t *testing.T) {
	previousCfg := cfg
	previousDryRun := agentDryRun
	previousHorizon := agentHorizonDays
	previousRequestID := agentRequestID
	defer func() {
		cfg = previousCfg
		agentDryRun = previousDryRun
		agentHorizonDays = previousHorizon
		agentRequestID = previousRequestID
	}()

	tmp := t.TempDir()
	cfg = config.DefaultConfig()
	cfg.Storage.Path = tmp
	cfg.Storage.File.Format = "json"
	agentDryRun = false
	agentHorizonDays = 14
	agentRequestID = "req_test"

	output := captureStdout(t, func() {
		if err := runAgentPlan(nil, []string{"学习 OpenClaw"}); err != nil {
			t.Fatalf("runAgentPlan: %v", err)
		}
	})

	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("unmarshal output: %v\n%s", err, output)
	}
	if envelope["dry_run"] != false {
		t.Fatalf("expected dry_run=false, got %v", envelope["dry_run"])
	}
	result := envelope["result"].(map[string]interface{})
	projectID, _ := result["project_id"].(string)
	planID, _ := result["plan_id"].(string)
	if projectID == "" || planID == "" || result["created_project"] != true || result["created_plan"] != true {
		t.Fatalf("expected persisted project and plan ids, got %+v", result)
	}

	store, err := project.NewFileStore(tmp)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	item, err := store.GetProject(context.Background(), projectID)
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	if item.Status != project.StatusSplitSuggested || item.LatestPlanID != planID {
		t.Fatalf("unexpected project: %+v", item)
	}
	if _, err := store.GetPlan(context.Background(), projectID, planID); err != nil {
		t.Fatalf("GetPlan: %v", err)
	}
}
